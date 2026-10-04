package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// DraftHandler 处理督导评估草稿接口（草稿箱与评估页保存）。
type DraftHandler struct {
	drafts service.DraftService
}

// NewDraftHandler 构造草稿 handler。
func NewDraftHandler(drafts service.DraftService) *DraftHandler {
	return &DraftHandler{drafts: drafts}
}

// Save 处理 PUT /api/v1/sessions/:id/draft（仅 supervisor）。
func (h *DraftHandler) Save(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.DraftUpsertReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	supervisorID, _ := middleware.UserID(c)
	data, err := h.drafts.Save(c.Request.Context(), supervisorID, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// GetBySession 处理 GET /api/v1/sessions/:id/draft（仅 supervisor，无草稿返回 null）。
func (h *DraftHandler) GetBySession(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	supervisorID, _ := middleware.UserID(c)
	data, err := h.drafts.GetBySession(c.Request.Context(), supervisorID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// List 处理 GET /api/v1/drafts（仅 supervisor，按课程名查询）。
func (h *DraftHandler) List(c *gin.Context) {
	var q dto.DraftListQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	supervisorID, _ := middleware.UserID(c)
	data, err := h.drafts.List(c.Request.Context(), supervisorID, q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Delete 处理 DELETE /api/v1/drafts/:id（仅 supervisor）。
func (h *DraftHandler) Delete(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	supervisorID, _ := middleware.UserID(c)
	if err := h.drafts.Delete(c.Request.Context(), supervisorID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// Submit 处理 POST /api/v1/drafts/:id/submit（仅 supervisor，五维齐全后转正式评价）。
func (h *DraftHandler) Submit(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	supervisorID, _ := middleware.UserID(c)
	data, err := h.drafts.Submit(c.Request.Context(), supervisorID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}
