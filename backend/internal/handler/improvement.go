package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// ImprovementHandler 处理待改进清单（N4）。
type ImprovementHandler struct {
	svc service.ImprovementService
}

// NewImprovementHandler 构造待改进清单 handler。
func NewImprovementHandler(svc service.ImprovementService) *ImprovementHandler {
	return &ImprovementHandler{svc: svc}
}

// List 处理 GET /api/v1/improvement-items。
func (h *ImprovementHandler) List(c *gin.Context) {
	var q dto.ImprovementListQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	data, err := h.svc.List(c.Request.Context(), role, userID, q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Create 处理 POST /api/v1/improvement-items（仅教师）。
func (h *ImprovementHandler) Create(c *gin.Context) {
	var req dto.ImprovementCreateReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	data, err := h.svc.Create(c.Request.Context(), role, userID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Update 处理 PUT /api/v1/improvement-items/:id（仅教师）。
func (h *ImprovementHandler) Update(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ImprovementUpdateReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	data, err := h.svc.Update(c.Request.Context(), role, userID, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Complete 处理 PUT /api/v1/improvement-items/:id/complete（教师自评完成）。
func (h *ImprovementHandler) Complete(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ImprovementCompleteReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	data, err := h.svc.Complete(c.Request.Context(), role, userID, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Delete 处理 DELETE /api/v1/improvement-items/:id（仅创建教师本人）。
func (h *ImprovementHandler) Delete(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	if err := h.svc.Delete(c.Request.Context(), role, userID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"id": id})
}

// ReflectionHandler 处理教学反思（N5）。
type ReflectionHandler struct {
	svc service.ReflectionService
}

// NewReflectionHandler 构造教学反思 handler。
func NewReflectionHandler(svc service.ReflectionService) *ReflectionHandler {
	return &ReflectionHandler{svc: svc}
}

// List 处理 GET /api/v1/teaching-reflections。
func (h *ReflectionHandler) List(c *gin.Context) {
	var q dto.ReflectionListQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	data, err := h.svc.List(c.Request.Context(), role, userID, q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Create 处理 POST /api/v1/teaching-reflections。
func (h *ReflectionHandler) Create(c *gin.Context) {
	var req dto.ReflectionUpsertReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	data, err := h.svc.Create(c.Request.Context(), role, userID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Update 处理 PUT /api/v1/teaching-reflections/:id。
func (h *ReflectionHandler) Update(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ReflectionUpsertReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	data, err := h.svc.Update(c.Request.Context(), role, userID, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// NotificationHandler 处理站内信（B12）。
type NotificationHandler struct {
	svc service.NotificationService
}

// NewNotificationHandler 构造站内信 handler。
func NewNotificationHandler(svc service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// List 处理 GET /api/v1/notifications。
func (h *NotificationHandler) List(c *gin.Context) {
	var q dto.NotificationListQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	userID, _ := middleware.UserID(c)
	data, err := h.svc.List(c.Request.Context(), userID, q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// MarkAllRead 处理 PUT /api/v1/notifications/read-all（注意注册顺序先于 /:id/read）。
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	if err := h.svc.MarkAllRead(c.Request.Context(), userID); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}

// MarkRead 处理 PUT /api/v1/notifications/:id/read。
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	userID, _ := middleware.UserID(c)
	if err := h.svc.MarkRead(c.Request.Context(), userID, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{})
}
