package model

import "time"

// TeachingSession 对应 teaching_sessions 表（授课记录：某年某月某日第几节、某班级、某教师上的那一次课）。
// teacher_id 冗余自 courses，避免每次联表；plan_id 反向指向听评课计划（可空）。
type TeachingSession struct {
	ID          uint64    `gorm:"column:id;primaryKey"`
	CourseID    uint64    `gorm:"column:course_id"`
	ClassID     uint64    `gorm:"column:class_id"`
	TeacherID   uint64    `gorm:"column:teacher_id"`
	SessionDate time.Time `gorm:"column:session_date"`
	Period      string    `gorm:"column:period"`
	Topic       string    `gorm:"column:topic"`
	PlanID      *uint64   `gorm:"column:plan_id"`
	Status      string    `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (TeachingSession) TableName() string { return "teaching_sessions" }
