package model

import "time"

// UserNotification 对应 user_notifications 表：站内信（B12）。
// 只推 supervisor_comment（督导评语）与 ai_issue（AI 发现问题）；
// 进入详情自动已读，红点收敛到工作台清单与督导任务队列两处。
type UserNotification struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	UserID    uint64    `gorm:"column:user_id"`
	Type      string    `gorm:"column:type"`
	RefID     uint64    `gorm:"column:ref_id"`
	Payload   *string   `gorm:"column:payload"`
	IsRead    bool      `gorm:"column:is_read"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// TableName 指定表名。
func (UserNotification) TableName() string { return "user_notifications" }

// 站内信类型。
const (
	NotificationTypeSupervisorComment = "supervisor_comment"
	NotificationTypeAIIssue           = "ai_issue"
)
