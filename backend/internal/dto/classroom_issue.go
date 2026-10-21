package dto

// ClassroomIssueDTO 是课堂问题登记行。
type ClassroomIssueDTO struct {
	ID          uint64 `json:"id"`
	TeacherID   uint64 `json:"teacherId"`
	TeacherName string `json:"teacherName"`
	IssueType   string `json:"issueType"` // equipment | absence | progress | material | other
	Content     string `json:"content"`
	Status      string `json:"status"` // open | handled
	HandleNote  string `json:"handleNote"`
	HandlerID   uint64 `json:"handlerId"`
	HandlerName string `json:"handlerName"`
	HandledAt   string `json:"handledAt"`
	CreatedAt   string `json:"createdAt"`
}

// ClassroomIssueCreateReq 是 POST /classroom-issues 的请求体（教师）。
type ClassroomIssueCreateReq struct {
	IssueType string `json:"issueType" binding:"required,oneof=equipment absence progress material other"`
	Content   string `json:"content" binding:"required,max=2000"`
}

// ClassroomIssueHandleReq 是 POST /classroom-issues/:id/handle 的请求体（督导/主任）。
type ClassroomIssueHandleReq struct {
	HandleNote string `json:"handleNote" binding:"required,max=1000"`
}

// ClassroomIssueConfirmReq 是 POST /classroom-issues/:id/confirm 的请求体（教师）。
type ClassroomIssueConfirmReq struct {
	Resolved bool `json:"resolved"`
}
