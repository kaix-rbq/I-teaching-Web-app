package model

// 角色标识（与 JWT claims、RBAC 中间件、数据裁剪共用）。
const (
	RoleDirector   = "director"
	RoleTeacher    = "teacher"
	RoleSupervisor = "supervisor"
)

// 课程开课状态。
const (
	CourseStatusOpen   = "open"
	CourseStatusDraft  = "draft"
	CourseStatusClosed = "closed"
)

// 听评课状态。
const (
	PlanStatusPlanned   = "planned"
	PlanStatusCompleted = "completed"
)

// 授课记录状态。
const (
	SessionStatusScheduled = "scheduled"
	SessionStatusRecorded  = "recorded"
	SessionStatusEvaluated = "evaluated"
)

// 评价来源（evaluator_type）：supervisor 与 agent 同表同结构，聚合层只认该枚举。
const (
	EvaluatorSupervisor = "supervisor"
	EvaluatorAgent      = "agent"
)

// 资源类型（按扩展名映射）。
const (
	ResourceTypePDF   = "pdf"
	ResourceTypeDoc   = "doc"
	ResourceTypePPT   = "ppt"
	ResourceTypeVideo = "video"
	ResourceTypeZip   = "zip"
	ResourceTypeOther = "other"
)
