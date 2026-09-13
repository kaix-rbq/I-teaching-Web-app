package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/response"
)

// DashboardHandler 处理工作台聚合接口。
type DashboardHandler struct {
	dashboard service.DashboardService
}

// NewDashboardHandler 构造工作台 handler。
func NewDashboardHandler(dashboard service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

// Board 处理 GET /api/v1/dashboard（一套接口三种 DTO）。
func (h *DashboardHandler) Board(c *gin.Context) {
	role, ok := middleware.Role(c)
	userID, okID := middleware.UserID(c)
	if !ok || !okID {
		response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
		return
	}
	data, err := h.dashboard.Board(c.Request.Context(), role, middleware.DeptID(c), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, data)
}
