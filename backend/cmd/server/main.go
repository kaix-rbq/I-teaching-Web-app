// Command server 是「爱教学」后端的唯一服务入口。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/router"
	"aijiaoxue-api/pkg/jwtutil"
)

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径（可用 AIJIAOXUE_* 环境变量覆盖）")
	flag.Parse()

	setupLogger()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config failed", "error", err, "path", *configPath)
		os.Exit(1)
	}

	db, err := openDB(cfg)
	if err != nil {
		slog.Error("connect mysql failed", "error", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(cfg.Upload.Dir, 0o755); err != nil {
		slog.Error("create upload dir failed", "error", err, "dir", cfg.Upload.Dir)
		os.Exit(1)
	}

	jwtManager := jwtutil.New(cfg.JWT.Secret, cfg.JWT.TTLDuration())
	engine := router.New(db, cfg, jwtManager)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server started", "addr", srv.Addr, "mode", cfg.Server.Mode)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server listen failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("server shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}
	slog.Info("server stopped")
}

// setupLogger 初始化标准库 slog 的 JSON 输出。
func setupLogger() {
	level := slog.LevelInfo
	if os.Getenv("AIJIAOXUE_LOG_DEBUG") == "true" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}

// openDB 建立 GORM 连接、配置连接池并 Ping 校验；任何失败都直接退出。
func openDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		// 表结构以 database/schema.sql 为唯一事实源，禁止 GORM 自动迁移。
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	slog.Info("mysql connected")
	return db, nil
}
