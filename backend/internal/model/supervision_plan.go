package model

import "time"

// SupervisionPlan 对应 supervision_plans 表（听评课安排）。
type SupervisionPlan struct {
	ID           uint64    `gorm:"column:id;primaryKey"`
	CourseID     uint64    `gorm:"column:course_id"`
	SupervisorID uint64    `gorm:"column:supervisor_id"`
	PlannedDate  time.Time `gorm:"column:planned_date"`
	Status       string    `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (SupervisionPlan) TableName() string { return "supervision_plans" }
