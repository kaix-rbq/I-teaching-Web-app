package dto

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// courseCodeRe 是课程编码规则：2-4 位大写字母 + 4 位数字，如 SE3101。
var courseCodeRe = regexp.MustCompile(`^[A-Z]{2,4}\d{4}$`)

// RegisterValidators 注册自定义校验规则，进程启动时调用一次。
// 课程编码正则在 validator 中注册，禁止在 handler 里手写正则 if。
func RegisterValidators() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	_ = v.RegisterValidation("coursecode", func(fl validator.FieldLevel) bool {
		return courseCodeRe.MatchString(fl.Field().String())
	})
}
