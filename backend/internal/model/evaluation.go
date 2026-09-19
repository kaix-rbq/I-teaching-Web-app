package model

import "time"

// Evaluation 对应 evaluations 表（课堂评价）：督导与智能体同表，靠 EvaluatorType 区分，
// 向同一张表写同样的结构（可插拔 scorer）。维度分 1-5，NULL 表示该评分源无法评价此维度。
type Evaluation struct {
	ID                uint64    `gorm:"column:id;primaryKey"`
	SessionID         uint64    `gorm:"column:session_id"`
	EvaluatorType     string    `gorm:"column:evaluator_type"`
	EvaluatorID       uint64    `gorm:"column:evaluator_id"`
	AIModelVersion    string    `gorm:"column:ai_model_version"`
	FormulaVersion    string    `gorm:"column:formula_version"`
	ObjectiveScore    *uint8    `gorm:"column:objective_score"`
	ContentScore      *uint8    `gorm:"column:content_score"`
	InteractionScore  *uint8    `gorm:"column:interaction_score"`
	OrganizationScore *uint8    `gorm:"column:organization_score"`
	FrontierScore     *uint8    `gorm:"column:frontier_score"`
	TotalScore        *float64  `gorm:"column:total_score"`
	AIConfidence      *float64  `gorm:"column:ai_confidence"`
	Evidence          string    `gorm:"column:evidence"`
	Comment           string    `gorm:"column:comment"`
	Highlights        string    `gorm:"column:highlights"`
	Improvements      string    `gorm:"column:improvements"`
	Suggestions       string    `gorm:"column:suggestions"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (Evaluation) TableName() string { return "evaluations" }
