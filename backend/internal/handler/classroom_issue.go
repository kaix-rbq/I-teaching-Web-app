package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// ClassroomIssueHandler 处理课堂问题登记与处理闭环。
type ClassroomIssueHandler struct {
	svc service.ClassroomIssueService
}

// NewClassroomIssueHandler 构造课堂问题 handler。
func NewClassroomIssueHandler(svc service.ClassroomIssueService) *ClassroomIssueHandler {
	return &ClassroomIssueHandler{svc: svc}
}

// List 处理 GET /api/v1/classroom-issues。
func (h *ClassroomIssueHandler) List(c *gin.Context) {
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	data, err := h.svc.List(c.Request.Context(), role, uid)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Create 处理 POST /api/v1/classroom-issues（教师）。
func (h *ClassroomIssueHandler) Create(c *gin.Context) {
	var req dto.ClassroomIssueCreateReq
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

// Handle 处理 POST /api/v1/classroom-issues/:id/handle（督导/主任）。
func (h *ClassroomIssueHandler) Handle(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ClassroomIssueHandleReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	data, err := h.svc.Handle(c.Request.Context(), role, uid, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Confirm 处理 POST /api/v1/classroom-issues/:id/confirm（教师本人）。
func (h *ClassroomIssueHandler) Confirm(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.ClassroomIssueConfirmReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	if err := h.svc.Confirm(c.Request.Context(), role, uid, id, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"id": id, "resolved": req.Resolved})
}
