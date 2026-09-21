// Package errcode 定义全局错误码、HTTP 状态映射与业务错误类型。
// 错误码表见 docs/backend_AGENTS.md §7，前后端共用。
package errcode

import (
	"errors"
	"fmt"
	"net/http"
)

// Code 是业务错误码。
type Code int

// 业务错误码常量（0 表示成功，其余与 §7 表格一一对应）。
const (
	OK                    Code = 0
	Params                Code = 40001
	BizRule               Code = 40002
	Unauthorized          Code = 40101
	ForbiddenRole         Code = 40301
	ForbiddenData         Code = 40302
	NotFound              Code = 40401
	Conflict              Code = 40901
	Internal              Code = 50001
	NotImplemented        Code = 50002
	DependencyUnavailable Code = 50003
)

var messages = map[Code]string{
	OK:            "ok",
	Params:        "参数校验失败",
	BizRule:       "业务规则校验失败",
	Unauthorized:  "登录状态已失效，请重新登录",
	ForbiddenRole: "当前角色无权访问该功能",
	ForbiddenData: "无权操作该数据",
	NotFound:      "请求的资源不存在",
	Conflict:      "数据已存在，请勿重复提交",
	Internal:      "服务器内部错误，请稍后重试",
}

func init() {
	messages[NotImplemented] = "功能尚未实现"
	messages[DependencyUnavailable] = "转写服务不可用"
}

// Message 返回错误码对应的用户可读中文文案。
func (c Code) Message() string {
	if msg, ok := messages[c]; ok {
		return msg
	}
	return messages[Internal]
}

// HTTPStatus 返回错误码对应的 HTTP 状态码。
func (c Code) HTTPStatus() int {
	switch c {
	case OK:
		return http.StatusOK
	case Params, BizRule:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case ForbiddenRole, ForbiddenData:
		return http.StatusForbidden
	case NotFound:
		return http.StatusNotFound
	case Conflict:
		return http.StatusConflict
	case NotImplemented:
		return http.StatusNotImplemented
	case DependencyUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Error 是携带业务错误码的错误类型，service 层统一返回它。
type Error struct {
	Code Code
	Msg  string
	Err  error
}

// Error 实现 error 接口。
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Msg, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// Unwrap 支持 errors.Is / errors.As 追溯底层错误。
func (e *Error) Unwrap() error { return e.Err }

// New 构造一个带自定义文案的业务错误。
func New(code Code, msg string) *Error {
	if msg == "" {
		msg = code.Message()
	}
	return &Error{Code: code, Msg: msg}
}

// Newf 构造一个带格式化文案的业务错误。
func Newf(code Code, format string, args ...any) *Error {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap 在业务错误上附带底层错误（仅进日志，不外泄）。
func Wrap(code Code, msg string, err error) *Error {
	return &Error{Code: code, Msg: msg, Err: err}
}

// From 从任意 error 中提取业务错误，提取不到时返回 Internal 兜底。
func From(err error) *Error {
	if err == nil {
		return New(OK, "")
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return Wrap(Internal, Internal.Message(), err)
}
