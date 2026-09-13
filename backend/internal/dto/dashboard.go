package dto

// DirectorDashboard 是主任工作台聚合数据。
type DirectorDashboard struct {
	CourseCount   int              `json:"courseCount"`
	TeacherCount  int              `json:"teacherCount"`
	ClassCount    int              `json:"classCount"`
	ResourceCount int              `json:"resourceCount"`
	RecentCourses []CourseListItem `json:"recentCourses"`
}

// TeacherDashboard 是教师工作台聚合数据。
// RecentResources 为前端 §7.3「近期上传资源」侧栏所需的补充字段。
type TeacherDashboard struct {
	CourseCount     int              `json:"courseCount"`
	ClassCount      int              `json:"classCount"`
	StudentCount    int              `json:"studentCount"`
	ResourceCount   int              `json:"resourceCount"`
	MyCourses       []CourseListItem `json:"myCourses"`
	RecentResources []ResourceDTO    `json:"recentResources"`
}

// SupervisorDashboard 是督导工作台聚合数据。
// ByDepartment 为前端 CoverageCard「按教研室覆盖率排行」所需的补充字段。
type SupervisorDashboard struct {
	CourseCount    int        `json:"courseCount"`
	PlanCount      int        `json:"planCount"`
	CompletedCount int        `json:"completedCount"`
	CoverageRate   float64    `json:"coverageRate"`
	RecentPlans    []PlanItem `json:"recentPlans"`
	ByDepartment   []DeptRate `json:"byDepartment"`
}
