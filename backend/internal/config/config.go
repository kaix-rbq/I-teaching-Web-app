// Package config 负责加载 config.yaml 与环境变量覆盖（前缀 AIJIAOXUE_）。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

	"aijiaoxue-api/pkg/scoring"
)

// Config 是应用的全部可配置项。
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	MySQL         MySQLConfig         `mapstructure:"mysql"`
	JWT           JWTConfig           `mapstructure:"jwt"`
	Upload        UploadConfig        `mapstructure:"upload"`
	CORS          CORSConfig          `mapstructure:"cors"`
	Evaluation    EvaluationConfig    `mapstructure:"evaluation"`
	Transcription TranscriptionConfig `mapstructure:"transcription"`
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

type TranscriptionConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Engine        string `mapstructure:"engine"`
	EngineVersion string `mapstructure:"engineVersion"`
}

// CORSConfig 描述跨域白名单。
type CORSConfig struct {
	Origins []string `mapstructure:"origins"`
}

// EvaluationConfig 描述课堂评价计分口径（开发计划 §2.2）。
// 变更口径必须递增 FormulaVersion：历史分按旧口径保留，聚合按版本隔离。
type EvaluationConfig struct {
	Weights          WeightConfig `mapstructure:"weights"`
	SupervisorWeight float64      `mapstructure:"supervisorWeight"`
	AgentWeight      float64      `mapstructure:"agentWeight"`
	FormulaVersion   string       `mapstructure:"formulaVersion"`
	MinSampleSize    int          `mapstructure:"minSampleSize"`
}

// WeightConfig 是 5 个维度的生效权重；非零项合计必须为 1.00。
type WeightConfig struct {
	Objective    float64 `mapstructure:"objective"`
	Content      float64 `mapstructure:"content"`
	Interaction  float64 `mapstructure:"interaction"`
	Organization float64 `mapstructure:"organization"`
	Frontier     float64 `mapstructure:"frontier"`
}

// ScoringWeights 转换为 pkg/scoring 的权重结构。
func (e EvaluationConfig) ScoringWeights() scoring.Weights {
	return scoring.Weights{
		Objective:    e.Weights.Objective,
		Content:      e.Weights.Content,
		Interaction:  e.Weights.Interaction,
		Organization: e.Weights.Organization,
		Frontier:     e.Weights.Frontier,
	}
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
	v.SetDefault("transcription.enabled", false)
	v.SetDefault("transcription.engine", "")
	v.SetDefault("transcription.engineVersion", "")

	// 评分口径默认值 = 开发计划 §2.2（frontier 为观测项，权重 0）。
	v.SetDefault("evaluation.weights.objective", 0.30)
	v.SetDefault("evaluation.weights.content", 0.30)
	v.SetDefault("evaluation.weights.interaction", 0.20)
	v.SetDefault("evaluation.weights.organization", 0.20)
	v.SetDefault("evaluation.weights.frontier", 0.00)
	v.SetDefault("evaluation.supervisorWeight", 0.5)
	v.SetDefault("evaluation.agentWeight", 0.5)
	v.SetDefault("evaluation.formulaVersion", "v1")
	v.SetDefault("evaluation.minSampleSize", 3)

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
	// §2.2 硬性约定：维度权重非零项合计必须为 1.00，服务启动时校验。
	if err := cfg.Evaluation.ScoringWeights().Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &cfg, nil
}
