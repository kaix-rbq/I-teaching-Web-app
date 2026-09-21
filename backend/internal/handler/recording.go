package handler

import (
	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/response"
	"github.com/gin-gonic/gin"
	"net/http"
)

type RecordingHandler struct{ recordings service.RecordingService }

func NewRecordingHandler(s service.RecordingService) *RecordingHandler {
	return &RecordingHandler{recordings: s}
}
func (h *RecordingHandler) Upload(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, errcode.New(errcode.Params, "缺少上传文件字段 file"))
		return
	}
	uid, _ := middleware.UserID(c)
	v, err := h.recordings.Upload(c.Request.Context(), uid, id, f)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, v)
}
func (h *RecordingHandler) Transcript(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	r, t, err := h.recordings.Media(c.Request.Context(), role, middleware.DeptID(c), uid, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"recording": r, "transcript": t})
}
func (h *RecordingHandler) Retry(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	if err := h.recordings.Retry(c.Request.Context(), role, middleware.DeptID(c), uid, id); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"status": "pending"})
}
func (h *RecordingHandler) Stream(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	uid, _ := middleware.UserID(c)
	r, err := h.recordings.Stream(c.Request.Context(), role, middleware.DeptID(c), uid, id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", mimeForAudio(r.Format))
	http.ServeFile(c.Writer, c.Request, r.FilePath)
}
func mimeForAudio(ext string) string {
	switch ext {
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "m4a":
		return "audio/mp4"
	default:
		return "application/octet-stream"
	}
}
