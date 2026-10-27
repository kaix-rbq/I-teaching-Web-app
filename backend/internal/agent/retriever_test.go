package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/config"
)

type fakeRetrieverEmbedder struct {
	vec  []float32
	err  error
	call int
}

func (f *fakeRetrieverEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	f.call++
	if f.err != nil {
		return nil, f.err
	}
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = append([]float32(nil), f.vec...)
	}
	return out, nil
}

type fakeVectorStore struct {
	chunks []StoredChunk
	err    error
	calls  int
	last   VectorQuery
}

func (f *fakeVectorStore) LoadVectors(_ context.Context, q VectorQuery) ([]StoredChunk, error) {
	f.calls++
	f.last = q
	if f.err != nil {
		return nil, f.err
	}
	return append([]StoredChunk(nil), f.chunks...), nil
}

func retrieverConfig(topK int, minScore float64) config.EmbeddingConfig {
	return config.EmbeddingConfig{Model: "test-embedding", Dim: 2, TopK: topK, MinScore: minScore}
}

func TestVectorRetrieverTopKAndThreshold(t *testing.T) {
	embedder := &fakeRetrieverEmbedder{vec: []float32{1, 0}}
	store := &fakeVectorStore{chunks: []StoredChunk{
		{ID: 1, Collection: "kb-rubric", Heading: "低", Content: "低分", Embedding: []float32{0.30, 0.9539392}},
		{ID: 2, Collection: "kb-rubric", Heading: "中", Content: "中分", Embedding: []float32{0.70, 0.7141428}},
		{ID: 3, Collection: "kb-rubric", Heading: "高", Content: "高分", Embedding: []float32{0.90, 0.4358899}},
	}}
	r := NewRetriever(embedder, store, retrieverConfig(2, 0.35))

	hits, err := r.Retrieve(context.Background(), "互动锚点", ChunkFilter{Collections: []string{"kb-rubric"}})
	require.NoError(t, err)
	require.Len(t, hits, 2)
	assert.Equal(t, "kb-rubric#3", hits[0].ChunkID)
	assert.Equal(t, "kb-rubric#2", hits[1].ChunkID)
	assert.InDelta(t, 0.90, hits[0].Score, 1e-5)
	assert.Greater(t, hits[0].Score, hits[1].Score)
}

func TestVectorRetrieverDropsBelowThreshold(t *testing.T) {
	embedder := &fakeRetrieverEmbedder{vec: []float32{1, 0}}
	store := &fakeVectorStore{chunks: []StoredChunk{
		{ID: 1, Collection: "kb-rubric", Content: "高于阈值", Embedding: []float32{0.36, 0.933}},
		{ID: 2, Collection: "kb-rubric", Content: "低于阈值", Embedding: []float32{0.349, 0.9371}},
	}}
	r := NewRetriever(embedder, store, retrieverConfig(8, 0.35))

	hits, err := r.Retrieve(context.Background(), "q", ChunkFilter{Collections: []string{"kb-rubric"}})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "kb-rubric#1", hits[0].ChunkID)
}

func TestVectorRetrieverCachesCandidatesNotQuery(t *testing.T) {
	embedder := &fakeRetrieverEmbedder{vec: []float32{1, 0}}
	store := &fakeVectorStore{chunks: []StoredChunk{
		{ID: 1, Collection: "kb-rubric", Content: "缓存命中", Embedding: []float32{0.9, 0.4358899}},
	}}
	r := NewRetriever(embedder, store, retrieverConfig(8, 0.35))
	filter := ChunkFilter{Collections: []string{"kb-rubric"}}

	_, err := r.Retrieve(context.Background(), "第一次", filter)
	require.NoError(t, err)
	_, err = r.Retrieve(context.Background(), "第二次", filter)
	require.NoError(t, err)

	assert.Equal(t, 1, store.calls, "同一过滤条件第二次必须命中候选缓存")
	assert.Equal(t, 2, embedder.call, "query 向量不得缓存，每次检索都要重新计算")
}

func TestVectorRetrieverQueryModelAndDimIsolation(t *testing.T) {
	embedder := &fakeRetrieverEmbedder{vec: []float32{1, 0, 0, 0}}
	store := &fakeVectorStore{}
	r := NewRetriever(embedder, store, config.EmbeddingConfig{Model: "cfg-model", Dim: 4, TopK: 8, MinScore: 0.35})
	courseID := uint64(12)

	_, err := r.Retrieve(context.Background(), "q", ChunkFilter{
		Collections:    []string{"kb-rubric"},
		CourseID:       &courseID,
		Subject:        "软件工程",
		EmbeddingModel: "filter-model",
		EmbeddingDim:   4,
	})
	require.NoError(t, err)
	assert.Equal(t, "filter-model", store.last.EmbeddingModel)
	assert.Equal(t, 4, store.last.EmbeddingDim)
	require.NotNil(t, store.last.CourseID)
	assert.Equal(t, uint64(12), *store.last.CourseID)
	assert.Equal(t, "软件工程", store.last.Subject)
	assert.Equal(t, []string{"kb-rubric"}, store.last.Collections)
}

func TestVectorRetrieverEmptyInputReturnsEmpty(t *testing.T) {
	embedder := &fakeRetrieverEmbedder{vec: []float32{1, 0}}
	store := &fakeVectorStore{}
	r := NewRetriever(embedder, store, retrieverConfig(8, 0.35))

	hits, err := r.Retrieve(context.Background(), "", ChunkFilter{Collections: []string{"kb-rubric"}})
	require.NoError(t, err)
	assert.Empty(t, hits)

	hits, err = r.Retrieve(context.Background(), "q", ChunkFilter{})
	require.NoError(t, err)
	assert.Empty(t, hits)
	assert.Zero(t, store.calls)
	assert.Zero(t, embedder.call)
}
