package handler

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/pkg/response"
)

// Health 处理 GET /healthz（无鉴权），供部署探活与 S1.1 验收。
func Health(c *gin.Context) {
	response.OK(c, gin.H{"status": "up"})
}
