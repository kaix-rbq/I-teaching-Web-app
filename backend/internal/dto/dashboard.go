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
// 只保留督导核心任务「记录课程并评估」所需的信息：待评课队列 + 最近草稿。
// 课程数 / 覆盖率等统计已移除（与听评课核心任务无关）。
type SupervisorDashboard struct {
	RecentPlans     []PlanItem           `json:"recentPlans"`
	RecentDrafts    []DraftDTO           `json:"recentDrafts"`
	PendingSessions []PendingSessionItem `json:"pendingSessions"`
}
