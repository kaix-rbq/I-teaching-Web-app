// Package asr 是语音识别（ASR）适配层：把本地音频文件转成带时间戳的文本。
//
// 依赖方向：service → asr（叶子包，不反向依赖 service/handler）。
// 流程（与 AImodel/audio/asr_test.py 验证过的链路一致）：
//  1. 取上传凭证，把音频传到 DashScope 临时存储，得到 oss:// 临时 URL；
//  2. 提交异步识别任务（须带 X-DashScope-Async 与 X-DashScope-OssResourceResolve）；
//  3. 轮询任务状态直至 SUCCEEDED / FAILED / CANCELED；
//  4. 下载 transcription_url 指向的 JSON 并解析为纯文本 + 句级分段。
//
// 本包不得记录 APIKey。
package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"aijiaoxue-api/internal/config"
)

// ErrEngineUnavailable 表示引擎侧不可用（未配置、鉴权失败、上游 5xx、任务失败或超时）。
// service 层据此把转录置为 failed 并映射 50003。
var ErrEngineUnavailable = errors.New("asr: 引擎不可用")

const (
	// policyURL 是公共域名的上传凭证接口（与业务空间域名无关）。
	policyURL = "https://dashscope.aliyuncs.com/api/v1/uploads"
	// publicBase 是未配置业务空间时的公共域名前缀。
	publicBase = "https://dashscope.aliyuncs.com"
)

// Segment 是一句转写结果，Start/End 单位为秒（与 dto.TranscriptSegment 对齐）。
type Segment struct {
	Start   float64
	End     float64
	Speaker string
	Text    string
}

// Result 是一次转写的完整产物。
type Result struct {
	Content    string    // 纯文本全文
	Segments   []Segment // 句级分段
	DurationMS int       // 原始音频时长（毫秒），用于回写 recordings.duration_sec
}

// Client 是 DashScope 非实时语音识别（Filetrans）HTTP 客户端。
type Client struct {
	apiKey  string
	model   string
	baseURL string
	// policyURL 是上传凭证接口地址；独立成字段便于测试注入。
	policyURL string
	diarize   bool
	timeout   time.Duration
	interval  time.Duration
	http      *http.Client
}

// NewClient 依据配置构造客户端；配置不完整时返回 nil，由调用方降级（不阻断主流程）。
func NewClient(cfg config.TranscriptionConfig) *Client {
	if !cfg.Enabled || cfg.APIKey == "" || cfg.Model == "" {
		return nil
	}
	base := cfg.BaseURL
	if base == "" {
		if cfg.WorkspaceID != "" {
			base = fmt.Sprintf("https://%s.cn-beijing.maas.aliyuncs.com", cfg.WorkspaceID)
		} else {
			base = publicBase
		}
	}
	return &Client{
		apiKey:    cfg.APIKey,
		model:     cfg.Model,
		baseURL:   base,
		policyURL: policyURL,
		diarize:   cfg.Diarization,
		timeout:   cfg.TimeoutDuration(),
		interval:  cfg.PollIntervalDuration(),
		// 单次 HTTP 交互超时；任务总时长由外层 context 控制。
		http: &http.Client{Timeout: 5 * time.Minute},
	}
}

// Timeout 暴露单任务总超时，供 service 层设置 context 期限。
func (c *Client) Timeout() time.Duration { return c.timeout }

