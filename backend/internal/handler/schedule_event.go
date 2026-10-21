package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// ScheduleEventHandler 处理主任事务（周视图事务管理卡）。
type ScheduleEventHandler struct {
	svc service.ScheduleEventService
}

// NewScheduleEventHandler 构造主任事务 handler。
func NewScheduleEventHandler(svc service.ScheduleEventService) *ScheduleEventHandler {
	return &ScheduleEventHandler{svc: svc}
}

// List 处理 GET /api/v1/schedule-events?from=&to=。
func (h *ScheduleEventHandler) List(c *gin.Context) {
	var q dto.ScheduleEventListQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	data, err := h.svc.List(c.Request.Context(), role, uid, q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Create 处理 POST /api/v1/schedule-events。
func (h *ScheduleEventHandler) Create(c *gin.Context) {
	var req dto.ScheduleEventCreateReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	data, err := h.svc.Create(c.Request.Context(), role, uid, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Update 处理 PATCH /api/v1/schedule-events/:id（仅备注）。
func (h *ScheduleEventHandler) Update(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ScheduleEventRemarkReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	data, err := h.svc.UpdateRemark(c.Request.Context(), role, uid, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Delete 处理 DELETE /api/v1/schedule-events/:id。
func (h *ScheduleEventHandler) Delete(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	if err := h.svc.Delete(c.Request.Context(), role, uid, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"id": id})
}
