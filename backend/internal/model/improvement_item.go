package model

import "time"

// ImprovementItem 对应 improvement_items 表：待改进清单。
// 2026-10-20 简化：教师自评完成直接 closed，不再经过 done_self / 督导确认。
// done_self 仅用于兼容历史数据。
type ImprovementItem struct {
	ID        uint64     `gorm:"column:id;primaryKey"`
	TeacherID uint64     `gorm:"column:teacher_id"`
	SessionID uint64     `gorm:"column:session_id"`
	Title     string     `gorm:"column:title"`
	Source    string     `gorm:"column:source"`
	Status    string     `gorm:"column:status"`
	DoneNote  string     `gorm:"column:done_note"`
	ClosedBy  *uint64    `gorm:"column:closed_by"`
	ClosedAt  *time.Time `gorm:"column:closed_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (ImprovementItem) TableName() string { return "improvement_items" }

// 改进项状态。
const (
	ImprovementStatusOpen     = "open"
	ImprovementStatusDoneSelf = "done_self"
	ImprovementStatusClosed   = "closed"
)

// 改进项来源。
const (
	ImprovementSourceAI         = "ai"
	ImprovementSourceSupervisor = "supervisor"
	ImprovementSourceSelf       = "self"
)
