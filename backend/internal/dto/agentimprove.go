package dto

// AgentSuggestionQuery 是 GET /courses/:id/agent-suggestions 的查询参数。
type AgentSuggestionQuery struct {
	Semester string `form:"semester"`
}

// AgentSuggestionDTO 与 frontend/src/types/agent.ts 的 AgentSuggestion 逐字对齐。
// summary/highlights/improvements 直接取 evaluations 同名列；evidence 是后端渲染后的
// 人类可读字符串（禁止把 evidence JSON 原样返回）。
type AgentSuggestionDTO struct {
	ID           uint64  `json:"id"`
	CourseID     uint64  `json:"courseId"`
	SessionID    uint64  `json:"sessionId"`
	SessionDate  string  `json:"sessionDate"`
	Period       string  `json:"period"`
	Topic        string  `json:"topic"`
	Summary      string  `json:"summary"`
	Highlights   string  `json:"highlights"`
	Improvements string  `json:"improvements"`
	Evidence     string  `json:"evidence"`
	ModelVersion string  `json:"modelVersion"`
	Confidence   float64 `json:"confidence"`
	CreatedAt    string  `json:"createdAt"`
}

// ScoreTrendQuery 是 GET /teachers/:id/score-trend 的查询参数。
// Dimension 缺省返回全部序列；合法值见 service 层固定序列表。
type ScoreTrendQuery struct {
	Semester  string `form:"semester"`
	CourseID  uint64 `form:"courseId"`
	Dimension string `form:"dimension"`
}

// ScoreTrendPointDTO 是趋势序列中的单个数据点；Value 为 nil 表示该课次无 AI 评价，
// 前端据此断开折线，不得补 0。
type ScoreTrendPointDTO struct {
	Label string   `json:"label"`
	Date  string   `json:"date"`
	Value *float64 `json:"value"`
}

// ScoreTrendSeriesDTO 与 frontend/src/types/teacher.ts 的 ScoreTrendSeries 逐字对齐。
type ScoreTrendSeriesDTO struct {
	Key           string               `json:"key"`
	Name          string               `json:"name"`
	IsObservation bool                 `json:"isObservation"`
	Points        []ScoreTrendPointDTO `json:"points"`
}
