package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/service"
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
