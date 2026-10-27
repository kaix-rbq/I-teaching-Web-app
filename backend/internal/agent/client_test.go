package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/config"
)

// newTestClient 构造指向 httptest 假上游的客户端。
func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	client := NewClient(config.AgentConfig{
		Enabled: true, APIKey: "test-key", LLMModel: "fake-llm", BaseURL: baseURL,
		Timeout: "3s", MaxConcurrency: 1, PromptVersion: "v1",
	})
	require.NotNil(t, client)
	return client
}

// chatCompletionBody 组装 OpenAI 兼容响应，content 为内部 S1 JSON。
func chatCompletionBody(t *testing.T, content string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"choices": []any{map[string]any{
			"message": map[string]any{"content": content},
		}},
	})
	require.NoError(t, err)
	return string(raw)
}

func TestAnalyzeSessionFiltersCitedChunksAgainstKnowledge(t *testing.T) {
	response := `{"choices":[{"message":{"content":"{\"dimensions\":{},\"confidence\":0.8,\"citedChunks\":[\"kb-rubric#13\",\"kb-rubric#999\",\"kb-rubric#13\"]}"}}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, response)
	}))
	defer srv.Close()

	client := NewClient(config.AgentConfig{
		Enabled: true, APIKey: "test-key", LLMModel: "fake-llm", BaseURL: srv.URL,
		Timeout: "3s", MaxConcurrency: 1, PromptVersion: "v1",
	})
	require.NotNil(t, client)

	res, err := client.AnalyzeSession(context.Background(), EvalInput{
		Transcript: "今天我们讲用户故事。",
		Knowledge:  []RetrievedChunk{{ChunkID: "kb-rubric#13"}},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"kb-rubric#13"}, res.Evidence.CitedChunks)

	res, err = client.AnalyzeSession(context.Background(), EvalInput{
		Transcript: "今天我们讲用户故事。",
	})
	require.NoError(t, err)
	assert.Empty(t, res.Evidence.CitedChunks, "无 KB 时必须清空模型返回的 citedChunks")
}

// TestAnalyzeSessionUsesMaxTokens3000 锁定 P1.4：S1 输出空间必须为 3000。
func TestAnalyzeSessionUsesMaxTokens3000(t *testing.T) {
	valid := chatCompletionBody(t, `{"dimensions":{},"confidence":0.5,"comment":"ok"}`)
	var got chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("解析请求体失败: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, valid)
	}))
	defer srv.Close()

	res, err := newTestClient(t, srv.URL).AnalyzeSession(context.Background(), EvalInput{
		Transcript: "今天我们讲用户故事。",
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", res.Comment)
	assert.Equal(t, 3000, got.MaxTokens)
}

// TestAnalyzeSessionRetriesOnceOnParseFailure 锁定 P1.4：第一次截断/非法 JSON
// 自动重试一次，第二次合法即成功；总共调用 2 次。
func TestAnalyzeSessionRetriesOnceOnParseFailure(t *testing.T) {
	bad := chatCompletionBody(t, `{"dimensions":{`)
	good := chatCompletionBody(t, `{"dimensions":{},"confidence":0.7,"comment":"重试成功"}`)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			// 上游 200，但 content 是截断的 S1 JSON。
			fmt.Fprint(w, bad)
			return
		}
		fmt.Fprint(w, good)
	}))
	defer srv.Close()

	res, err := newTestClient(t, srv.URL).AnalyzeSession(context.Background(), EvalInput{
		Transcript: "今天我们讲用户故事。",
	})
	require.NoError(t, err)
	assert.Equal(t, "重试成功", res.Comment)
	assert.Equal(t, 2, calls, "解析失败必须恰好重试一次")
}

// TestAnalyzeSessionStopsAfterSecondParseFailure 验证只重试一次：
// 两次都解析失败时返回错误，不再发起第三次调用。
func TestAnalyzeSessionStopsAfterSecondParseFailure(t *testing.T) {
	bad := chatCompletionBody(t, `{"dimensions":{`)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, bad)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv.URL).AnalyzeSession(context.Background(), EvalInput{
		Transcript: "今天我们讲用户故事。",
	})
	require.Error(t, err)
	assert.Equal(t, 2, calls, "只允许自动重试一次，总调用次数上限 2")
}

// TestAnalyzeSessionDoesNotRetryHTTPFailures 锁定 P1.4：HTTP 错误不自动重试，
// 避免重复计费。
func TestAnalyzeSessionDoesNotRetryHTTPFailures(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		http.Error(w, "upstream unavailable", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestClient(t, srv.URL).AnalyzeSession(context.Background(), EvalInput{
		Transcript: "今天我们讲用户故事。",
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls, "HTTP 500 不得自动重试")
}
