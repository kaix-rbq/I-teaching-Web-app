package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/config"
)

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
