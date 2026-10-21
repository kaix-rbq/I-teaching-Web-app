package model

import "time"

// ClassroomIssue 对应 classroom_issues 表：课堂问题登记。
// 状态机：open（进行中）→ handled（待教师确认）→ 教师确认解决即删除 / 未解决回退 open。
type ClassroomIssue struct {
	ID         uint64     `gorm:"column:id;primaryKey"`
	TeacherID  uint64     `gorm:"column:teacher_id"`
	IssueType  string     `gorm:"column:issue_type"`
	Content    string     `gorm:"column:content"`
	Status     string     `gorm:"column:status"`
	HandleNote string     `gorm:"column:handle_note"`
	HandlerID  uint64     `gorm:"column:handler_id"`
	HandledAt  *time.Time `gorm:"column:handled_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (ClassroomIssue) TableName() string { return "classroom_issues" }

// 课堂问题类型与状态。
const (
	IssueTypeEquipment = "equipment"
	IssueTypeAbsence   = "absence"
	IssueTypeProgress  = "progress"
	IssueTypeMaterial  = "material"
	IssueTypeOther     = "other"

	IssueStatusOpen    = "open"
	IssueStatusHandled = "handled"
)
