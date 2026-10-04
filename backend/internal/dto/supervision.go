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
// SessionID 为关联授课记录（null = 尚无授课记录，需先创建再评估）；
// Evaluated 表示关联授课记录是否已完成督导评价（前端据此暴露/隐藏「去评估」入口）。
type PlanItem struct {
	ID             uint64  `json:"id"`
	CourseID       uint64  `json:"courseId"`
	CourseName     string  `json:"courseName"`
	TeacherName    string  `json:"teacherName"`
	SupervisorName string  `json:"supervisorName"`
	PlannedDate    string  `json:"plannedDate"`
	Status         string  `json:"status"`
	SessionID      *uint64 `json:"sessionId"`
	Evaluated      bool    `json:"evaluated"`
}

// PlanListQuery 是 GET /supervision/plans 的查询参数。
type PlanListQuery struct {
	Status   string `form:"status" binding:"omitempty,oneof=planned completed"`
	DateFrom string `form:"dateFrom"`
	DateTo   string `form:"dateTo"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}
