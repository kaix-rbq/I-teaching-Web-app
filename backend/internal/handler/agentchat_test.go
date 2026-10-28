package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/pkg/errcode"
)

type fakeAgentChatSvc struct {
	deltas    []string
	citations []dto.AgentChatCitation
	err       error
}

func (f *fakeAgentChatSvc) Chat(
	_ context.Context, _ string, _, _ uint64, _ dto.AgentChatReq, onDelta func(string) error,
) ([]dto.AgentChatCitation, error) {
	for _, delta := range f.deltas {
		if err := onDelta(delta); err != nil {
			return nil, err
		}
	}
	return f.citations, f.err
}

func TestAgentChatHandlerSSESuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeAgentChatSvc{
		deltas: []string{"你", "好"},
		citations: []dto.AgentChatCitation{
			{ChunkID: "kb-rubric#3", Title: "互动锚点", Quote: "留思考时间"},
		},
	}
	h := NewAgentChatHandler(svc)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/agent/chat",
		strings.NewReader(`{"courseId":1,"question":"q"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	body := w.Body.String()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "no-cache")
	assert.Contains(t, body, "data: {\"delta\":\"你\"}\n\n")
	assert.Contains(t, body, "data: {\"delta\":\"好\"}\n\n")
	assert.Contains(t, body, "\"done\":true")
	assert.Contains(t, body, "kb-rubric#3")
	assert.Contains(t, body, "data: [DONE]\n\n")
}

func TestAgentChatHandlerSSEError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAgentChatHandler(&fakeAgentChatSvc{err: errcode.NewDependencyUnavailable("智能体暂时不可用")})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/agent/chat",
		strings.NewReader(`{"courseId":1,"question":"q"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	body := w.Body.String()
	assert.Equal(t, http.StatusOK, w.Code, "SSE 错误帧仍应是 HTTP 200")
	assert.Contains(t, body, `"error"`)
	assert.Contains(t, body, "50003")
	assert.Contains(t, body, "智能体暂时不可用")
	assert.Contains(t, body, "data: [DONE]\n\n")
	require.NotContains(t, body, `"code":50001`)
}
