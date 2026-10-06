package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/pkg/jwtutil"
)

// newOfflineDB 构造一个不会真正连接的 GORM 实例。
// 路由注册阶段不访问数据库，因此可以在无 MySQL 的环境里测试「路由 + 中间件」装配。
func newOfflineDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "user:pass@tcp(127.0.0.1:1)/nope?timeout=1ms",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true, Logger: gormlogger.Discard})
	require.NoError(t, err, "离线 GORM 实例不应在打开阶段连接数据库")
	return db
}

func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{Port: 0, Mode: gin.TestMode},
		Upload: config.UploadConfig{Dir: "./uploads"},
		// CORS 白名单为空会让 gin-contrib/cors 直接 panic（all origins disabled）。
		CORS: config.CORSConfig{Origins: []string{"http://localhost:5173"}},
	}
}

// TestStreamRouteUsesTicketAuth 是集成回归用例。
//
// 真实踩到的 bug：`/recordings/:id/stream` 曾注册在挂了 Auth 的 authed 组内，
// 于是组上的 Auth 先执行并因缺少 Authorization 头直接 401，
// AuthOrPlaybackTicket 永远没机会运行 —— 表现为「票据签发正常但音频播不出来」。
// 该 bug 单测（只挂单个中间件的 gin 引擎）无法发现，必须测真实路由表。
func TestStreamRouteUsesTicketAuth(t *testing.T) {
	mgr := jwtutil.New("router-test-secret", time.Hour)
	engine, _ := New(newOfflineDB(t), testConfig(), mgr)

	ticket, err := mgr.SignPlayback(5, "supervisor", 0, 1, time.Hour, time.Now())
	require.NoError(t, err)

	t.Run("无凭据必须 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/recordings/1/stream", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("有效票据不得被 401 拦截", func(t *testing.T) {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
			"/api/v1/recordings/1/stream?ticket="+ticket, nil))
		// 无数据库时后续会失败为 5xx，这里只断言「已越过鉴权中间件」。
		assert.NotEqual(t, http.StatusUnauthorized, w.Code,
			"带票据的流式请求被 401：说明该路由又回到了 Auth 组内")
	})

	t.Run("会话令牌仍然可用", func(t *testing.T) {
		session, err := mgr.Sign(5, "supervisor", 0, time.Now())
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/recordings/1/stream", nil)
		req.Header.Set("Authorization", "Bearer "+session)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusUnauthorized, w.Code)
	})
}
