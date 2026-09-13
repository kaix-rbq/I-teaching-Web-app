// Package response 提供统一的响应信封：{code, message, data}。
// handler 层禁止手写 c.JSON 拼信封。
package response

import (
	"github.com/gin-gonic/gin"

	"aijiaoxue-api/pkg/errcode"
)

// Envelope 是所有业务接口的响应结构。
type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK 返回成功响应（HTTP 200 + code 0）。
func OK(c *gin.Context, data any) {
	c.JSON(errcode.OK.HTTPStatus(), Envelope{
		Code:    int(errcode.OK),
		Message: errcode.OK.Message(),
		Data:    data,
	})
}

// Fail 返回失败响应；err 会被解析为业务错误码并映射对应 HTTP 状态。
func Fail(c *gin.Context, err error) {
	appErr := errcode.From(err)
	c.JSON(appErr.Code.HTTPStatus(), Envelope{
		Code:    int(appErr.Code),
		Message: appErr.Msg,
		Data:    nil,
	})
}
