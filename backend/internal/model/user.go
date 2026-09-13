package model

import "time"

// User 对应 users 表（三角色账号）。
type User struct {
	ID           uint64    `gorm:"column:id;primaryKey"`
	Username     string    `gorm:"column:username"`
	PasswordHash string    `gorm:"column:password_hash"`
	Name         string    `gorm:"column:name"`
	Role         string    `gorm:"column:role"`
	DepartmentID *uint64   `gorm:"column:department_id"`
	JobNo        string    `gorm:"column:job_no"`
	Status       uint8     `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名。
func (User) TableName() string { return "users" }

// DeptID 返回可安全参与数据裁剪的部门 id（督导为 0）。
func (u *User) DeptID() uint64 {
	if u.DepartmentID == nil {
		return 0
	}
	return *u.DepartmentID
}
