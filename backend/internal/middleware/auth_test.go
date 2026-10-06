package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/pkg/jwtutil"
)

func newTestManager() *jwtutil.Manager {
	return jwtutil.New("test-secret-for-unit-test", time.Hour)
}

// newTestRouter 挂上被测中间件，命中即返回 200，被拦截则返回中间件写的错误响应。
func newTestRouter(mw gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/recordings/:id/stream", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func doGet(r *gin.Engine, target, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestAuthOrPlaybackTicket 覆盖音频流鉴权的四条关键路径。
func TestAuthOrPlaybackTicket(t *testing.T) {
	mgr := newTestManager()
	ticket, err := mgr.SignPlayback(5, "supervisor", 0, 42, time.Hour, time.Now())
	require.NoError(t, err)

	otherTicket, err := mgr.SignPlayback(5, "supervisor", 0, 99, time.Hour, time.Now())
	require.NoError(t, err)

	sessionToken, err := mgr.Sign(5, "supervisor", 0, time.Now())
	require.NoError(t, err)

	expired, err := mgr.SignPlayback(5, "supervisor", 0, 42, time.Hour, time.Now().Add(-2*time.Hour))
	require.NoError(t, err)

	cases := []struct {
		name       string
		target     string
		bearer     string
		wantStatus int
	}{
		{"票据匹配录音 id 放行", "/recordings/42/stream?ticket=" + ticket, "", http.StatusOK},
		{"会话令牌同样可用", "/recordings/42/stream", sessionToken, http.StatusOK},
		{"🔴 票据 id 与路径不符必须拒绝", "/recordings/99/stream?ticket=" + ticket, "", http.StatusUnauthorized},
		{"他人录音的票据不可用", "/recordings/42/stream?ticket=" + otherTicket, "", http.StatusUnauthorized},
		{"无任何凭据拒绝", "/recordings/42/stream", "", http.StatusUnauthorized},
		{"过期票据拒绝", "/recordings/42/stream?ticket=" + expired, "", http.StatusUnauthorized},
		{"伪造票据拒绝", "/recordings/42/stream?ticket=not-a-jwt", "", http.StatusUnauthorized},
	}

	r := newTestRouter(AuthOrPlaybackTicket(mgr))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantStatus, doGet(r, tc.target, tc.bearer).Code)
		})
	}
}

// TestAuthRejectsPlaybackTicket 是安全回归：
// 播放票据是为查询串设计的窄用途凭据，绝不能当会话令牌换取完整账号权限。
func TestAuthRejectsPlaybackTicket(t *testing.T) {
	mgr := newTestManager()
	ticket, err := mgr.SignPlayback(5, "supervisor", 0, 42, 4*time.Hour, time.Now())
	require.NoError(t, err)
	session, err := mgr.Sign(5, "supervisor", 0, time.Now())
	require.NoError(t, err)

	r := newTestRouter(Auth(mgr))

	assert.Equal(t, http.StatusUnauthorized, doGet(r, "/recordings/42/stream", ticket).Code,
		"播放票据不得作为会话令牌使用")
	assert.Equal(t, http.StatusOK, doGet(r, "/recordings/42/stream", session).Code)
}

// TestRedactQuery 覆盖访问日志脱敏：票据进日志等于鉴权形同虚设。
func TestRedactQuery(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"空查询串", "", ""},
		{"普通参数原样保留", "page=1&pageSize=10", "?page=1&pageSize=10"},
		{"ticket 必须掩码", "ticket=eyJhbGciOiJIUzI1NiJ9.abc.def", "?ticket=%2A%2A%2A"},
		{"token 必须掩码", "token=secret-jwt", "?token=%2A%2A%2A"},
		{"api_key 必须掩码", "api_key=sk-123456", "?api_key=%2A%2A%2A"},
		{"混合时只掩码敏感项", "page=2&ticket=abc&keyword=x", "?keyword=x&page=2&ticket=%2A%2A%2A"},
		{"无法解析时不落盘原串", "%zz=bad", "?<unparsable>"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactQuery(tc.in)
			assert.Equal(t, tc.want, got)
			assert.NotContains(t, got, "eyJhbGci", "JWT 片段绝不能出现在日志路径中")
		})
	}
}
