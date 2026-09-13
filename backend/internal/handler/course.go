package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/response"
)

// CourseHandler 处理课程相关接口。
type CourseHandler struct {
	courses service.CourseService
}

// NewCourseHandler 构造课程 handler。
func NewCourseHandler(courses service.CourseService) *CourseHandler {
	return &CourseHandler{courses: courses}
}

// List 处理 GET /api/v1/courses（数据范围按角色裁剪）。
func (h *CourseHandler) List(c *gin.Context) {
	var q dto.CourseListQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	result, err := h.courses.List(c.Request.Context(), role, middleware.DeptID(c), userID, q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

// Detail 处理 GET /api/v1/courses/:id。
func (h *CourseHandler) Detail(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	detail, svcErr := h.courses.Detail(c.Request.Context(), role, middleware.DeptID(c), userID, id)
	if svcErr != nil {
		response.Fail(c, svcErr)
		return
	}
	response.OK(c, detail)
}

// Create 处理 POST /api/v1/courses（仅 director）。
func (h *CourseHandler) Create(c *gin.Context) {
	var req dto.CourseUpsertReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	detail, err := h.courses.Create(c.Request.Context(), middleware.DeptID(c), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, detail)
}

// Update 处理 PUT /api/v1/courses/:id（仅 director，code 不可改）。
func (h *CourseHandler) Update(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.CourseUpsertReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	detail, svcErr := h.courses.Update(c.Request.Context(), middleware.DeptID(c), id, req)
	if svcErr != nil {
		response.Fail(c, svcErr)
		return
	}
	response.OK(c, detail)
}

// pathID 解析路径中的数字 id。
func pathID(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errcode.New(errcode.Params, "路径参数 id 非法")
	}
	return id, nil
}
