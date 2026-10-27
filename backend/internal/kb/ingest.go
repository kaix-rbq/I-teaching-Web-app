package kb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"aijiaoxue-api/internal/agent"
	"aijiaoxue-api/internal/model"
)

// 允许的知识库集合；目录名与集合名一致，减少手误。
var allowedCollections = map[string]struct{}{
	"kb-rubric":   {},
	"kb-standard": {},
	"kb-textbook": {},
	"kb-strategy": {},
	"kb-case":     {},
}

// ValidateCollection 校验集合名。
func ValidateCollection(name string) error {
	if _, ok := allowedCollections[name]; !ok {
		return fmt.Errorf("kb: 未知集合 %q（允许：kb-rubric/kb-standard/kb-textbook/kb-strategy/kb-case）", name)
	}
	return nil
}

// Source 是一份待摄取源文件。
type Source struct {
	Path       string // 绝对或相对路径
	RelPath    string // 相对知识库根目录的路径，用于 doc_key / source_path
	Collection string
	CourseID   uint64
	Subject    string
}

// IngestResult 是单文件摄取结果。
type IngestResult struct {
	DocKey  string
	Title   string
	Format  string
	Chunks  int
	Skipped bool
}

// Ingester 负责单文件摄取：解析 → 脱敏 → 切片 → 向量化 → 事务写入。
type Ingester struct {
	Store      *Store
	Embedder   agent.Embedder
	Scrub      func(string) string
	ChunkRunes int
	Overlap    int
}

// Ingest 摄取一份文件；同一 doc_key 且 checksum/模型/维度都没变时跳过。
func (i *Ingester) Ingest(ctx context.Context, src Source) (*IngestResult, error) {
	if i == nil || i.Store == nil {
		return nil, errors.New("kb: ingester/store 未初始化")
	}
	if i.Embedder == nil {
		return nil, errors.New("kb: embedder 未初始化")
	}
	if err := ValidateCollection(src.Collection); err != nil {
		return nil, err
	}
	if strings.TrimSpace(src.Path) == "" {
		return nil, errors.New("kb: 源文件路径为空")
	}
	rel := filepath.ToSlash(strings.TrimSpace(src.RelPath))
	if rel == "" {
		rel = filepath.Base(src.Path)
	}
	docKey := BuildDocKey(src.Collection, src.CourseID, src.Subject, rel)
	fallbackTitle := strings.TrimSuffix(filepath.Base(src.Path), filepath.Ext(src.Path))
	embeddingModel := embeddingModelOf(i.Embedder)
	embeddingDim := embeddingDimOf(i.Embedder)
	if embeddingModel == "" || embeddingDim <= 0 {
		return nil, errors.New("kb: embedder 未提供 model/dim 元数据，无法写入向量模型版本")
	}
	formatHint := sourceFormatOfPath(src.Path)

	text, format, err := ExtractFile(src.Path)
	if err != nil {
		i.recordFailure(ctx, src, rel, docKey, fallbackTitle, formatHint, embeddingModel, embeddingDim, err)
		return nil, err
	}
	if i.Scrub != nil {
		text = i.Scrub(text)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		err = errors.New("kb: 抽取后文本为空")
		i.recordFailure(ctx, src, rel, docKey, fallbackTitle, format, embeddingModel, embeddingDim, err)
		return nil, err
	}
	checksum := sha256Hex(text)
	title := DeriveTitle(text, fallbackTitle)
	if title == "" {
		title = fallbackTitle
	}

	existing, err := i.Store.FindDocument(ctx, docKey)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Checksum == checksum &&
		existing.EmbeddingModel == embeddingModel &&
		existing.EmbeddingDim == embeddingDim &&
		existing.Status == "done" && existing.ChunkCount > 0 {
		return &IngestResult{DocKey: docKey, Title: existing.Title, Format: format, Chunks: existing.ChunkCount, Skipped: true}, nil
	}

	chunks := ChunkDocument(text, i.chunkRunes(), i.overlapRunes())
	if len(chunks) == 0 {
		err = errors.New("kb: 切片结果为空")
		i.recordFailure(ctx, src, rel, docKey, title, format, embeddingModel, embeddingDim, err)
		return nil, err
	}
	texts := make([]string, len(chunks))
	for idx := range chunks {
		texts[idx] = chunks[idx].Content
	}
	vectors, err := i.Embedder.Embed(ctx, texts)
	if err != nil {
		i.recordFailure(ctx, src, rel, docKey, title, format, embeddingModel, embeddingDim, err)
		return nil, err
	}
	if len(vectors) != len(chunks) {
		err = fmt.Errorf("kb: 向量数 %d 与切片数 %d 不一致", len(vectors), len(chunks))
		i.recordFailure(ctx, src, rel, docKey, title, format, embeddingModel, embeddingDim, err)
		return nil, err
	}
	for idx := range vectors {
		if len(vectors[idx]) != embeddingDim {
			err = fmt.Errorf("kb: 第 %d 个向量维度 %d，期望 %d", idx, len(vectors[idx]), embeddingDim)
			i.recordFailure(ctx, src, rel, docKey, title, format, embeddingModel, embeddingDim, err)
			return nil, err
		}
	}

	now := time.Now()
	doc := &model.KnowledgeDocument{
		DocKey:         docKey,
		Title:          title,
		Collection:     src.Collection,
		CourseID:       src.CourseID,
		Subject:        src.Subject,
		SourceFormat:   format,
		SourcePath:     truncateRunes(rel, 255),
		Checksum:       checksum,
		ChunkCount:     len(chunks),
		EmbeddingModel: embeddingModel,
		EmbeddingDim:   embeddingDim,
		Status:         "done",
		ErrorMessage:   "",
		IngestedAt:     &now,
	}
	if existing != nil {
		doc.ID = existing.ID
	}
	rows := make([]model.KnowledgeChunk, len(chunks))
	for idx, c := range chunks {
		rows[idx] = model.KnowledgeChunk{
			Collection:     src.Collection,
			CourseID:       src.CourseID,
			Subject:        src.Subject,
			Seq:            idx,
			Heading:        c.Heading,
			Content:        c.Content,
			TokenCount:     c.TokenCount,
			Embedding:      EncodeVector(vectors[idx]),
			EmbeddingModel: embeddingModel,
			EmbeddingDim:   embeddingDim,
		}
	}
	if err := i.Store.SaveDocument(ctx, doc, rows); err != nil {
		i.recordFailure(ctx, src, rel, docKey, title, format, embeddingModel, embeddingDim, err)
		return nil, err
	}
	return &IngestResult{DocKey: docKey, Title: doc.Title, Format: format, Chunks: len(chunks)}, nil
}

