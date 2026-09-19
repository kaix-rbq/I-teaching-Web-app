// Package migrations 以 embed 方式打包全部 SQL 迁移脚本，供 golang-migrate 执行。
// 目录内文件命名遵循 golang-migrate 默认约定：<版本>_<名称>.up.sql / .down.sql
// （正则 ^([0-9]+)_(.*)\.(up|down)\.(.*)$）。
// 注意：Flyway 风格的 V1__xxx.up.sql 不被标识为迁移，会导致 migrate up 报
// "first .: file does not exist"，禁止改回该命名。
package migrations

import "embed"

// FS 是迁移脚本集合（V1 基线 + V2 起的增量）。
//
//go:embed *.sql
var FS embed.FS
