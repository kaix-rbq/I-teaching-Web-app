package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// SupervisionHandler 处理督导相关接口。
type SupervisionHandler struct {
	supervision service.SupervisionService
}

// NewSupervisionHandler 构造督导 handler。
func NewSupervisionHandler(supervision service.SupervisionService) *SupervisionHandler {
	return &SupervisionHandler{supervision: supervision}
}

// Coverage 处理 GET /api/v1/supervision/coverage。
func (h *SupervisionHandler) Coverage(c *gin.Context) {
	data, err := h.supervision.Coverage(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}

// Plans 处理 GET /api/v1/supervision/plans。
func (h *SupervisionHandler) Plans(c *gin.Context) {
	var q dto.PlanListQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := h.supervision.Plans(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}
