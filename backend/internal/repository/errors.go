// Package repository 是数据层：只做 GORM 查询与持久化，不写业务分支。
package repository

import (
	"errors"

	"gorm.io/gorm"
)

// IsNotFound 判断是否为「记录不存在」。
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// ErrNotImplemented 供测试替身使用，表示该方法未注入行为。
var ErrNotImplemented = errors.New("repository: not implemented")
