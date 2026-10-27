package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// AgentImproveHandler 处理提优读接口（§8.15 P2）：
// 课程 AI 提优建议与教师评分趋势。权限与数据裁剪均在 service 层完成。
type AgentImproveHandler struct {
	improve service.AgentImproveService
}

// NewAgentImproveHandler 构造提优读接口 handler。
func NewAgentImproveHandler(improve service.AgentImproveService) *AgentImproveHandler {
	return &AgentImproveHandler{improve: improve}
}

// Suggestions 处理 GET /api/v1/courses/:id/agent-suggestions?semester=
// 返回 AgentSuggestion[]（v1 数组，不分页）。
func (h *AgentImproveHandler) Suggestions(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var q dto.AgentSuggestionQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	data, err := h.improve.CourseSuggestions(
		c.Request.Context(), role, middleware.DeptID(c), userID, id, q.Semester)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// ScoreTrend 处理 GET /api/v1/teachers/:id/score-trend?semester=&courseId=&dimension=
// 返回 ScoreTrendSeries[]（composite + 五维，缺失点 value=null）。
func (h *AgentImproveHandler) ScoreTrend(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var q dto.ScoreTrendQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	data, err := h.improve.TeacherScoreTrend(
		c.Request.Context(), role, middleware.DeptID(c), userID, id, q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}
