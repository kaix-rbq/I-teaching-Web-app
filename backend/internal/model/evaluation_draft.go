package model

import "time"

// EvaluationDraft 对应 evaluation_drafts 表（督导评估草稿）。
// 与 Evaluation 不同：草稿是未生效的私人工作副本，维度分允许 NULL、无总分与口径版本，
// 不进入任何聚合；提交后才写入 evaluations 并删除本行。
type EvaluationDraft struct {
	ID                uint64    `gorm:"column:id;primaryKey"`
	SessionID         uint64    `gorm:"column:session_id"`
	SupervisorID      uint64    `gorm:"column:supervisor_id"`
	ObjectiveScore    *uint8    `gorm:"column:objective_score"`
	ContentScore      *uint8    `gorm:"column:content_score"`
	InteractionScore  *uint8    `gorm:"column:interaction_score"`
	OrganizationScore *uint8    `gorm:"column:organization_score"`
	FrontierScore     *uint8    `gorm:"column:frontier_score"`
	Comment           string    `gorm:"column:comment"`
	Highlights        string    `gorm:"column:highlights"`
	Improvements      string    `gorm:"column:improvements"`
	Suggestions       string    `gorm:"column:suggestions"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (EvaluationDraft) TableName() string { return "evaluation_drafts" }