// Transcribe 走完「上传 → 提交 → 轮询 → 下载」全流程。
func (c *Client) Transcribe(ctx context.Context, filePath string) (*Result, error) {
	ossURL, err := c.upload(ctx, filePath)
	if err != nil {
		return nil, err
	}
	taskID, err := c.submit(ctx, ossURL)
	if err != nil {
		return nil, err
	}
	transcriptionURL, err := c.poll(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return c.fetchResult(ctx, transcriptionURL)
}

// --- 步骤 1：取上传凭证并上传到 DashScope 临时存储 ---

func (c *Client) upload(ctx context.Context, filePath string) (string, error) {
	u, err := url.Parse(c.policyURL)
	if err != nil {
		return "", fmt.Errorf("%w: 解析上传凭证地址: %v", ErrEngineUnavailable, err)
	}
	q := u.Query()
	q.Set("action", "getPolicy")
	q.Set("model", c.model)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("%w: 构造上传凭证请求: %v", ErrEngineUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: 获取上传凭证: %v", ErrEngineUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// 响应体不含 Key；截断后返回便于排障。
		return "", fmt.Errorf("%w: 获取上传凭证 HTTP %d: %s", ErrEngineUnavailable, resp.StatusCode, snippet(resp.Body))
	}

	var policyResp struct {
		Data struct {
			UploadHost          string `json:"upload_host"`
			UploadDir           string `json:"upload_dir"`
			OSSAccessKeyID      string `json:"oss_access_key_id"`
			Signature           string `json:"signature"`
			Policy              string `json:"policy"`
			XOSSAcl             string `json:"x_oss_object_acl"`
			XOSSForbidOverwrite string `json:"x_oss_forbid_overwrite"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&policyResp); err != nil {
		return "", fmt.Errorf("%w: 解析上传凭证: %v", ErrEngineUnavailable, err)
	}
	p := policyResp.Data
	if p.UploadHost == "" || p.UploadDir == "" {
		return "", fmt.Errorf("%w: 上传凭证缺少 upload_host/upload_dir", ErrEngineUnavailable)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("asr: 打开音频失败: %w", err)
	}
	defer f.Close()

	// OSS object key 用磁盘上的 UUID 文件名，天然规避中文原名带来的编码问题。
	fileName := filepath.Base(filePath)
	key := fmt.Sprintf("%s/%s", p.UploadDir, fileName)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, v := range map[string]string{
		"OSSAccessKeyId":         p.OSSAccessKeyID,
		"Signature":              p.Signature,
		"policy":                 p.Policy,
		"x-oss-object-acl":       p.XOSSAcl,
		"x-oss-forbid-overwrite": p.XOSSForbidOverwrite,
		"key":                    key,
		"success_action_status":  "200",
	} {
		if err := w.WriteField(k, v); err != nil {
			return "", fmt.Errorf("%w: 构造上传表单: %v", ErrEngineUnavailable, err)
		}
	}
	part, err := w.CreateFormFile("file", fileName)
	if err != nil {
		return "", fmt.Errorf("%w: 构造上传表单: %v", ErrEngineUnavailable, err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", fmt.Errorf("asr: 读取音频失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("%w: 结束上传表单: %v", ErrEngineUnavailable, err)
	}

	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.UploadHost, &body)
	if err != nil {
		return "", fmt.Errorf("%w: 构造上传请求: %v", ErrEngineUnavailable, err)
	}
	upReq.Header.Set("Content-Type", w.FormDataContentType())

	upResp, err := c.http.Do(upReq)
	if err != nil {
		return "", fmt.Errorf("%w: 上传音频: %v", ErrEngineUnavailable, err)
	}
	defer upResp.Body.Close()
	if upResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: 上传音频 HTTP %d: %s", ErrEngineUnavailable, upResp.StatusCode, snippet(upResp.Body))
	}
	return "oss://" + key, nil
}

// --- 步骤 2：提交异步识别任务 ---

func (c *Client) submit(ctx context.Context, ossURL string) (string, error) {
	// 说话人分离仅支持单声道，由配置显式开启。
	params := map[string]any{"channel_id": []int{0}}
	if c.diarize {
		params["diarization_enabled"] = true
	}
	payload := map[string]any{
		"model":      c.model,
		"input":      map[string]any{"file_urls": []string{ossURL}},
		"parameters": params,
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("%w: 序列化任务请求: %v", ErrEngineUnavailable, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/services/audio/asr/transcription", bytes.NewReader(buf))
	if err != nil {
		return "", fmt.Errorf("%w: 构造提交请求: %v", ErrEngineUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DashScope-Async", "enable")
	// 使用 oss:// 临时 URL 时必须携带，缺失会被上游拒绝。
	req.Header.Set("X-DashScope-OssResourceResolve", "enable")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: 提交识别任务: %v", ErrEngineUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: 提交识别任务 HTTP %d: %s", ErrEngineUnavailable, resp.StatusCode, snippet(resp.Body))
	}

	var out struct {
		Output struct {
			TaskID     string `json:"task_id"`
			TaskStatus string `json:"task_status"`
		} `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("%w: 解析提交响应: %v", ErrEngineUnavailable, err)
	}
	if out.Output.TaskID == "" {
		return "", fmt.Errorf("%w: 提交响应缺少 task_id", ErrEngineUnavailable)
	}
	return out.Output.TaskID, nil
}

// --- 步骤 3：轮询任务状态（受 ctx 总超时保护） ---

func (c *Client) poll(ctx context.Context, taskID string) (string, error) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		transcriptionURL, done, err := c.queryOnce(ctx, taskID)
		if err != nil {
			return "", err
		}
		if done {
			return transcriptionURL, nil
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("%w: 轮询超时: %v", ErrEngineUnavailable, ctx.Err())
		case <-ticker.C:
		}
	}
}

