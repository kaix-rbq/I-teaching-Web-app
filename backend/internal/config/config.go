// Package config 负责加载 config.yaml 与环境变量覆盖（前缀 AIJIAOXUE_）。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 是应用的全部可配置项。
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	MySQL  MySQLConfig  `mapstructure:"mysql"`
	JWT    JWTConfig    `mapstructure:"jwt"`
	Upload UploadConfig `mapstructure:"upload"`
	CORS   CORSConfig   `mapstructure:"cors"`
}

// ServerConfig 描述 HTTP 服务监听参数。
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// MySQLConfig 描述数据库连接与连接池参数。
type MySQLConfig struct {
	DSN          string `mapstructure:"dsn"`
	MaxOpenConns int    `mapstructure:"maxOpenConns"`
	MaxIdleConns int    `mapstructure:"maxIdleConns"`
}

// JWTConfig 描述令牌签发参数。
type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	TTL    string `mapstructure:"ttl"`
}

// UploadConfig 描述课程资源上传限制。
type UploadConfig struct {
	Dir      string   `mapstructure:"dir"`
	MaxSize  int64    `mapstructure:"maxSize"`
	AllowExt []string `mapstructure:"allowExt"`
}

// CORSConfig 描述跨域白名单。
type CORSConfig struct {
	Origins []string `mapstructure:"origins"`
}

// TTLDuration 把 jwt.ttl 解析为 time.Duration。
func (c JWTConfig) TTLDuration() time.Duration {
	d, err := time.ParseDuration(c.TTL)
	if err != nil || d <= 0 {
		return 24 * time.Hour
	}
	return d
}

// Load 从 path 指向的 yaml 读取配置；文件缺失时报错而非静默使用默认值。
// 环境变量 AIJIAOXUE_XXX_YYY 可覆盖任意键。
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("mysql.maxOpenConns", 20)
	v.SetDefault("mysql.maxIdleConns", 5)
	v.SetDefault("jwt.ttl", "24h")
	v.SetDefault("upload.dir", "./uploads")
	v.SetDefault("upload.maxSize", 100*1024*1024)

	v.SetEnvPrefix("AIJIAOXUE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if cfg.MySQL.DSN == "" {
		return nil, fmt.Errorf("config: mysql.dsn 不能为空")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("config: jwt.secret 不能为空")
	}
	return &cfg, nil
}
