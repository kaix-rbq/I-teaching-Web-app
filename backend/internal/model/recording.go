package model

import "time"

type Recording struct {
	ID           uint64    `gorm:"column:id;primaryKey"`
	SessionID    uint64    `gorm:"column:session_id"`
	FilePath     string    `gorm:"column:file_path"`
	OriginalName string    `gorm:"column:original_name"`
	Format       string    `gorm:"column:format"`
	Size         int64     `gorm:"column:size"`
	DurationSec  int       `gorm:"column:duration_sec"`
	UploadedBy   uint64    `gorm:"column:uploaded_by"`
	UploadedAt   time.Time `gorm:"column:uploaded_at"`
}

func (Recording) TableName() string { return "recordings" }

type Transcript struct {
	ID            uint64    `gorm:"column:id;primaryKey"`
	SessionID     uint64    `gorm:"column:session_id"`
	RecordingID   uint64    `gorm:"column:recording_id"`
	Content       string    `gorm:"column:content"`
	Segments      *string   `gorm:"column:segments"`
	Engine        string    `gorm:"column:engine"`
	EngineVersion string    `gorm:"column:engine_version"`
	Status        string    `gorm:"column:status"`
	ErrorMessage  string    `gorm:"column:error_message"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (Transcript) TableName() string { return "transcripts" }

const (
	TranscriptPending = "pending"
	TranscriptRunning = "running"
	TranscriptDone    = "done"
	TranscriptFailed  = "failed"
)
