// Package service 是业务层：权限判断、业务校验、事务边界与聚合计算。
// 禁止读写 gin.Context，禁止拼 SQL 字符串。
package service

import (
	"math"

	"aijiaoxue-api/internal/model"
)

// 角色、状态、类型枚举在 service 层统一以常量引用（值定义于 model，供数据层复用）。
const (
	RoleDirector   = model.RoleDirector
	RoleTeacher    = model.RoleTeacher
	RoleSupervisor = model.RoleSupervisor

	CourseStatusOpen   = model.CourseStatusOpen
	CourseStatusDraft  = model.CourseStatusDraft
	CourseStatusClosed = model.CourseStatusClosed

	PlanStatusPlanned   = model.PlanStatusPlanned
	PlanStatusCompleted = model.PlanStatusCompleted
)

// CurrentSemester 是 Sprint 1 的当前学期（覆盖率等口径的分母范围）。
const CurrentSemester = "2026-2027-1"

// Semesters 是可选学期列表（Sprint 1 由后端常量维护，前端同源展示）。
var Semesters = []string{"2026-2027-1", "2025-2026-2", "2025-2026-1"}

// Scope 是后端强制的数据范围；零值表示不限制（督导）。
type Scope struct {
	DepartmentID uint64
	TeacherID    uint64
}

// ScopeFor 按角色计算课程数据范围（铁律：裁剪只能发生在后端）。
//
//	director   → 本教研室全部课程
//	teacher    → 本人授课课程
//	supervisor → 全校课程
func ScopeFor(role string, deptID, userID uint64) Scope {
	switch role {
	case RoleDirector:
		return Scope{DepartmentID: deptID}
	case RoleTeacher:
		return Scope{TeacherID: userID}
	default:
		return Scope{}
	}
}

// normalizePage 归一化分页参数：page 默认 1，pageSize 默认 10、上限 50。
func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	return page, pageSize
}

// round2 保留两位小数（覆盖率口径）。
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
