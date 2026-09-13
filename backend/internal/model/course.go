package model

import "time"

// Course 对应 courses 表（课程主表）。
type Course struct {
	ID           uint64    `gorm:"column:id;primaryKey"`
	Code         string    `gorm:"column:code"`
	Name         string    `gorm:"column:name"`
	Credit       int       `gorm:"column:credit"`
	Hours        int       `gorm:"column:hours"`
	Semester     string    `gorm:"column:semester"`
	DepartmentID uint64    `gorm:"column:department_id"`
	TeacherID    uint64    `gorm:"column:teacher_id"`
	Description  string    `gorm:"column:description"`
	Status       string    `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (Course) TableName() string { return "courses" }
