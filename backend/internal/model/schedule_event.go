package model

import "time"

// ScheduleEvent 对应 schedule_events 表：主任事务（会议/公开课/讲座/其他）。
// start_time/end_time 用字符串保存 HH:MM，避免 MySQL TIME 与 Go time.Time 的时区转换歧义。
type ScheduleEvent struct {
	ID          uint64    `gorm:"column:id;primaryKey"`
	OwnerUserID uint64    `gorm:"column:owner_user_id"`
	EventDate   time.Time `gorm:"column:event_date"`
	StartTime   string    `gorm:"column:start_time"`
	EndTime     string    `gorm:"column:end_time"`
	EventType   string    `gorm:"column:event_type"`
	TypeText    string    `gorm:"column:type_text"`
	Remark      string    `gorm:"column:remark"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (ScheduleEvent) TableName() string { return "schedule_events" }

// 事务类型。
const (
	ScheduleEventMeeting   = "meeting"
	ScheduleEventOpenClass = "open_class"
	ScheduleEventLecture   = "lecture"
	ScheduleEventOther     = "other"
)
