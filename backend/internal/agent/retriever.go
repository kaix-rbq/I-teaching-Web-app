package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"aijiaoxue-api/internal/config"
)

// vectorCacheTTL 是候选向量缓存的有效期。
//
// CLI 重新摄取知识库后，本进程最多 30 秒后看到新向量；该延迟用于避免为每次
// 请求都全量扫描 knowledge_chunks，同时不需要跨进程失效机制。
const vectorCacheTTL = 30 * time.Second

// ChunkFilter 描述知识库检索过滤条件。
type ChunkFilter struct {
	// Collections 限定知识库集合；为空时检索直接返回空结果。
	Collections []string
	// CourseID 非空时同时匹配通用切片（course_id=0）与指定课程切片。
	CourseID *uint64
	// Subject 非空时按学科过滤。
	Subject string
	// EmbeddingModel 指定向量模型；为空时使用 Retriever 配置中的模型。
	EmbeddingModel string
	// EmbeddingDim 指定向量维度；<=0 时使用 Retriever 配置中的维度。
	EmbeddingDim int
}

// VectorQuery 是 store 加载候选向量的只读查询条件。
type VectorQuery struct {
	Collections    []string
	CourseID       *uint64
	Subject        string
	EmbeddingModel string
	EmbeddingDim   int
}

// StoredChunk 是 store 返回的候选切片及其已归一化向量。
type StoredChunk struct {
	ID         uint64
	Collection string
	Heading    string
	Content    string
	Embedding  []float32
}

// ChunkID 返回稳定的知识库引用 id，格式为 collection#chunkID。
func (c StoredChunk) ChunkID() string {
	return fmt.Sprintf("%s#%d", c.Collection, c.ID)
}

// RetrievedChunk 是检索命中片段。
type RetrievedChunk struct {
	ChunkID    string
	Collection string
	Heading    string
	Content    string
	Score      float64
}

// VectorStore 是 retriever 依赖的只读向量查询能力；kb.Store 实现它。
type VectorStore interface {
	LoadVectors(ctx context.Context, q VectorQuery) ([]StoredChunk, error)
}

// Retriever 是知识库检索能力接口；测试可用假实现替换。
type Retriever interface {
	Retrieve(ctx context.Context, query string, f ChunkFilter) ([]RetrievedChunk, error)
}

// VectorRetriever 基于查询向量与内存候选缓存实现余弦 topK 检索。
type VectorRetriever struct {
	embedder Embedder
	store    VectorStore
	cfg      config.EmbeddingConfig

	mu    sync.Mutex
	cache map[vectorCacheKey]vectorCacheEntry
}

type vectorCacheKey struct {
	Collections string
	CourseID    uint64
	HasCourse   bool
	Subject     string
	Model       string
	Dim         int
}

type vectorCacheEntry struct {
	chunks    []StoredChunk
	expiresAt time.Time
}

// NewRetriever 构造知识库检索器。embedder 与 store 均不应为 nil；
// 生产装配仅在 embedding 客户端可用时构造 retriever。
func NewRetriever(embedder Embedder, store VectorStore, cfg config.EmbeddingConfig) *VectorRetriever {
	return &VectorRetriever{
		embedder: embedder,
		store:    store,
		cfg:      cfg,
		cache:    make(map[vectorCacheKey]vectorCacheEntry),
	}
}

