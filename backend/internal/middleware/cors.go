package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS 按配置白名单放行跨域请求，并允许携带 Authorization 头。
//
// allowAnyOrigin=true 时回显请求来源（任意来源都放行），用于本地/局域网联调：
// 前端 dev server 除 localhost 外还监听内网 IP，用内网地址打开页面时
// Origin 不在白名单里，POST 会被拦成 403（curl 不带 Origin，故本地联调测不出来）。
//
// 🔴 注意：本中间件 AllowCredentials=true，因此放行任意来源时**不能**用 `AllowOrigins: ["*"]`
// —— gin-contrib/cors 会因 "credentials + wildcard" 直接 panic。
// 必须用 AllowOriginFunc 回显具体来源，这也是浏览器在带凭据时唯一接受的写法。
func CORS(origins []string, allowAnyOrigin bool) gin.HandlerFunc {
	cfg := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	if allowAnyOrigin {
		cfg.AllowOriginFunc = func(string) bool { return true }
	} else {
		cfg.AllowOrigins = origins
	}
	return cors.New(cfg)
}
