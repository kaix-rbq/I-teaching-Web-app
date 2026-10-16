package model

import "time"

// TeachingReflection 对应 teaching_reflections 表：教师私有的教学反思（N5）。
// source_eval_id 指向被收藏的评价行（evaluations.id）；督导在关闭改进项时可读关联反思。
type TeachingReflection struct {
	ID           uint64    `gorm:"column:id;primaryKey"`
	TeacherID    uint64    `gorm:"column:teacher_id"`
	SessionID    uint64    `gorm:"column:session_id"`
	SourceEvalID uint64    `gorm:"column:source_eval_id"`
	Content      string    `gorm:"column:content"`
	Plan         string    `gorm:"column:plan"`
	NextAction   string    `gorm:"column:next_action"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (TeachingReflection) TableName() string { return "teaching_reflections" }
