package model

import "time"

// KnowledgeDocument 对应 knowledge_documents（RAG §4.2）。
// 表结构以 migrations/16_knowledge_base.up.sql 为唯一事实源，禁止 AutoMigrate。
type KnowledgeDocument struct {
	ID             uint64     `gorm:"primaryKey" json:"id"`
	DocKey         string     `gorm:"column:doc_key" json:"docKey"`
	Title          string     `json:"title"`
	Collection     string     `json:"collection"`
	CourseID       uint64     `gorm:"column:course_id" json:"courseId"`
	Subject        string     `json:"subject"`
	SourceFormat   string     `gorm:"column:source_format" json:"sourceFormat"`
	SourcePath     string     `gorm:"column:source_path" json:"sourcePath"`
	Checksum       string     `json:"checksum"`
	ChunkCount     int        `gorm:"column:chunk_count" json:"chunkCount"`
	EmbeddingModel string     `gorm:"column:embedding_model" json:"embeddingModel"`
	EmbeddingDim   int        `gorm:"column:embedding_dim" json:"embeddingDim"`
	Status         string     `json:"status"`
	ErrorMessage   string     `gorm:"column:error_message" json:"errorMessage"`
	IngestedAt     *time.Time `gorm:"column:ingested_at" json:"ingestedAt"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}

// TableName 固定表名，避免 GORM 复数推导。
func (KnowledgeDocument) TableName() string { return "knowledge_documents" }

// KnowledgeChunk 对应 knowledge_chunks；Embedding 为 float32 小端紧凑排列的 LONGBLOB。
type KnowledgeChunk struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	DocumentID     uint64    `gorm:"column:document_id" json:"documentId"`
	Collection     string    `json:"collection"`
	CourseID       uint64    `gorm:"column:course_id" json:"courseId"`
	Subject        string    `json:"subject"`
	Seq            int       `json:"seq"`
	Heading        string    `json:"heading"`
	Content        string    `json:"content"`
	TokenCount     int       `gorm:"column:token_count" json:"tokenCount"`
	Embedding      []byte    `json:"-"`
	EmbeddingModel string    `gorm:"column:embedding_model" json:"embeddingModel"`
	EmbeddingDim   int       `gorm:"column:embedding_dim" json:"embeddingDim"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"createdAt"`
}

// TableName 固定表名。
func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }
