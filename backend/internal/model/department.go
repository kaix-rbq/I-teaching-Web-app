// Package model 定义与 MySQL 表一一对应的 GORM 模型。
// 表结构以 database/schema.sql 为唯一事实源，禁止使用 AutoMigrate。
package model

import "time"

// Department 对应 departments 表（教研室）。
type Department struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	Code      string    `gorm:"column:code"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (Department) TableName() string { return "departments" }
