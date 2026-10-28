package errcode

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHTTPStatus 锁定错误码 → HTTP 状态映射（docs/backend_AGENTS.md §7 / 开发计划 §4）。
// 回归点：BizRule(40002) 曾漏配，落 default 返回 500，导致业务校验失败被记为服务端错误。
func TestHTTPStatus(t *testing.T) {
	cases := []struct {
		code Code
		want int
	}{
		{OK, http.StatusOK},
		{Params, http.StatusBadRequest},
		{BizRule, http.StatusBadRequest},
		{Unauthorized, http.StatusUnauthorized},
		{ForbiddenRole, http.StatusForbidden},
		{ForbiddenData, http.StatusForbidden},
		{NotFound, http.StatusNotFound},
		{Conflict, http.StatusConflict},
		{Internal, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.code.Message(), func(t *testing.T) {
			assert.Equal(t, tc.want, tc.code.HTTPStatus())
		})
	}
}

// TestFrom 校验 From 能提取业务错误并兜底为 Internal。
func TestFrom(t *testing.T) {
	appErr := From(New(BizRule, "业务规则校验失败"))
	assert.Equal(t, BizRule, appErr.Code)

	wrapped := From(errors.New("plain error"))
	assert.Equal(t, Internal, wrapped.Code)
	assert.ErrorContains(t, wrapped, "plain error")
}

// TestNewDependencyUnavailable 验证 50003 文案可参数化，且默认 ASR 文案保持不变。
func TestNewDependencyUnavailable(t *testing.T) {
	aiErr := NewDependencyUnavailable("智能体暂时不可用")
	assert.Equal(t, DependencyUnavailable, aiErr.Code)
	assert.Equal(t, "智能体暂时不可用", aiErr.Msg)
	assert.Equal(t, "转写服务不可用", DependencyUnavailable.Message(),
		"ASR 既有默认文案不得被 AI 链路改写")
}
