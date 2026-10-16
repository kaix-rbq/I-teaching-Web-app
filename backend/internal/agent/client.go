package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"aijiaoxue-api/internal/config"
)

// SessionAnalyzer 是 AI 单次课评析能力接口；*Client 实现它，测试用假实现替换。
type SessionAnalyzer interface {
	AnalyzeSession(ctx context.Context, in EvalInput) (*Result, error)
}

// Client 是百炼 OpenAI 兼容接口的最小客户端（只做 S1 单次课评价）。
type Client struct {
	cfg    config.AgentConfig
	http   *http.Client
	sem    chan struct{}
	apiURL string
}

// NewClient 按配置构造客户端；未启用或缺 Key/模型时返回 nil（调用方必须显式判空）。
//
// 与 asr.NewClient 同构：返回具体指针类型而非接口，避免「持有 nil 指针的非 nil 接口」陷阱。
func NewClient(cfg config.AgentConfig) *Client {
	if !cfg.Enabled || strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.LLMModel) == "" {
		return nil
	}
	return &Client{
		cfg:    cfg,
		http:   &http.Client{Timeout: cfg.TimeoutDuration()},
		sem:    make(chan struct{}, cfg.Concurrency()),
		apiURL: chatCompletionsURL(cfg),
	}
}

// chatCompletionsURL 推导 OpenAI 兼容的 chat/completions 地址。
// baseUrl 显式配置时以它为准（便于联调与切换地域），否则按 workspaceId 推导北京地域。
func chatCompletionsURL(cfg config.AgentConfig) string {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = fmt.Sprintf("https://%s.cn-beijing.maas.aliyuncs.com/compatible-mode/v1", cfg.WorkspaceID)
	}
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	Temperature    float64       `json:"temperature"`
	ResponseFormat *respFormat   `json:"response_format,omitempty"`
	MaxTokens      int           `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type respFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

// AnalyzeSession 调用 LLM 评析单次课并解析为结构化结果。
// 上游失败统一包装为普通 error，由 service 层记 ai_tasks.failed，绝不向 HTTP 主链路传播。
func (c *Client) AnalyzeSession(ctx context.Context, in EvalInput) (*Result, error) {
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	body, err := json.Marshal(chatRequest{
		Model: c.cfg.LLMModel,
		Messages: []chatMessage{
			{Role: "system", Content: s1SystemPrompt},
			{Role: "user", Content: buildUserPrompt(in)},
		},
		Temperature:    0.2,
		ResponseFormat: &respFormat{Type: "json_object"},
		MaxTokens:      2000,
	})
	if err != nil {
		return nil, fmt.Errorf("agent: 构造请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("agent: 构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent: 调用上游失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent: 上游返回 %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("agent: 解析上游响应失败: %w", err)
	}
	if cr.Error != nil {
		return nil, fmt.Errorf("agent: 上游错误: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("agent: 上游返回空 choices")
	}
	promptVersion := c.cfg.PromptVersion
	if promptVersion == "" {
		promptVersion = "v1"
	}
	return ParseS1Response(cr.Choices[0].Message.Content, in.Transcript, promptVersion, c.cfg.LLMModel)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
