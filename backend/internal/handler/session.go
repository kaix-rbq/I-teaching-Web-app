package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// SessionHandler 处理授课记录与督导评分接口（开发计划 §4.1）。
type SessionHandler struct {
	sessions   service.SessionService
	recordings service.RecordingService
}

// NewSessionHandler 构造授课记录 handler。
func NewSessionHandler(sessions service.SessionService, recordings ...service.RecordingService) *SessionHandler {
	h := &SessionHandler{sessions: sessions}
	if len(recordings) > 0 {
		h.recordings = recordings[0]
	}
	return h
}

// Create 处理 POST /api/v1/sessions（仅 supervisor，S6.1）。
func (h *SessionHandler) Create(c *gin.Context) {
	var req dto.SessionCreateReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.sessions.Create(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

// ListByCourse 处理 GET /api/v1/courses/:id/sessions（登录，数据裁剪，S6.2）。
func (h *SessionHandler) ListByCourse(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var q dto.SessionListQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	result, err := h.sessions.ListByCourse(c.Request.Context(), role, middleware.DeptID(c), userID, id, q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

// Detail 处理 GET /api/v1/sessions/:id（登录，数据裁剪）。
func (h *SessionHandler) Detail(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	detail, err := h.sessions.Detail(c.Request.Context(), role, middleware.DeptID(c), userID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

// Evaluation 处理 GET /api/v1/sessions/:id/evaluation（登录，数据裁剪）。
func (h *SessionHandler) Evaluation(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	data, err := h.sessions.Evaluation(c.Request.Context(), role, middleware.DeptID(c), userID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if h.recordings != nil {
		// 录音/转写是阶段②增强项：其查询失败（如迁移未执行）不得阻断阶段①评估数据展示。
		recording, transcript, mediaErr := h.recordings.Media(c.Request.Context(), role, middleware.DeptID(c), userID, id)
		if mediaErr == nil {
			data.Recording, data.Transcript = recording, transcript
		}
	}
	response.OK(c, data)
}

// Submit 处理 PUT /api/v1/sessions/:id/supervisor-evaluation（仅 supervisor，S6.3）。
// PUT 幂等覆盖：重复提交整行更新而非报错（§3.1 无草稿态约定）。
func (h *SessionHandler) Submit(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.SupervisorEvaluationReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	supervisorID, _ := middleware.UserID(c)

	data, err := h.sessions.SubmitSupervisorEvaluation(c.Request.Context(), supervisorID, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}
