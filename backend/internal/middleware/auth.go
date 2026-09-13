package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/jwtutil"
	"aijiaoxue-api/pkg/response"
)

// gin.Context 中保存鉴权信息的键。
const (
	ctxUserID = "userID"
	ctxRole   = "role"
	ctxDeptID = "deptID"
)

// Auth 解析 Bearer 令牌，把 userID / role / deptID 注入 context；失败返回 401 + 40101。
func Auth(manager *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if raw == "" || !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
			c.Abort()
			return
		}

		claims, err := manager.Parse(strings.TrimSpace(raw[len("Bearer "):]))
		if err != nil {
			response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
			c.Abort()
			return
		}
		userID, err := claims.UserID()
		if err != nil {
			response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
			c.Abort()
			return
		}

		c.Set(ctxUserID, userID)
		c.Set(ctxRole, claims.Role)
		c.Set(ctxDeptID, claims.DeptID)
		c.Next()
	}
}

// UserID 从 context 读取当前用户 id。
func UserID(c *gin.Context) (uint64, bool) {
	value, ok := c.Get(ctxUserID)
	if !ok {
		return 0, false
	}
	id, ok := value.(uint64)
	return id, ok
}

// Role 从 context 读取当前用户角色。
func Role(c *gin.Context) (string, bool) {
	value, ok := c.Get(ctxRole)
	if !ok {
		return "", false
	}
	role, ok := value.(string)
	return role, ok
}

// DeptID 从 context 读取当前用户所属教研室 id（督导为 0）。
func DeptID(c *gin.Context) uint64 {
	value, ok := c.Get(ctxDeptID)
	if !ok {
		return 0
	}
	deptID, _ := value.(uint64)
	return deptID
}
