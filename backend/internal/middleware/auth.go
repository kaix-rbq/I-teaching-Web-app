package middleware

import (
	"strconv"
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
//
// 播放票据（scope=playback）会被拒绝：它是为 <audio> 查询串设计的窄用途凭据，
// 若能当会话令牌用，等于用一条录音的短期票据换到 4 小时的完整账号权限。
func Auth(manager *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if raw == "" || !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
			c.Abort()
			return
		}
		claims, err := manager.Parse(strings.TrimSpace(raw[len("Bearer "):]))
		if err != nil || claims.IsPlaybackTicket() {
			response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
			c.Abort()
			return
		}
		if !injectClaims(c, claims) {
			return
		}
		c.Next()
	}
}

// AuthOrPlaybackTicket 与 Auth 等价，但额外允许用 ?ticket= 短时票据鉴权。
//
// 仅挂在 GET /recordings/:id/stream：<audio> 标签无法自定义请求头，只能靠查询串。
// 票据必须 scope=playback 且 rid 与路径 :id 一致，防止拿 A 的票据听 B 的录音。
func AuthOrPlaybackTicket(manager *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		if raw := c.GetHeader("Authorization"); strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			token = strings.TrimSpace(raw[len("Bearer "):])
		} else if t := c.Query("ticket"); t != "" {
			token = t
		}
		if token == "" {
			response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
			c.Abort()
			return
		}
		claims, err := manager.Parse(token)
		if err != nil {
			response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
			c.Abort()
			return
		}
		// 走票据通道时必须校验录音 id 绑定，防止越权重放。
		if claims.IsPlaybackTicket() {
			id, convErr := strconv.ParseUint(c.Param("id"), 10, 64)
			if convErr != nil || claims.RecID != id {
				response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
				c.Abort()
				return
			}
		}
		if !injectClaims(c, claims) {
			return
		}
		c.Next()
	}
}

// injectClaims 把令牌载荷写入 gin context；subject 非法时直接结束请求并返回 false。
func injectClaims(c *gin.Context, claims *jwtutil.Claims) bool {
	userID, err := claims.UserID()
	if err != nil {
		response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
		c.Abort()
		return false
	}
	c.Set(ctxUserID, userID)
	c.Set(ctxRole, claims.Role)
	c.Set(ctxDeptID, claims.DeptID)
	return true
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
