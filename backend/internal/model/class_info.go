package model

import "time"

// ClassInfo 对应 course_classes 表（开课班级）。
type ClassInfo struct {
	ID           uint64    `gorm:"column:id;primaryKey"`
	CourseID     uint64    `gorm:"column:course_id"`
	ClassName    string    `gorm:"column:class_name"`
	Schedule     string    `gorm:"column:schedule"`
	Location     string    `gorm:"column:location"`
	StudentCount int       `gorm:"column:student_count"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (ClassInfo) TableName() string { return "course_classes" }
