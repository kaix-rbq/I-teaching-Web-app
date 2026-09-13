// Package dto 定义请求与响应结构体。
// json tag 必须与前端 src/types/ 字段名逐字对齐（camelCase）。
package dto

// LoginReq 是 POST /auth/login 的请求体。
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserDTO 是登录与 /auth/me 返回的用户信息。
type UserDTO struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	DepartmentID uint64 `json:"departmentId,omitempty"`
	Department   string `json:"department,omitempty"`
	JobNo        string `json:"jobNo,omitempty"`
}

// LoginResp 是登录成功的响应。
type LoginResp struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}
