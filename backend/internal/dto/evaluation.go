package dto

// SessionListQuery 是 GET /courses/:id/sessions 的查询参数。
type SessionListQuery struct {
	Semester string `form:"semester"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// SessionCreateReq 是 POST /sessions 的请求体（督导创建授课记录）。
type SessionCreateReq struct {
	CourseID    uint64  `json:"courseId" binding:"required"`
	ClassID     uint64  `json:"classId"`
	SessionDate string  `json:"sessionDate" binding:"required"`
	Period      string  `json:"period" binding:"required,max=32"`
	Topic       string  `json:"topic" binding:"max=128"`
	PlanID      *uint64 `json:"planId"`
}

// SessionDetail 是 GET /sessions/:id 与创建后的响应。
type SessionDetail struct {
	ID          uint64 `json:"id"`
	CourseID    uint64 `json:"courseId"`
	CourseCode  string `json:"courseCode"`
	CourseName  string `json:"courseName"`
	ClassID     uint64 `json:"classId"`
	ClassName   string `json:"className"`
	TeacherID   uint64 `json:"teacherId"`
	TeacherName string `json:"teacherName"`
	Semester    string `json:"semester"`
	SessionDate string `json:"sessionDate"`
	Period      string `json:"period"`
	Topic       string `json:"topic"`
	Status      string `json:"status"`
}

// SessionListItem 是课程历史授课记录行，附双侧评分摘要（无评价侧为 null）。
type SessionListItem struct {
	ID              uint64   `json:"id"`
	SessionDate     string   `json:"sessionDate"`
	Period          string   `json:"period"`
	Topic           string   `json:"topic"`
	Status          string   `json:"status"`
	SupervisorScore *float64 `json:"supervisorScore"`
	AgentScore      *float64 `json:"agentScore"`
	EvaluationCount int      `json:"evaluationCount"`
}

// SupervisorEvaluationReq 是 PUT /sessions/:id/supervisor-evaluation 的请求体。
// 督导评分强制 5 个维度全部录入：维度分 1-5 整数，任一维度缺失返回 40002，越界返回 40001。
// （仅智能体侧允许维度为 null，见开发计划 §2.1。）
type SupervisorEvaluationReq struct {
	Objective    *int   `json:"objective" binding:"omitempty,min=1,max=5"`
	Content      *int   `json:"content" binding:"omitempty,min=1,max=5"`
	Interaction  *int   `json:"interaction" binding:"omitempty,min=1,max=5"`
	Organization *int   `json:"organization" binding:"omitempty,min=1,max=5"`
	Frontier     *int   `json:"frontier" binding:"omitempty,min=1,max=5"`
	Comment      string `json:"comment" binding:"max=2000"`
	Highlights   string `json:"highlights" binding:"max=1000"`
	Improvements string `json:"improvements" binding:"max=1000"`
	Suggestions  string `json:"suggestions" binding:"max=1000"`
}

// EvaluationDTO 是一条评价的完整展示（督导评分表单回填 / 智能体参考面板共用）。
type EvaluationDTO struct {
	EvaluatorID    uint64   `json:"evaluatorId"`
	EvaluatorName  string   `json:"evaluatorName"`
	EvaluatorType  string   `json:"evaluatorType"`
	AIModelVersion string   `json:"aiModelVersion"`
	AIConfidence   *float64 `json:"aiConfidence"`
	FormulaVersion string   `json:"formulaVersion"`
	Objective      *int     `json:"objective"`
	Content        *int     `json:"content"`
	Interaction    *int     `json:"interaction"`
	Organization   *int     `json:"organization"`
	Frontier       *int     `json:"frontier"`
	TotalScore     *float64 `json:"totalScore"`
	Comment        string   `json:"comment"`
	Highlights     string   `json:"highlights"`
	Improvements   string   `json:"improvements"`
	Suggestions    string   `json:"suggestions"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}

// SessionEvaluation 是 GET /sessions/:id/evaluation 的响应：当堂课评估页聚合。
// 阶段二将追加 recording / transcript 字段，阶段一保持 null。
type SessionEvaluation struct {
	Session          SessionDetail   `json:"session"`
	SupervisorScores []EvaluationDTO `json:"supervisorScores"`
	AgentScore       *EvaluationDTO  `json:"agentScore"`
}

// ScoreWeights 是综合分的双侧权重展示（α 与 1−α）。
type ScoreWeights struct {
	Supervisor float64 `json:"supervisor"`
	Agent      float64 `json:"agent"`
}

