package dto

// DeptRate 是分教研室覆盖率。
type DeptRate struct {
	Department string  `json:"department"`
	Rate       float64 `json:"rate"`
}

// CoverageDTO 是 GET /supervision/coverage 的响应。
type CoverageDTO struct {
	TotalCourses      int        `json:"totalCourses"`
	SupervisedCourses int        `json:"supervisedCourses"`
	Rate              float64    `json:"rate"`
	ByDepartment      []DeptRate `json:"byDepartment"`
}

// PlanItem 是听评课安排行。
type PlanItem struct {
	ID             uint64 `json:"id"`
	CourseID       uint64 `json:"courseId"`
	CourseName     string `json:"courseName"`
	TeacherName    string `json:"teacherName"`
	SupervisorName string `json:"supervisorName"`
	PlannedDate    string `json:"plannedDate"`
	Status         string `json:"status"`
}

// PlanListQuery 是 GET /supervision/plans 的查询参数。
type PlanListQuery struct {
	Status   string `form:"status" binding:"omitempty,oneof=planned completed"`
	DateFrom string `form:"dateFrom"`
	DateTo   string `form:"dateTo"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}
