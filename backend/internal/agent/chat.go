package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ChatMessage 是 S3 对话消息；agent 角色在 OpenAI 兼容接口中映射为 assistant。
type ChatMessage struct {
	Role    string
	Content string
}

// ChatStreamer 是流式提优对话能力接口；*Client 实现它，测试用假实现替换。
type ChatStreamer interface {
	// StreamChat 逐段回调 delta；返回时表示上游成功结束。
	StreamChat(ctx context.Context, messages []ChatMessage, onDelta func(string) error) error
}

// chatStreamRequest 是 OpenAI 兼容的流式 chat/completions 请求体。
type chatStreamRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

// StreamChat 调用百炼 OpenAI 兼容流式接口并逐帧回调 delta。
// 复用 Client 的 http.Client 与 sem，不创建第二套连接池。
func (c *Client) StreamChat(ctx context.Context, messages []ChatMessage, onDelta func(string) error) error {
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return ctx.Err()
	}

	reqBody := make([]chatMessage, 0, len(messages))
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role == "agent" {
			role = "assistant"
		}
		reqBody = append(reqBody, chatMessage{Role: role, Content: m.Content})
	}
	body, err := json.Marshal(chatStreamRequest{
		Model:       c.cfg.LLMModel,
		Messages:    reqBody,
		Temperature: 0.4,
		Stream:      true,
		MaxTokens:   2000,
	})
	if err != nil {
		return fmt.Errorf("agent: 构造 S3 请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("agent: 构造 S3 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("agent: 调用 S3 上游失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("agent: S3 上游返回 %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			return nil
		}
		var frame chatStreamResponse
		if err := json.Unmarshal([]byte(payload), &frame); err != nil {
			return fmt.Errorf("agent: 解析 S3 流帧失败: %w", err)
		}
		if frame.Error != nil {
			return fmt.Errorf("agent: S3 上游错误: %s", frame.Error.Message)
		}
		if len(frame.Choices) == 0 {
			continue
		}
		delta := frame.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		if err := onDelta(delta); err != nil {
			return err
		}
	}
	return scanner.Err()
}
