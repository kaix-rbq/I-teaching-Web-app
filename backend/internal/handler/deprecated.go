package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/response"
)

// Deprecated 返回一个统一返回 50002（HTTP 501，功能未实现）的处理器。
//
// 2026-10-16 重构（计划 §4.5）：督导五维评分与评估草稿整套停用，
// 后端路由暂留并明确返回 50002（而非 404），下轮删除。
func Deprecated(message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Fail(c, errcode.New(errcode.NotImplemented, message))
	}
}
