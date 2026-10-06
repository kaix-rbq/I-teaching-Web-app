package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// newCORSRouter 构造只挂 CORS 的最小引擎，POST /login 固定返回 200。
func newCORSRouter(origins []string, allowAnyOrigin bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(origins, allowAnyOrigin))
	r.POST("/login", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func doLoginWithOrigin(r *gin.Engine, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestCORSWhitelist 是回归用例。
//
// 真实踩到的 bug：前端 dev server 会同时监听内网 IP（Vite 的 Network 地址），
// 用内网地址打开页面时浏览器带 `Origin: http://<内网IP>:5173`，
// 而白名单只有 localhost/127.0.0.1，于是登录等 POST 被 CORS 拦成 403 ——
// 表现是「页面能打开、一点登录就 403」。curl 不带 Origin，所以接口自测全过。
func TestCORSWhitelist(t *testing.T) {
	whitelist := []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	r := newCORSRouter(whitelist, false)

	t.Run("白名单内放行", func(t *testing.T) {
		for _, origin := range whitelist {
			w := doLoginWithOrigin(r, origin)
			assert.Equal(t, http.StatusOK, w.Code, origin)
			assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("白名单外拦截为 403", func(t *testing.T) {
		w := doLoginWithOrigin(r, "http://172.22.81.206:5173")
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("无 Origin（curl / 服务端调用）不受影响", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, doLoginWithOrigin(r, "").Code)
	})
}

// TestCORSAllowAnyOrigin 覆盖联调开关：开启后任意来源都放行，且回显具体来源。
//
// 回显而非返回 "*" 是硬要求：AllowCredentials=true 时浏览器只接受具体来源，
// 且 gin-contrib/cors 对 "credentials + wildcard" 会直接 panic。
func TestCORSAllowAnyOrigin(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:5173"}, true)

	for _, origin := range []string{
		"http://localhost:5173",
		"http://172.22.81.206:5173",
		"http://26.82.138.248:5173",
	} {
		w := doLoginWithOrigin(r, origin)
		assert.Equal(t, http.StatusOK, w.Code, origin)
		assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"),
			"必须回显具体来源，不能是 *")
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	}
}

// TestCORSPreflight 验证预检请求（浏览器发 POST+JSON 前会先 OPTIONS）。
func TestCORSPreflight(t *testing.T) {
	r := newCORSRouter(nil, true)

	req := httptest.NewRequest(http.MethodOptions, "/login", nil)
	req.Header.Set("Origin", "http://172.22.81.206:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://172.22.81.206:5173", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}
