// Package handler 是 HTTP 层：绑定参数、调用 service、组装统一响应。
// 禁止直接调用 repository、写 SQL 或业务分支。
package handler

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"aijiaoxue-api/pkg/errcode"
)

// bindJSON 绑定并校验请求体，把 validator 错误转换为可读的 40001。
func bindJSON(c *gin.Context, req any) error {
	if err := c.ShouldBindJSON(req); err != nil {
		return errcode.New(errcode.Params, friendlyBindError(err))
	}
	return nil
}

// bindQuery 绑定并校验查询参数。
func bindQuery(c *gin.Context, req any) error {
	if err := c.ShouldBindQuery(req); err != nil {
		return errcode.New(errcode.Params, friendlyBindError(err))
	}
	return nil
}

// friendlyBindError 把 validator 的结构化错误翻译为面向用户的中文提示。
func friendlyBindError(err error) string {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) && len(validationErrs) > 0 {
		first := validationErrs[0]
		return fmt.Sprintf("参数校验失败：字段 %s 不满足 %s 约束", first.Field(), first.Tag())
	}
	return "参数校验失败，请检查请求内容"
}
