package dto

// ImprovementItemDTO 是待改进清单行（N4 简化版：标题+来源+完成说明+三态）。
type ImprovementItemDTO struct {
	ID           uint64 `json:"id"`
	TeacherID    uint64 `json:"teacherId"`
	TeacherName  string `json:"teacherName"`
	SessionID    uint64 `json:"sessionId"`
	Title        string `json:"title"`
	Source       string `json:"source"`   // ai / supervisor / self
	Status       string `json:"status"`   // open / done_self / closed
	DoneNote     string `json:"doneNote"` // 完成说明（教师自评）
	ClosedBy     uint64 `json:"closedBy"`
	ClosedByName string `json:"closedByName"`
	ClosedAt     string `json:"closedAt"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// ImprovementCreateReq 是 POST /improvement-items 的请求体（教师创建）。
type ImprovementCreateReq struct {
	Title     string `json:"title" binding:"required,max=255"`
	SessionID uint64 `json:"sessionId"`
	Source    string `json:"source" binding:"omitempty,oneof=ai supervisor self"`
}

// ImprovementUpdateReq 是 PUT /improvement-items/:id 的请求体（教师改标题）。
type ImprovementUpdateReq struct {
	Title string `json:"title" binding:"required,max=255"`
}

// ImprovementCompleteReq 是 PUT /improvement-items/:id/complete 的请求体（教师自评完成）。
type ImprovementCompleteReq struct {
	DoneNote string `json:"doneNote" binding:"max=1000"`
}

// ImprovementCloseReq 是 PUT /improvement-items/:id/close 的请求体（督导确认关闭，可留言）。
type ImprovementCloseReq struct {
	Note string `json:"note" binding:"max=1000"`
}

// ImprovementListQuery 是 GET /improvement-items 的查询参数。
type ImprovementListQuery struct {
	TeacherID uint64 `form:"teacherId"`
	Status    string `form:"status" binding:"omitempty,oneof=open done_self closed"`
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
}

// ReflectionDTO 是教学反思（N5，教师私有）。
type ReflectionDTO struct {
	ID           uint64 `json:"id"`
	TeacherID    uint64 `json:"teacherId"`
	SessionID    uint64 `json:"sessionId"`
	SourceEvalID uint64 `json:"sourceEvalId"`
	Content      string `json:"content"`
	Plan         string `json:"plan"`
	NextAction   string `json:"nextAction"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// ReflectionUpsertReq 是 POST/PUT /teaching-reflections 的请求体。
type ReflectionUpsertReq struct {
	SessionID    uint64 `json:"sessionId"`
	SourceEvalID uint64 `json:"sourceEvalId"`
	Content      string `json:"content" binding:"max=4000"`
	Plan         string `json:"plan" binding:"max=2000"`
	NextAction   string `json:"nextAction" binding:"max=2000"`
}

// ReflectionListQuery 是 GET /teaching-reflections 的查询参数。
// 督导关闭清单时可带 sessionId 读取关联反思（§5 权限矩阵）。
type ReflectionListQuery struct {
	SessionID uint64 `form:"sessionId"`
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
}

// NotificationDTO 是站内信行（B12）。
type NotificationDTO struct {
	ID        uint64 `json:"id"`
	Type      string `json:"type"` // supervisor_comment / ai_issue
	RefID     uint64 `json:"refId"`
	Title     string `json:"title"`
	IsRead    bool   `json:"isRead"`
	CreatedAt string `json:"createdAt"`
}

// NotificationListDTO 是 GET /notifications 的响应（含未读数，红点收敛依据）。
type NotificationListDTO struct {
	List     []NotificationDTO `json:"list"`
	Total    int64             `json:"total"`
	Unread   int64             `json:"unread"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// NotificationListQuery 是 GET /notifications 的查询参数。
type NotificationListQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
}

// DemoSeedResp 是 POST /demo/seed 的响应（B6/N7）。
type DemoSeedResp struct {
	Recordings    int    `json:"recordings"`
	Transcripts   int    `json:"transcripts"`
	Improvements  int    `json:"improvements"`
	Reflections   int    `json:"reflections"`
	Notifications int    `json:"notifications"`
	Message       string `json:"message"`
}
