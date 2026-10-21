package dto

// ScheduleEventDTO 是主任事务行（GET/POST/PATCH /schedule-events）。
type ScheduleEventDTO struct {
	ID          uint64 `json:"id"`
	OwnerUserID uint64 `json:"ownerUserId"`
	EventDate   string `json:"eventDate"` // YYYY-MM-DD
	StartTime   string `json:"startTime"` // HH:MM
	EndTime     string `json:"endTime"`   // HH:MM（24:00 合法）
	EventType   string `json:"eventType"` // meeting | open_class | lecture | other
	TypeText    string `json:"typeText"`  // 「其他」的自填名称
	Remark      string `json:"remark"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// ScheduleEventCreateReq 是 POST /schedule-events 的请求体。
type ScheduleEventCreateReq struct {
	EventDate string `json:"eventDate" binding:"required"`
	StartTime string `json:"startTime" binding:"required"`
	EndTime   string `json:"endTime" binding:"required"`
	EventType string `json:"eventType" binding:"required,oneof=meeting open_class lecture other"`
	TypeText  string `json:"typeText" binding:"max=32"`
	Remark    string `json:"remark" binding:"max=255"`
}

// ScheduleEventListQuery 是 GET /schedule-events 的查询参数（缺省当前周）。
type ScheduleEventListQuery struct {
	From string `form:"from"`
	To   string `form:"to"`
}

// ScheduleEventRemarkReq 是 PATCH /schedule-events/:id 的请求体（仅备注）。
type ScheduleEventRemarkReq struct {
	Remark string `json:"remark" binding:"max=255"`
}
