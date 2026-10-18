package model

import "time"

// CourseTimetable 对应 course_timetable 表：最小课表（B11）。
// 仅用于创建授课记录时的课程/班级下拉，不做同步与冲突校验。
type CourseTimetable struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	CourseID  uint64    `gorm:"column:course_id"`
	ClassID   uint64    `gorm:"column:class_id"`
	TeacherID uint64    `gorm:"column:teacher_id"`
	Semester  string    `gorm:"column:semester"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (CourseTimetable) TableName() string { return "course_timetable" }
