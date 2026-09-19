// Command migrate 用 golang-migrate 执行 migrations/ 下的数据库迁移。
//
// 用法：
//
//	go run ./cmd/migrate [-config config.yaml] up        # 升级到最新版本（默认）
//	go run ./cmd/migrate [-config config.yaml] down [N]  # 回滚 N 个版本（默认 1）
//	go run ./cmd/migrate [-config config.yaml] version    # 查看当前版本
//
// 迁移脚本经 embed 打包进二进制，不依赖外部文件路径；版本记录表 schema_migrations
// 由 golang-migrate 自动维护。建库与开发账号不在迁移范围内（见 scripts/init_db.sh）。
package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/migrations"
)

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	cmd := "up"
	if args := flag.Args(); len(args) > 0 {
		cmd = args[0]
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config failed", "error", err, "path", *configPath)
		os.Exit(1)
	}

	db, err := sql.Open("mysql", withMultiStatements(cfg.MySQL.DSN))
	if err != nil {
		slog.Error("open mysql failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		slog.Error("load migrations failed", "error", err)
		os.Exit(1)
	}
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		slog.Error("init migrate driver failed", "error", err)
		os.Exit(1)
	}
	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	if err != nil {
		slog.Error("init migrate failed", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && !isNoChange(err) {
			slog.Error("migrate up failed", "error", err)
			os.Exit(1)
		}
		version, dirty, verr := m.Version()
		if verr != nil {
			slog.Error("read version failed", "error", verr)
			os.Exit(1)
		}
		slog.Info("migrate up done", "version", version, "dirty", dirty)
	case "down":
		steps := 1
		if args := flag.Args(); len(args) > 1 {
			if n, perr := strconv.Atoi(args[1]); perr == nil && n > 0 {
				steps = n
			}
		}
		if err := m.Steps(-steps); err != nil && !isNoChange(err) {
			slog.Error("migrate down failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migrate down done", "steps", steps)
	case "version":
		version, dirty, verr := m.Version()
		if verr != nil {
			fmt.Printf("无已应用版本（%v）\n", verr)
			return
		}
		fmt.Printf("当前版本: %d dirty: %v\n", version, dirty)
	default:
		fmt.Fprintf(os.Stderr, "未知子命令 %q（支持 up / down [N] / version）\n", cmd)
		os.Exit(2)
	}
}

// withMultiStatements 为 DSN 追加 multiStatements=true：go-sql-driver 默认一次只执行
// 一条语句，而迁移脚本包含多条 DDL，不开启会报 "Error 1064"。
func withMultiStatements(dsn string) string {
	if strings.Contains(dsn, "multiStatements") {
		return dsn
	}
	if strings.Contains(dsn, "?") {
		return dsn + "&multiStatements=true"
	}
	return dsn + "?multiStatements=true"
}

// isNoChange 判断「无待执行迁移」的哨兵错误（Up/Steps 到顶时返回，非失败）。
func isNoChange(err error) bool {
	return errors.Is(err, migrate.ErrNoChange) || errors.Is(err, migrate.ErrNilVersion)
}