func (i *Ingester) chunkRunes() int {
	if i.ChunkRunes <= 0 {
		return DefaultChunkRunes
	}
	return i.ChunkRunes
}

func (i *Ingester) overlapRunes() int {
	if i.Overlap < 0 {
		return 0
	}
	if i.Overlap == 0 {
		return DefaultOverlapRunes
	}
	return i.Overlap
}

func (i *Ingester) recordFailure(ctx context.Context, src Source, rel, docKey, title, format, embeddingModel string, embeddingDim int, cause error) {
	if i.Store == nil {
		return
	}
	doc := &model.KnowledgeDocument{
		DocKey:         docKey,
		Title:          title,
		Collection:     src.Collection,
		CourseID:       src.CourseID,
		Subject:        src.Subject,
		SourceFormat:   format,
		SourcePath:     truncateRunes(rel, 255),
		EmbeddingModel: embeddingModel,
		EmbeddingDim:   embeddingDim,
		Status:         "failed",
	}
	_ = i.Store.MarkFailed(ctx, doc, cause.Error())
}

// BuildDocKey 生成稳定标识；超长时退化为集合 + 内容路径哈希，仍保持可重复。
func BuildDocKey(collection string, courseID uint64, subject, relPath string) string {
	raw := fmt.Sprintf("%s:%d:%s:%s", collection, courseID, strings.TrimSpace(subject), filepath.ToSlash(relPath))
	if len([]rune(raw)) <= 128 {
		return raw
	}
	sum := sha256.Sum256([]byte(raw))
	return truncateRunes(collection+":", 32) + hex.EncodeToString(sum[:16])
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// sourceFormatOfPath 为失败记录提供合法的 source_format；未知扩展名按 txt 兜底。
func sourceFormatOfPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return "md"
	case ".docx":
		return "docx"
	case ".pdf":
		return "pdf"
	default:
		return "txt"
	}
}

// embeddingModelOf / embeddingDimOf 从可选的元数据接口读取；真实客户端实现 Model()/Dim()。
// 用接口断言而不是具体类型，方便测试替身与后续替换实现。
func embeddingModelOf(e agent.Embedder) string {
	if m, ok := e.(interface{ Model() string }); ok {
		return m.Model()
	}
	return ""
}

func embeddingDimOf(e agent.Embedder) int {
	if d, ok := e.(interface{ Dim() int }); ok {
		return d.Dim()
	}
	return 0
}
