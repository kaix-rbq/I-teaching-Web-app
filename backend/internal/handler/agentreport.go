package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// AgentReportHandler 处理教师级 AI 报告（S2）读写接口。
type AgentReportHandler struct {
	reports service.ReportService
}

// NewAgentReportHandler 构造教师级报告 handler。
func NewAgentReportHandler(reports service.ReportService) *AgentReportHandler {
	return &AgentReportHandler{reports: reports}
}

// Get 处理 GET /api/v1/teachers/:id/ai-report?semester=。
// 权限在 service 层：teacher 本人 / director 本室 / supervisor 全校。
func (h *AgentReportHandler) Get(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	data, err := h.reports.GetReport(
		c.Request.Context(), role, middleware.DeptID(c), userID, id, c.Query("semester"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Regenerate 处理 POST /api/v1/teachers/:id/ai-report/regenerate。
// 仅 supervisor/director；复位 pending 后异步生成，不等待完成。
func (h *AgentReportHandler) Regenerate(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	data, err := h.reports.Regenerate(
		c.Request.Context(), role, middleware.DeptID(c), userID, id, c.Query("semester"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}
