package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// TeacherScoreHandler 处理教师级/课程级评分聚合接口（开发计划 §4.2）。
//
// 路由说明：教师评分列表挂在 /teacher-scores 而非契约原文的 /teachers——
// Sprint 1 已用 GET /teachers 承载教研室教师字典（前端课程表单依赖，返回数组），
// 为守住「Sprint 1 接口无回归」的验收线，评分列表使用独立路径。
type TeacherScoreHandler struct {
	teacherScores service.TeacherScoreService
}

// NewTeacherScoreHandler 构造教师评分 handler。
func NewTeacherScoreHandler(teacherScores service.TeacherScoreService) *TeacherScoreHandler {
	return &TeacherScoreHandler{teacherScores: teacherScores}
}

// List 处理 GET /api/v1/teacher-scores（director 本室 / supervisor 全校，S6.4）。
func (h *TeacherScoreHandler) List(c *gin.Context) {
	var q dto.TeacherScoreListQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)

	result, err := h.teacherScores.List(c.Request.Context(), role, middleware.DeptID(c), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

// TeacherSummary 处理 GET /api/v1/teachers/:id/evaluation-summary
// （director 本室 / teacher 仅自己 / supervisor 全校，S6.5）。
func (h *TeacherScoreHandler) TeacherSummary(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	data, err := h.teacherScores.TeacherSummary(c.Request.Context(), role, middleware.DeptID(c), userID, id, c.Query("semester"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// CourseSummary 处理 GET /api/v1/courses/:id/evaluation-summary（登录，数据裁剪，S6.6）。
func (h *TeacherScoreHandler) CourseSummary(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	data, err := h.teacherScores.CourseSummary(c.Request.Context(), role, middleware.DeptID(c), userID, id, c.Query("semester"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// TeacherEvaluations 处理 GET /api/v1/teachers/:id/evaluations（历次评价时间线，S6.5 / T1.13）。
// 权限与 /teachers/:id/evaluation-summary 一致：director 本室 / teacher 仅自己 / supervisor 全校。
func (h *TeacherScoreHandler) TeacherEvaluations(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var q dto.TeacherEvaluationTimelineQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	data, err := h.teacherScores.TeacherEvaluations(c.Request.Context(), role, middleware.DeptID(c), userID, id, q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}
