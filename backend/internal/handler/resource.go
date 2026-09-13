package handler

import (
	"path/filepath"

	"github.com/gin-gonic/gin"

	"aijiaoxue-api/internal/middleware"
	"aijiaoxue-api/internal/service"
	"aijiaoxue-api/pkg/errcode"
	"aijiaoxue-api/pkg/response"
)

// ResourceHandler 处理课程资源相关接口。
type ResourceHandler struct {
	resources service.ResourceService
}

// NewResourceHandler 构造资源 handler。
func NewResourceHandler(resources service.ResourceService) *ResourceHandler {
	return &ResourceHandler{resources: resources}
}

// List 处理 GET /api/v1/courses/:id/resources。
func (h *ResourceHandler) List(c *gin.Context) {
	courseID, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	list, svcErr := h.resources.List(c.Request.Context(), role, middleware.DeptID(c), userID, courseID)
	if svcErr != nil {
		response.Fail(c, svcErr)
		return
	}
	response.OK(c, list)
}

// Upload 处理 POST /api/v1/courses/:id/resources（multipart/form-data，字段名 file）。
func (h *ResourceHandler) Upload(c *gin.Context) {
	courseID, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, errcode.New(errcode.Params, "缺少上传文件字段 file"))
		return
	}
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
		return
	}

	dto, svcErr := h.resources.Upload(c.Request.Context(), userID, courseID, file)
	if svcErr != nil {
		response.Fail(c, svcErr)
		return
	}
	response.OK(c, dto)
}

// Delete 处理 DELETE /api/v1/resources/:id。
func (h *ResourceHandler) Delete(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Fail(c, errcode.New(errcode.Unauthorized, errcode.Unauthorized.Message()))
		return
	}
	if svcErr := h.resources.Delete(c.Request.Context(), userID, id); svcErr != nil {
		response.Fail(c, svcErr)
		return
	}
	response.OK(c, nil)
}

// Download 处理 GET /api/v1/resources/:id/download（文件名做 filepath.Base 清洗）。
func (h *ResourceHandler) Download(c *gin.Context) {
	id, err := pathID(c)
	if err != nil {
		response.Fail(c, err)
		return
	}
	role, _ := middleware.Role(c)
	userID, _ := middleware.UserID(c)

	record, svcErr := h.resources.ForDownload(c.Request.Context(), role, middleware.DeptID(c), userID, id)
	if svcErr != nil {
		response.Fail(c, svcErr)
		return
	}
	c.FileAttachment(record.FilePath, filepath.Base(record.Name))
}
