// Package repository 是数据层：只做 GORM 查询与持久化，不写业务分支。
package repository

import (
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// IsNotFound 判断是否为「记录不存在」。
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// IsDuplicate 判断是否为唯一键冲突（MySQL 1062）。
// 唯一性预检存在并发窗口，最终仍可能由数据库抛出 1062；service 据此返回 40901 而非 50001。
func IsDuplicate(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// ErrNotImplemented 供测试替身使用，表示该方法未注入行为。
var ErrNotImplemented = errors.New("repository: not implemented")
