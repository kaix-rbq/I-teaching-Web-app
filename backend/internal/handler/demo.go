package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/response"
)

// DemoHandler 处理演示模式（B6/N7）：仅本地/dev 开启。
type DemoHandler struct {
	svc service.DemoService
}

// NewDemoHandler 构造演示模式 handler。
func NewDemoHandler(svc service.DemoService) *DemoHandler {
	return &DemoHandler{svc: svc}
}

// Seed 处理 POST /api/v1/demo/seed：一键载入示例数据（前端需二次确认）。
func (h *DemoHandler) Seed(c *gin.Context) {
	data, err := h.svc.Seed(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}
