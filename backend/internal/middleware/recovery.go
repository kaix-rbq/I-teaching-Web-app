// Package middleware 提供 Recovery / Logger / CORS / Auth / RBAC 中间件。
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/response"
)

// Recovery 捕获 panic，返回 500 + code 50001，并把堆栈写入日志。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"error", err,
					"stack", string(debug.Stack()),
				)
				response.Fail(c, errcode.New(errcode.Internal, errcode.Internal.Message()))
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
