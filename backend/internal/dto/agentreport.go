package dto

// AiReportDTO 是教师级综合报告（teacher_ai_reports，teacher_id+semester+course_id=0 唯一）。
// 字段与 docs/backend_AGENTS.md §8.15 逐字一致；JSON 数组字段缺失时由 service 归一化为 []。
type AiReportDTO struct {
	TeacherID        uint64   `json:"teacherId"`
	Semester         string   `json:"semester"`
	CourseID         uint64   `json:"courseId"`
	Summary          string   `json:"summary"`
	Directions       []string `json:"directions"`
	Suggestions      []string `json:"suggestions"`
	Highlights       string   `json:"highlights"`
	TrendComment     string   `json:"trendComment"`
	Confidence       *float64 `json:"confidence"`
	EvaluatedCount   int      `json:"evaluatedCount"`
	SessionCount     int      `json:"sessionCount"`
	SourceSessionIDs []uint64 `json:"sourceSessionIds"`
	ModelVersion     string   `json:"modelVersion"`
	PromptVersion    string   `json:"promptVersion"`
	Status           string   `json:"status"`
	ErrorMessage     string   `json:"errorMessage"`
	GeneratedAt      string   `json:"generatedAt"`
}

// AiReportEnvelope 是 GET /teachers/:id/ai-report 响应；无报告时 report=null、status=pending。
type AiReportEnvelope struct {
	Status       string       `json:"status"`
	Report       *AiReportDTO `json:"report"`
	ErrorMessage string       `json:"errorMessage"`
}

// AiReportRegenerateResp 是 POST /teachers/:id/ai-report/regenerate 的响应。
type AiReportRegenerateResp struct {
	Status string `json:"status"`
}
