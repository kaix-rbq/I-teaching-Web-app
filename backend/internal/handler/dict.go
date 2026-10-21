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

// DictHandler 处理字典接口（供前端筛选下拉）。
type DictHandler struct {
	dict service.DictService
}

// NewDictHandler 构造字典 handler。
func NewDictHandler(dict service.DictService) *DictHandler {
	return &DictHandler{dict: dict}
}

// Departments 处理 GET /api/v1/departments。
func (h *DictHandler) Departments(c *gin.Context) {
	data, err := h.dict.Departments(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Teachers 处理 GET /api/v1/teachers?departmentId=。
func (h *DictHandler) Teachers(c *gin.Context) {
	var departmentID uint64
	if raw := c.Query("departmentId"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err == nil {
			departmentID = parsed
		}
	}
	data, err := h.dict.Teachers(c.Request.Context(), departmentID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// MyTimetable 处理 GET /api/v1/timetable/mine?teacherId=&semester=（B11 建课下拉）。
func (h *DictHandler) MyTimetable(c *gin.Context) {
	var teacherID uint64
	if raw := c.Query("teacherId"); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, 64); err == nil {
			teacherID = parsed
		}
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	data, err := h.dict.MyTimetable(c.Request.Context(), role, middleware.DeptID(c), uid, teacherID, c.Query("semester"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// CreateTimetable 处理 POST /api/v1/timetable（director / supervisor 维护）。
func (h *DictHandler) CreateTimetable(c *gin.Context) {
	var req dto.TimetableCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.New(errcode.Params, "请求参数错误"))
		return
	}
	data, err := h.dict.CreateTimetableEntry(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// UpdateTimetableRemark 处理 PATCH /api/v1/timetable/:id/remark（仅本人教师）。
func (h *DictHandler) UpdateTimetableRemark(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req dto.TimetableRemarkReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	data, err := h.dict.UpdateTimetableRemark(c.Request.Context(), role, uid, id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// DeleteTimetable 处理 DELETE /api/v1/timetable/:id。
func (h *DictHandler) DeleteTimetable(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.dict.DeleteTimetableEntry(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"id": id})
}
