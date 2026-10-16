package model

import "time"

// ImprovementItem 对应 improvement_items 表：待改进清单（N4 简化版）。
// 状态机 open → done_self → closed；不支持打回，打回即退回 open 并留言。
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
