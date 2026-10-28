package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/dto"
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/response"
)

// AgentChatHandler 处理 S3 教学提优流式对话（SSE）。
type AgentChatHandler struct {
	chats service.AgentChatService
}

// NewAgentChatHandler 构造 S3 对话 handler。
func NewAgentChatHandler(chats service.AgentChatService) *AgentChatHandler {
	return &AgentChatHandler{chats: chats}
}

// Chat 处理 POST /api/v1/agent/chat，按 SSE 契约输出 delta / done / [DONE]。
// 关键：不能先 response.OK 再写流；服务层错误也必须以 SSE 错误帧返回，不裸 500。
func (h *AgentChatHandler) Chat(c *gin.Context) {
	var req dto.AgentChatReq
	if err := bindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	writeData := func(v any) error {
		if err := c.Request.Context().Err(); err != nil {
			return err
		}
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", raw); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
	writeDoneMarker := func() error {
		if err := c.Request.Context().Err(); err != nil {
			return err
		}
		if _, err := fmt.Fprint(c.Writer, "data: [DONE]\n\n"); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)
	citations, err := h.chats.Chat(
		c.Request.Context(), role, middleware.DeptID(c), userID, req,
		func(delta string) error {
			return writeData(gin.H{"delta": delta})
		},
	)
	if err != nil {
		if c.Request.Context().Err() != nil {
			return
		}
		appErr := errcode.From(err)
		_ = writeData(gin.H{"error": gin.H{"code": int(appErr.Code), "message": appErr.Msg}})
		_ = writeDoneMarker()
		return
	}
	if citations == nil {
		citations = []dto.AgentChatCitation{}
	}
	// 使用结构体而非 map，保证帧字段顺序与契约一致：done → citations。
	if err := writeData(struct {
		Done      bool                    `json:"done"`
		Citations []dto.AgentChatCitation `json:"citations"`
	}{Done: true, Citations: citations}); err != nil {
		return
	}
	_ = writeDoneMarker()
}