// queryOnce 查询一次任务状态：done=true 表示已有终态结论。
func (c *Client) queryOnce(ctx context.Context, taskID string) (transcriptionURL string, done bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/tasks/"+taskID, nil)
	if err != nil {
		return "", false, fmt.Errorf("%w: 构造查询请求: %v", ErrEngineUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("%w: 查询任务: %v", ErrEngineUnavailable, err)
	}
	defer resp.Body.Close()

	var out struct {
		Output struct {
			TaskStatus string `json:"task_status"`
			Results    []struct {
				SubtaskStatus    string `json:"subtask_status"`
				Code             string `json:"code"`
				Message          string `json:"message"`
				TranscriptionURL string `json:"transcription_url"`
			} `json:"results"`
		} `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", false, fmt.Errorf("%w: 解析任务状态: %v", ErrEngineUnavailable, err)
	}

	switch out.Output.TaskStatus {
	case "SUCCEEDED":
		// 整体 SUCCEEDED 不代表每个子任务都成功，必须看 subtask_status。
		for _, r := range out.Output.Results {
			if r.SubtaskStatus == "SUCCEEDED" && r.TranscriptionURL != "" {
				return r.TranscriptionURL, true, nil
			}
			return "", false, fmt.Errorf("%w: 子任务失败 code=%s message=%s", ErrEngineUnavailable, r.Code, r.Message)
		}
		return "", false, fmt.Errorf("%w: 任务成功但无识别结果", ErrEngineUnavailable)
	case "FAILED", "CANCELED":
		return "", false, fmt.Errorf("%w: 任务状态 %s", ErrEngineUnavailable, out.Output.TaskStatus)
	default:
		return "", false, nil
	}
}

// --- 步骤 4：下载并解析识别结果 ---

func (c *Client) fetchResult(ctx context.Context, transcriptionURL string) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, transcriptionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: 构造结果下载请求: %v", ErrEngineUnavailable, err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: 下载识别结果: %v", ErrEngineUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: 下载识别结果 HTTP %d", ErrEngineUnavailable, resp.StatusCode)
	}

	var raw struct {
		Properties struct {
			OriginalDurationMS int `json:"original_duration_in_milliseconds"`
		} `json:"properties"`
		Transcripts []struct {
			Text      string `json:"text"`
			Sentences []struct {
				BeginTime int    `json:"begin_time"` // 毫秒
				EndTime   int    `json:"end_time"`   // 毫秒
				Text      string `json:"text"`
				SpeakerID *int   `json:"speaker_id"` // 未开启说话人分离时缺省
			} `json:"sentences"`
		} `json:"transcripts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: 解析识别结果: %v", ErrEngineUnavailable, err)
	}

	res := &Result{DurationMS: raw.Properties.OriginalDurationMS}
	var content bytes.Buffer
	for _, tr := range raw.Transcripts {
		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString(tr.Text)
		for _, s := range tr.Sentences {
			if s.Text == "" {
				continue
			}
			speaker := ""
			if s.SpeakerID != nil {
				// 转成「说话人1/2…」便于前端 TranscriptViewer 区分气泡。
				speaker = fmt.Sprintf("说话人%d", *s.SpeakerID+1)
			}
			res.Segments = append(res.Segments, Segment{
				Start:   float64(s.BeginTime) / 1000, // 毫秒 → 秒
				End:     float64(s.EndTime) / 1000,
				Speaker: speaker,
				Text:    s.Text,
			})
		}
	}
	res.Content = content.String()
	return res, nil
}

// snippet 截断上游响应体，避免把大段无关节流写进 transcripts.error_message（VARCHAR(255)）。
func snippet(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 200))
	return string(b)
}
