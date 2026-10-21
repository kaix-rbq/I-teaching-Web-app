package model

import "time"

// CourseTimetable 对应 course_timetable 表：教师周课表（V13 扩展）。
// weekday 1=周一…7=周日；0=历史数据未排课。period_start/period_end 1-11；0=未指定。
type CourseTimetable struct {
	ID          uint64    `gorm:"column:id;primaryKey"`
	CourseID    uint64    `gorm:"column:course_id"`
	ClassID     uint64    `gorm:"column:class_id"`
	TeacherID   uint64    `gorm:"column:teacher_id"`
	Semester    string    `gorm:"column:semester"`
	Weekday     int       `gorm:"column:weekday"`
	PeriodStart int       `gorm:"column:period_start"`
	PeriodEnd   int       `gorm:"column:period_end"`
	Classroom   string    `gorm:"column:classroom"`
	Remark      string    `gorm:"column:remark"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (CourseTimetable) TableName() string { return "course_timetable" }

// 星期与节次边界（service 层校验复用）。
const (
	WeekdayMin = 1
	WeekdayMax = 7
	PeriodMin  = 1
	PeriodMax  = 11
)
