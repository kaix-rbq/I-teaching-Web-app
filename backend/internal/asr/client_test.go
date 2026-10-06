package asr

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/config"
)

// newTestClient 构造指向 httptest 服务端的客户端（同包测试可直接写非导出字段）。
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Client{
		apiKey:    "test-key",
		model:     "qwen-audio-3.1-asr-flash-filetrans",
		baseURL:   srv.URL,
		policyURL: srv.URL + "/api/v1/uploads",
		interval:  time.Millisecond,
		http:      srv.Client(),
	}
}

// TestFetchResultParse 验证识别结果解析：
// 毫秒→秒转换、全文拼接、说话人分离字段（未开启时 speaker_id 缺省）。
func TestFetchResultParse(t *testing.T) {
	payload := `{
      "properties": {"original_duration_in_milliseconds": 183000},
      "transcripts": [{
        "channel_id": 0,
        "text": "同学们好，今天讲进程调度。",
        "sentences": [
          {"begin_time": 1200, "end_time": 3400, "text": "同学们好，", "speaker_id": 0},
          {"begin_time": 3400, "end_time": 8100, "text": "今天讲进程调度。", "speaker_id": 1},
          {"begin_time": 8100, "end_time": 9000, "text": ""}
        ]
      }]
    }`
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, payload)
	})

	res, err := client.fetchResult(context.Background(), client.baseURL+"/result.json")
	require.NoError(t, err)

	assert.Equal(t, 183000, res.DurationMS)
	assert.Equal(t, "同学们好，今天讲进程调度。", res.Content)
	require.Len(t, res.Segments, 2, "空文本句子必须被丢弃")

	// 毫秒 → 秒
	assert.InDelta(t, 1.2, res.Segments[0].Start, 1e-9)
	assert.InDelta(t, 3.4, res.Segments[0].End, 1e-9)
	// speaker_id 0/1 → 说话人1/2（前端按此分气泡）
	assert.Equal(t, "说话人1", res.Segments[0].Speaker)
	assert.Equal(t, "说话人2", res.Segments[1].Speaker)
}

// TestFetchResultWithoutDiarization 未开启说话人分离时 speaker_id 缺省，Speaker 应为空。
func TestFetchResultWithoutDiarization(t *testing.T) {
	payload := `{"properties":{"original_duration_in_milliseconds":1000},
      "transcripts":[{"text":"好的","sentences":[{"begin_time":0,"end_time":1000,"text":"好的"}]}]}`
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, payload)
	})

	res, err := client.fetchResult(context.Background(), client.baseURL+"/r.json")
	require.NoError(t, err)
	require.Len(t, res.Segments, 1)
	assert.Equal(t, "", res.Segments[0].Speaker)
	assert.Equal(t, "好的", res.Content)
}

// TestQueryOnceTerminalStates 验证任务终态判定。
// 关键点：整体 SUCCEEDED 不代表子任务成功，必须看 subtask_status。
func TestQueryOnceTerminalStates(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantURL   string
		wantDone  bool
		wantError bool
	}{
		{
			name:     "仍在排队",
			body:     `{"output":{"task_status":"PENDING"}}`,
			wantDone: false,
		},
		{
			name:     "运行中",
			body:     `{"output":{"task_status":"RUNNING"}}`,
			wantDone: false,
		},
		{
			name:     "成功",
			body:     `{"output":{"task_status":"SUCCEEDED","results":[{"subtask_status":"SUCCEEDED","transcription_url":"https://x/r.json"}]}}`,
			wantURL:  "https://x/r.json",
			wantDone: true,
		},
		{
			name:      "整体成功但子任务失败",
			body:      `{"output":{"task_status":"SUCCEEDED","results":[{"subtask_status":"FAILED","code":"InvalidFile","message":"格式不支持"}]}}`,
			wantError: true,
		},
		{
			name:      "任务失败",
			body:      `{"output":{"task_status":"FAILED"}}`,
			wantError: true,
		},
		{
			name:      "成功但无结果",
			body:      `{"output":{"task_status":"SUCCEEDED","results":[]}}`,
			wantError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body)
			})

			gotURL, done, err := client.queryOnce(context.Background(), "task-1")
			if tc.wantError {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrEngineUnavailable,
					"上游失败必须包装为 ErrEngineUnavailable，供 service 降级为 failed + 50003")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantDone, done)
			assert.Equal(t, tc.wantURL, gotURL)
		})
	}
}

