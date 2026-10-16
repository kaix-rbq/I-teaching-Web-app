package model

import "time"

// AiTask 对应 ai_tasks 表：AI 任务（可观测、可重试、幂等）。
// uk_ai_task(kind,ref_type,ref_id) 保证同一目标同一类任务至多一条；
// 重新生成 = 把 status 复位为 pending，而非新增行。
type AiTask struct {
	ID           uint64     `gorm:"column:id;primaryKey"`
	Kind         string     `gorm:"column:kind"`
	RefType      string     `gorm:"column:ref_type"`
	RefID        uint64     `gorm:"column:ref_id"`
	Status       string     `gorm:"column:status"`
	Attempts     uint8      `gorm:"column:attempts"`
	ErrorMessage string     `gorm:"column:error_message"`
	Payload      *string    `gorm:"column:payload"`
	StartedAt    *time.Time `gorm:"column:started_at"`
	FinishedAt   *time.Time `gorm:"column:finished_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (AiTask) TableName() string { return "ai_tasks" }

// TeacherAiReport 对应 teacher_ai_reports 表：教师级 AI 综合报告（S2 产物）。
// JSON 列一律用 *string 承载：零值 string 会让 GORM 写入 ”，MySQL 对 JSON 列拒绝空串（ERROR 3140）。
type TeacherAiReport struct {
	ID               uint64     `gorm:"column:id;primaryKey"`
	TeacherID        uint64     `gorm:"column:teacher_id"`
	Semester         string     `gorm:"column:semester"`
	CourseID         uint64     `gorm:"column:course_id"`
	FormulaVersion   string     `gorm:"column:formula_version"`
	AIModelVersion   string     `gorm:"column:ai_model_version"`
	PromptVersion    string     `gorm:"column:prompt_version"`
	Summary          *string    `gorm:"column:summary"`
	Directions       *string    `gorm:"column:directions"`
	Suggestions      *string    `gorm:"column:suggestions"`
	Highlights       *string    `gorm:"column:highlights"`
	TrendComment     *string    `gorm:"column:trend_comment"`
	Evidence         *string    `gorm:"column:evidence"`
	SourceSessionIDs *string    `gorm:"column:source_session_ids"`
	Sample           *string    `gorm:"column:sample"`
	Confidence       *float64   `gorm:"column:confidence"`
	Status           string     `gorm:"column:status"`
	ErrorMessage     string     `gorm:"column:error_message"`
	GeneratedAt      *time.Time `gorm:"column:generated_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (TeacherAiReport) TableName() string { return "teacher_ai_reports" }

// AI 任务枚举（kind / status）。
const (
	AiTaskKindSessionEval   = "session_eval"
	AiTaskKindTeacherReport = "teacher_report"
	AiTaskKindEmbedDocument = "embed_document"

	AiTaskRefSession  = "session"
	AiTaskRefTeacher  = "teacher"
	AiTaskRefDocument = "document"

	AiTaskStatusPending = "pending"
	AiTaskStatusRunning = "running"
	AiTaskStatusDone    = "done"
	AiTaskStatusFailed  = "failed"

	// AiTaskMaxAttempts 是自动重试上限，达到后置 failed，由人工「重新生成」复位。
	AiTaskMaxAttempts = 3
)

// IsTerminalAiTaskStatus 判断任务是否已到终态（done/failed）。
func IsTerminalAiTaskStatus(status string) bool {
	return status == AiTaskStatusDone || status == AiTaskStatusFailed
}
