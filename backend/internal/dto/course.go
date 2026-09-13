package dto

// CourseListQuery 是 GET /courses 的查询参数。
type CourseListQuery struct {
	Semester     string `form:"semester"`
	DepartmentID uint64 `form:"departmentId"`
	TeacherID    uint64 `form:"teacherId"`
	Status       string `form:"status" binding:"omitempty,oneof=open draft closed"`
	Keyword      string `form:"keyword"`
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
}

// CourseListItem 是课程列表行。
// 说明：credit/hours/description/teacherId/departmentId 为满足前端列表与编辑回填的字段，
// 仍全部来自 courses 主表，不引入冗余汇总列。
type CourseListItem struct {
	ID            uint64 `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Credit        int    `json:"credit"`
	Hours         int    `json:"hours"`
	Description   string `json:"description"`
	TeacherID     uint64 `json:"teacherId"`
	TeacherName   string `json:"teacherName"`
	DepartmentID  uint64 `json:"departmentId"`
	Department    string `json:"department"`
	Semester      string `json:"semester"`
	ClassCount    int    `json:"classCount"`
	StudentCount  int    `json:"studentCount"`
	ResourceCount int    `json:"resourceCount"`
	Status        string `json:"status"`
}

// ClassInfoDTO 是课程详情中的开课班级。
type ClassInfoDTO struct {
	ID           uint64 `json:"id"`
	ClassName    string `json:"className"`
	Schedule     string `json:"schedule"`
	Location     string `json:"location"`
	StudentCount int    `json:"studentCount"`
}

// CourseDetail 是 GET /courses/:id 的响应。
type CourseDetail struct {
	CourseListItem
	Classes []ClassInfoDTO `json:"classes"`
}

// CourseUpsertReq 是 POST/PUT /courses 的请求体。
type CourseUpsertReq struct {
	Code         string `json:"code" binding:"required,max=12,coursecode"`
	Name         string `json:"name" binding:"required,max=128"`
	DepartmentID uint64 `json:"departmentId" binding:"required"`
	TeacherID    uint64 `json:"teacherId" binding:"required"`
	Semester     string `json:"semester" binding:"required,max=16"`
	Credit       int    `json:"credit" binding:"required,min=1,max=6"`
	Hours        int    `json:"hours" binding:"required,min=16,max=128"`
	Description  string `json:"description" binding:"max=500"`
	Status       string `json:"status" binding:"required,oneof=open draft closed"`
}