// DimensionScoreDTO 是聚合后的单个维度（0-100 制；缺侧为 null）。
type DimensionScoreDTO struct {
	Key             string   `json:"key"`
	Name            string   `json:"name"`
	Weight          float64  `json:"weight"`
	IsObservation   bool     `json:"isObservation"`
	Score           *float64 `json:"score"`
	SupervisorScore *float64 `json:"supervisorScore"`
	AgentScore      *float64 `json:"agentScore"`
}

// SampleDTO 是样本量与覆盖度（alignedCount 必须暴露，§2.5.3）。
// sampleSufficient 以 evaluatedCount（已评价场次）为准（§2.5.5）。
type SampleDTO struct {
	SessionCount     int  `json:"sessionCount"`
	EvaluatedCount   int  `json:"evaluatedCount"`
	SupervisorCount  int  `json:"supervisorCount"`
	AgentCount       int  `json:"agentCount"`
	AlignedCount     int  `json:"alignedCount"`
	SampleSufficient bool `json:"sampleSufficient"`
}

// ScoreSummary 是课程级与教师级共用的评分聚合结果（§4.2）。
type ScoreSummary struct {
	CompositeScore  *float64            `json:"compositeScore"`
	SupervisorScore *float64            `json:"supervisorScore"`
	AgentScore      *float64            `json:"agentScore"`
	Dimensions      []DimensionScoreDTO `json:"dimensions"`
	Sample          SampleDTO           `json:"sample"`
	Flags           []string            `json:"flags"`
	Weights         ScoreWeights        `json:"weights"`
	FormulaVersion  string              `json:"formulaVersion"`
}

// TeacherScoreListQuery 是教师评分列表的查询参数。
type TeacherScoreListQuery struct {
	DepartmentID uint64 `form:"departmentId"`
	Semester     string `form:"semester"`
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
}

// TeacherScoreItem 是教师管理列表行（默认按姓名排序，n<3 标注样本不足）。
type TeacherScoreItem struct {
	TeacherID   uint64 `json:"teacherId"`
	TeacherName string `json:"teacherName"`
	JobNo       string `json:"jobNo"`
	ScoreSummary
}

// CourseScoreItem 是教师面板「按课程明细」的行。
type CourseScoreItem struct {
	CourseID   uint64 `json:"courseId"`
	CourseCode string `json:"courseCode"`
	CourseName string `json:"courseName"`
	ScoreSummary
}

// TeacherEvaluationSummary 是 GET /teachers/:id/evaluation-summary 的响应。
type TeacherEvaluationSummary struct {
	TeacherID   uint64            `json:"teacherId"`
	TeacherName string            `json:"teacherName"`
	Semester    string            `json:"semester"`
	Courses     []CourseScoreItem `json:"courses"`
	ScoreSummary
}

// CourseEvaluationSummary 是 GET /courses/:id/evaluation-summary 的响应（教学提优页）。
type CourseEvaluationSummary struct {
	CourseID    uint64 `json:"courseId"`
	CourseCode  string `json:"courseCode"`
	CourseName  string `json:"courseName"`
	TeacherID   uint64 `json:"teacherId"`
	TeacherName string `json:"teacherName"`
	Semester    string `json:"semester"`
	ScoreSummary
}

// TeacherEvaluationTimelineQuery 是 GET /teachers/:id/evaluations 的查询参数。
type TeacherEvaluationTimelineQuery struct {
	Semester string `form:"semester"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// TeacherEvaluationTimelineItem 是教师「历次评价时间线」的一行（一场课）。
// 一次拉取即可渲染「时间 / 课程·课次 / 督导评语 / 分数」，无需前端逐场拼装。
type TeacherEvaluationTimelineItem struct {
	SessionID   uint64 `json:"sessionId"`
	SessionDate string `json:"sessionDate"` // YYYY-MM-DD，时间线倒序依据
	Period      string `json:"period"`
	Topic       string `json:"topic"`
	CourseID    uint64 `json:"courseId"`
	CourseCode  string `json:"courseCode"`
	CourseName  string `json:"courseName"`
	Status      string `json:"status"`
	// 场次综合分（双侧融合后加权，口径与教师级聚合一致；无评价为 null）
	CompositeScore *float64 `json:"compositeScore"`
	// 该场督导单次总分均值（同场多督导取 AVG）与该场智能体总分
	SupervisorScore *float64 `json:"supervisorScore"`
	AgentScore      *float64 `json:"agentScore"`
	// 结构化评语明细（督导可能多人；阶段一智能体至多一条）
	SupervisorEvaluations []EvaluationDTO `json:"supervisorEvaluations"`
	AgentEvaluation       *EvaluationDTO  `json:"agentEvaluation"`
}
