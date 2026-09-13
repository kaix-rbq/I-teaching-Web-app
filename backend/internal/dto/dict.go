package dto

// DeptOption 是教研室字典项。
type DeptOption struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// TeacherOption 是教师字典项。
// departmentId 为前端「按教研室过滤教师下拉」所需的补充字段（与前端的 User.departmentId 对齐）。
type TeacherOption struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	DepartmentID uint64 `json:"departmentId,omitempty"`
}
