// Command seed 用 bcrypt 覆写演示账号密码哈希。
// 用法：go run ./cmd/seed [-config config.yaml] [-password 123456] [-username teacher]
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"aijiaoxue-api/internal/config"
	"aijiaoxue-api/internal/model"
)

// demoUsernames 是种子数据中的演示账号。
var demoUsernames = []string{"director", "teacher", "supervisor", "zhanghua", "liuyang", "zhaolei"}

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	password := flag.String("password", "123456", "要写入的明文密码")
	username := flag.String("username", "", "只更新指定账号（默认更新全部演示账号）")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接数据库失败: %v\n", err)
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成 bcrypt 哈希失败: %v\n", err)
		os.Exit(1)
	}

	targets := demoUsernames
	if *username != "" {
		targets = []string{*username}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := db.WithContext(ctx).
		Model(&model.User{}).
		Where("username IN ?", targets).
		Update("password_hash", string(hash))
	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "更新密码哈希失败: %v\n", result.Error)
		os.Exit(1)
	}
	if result.RowsAffected == 0 {
		fmt.Fprintf(os.Stderr, "未匹配到任何账号（%v），请先执行 database/seed.sql\n", targets)
		os.Exit(1)
	}

	slog.Info("password hash updated", "accounts", targets, "rows", result.RowsAffected)
	fmt.Printf("已为 %d 个账号写入密码 %q 的 bcrypt 哈希\n", result.RowsAffected, *password)
}
