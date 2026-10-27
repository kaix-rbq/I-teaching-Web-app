package kb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"aijiaoxue-api/internal/agent"
	"aijiaoxue-api/internal/model"
)

// Store 封装 knowledge_documents / knowledge_chunks 的读写。
// 表结构由 V16 迁移维护；本类型不做 AutoMigrate。
type Store struct {
	db *gorm.DB
}

// NewStore 构造知识库存储。
func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

// HasTables 检查 V16 是否已执行；cmd/kb 在启动时用它给出可操作的错误。
func (s *Store) HasTables(ctx context.Context) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("kb: store 未初始化")
	}
	m := s.db.WithContext(ctx).Migrator()
	okDoc := m.HasTable(&model.KnowledgeDocument{})
	okChunk := m.HasTable(&model.KnowledgeChunk{})
	return okDoc && okChunk, nil
}

// FindDocument 按 doc_key 查文档；不存在返回 (nil, nil)。
func (s *Store) FindDocument(ctx context.Context, docKey string) (*model.KnowledgeDocument, error) {
	var doc model.KnowledgeDocument
	tx := s.db.WithContext(ctx).Where("doc_key = ?", docKey).Limit(1).Find(&doc)
	if tx.Error != nil {
		return nil, fmt.Errorf("查询知识库文档失败: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}
	return &doc, nil
}

// SaveDocument 在一个事务内 upsert 文档并替换其切片。
// chunks 为 nil 表示只更新文档元数据（如标记 failed 时保留旧切片）。
func (s *Store) SaveDocument(ctx context.Context, doc *model.KnowledgeDocument, chunks []model.KnowledgeChunk) error {
	if doc == nil {
		return errors.New("kb: doc 不能为空")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if doc.ID == 0 {
			if err := tx.Create(doc).Error; err != nil {
				return fmt.Errorf("创建知识库文档失败: %w", err)
			}
		} else {
			updates := map[string]any{
				"doc_key":         doc.DocKey,
				"title":           doc.Title,
				"collection":      doc.Collection,
				"course_id":       doc.CourseID,
				"subject":         doc.Subject,
				"source_format":   doc.SourceFormat,
				"source_path":     doc.SourcePath,
				"checksum":        doc.Checksum,
				"chunk_count":     doc.ChunkCount,
				"embedding_model": doc.EmbeddingModel,
				"embedding_dim":   doc.EmbeddingDim,
				"status":          doc.Status,
				"error_message":   doc.ErrorMessage,
				"ingested_at":     doc.IngestedAt,
				"updated_at":      time.Now(),
			}
			if err := tx.Model(&model.KnowledgeDocument{}).Where("id = ?", doc.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("更新知识库文档失败: %w", err)
			}
		}
		if chunks == nil {
			return nil
		}
		if err := tx.Where("document_id = ?", doc.ID).Delete(&model.KnowledgeChunk{}).Error; err != nil {
			return fmt.Errorf("清理旧切片失败: %w", err)
		}
		if len(chunks) == 0 {
			return nil
		}
		for i := range chunks {
			chunks[i].DocumentID = doc.ID
		}
		if err := tx.CreateInBatches(chunks, 100).Error; err != nil {
			return fmt.Errorf("写入知识库切片失败: %w", err)
		}
		return nil
	})
}

// MarkFailed 记录失败文档；已有切片保留，便于排查或回滚，不参与检索（检索只看 done）。
func (s *Store) MarkFailed(ctx context.Context, doc *model.KnowledgeDocument, message string) error {
	if doc == nil {
		return errors.New("kb: doc 不能为空")
	}
	if len([]rune(message)) > 250 {
		message = string([]rune(message)[:250])
	}
	doc.Status = "failed"
	doc.ErrorMessage = message
	doc.UpdatedAt = time.Now()
	if doc.ID == 0 {
		if err := s.db.WithContext(ctx).Create(doc).Error; err != nil {
			return fmt.Errorf("记录失败文档失败: %w", err)
		}
		return nil
	}
	err := s.db.WithContext(ctx).Model(&model.KnowledgeDocument{}).
		Where("id = ?", doc.ID).
		Updates(map[string]any{
			"status":        "failed",
			"error_message": message,
			"updated_at":    time.Now(),
		}).Error
	if err != nil {
		return fmt.Errorf("记录失败文档失败: %w", err)
	}
	return nil
}

// ListDocuments 按集合列出文档；collection 为空时列出全部。
func (s *Store) ListDocuments(ctx context.Context, collection string) ([]model.KnowledgeDocument, error) {
	q := s.db.WithContext(ctx).Order("collection ASC, id ASC")
	if collection != "" {
		q = q.Where("collection = ?", collection)
	}
	var docs []model.KnowledgeDocument
	if err := q.Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("查询知识库文档失败: %w", err)
	}
	return docs, nil
}