// Retrieve 将 query 向量化后加载候选切片，按余弦相似度过滤阈值并返回 topK。
//
// 查询为空、集合为空或无候选/命中时均返回空切片而非错误。每次调用都会重新
// 计算 query 向量；只缓存按过滤条件加载的候选切片，TTL 为 30 秒。
func (r *VectorRetriever) Retrieve(ctx context.Context, query string, f ChunkFilter) ([]RetrievedChunk, error) {
	collections := normalizeCollections(f.Collections)
	if strings.TrimSpace(query) == "" || len(collections) == 0 {
		return []RetrievedChunk{}, nil
	}
	if r == nil || r.embedder == nil || r.store == nil {
		return nil, fmt.Errorf("agent: retriever 未正确初始化")
	}

	model, dim := r.effectiveModelDim(f)
	if model == "" {
		return nil, fmt.Errorf("agent: embedding model 未配置")
	}
	if dim <= 0 {
		return nil, fmt.Errorf("agent: embedding dim 非法: %d", dim)
	}

	vecs, err := r.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("agent: 查询向量化失败: %w", err)
	}
	if len(vecs) != 1 || len(vecs[0]) != dim {
		got := 0
		if len(vecs) > 0 {
			got = len(vecs[0])
		}
		return nil, fmt.Errorf("agent: 查询向量数量或维度不符（数量=%d，维度=%d，期望维度=%d）", len(vecs), got, dim)
	}

	q := VectorQuery{
		Collections:    collections,
		CourseID:       f.CourseID,
		Subject:        strings.TrimSpace(f.Subject),
		EmbeddingModel: model,
		EmbeddingDim:   dim,
	}
	candidates, err := r.loadCached(ctx, q)
	if err != nil {
		return nil, err
	}

	minScore := r.cfg.EffectiveMinScore()
	hits := make([]RetrievedChunk, 0, len(candidates))
	for _, c := range candidates {
		if len(c.Embedding) != dim {
			return nil, fmt.Errorf("agent: 候选向量 %s 维度 %d，期望 %d", c.ChunkID(), len(c.Embedding), dim)
		}
		score := dot(vecs[0], c.Embedding)
		if score < minScore {
			continue
		}
		hits = append(hits, RetrievedChunk{
			ChunkID:    c.ChunkID(),
			Collection: c.Collection,
			Heading:    c.Heading,
			Content:    c.Content,
			Score:      score,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	topK := r.cfg.EffectiveTopK()
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, nil
}

// effectiveModelDim 优先使用过滤条件中的模型/维度，缺省回落到配置值。
func (r *VectorRetriever) effectiveModelDim(f ChunkFilter) (string, int) {
	model := strings.TrimSpace(f.EmbeddingModel)
	if model == "" {
		model = strings.TrimSpace(r.cfg.Model)
	}
	dim := f.EmbeddingDim
	if dim <= 0 {
		dim = r.cfg.EffectiveDim()
	}
	return model, dim
}

// loadCached 返回过滤条件对应的候选切片；命中未过期缓存时不会访问 store。
func (r *VectorRetriever) loadCached(ctx context.Context, q VectorQuery) ([]StoredChunk, error) {
	key := vectorCacheKey{
		Collections: strings.Join(q.Collections, "\x00"),
		Subject:     q.Subject,
		Model:       q.EmbeddingModel,
		Dim:         q.EmbeddingDim,
	}
	if q.CourseID != nil {
		key.CourseID = *q.CourseID
		key.HasCourse = true
	}

	now := time.Now()
	r.mu.Lock()
	if entry, ok := r.cache[key]; ok && now.Before(entry.expiresAt) {
		cached := append([]StoredChunk(nil), entry.chunks...)
		r.mu.Unlock()
		return cached, nil
	}
	r.mu.Unlock()

	chunks, err := r.store.LoadVectors(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("agent: 加载知识库向量失败: %w", err)
	}
	// 缓存只读候选；复制 slice 头，避免 store 或调用方意外修改缓存结构。
	cached := append([]StoredChunk(nil), chunks...)
	r.mu.Lock()
	r.cache[key] = vectorCacheEntry{chunks: cached, expiresAt: now.Add(vectorCacheTTL)}
	r.mu.Unlock()
	return append([]StoredChunk(nil), cached...), nil
}

// normalizeCollections 去除空白项、排序并去重，使缓存 key 与查询条件稳定。
func normalizeCollections(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, c := range in {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// dot 计算两个等长向量的点积；入库向量已 L2 归一化，故等价余弦相似度。
func dot(a, b []float32) float64 {
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}
