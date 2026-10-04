package dto

// DraftUpsertReq 是 PUT /sessions/:id/draft 的请求体：保存督导评估草稿。
// 与正式提交不同，草稿允许任意维度为空（可先写一半），提交时再强制五维齐全。
type DraftUpsertReq struct {
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

// DraftListQuery 是 GET /drafts 的查询参数（可按课程名模糊查询）。
type DraftListQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// DraftDTO 是一条草稿的完整展示（草稿箱列表 + 编辑回填 + 工作台最近草稿）。
type DraftDTO struct {
	ID           uint64 `json:"id"`
	SessionID    uint64 `json:"sessionId"`
	CourseID     uint64 `json:"courseId"`
	CourseCode   string `json:"courseCode"`
	CourseName   string `json:"courseName"`
	TeacherName  string `json:"teacherName"`
	SessionDate  string `json:"sessionDate"`
	Period       string `json:"period"`
	Topic        string `json:"topic"`
	Objective    *int   `json:"objective"`
	Content      *int   `json:"content"`
	Interaction  *int   `json:"interaction"`
	Organization *int   `json:"organization"`
	Frontier     *int   `json:"frontier"`
	Comment      string `json:"comment"`
	Highlights   string `json:"highlights"`
	Improvements string `json:"improvements"`
	Suggestions  string `json:"suggestions"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}
