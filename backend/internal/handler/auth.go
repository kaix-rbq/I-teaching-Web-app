package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/response"
)

// AuthHandler 处理认证相关接口。
type AuthHandler struct {
	auth service.AuthService
}

// NewAuthHandler 构造认证 handler。
func NewAuthHandler(auth service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Login 处理 POST /api/v1/auth/login。
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	resp, err := h.auth.Login(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, resp)
}

// Me 处理 GET /api/v1/auth/me。
func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
		return
	}
	user, err := h.auth.Me(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

// Logout 处理 POST /api/v1/auth/logout（Sprint 1 无服务端黑名单，直接返回成功）。
func (h *AuthHandler) Logout(c *gin.Context) {
	response.OK(c, nil)
}
