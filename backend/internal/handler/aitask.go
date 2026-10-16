package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// AiTaskHandler 处理 AI 评析任务链接口（开发计划 §4.2 / D3）。
type AiTaskHandler struct {
	tasks service.AiTaskService
}

// NewAiTaskHandler 构造 AI 任务 handler。
func NewAiTaskHandler(tasks service.AiTaskService) *AiTaskHandler {
	return &AiTaskHandler{tasks: tasks}
}

// Analyze 处理 POST /api/v1/sessions/:id/ai-observation/analyze（入队 AI 评析，幂等）。
func (h *AiTaskHandler) Analyze(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	resp, err := h.tasks.Analyze(c.Request.Context(), role, middleware.DeptID(c), userID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, resp)
}

// Observation 处理 GET /api/v1/sessions/:id/ai-observation（AI 观察详情，教师侧风险脱敏）。
func (h *AiTaskHandler) Observation(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	data, err := h.tasks.Observation(c.Request.Context(), role, middleware.DeptID(c), userID, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// TaskStatus 处理 GET /api/v1/ai-tasks/:id（任务状态查询）。
func (h *AiTaskHandler) TaskStatus(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	data, err := h.tasks.TaskStatus(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Retry 处理 POST /api/v1/ai-tasks/:id/retry（仅 supervisor 路由组）。
func (h *AiTaskHandler) Retry(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	data, err := h.tasks.Retry(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}
