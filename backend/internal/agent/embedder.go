package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"aijiaoxue-api/internal/config"
)

// Embedder 是知识库向量化能力接口；测试用假实现替换，生产实现走百炼 OpenAI 兼容接口。
type Embedder interface {
	// Embed 返回与输入等长、顺序一致的向量；每个向量已做 L2 归一化。
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// EmbeddingClient 是 qwen3.7-text-embedding-flash 的最小客户端。
//
// 安全约定：API Key / WorkspaceID / BaseURL 复用 AgentConfig，不新增第二份凭据。
type EmbeddingClient struct {
	cfg    config.EmbeddingConfig
	creds  config.AgentConfig
	http   *http.Client
	apiURL string
}

// NewEmbeddingClient 按配置构造客户端；未启用或缺凭据/模型时返回 nil（调用方必须显式判空）。
func NewEmbeddingClient(creds config.AgentConfig, cfg config.EmbeddingConfig) *EmbeddingClient {
	if !cfg.Enabled || strings.TrimSpace(creds.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
		return nil
	}
	return &EmbeddingClient{
		cfg:    cfg,
		creds:  creds,
		http:   &http.Client{Timeout: creds.TimeoutDuration()},
		apiURL: embeddingsURL(creds),
	}
}

// Model 返回配置的向量模型名，供知识库写入 embedding_model。
func (c *EmbeddingClient) Model() string { return c.cfg.Model }

// Dim 返回配置的向量维度，供知识库写入 embedding_dim。
func (c *EmbeddingClient) Dim() int { return c.cfg.EffectiveDim() }

// embeddingsURL 推导 OpenAI 兼容的 embeddings 地址。
// baseUrl 显式配置时以它为准；否则按 workspaceId 推导北京地域。
func embeddingsURL(creds config.AgentConfig) string {
	base := strings.TrimRight(strings.TrimSpace(creds.BaseURL), "/")
	if base == "" {
		base = fmt.Sprintf("https://%s.cn-beijing.maas.aliyuncs.com/compatible-mode/v1", creds.WorkspaceID)
	}
	if strings.HasSuffix(base, "/embeddings") {
		return base
	}
	return base + "/embeddings"
}

type embeddingsRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions"`
}

type embeddingsResponse struct {
	Data []struct {
		Index     *int      `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Embed 按 batchSize 分批调用，单批失败即返回错误（由调用方决定是否把这个文档标 failed）。
func (c *EmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	batchSize := c.cfg.EffectiveBatchSize()
	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += batchSize {
		end := start + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := c.embedBatch(ctx, texts[start:end])
		if err != nil {
			return nil, fmt.Errorf("embedding: batch %d-%d: %w", start, end, err)
		}
		out = append(out, vecs...)
	}
	return out, nil
}

func (c *EmbeddingClient) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	body, err := json.Marshal(embeddingsRequest{
		Model:      c.cfg.Model,
		Input:      texts,
		Dimensions: c.cfg.EffectiveDim(),
	})
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.creds.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用上游失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("上游返回 %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var er embeddingsResponse
	if err := json.Unmarshal(raw, &er); err != nil {
		return nil, fmt.Errorf("解析上游响应失败: %w", err)
	}
	if er.Error != nil {
		return nil, fmt.Errorf("上游错误: %s", er.Error.Message)
	}
	if len(er.Data) != len(texts) {
		return nil, fmt.Errorf("上游返回 %d 条向量，期望 %d 条", len(er.Data), len(texts))
	}

	dim := c.cfg.EffectiveDim()
	out := make([][]float32, len(texts))
	seen := make([]bool, len(texts))
	for i, item := range er.Data {
		idx := i
		if item.Index != nil {
			idx = *item.Index
		}
		if idx < 0 || idx >= len(texts) {
			return nil, fmt.Errorf("上游返回越界 index=%d", idx)
		}
		if seen[idx] {
			return nil, fmt.Errorf("上游返回重复 index=%d", idx)
		}
		if len(item.Embedding) != dim {
			return nil, fmt.Errorf("向量维度 %d，期望 %d（model=%s）", len(item.Embedding), dim, c.cfg.Model)
		}
		seen[idx] = true
		vec := normalize(item.Embedding)
		if isZeroVector(vec) {
			return nil, fmt.Errorf("上游返回零向量 index=%d", idx)
		}
		out[idx] = vec
	}
	for i, ok := range seen {
		if !ok {
			return nil, fmt.Errorf("上游缺少 index=%d 的向量", i)
		}
	}
	return out, nil
}

// normalize 做 L2 归一化，确保检索时点积等价余弦；零向量由调用方判错。
func normalize(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return v
	}
	norm := float32(math.Sqrt(sum))
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = x / norm
	}
	return out
}

func isZeroVector(v []float32) bool {
	for _, x := range v {
		if x != 0 {
			return false
		}
	}
	return true
}
