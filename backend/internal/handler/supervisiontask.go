package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// SupervisionTaskHandler 处理督导抽样任务队列（§4.3 / R1.2）。
type SupervisionTaskHandler struct {
	svc service.SupervisionTaskService
}

// NewSupervisionTaskHandler 构造抽样任务 handler。
func NewSupervisionTaskHandler(svc service.SupervisionTaskService) *SupervisionTaskHandler {
	return &SupervisionTaskHandler{svc: svc}
}

// Queue 处理 GET /api/v1/supervision/ai-tasks（仅 supervisor，路由组守卫）。
func (h *SupervisionTaskHandler) Queue(c *gin.Context) {
	var q dto.SupervisorTaskListQuery
	if err := bindQuery(c, &q); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := h.svc.Queue(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}