// TestPollStopsOnContextTimeout 回归：轮询必须受 ctx 保护，不能在 ctx 取消后继续空转。
func TestPollStopsOnContextTimeout(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"output":{"task_status":"RUNNING"}}`)
	})
	client.interval = time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err := client.poll(ctx, "task-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEngineUnavailable)
}

// TestSubmitSendsRequiredHeaders 验证提交任务时两个必需请求头都在。
// X-DashScope-OssResourceResolve 缺失会被上游直接拒绝（oss:// 临时 URL 场景）。
func TestSubmitSendsRequiredHeaders(t *testing.T) {
	var gotAsync, gotResolve, gotAuth string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAsync = r.Header.Get("X-DashScope-Async")
		gotResolve = r.Header.Get("X-DashScope-OssResourceResolve")
		gotAuth = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `{"output":{"task_id":"t-1","task_status":"PENDING"}}`)
	})

	taskID, err := client.submit(context.Background(), "oss://bucket/key/audio.mp3")
	require.NoError(t, err)
	assert.Equal(t, "t-1", taskID)
	assert.Equal(t, "enable", gotAsync)
	assert.Equal(t, "enable", gotResolve)
	assert.Equal(t, "Bearer test-key", gotAuth)
}

// TestUploadUsesPolicyAndKeepsKeyOutOfBody 回归：API Key 只进 Authorization 头，
// 绝不能出现在上传表单字段里（OSS 表单会长期留存）。
func TestUploadUsesPolicyAndKeepsKeyOutOfBody(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "audio.mp3")
	require.NoError(t, os.WriteFile(audioPath, []byte("fake-mp3-bytes"), 0o600))

	var policyForm string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/uploads":
			// getPolicy：上传凭证
			_, _ = io.WriteString(w, `{"data":{
              "upload_host":"http://`+r.Host+`/oss-put",
              "upload_dir":"pre/filetrans-16k/2024","oss_access_key_id":"ak","signature":"sig",
              "policy":"pol","x_oss_object_acl":"private","x_oss_forbid_overwrite":"true"}}`)
		case "/oss-put":
			require.NoError(t, r.ParseMultipartForm(1<<20))
			policyForm = r.FormValue("Signature") + "|" + r.FormValue("policy") + "|" + r.FormValue("key")
			_, _ = io.WriteString(w, "ok")
		default:
			http.NotFound(w, r)
		}
	})

	ossURL, err := client.upload(context.Background(), audioPath)
	require.NoError(t, err)
	assert.Equal(t, "oss://pre/filetrans-16k/2024/audio.mp3", ossURL)
	assert.Equal(t, "sig|pol|pre/filetrans-16k/2024/audio.mp3", policyForm)
	assert.NotContains(t, policyForm, "test-key", "API Key 不得进入上传表单字段")
}

// TestNewClientDisabled 未启用/缺少凭据时必须返回 nil，由调用方降级（不阻断主流程）。
func TestNewClientDisabled(t *testing.T) {
	assert.Nil(t, NewClient(config.TranscriptionConfig{Enabled: false, APIKey: "k", Model: "m"}))
	assert.Nil(t, NewClient(config.TranscriptionConfig{Enabled: true, APIKey: "", Model: "m"}), "缺 Key 必须返回 nil")
	assert.Nil(t, NewClient(config.TranscriptionConfig{Enabled: true, APIKey: "k", Model: ""}), "缺模型必须返回 nil")

	c := NewClient(config.TranscriptionConfig{Enabled: true, APIKey: "k", Model: "m", WorkspaceID: "ws-1"})
	require.NotNil(t, c)
	assert.Equal(t, "https://ws-1.cn-beijing.maas.aliyuncs.com", c.baseURL, "配置业务空间时使用专属域名")
	assert.Equal(t, 30*time.Minute, c.Timeout(), "未配置 timeout 时缺省 30 分钟")

	c2 := NewClient(config.TranscriptionConfig{Enabled: true, APIKey: "k", Model: "m"})
	require.NotNil(t, c2)
	assert.Equal(t, publicBase, c2.baseURL, "未配置业务空间时回落到公共域名")
}
