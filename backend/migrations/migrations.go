// Package migrations 以 embed 方式打包全部 SQL 迁移脚本，供 golang-migrate 执行。
// 目录内文件命名遵循 golang-migrate 约定：V<版本>__<名称>.up.sql / .down.sql。
package migrations

import "embed"

// FS 是迁移脚本集合（V1 基线 + V2 起的增量）。
//
//go:embed *.sql
var FS embed.FS
