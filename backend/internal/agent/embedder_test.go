package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/config"
)

func TestEmbeddingClientBatchesAndNormalizes(t *testing.T) {
	var batches [][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/embeddings", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		var req embeddingsRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		batches = append(batches, req.Input)
		data := make([]map[string]any, len(req.Input))
		for i := range req.Input {
			data[i] = map[string]any{"index": i, "embedding": []float32{3, 4}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer srv.Close()

	c := NewEmbeddingClient(
		config.AgentConfig{APIKey: "test-key", BaseURL: srv.URL, Timeout: "5s"},
		config.EmbeddingConfig{Enabled: true, Model: "qwen-test", Dim: 2, BatchSize: 2},
	)
	require.NotNil(t, c)
	vecs, err := c.Embed(context.Background(), []string{"a", "b", "c"})
	require.NoError(t, err)
	require.Len(t, vecs, 3)
	assert.Equal(t, [][]string{{"a", "b"}, {"c"}}, batches, "必须按 batchSize=2 分批")
	assert.InDelta(t, 0.6, vecs[0][0], 1e-6)
	assert.InDelta(t, 0.8, vecs[0][1], 1e-6)
}

func TestEmbeddingClientRejectsWrongDimension(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1,2,3]}]}`))
	}))
	defer srv.Close()

	c := NewEmbeddingClient(
		config.AgentConfig{APIKey: "k", BaseURL: srv.URL, Timeout: "5s"},
		config.EmbeddingConfig{Enabled: true, Model: "m", Dim: 2, BatchSize: 20},
	)
	_, err := c.Embed(context.Background(), []string{"a"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "向量维度")
}

func TestEmbeddingClientDisabledReturnsNil(t *testing.T) {
	assert.Nil(t, NewEmbeddingClient(config.AgentConfig{}, config.EmbeddingConfig{Enabled: false}))
}
