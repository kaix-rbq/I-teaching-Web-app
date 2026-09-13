package middleware

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/response"
)

// RequireRoles 是角色守卫：路由分组挂载，禁止散落在 handler 里判断角色。
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		role, ok := Role(c)
		if !ok {
			response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
			c.Abort()
			return
		}
		if _, ok := allowed[role]; !ok {
			response.Fail(c, errcode.New(errcode.ForbiddenRole, errcode.ForbiddenRole.Message()))
			c.Abort()
			return
		}
		c.Next()
	}
}