// DeleteCollection 删除集合下全部文档与切片，返回删除的文档数。
func (s *Store) DeleteCollection(ctx context.Context, collection string) (int64, error) {
	if collection == "" {
		return 0, errors.New("kb: 清空集合时 collection 不能为空")
	}
	var deleted int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sub := tx.Model(&model.KnowledgeDocument{}).Select("id").Where("collection = ?", collection)
		if err := tx.Where("document_id IN (?)", sub).Delete(&model.KnowledgeChunk{}).Error; err != nil {
			return fmt.Errorf("删除切片失败: %w", err)
		}
		res := tx.Where("collection = ?", collection).Delete(&model.KnowledgeDocument{})
		if res.Error != nil {
			return fmt.Errorf("删除文档失败: %w", res.Error)
		}
		deleted = res.RowsAffected
		return nil
	})
	return deleted, err
}

// CountChunks 返回文档切片数；用于 list 校验 chunk_count 是否一致。
func (s *Store) CountChunks(ctx context.Context, documentID uint64) (int64, error) {
	var n int64
	err := s.db.WithContext(ctx).Model(&model.KnowledgeChunk{}).
		Where("document_id = ?", documentID).Count(&n).Error
	return n, err
}

// LoadVectors 只读加载符合过滤条件且已完成入库的知识库切片向量。
//
// 模型与维度必须同时匹配，避免不同向量空间混用；embedding 为 NULL 或
// 长度与查询维度不符时返回明确错误，不静默混入候选集。
func (s *Store) LoadVectors(ctx context.Context, q agent.VectorQuery) ([]agent.StoredChunk, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("kb: store 未初始化")
	}
	if len(q.Collections) == 0 {
		return []agent.StoredChunk{}, nil
	}
	if strings.TrimSpace(q.EmbeddingModel) == "" {
		return nil, errors.New("kb: 加载向量时 embedding model 不能为空")
	}
	if q.EmbeddingDim <= 0 {
		return nil, fmt.Errorf("kb: 加载向量时 embedding dim 非法: %d", q.EmbeddingDim)
	}

	var rows []struct {
		ID         uint64 `gorm:"column:id"`
		Collection string `gorm:"column:collection"`
		Heading    string `gorm:"column:heading"`
		Content    string `gorm:"column:content"`
		Embedding  []byte `gorm:"column:embedding"`
	}
	tx := s.db.WithContext(ctx).
		Table("knowledge_chunks AS kc").
		Select("kc.id AS id, kc.collection AS collection, kc.heading AS heading, kc.content AS content, kc.embedding AS embedding").
		Joins("JOIN knowledge_documents AS kd ON kd.id = kc.document_id").
		Where("kd.status = ?", "done").
		Where("kc.embedding IS NOT NULL").
		Where("kc.embedding_model = ? AND kc.embedding_dim = ?", strings.TrimSpace(q.EmbeddingModel), q.EmbeddingDim).
		Where("kc.collection IN ?", q.Collections)
	if q.CourseID != nil {
		tx = tx.Where("(kc.course_id = 0 OR kc.course_id = ?)", *q.CourseID)
	}
	if subject := strings.TrimSpace(q.Subject); subject != "" {
		tx = tx.Where("kc.subject = ?", subject)
	}
	if err := tx.Order("kc.id ASC").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询知识库向量失败: %w", err)
	}

	out := make([]agent.StoredChunk, 0, len(rows))
	for _, row := range rows {
		vec, err := DecodeVector(row.Embedding, q.EmbeddingDim)
		if err != nil {
			return nil, fmt.Errorf("kb: 解析向量 %s#%d 失败: %w", row.Collection, row.ID, err)
		}
		out = append(out, agent.StoredChunk{
			ID:         row.ID,
			Collection: row.Collection,
			Heading:    row.Heading,
			Content:    row.Content,
			Embedding:  vec,
		})
	}
	return out, nil
}
