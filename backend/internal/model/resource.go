package model

import "time"

// Resource 对应 resources 表（课程资源）。
type Resource struct {
	ID         uint64    `gorm:"column:id;primaryKey"`
	CourseID   uint64    `gorm:"column:course_id"`
	Name       string    `gorm:"column:name"`
	Type       string    `gorm:"column:type"`
	Size       int64     `gorm:"column:size"`
	FilePath   string    `gorm:"column:file_path"`
	UploaderID uint64    `gorm:"column:uploader_id"`
	UploadedAt time.Time `gorm:"column:uploaded_at"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (Resource) TableName() string { return "resources" }
