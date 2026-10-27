// Package config 负责加载 config.yaml 与环境变量覆盖（前缀 AIJIAOXUE_）。
package config

import (
	"fmt"
	"log/slog"
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
	Agent         AgentConfig         `mapstructure:"agent"`
	Embedding     EmbeddingConfig     `mapstructure:"embedding"`
	Demo          DemoConfig          `mapstructure:"demo"`
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

// TranscriptionConfig 描述课堂音频转写（ASR）引擎参数（Sprint 2.2 阶段②）。
//
// 安全约定：APIKey 只允许来自环境变量 AIJIAOXUE_TRANSCRIPTION_API_KEY，
// config.yaml 中保持空串；本结构体禁止出现在任何 DTO 中。
type TranscriptionConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Engine        string `mapstructure:"engine"`        // 逻辑引擎名，写入 transcripts.engine
	EngineVersion string `mapstructure:"engineVersion"` // 写入 transcripts.engine_version
	Model         string `mapstructure:"model"`
	// APIKey 只允许来自环境变量 AIJIAOXUE_TRANSCRIPTION_API_KEY。
	// config.yaml 中必须保持空串；排障只能打日志用 MaskedAPIKey()。
	APIKey       string `mapstructure:"apiKey"`
	WorkspaceID  string `mapstructure:"workspaceId"`
	BaseURL      string `mapstructure:"baseUrl"`
	Timeout      string `mapstructure:"timeout"`
	PollInterval string `mapstructure:"pollInterval"`
	// MaxConcurrency 限制同时进行的转写任务数，保护内存与上游配额。
	MaxConcurrency int `mapstructure:"maxConcurrency"`
	// Diarization 开启说话人分离（产出 speaker_id）；仅支持单声道音频。
	Diarization bool `mapstructure:"diarization"`
	// StudentNames 是脱敏词典；为空时仅依赖称谓正则（开发计划 §5.2-5）。
	StudentNames []string `mapstructure:"studentNames"`
}

// TimeoutDuration 解析单任务总超时，缺省 30 分钟（45 分钟课堂音频需留足余量）。
func (t TranscriptionConfig) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(t.Timeout)
	if err != nil || d <= 0 {
		return 30 * time.Minute
	}
	return d
}

// PollIntervalDuration 解析任务轮询间隔，缺省 5 秒。
func (t TranscriptionConfig) PollIntervalDuration() time.Duration {
	d, err := time.ParseDuration(t.PollInterval)
	if err != nil || d <= 0 {
		return 5 * time.Second
	}
	return d
}

// Concurrency 返回并发闸门容量，缺省 2，且不小于 1。
func (t TranscriptionConfig) Concurrency() int {
	if t.MaxConcurrency < 1 {
		return 2
	}
	return t.MaxConcurrency
}

// MaskedAPIKey 返回可安全写入日志的掩码形式；禁止记录原始 APIKey。
func (t TranscriptionConfig) MaskedAPIKey() string {
	if t.APIKey == "" {
		return "(未配置)"
	}
	if len(t.APIKey) <= 10 {
		return "***"
	}
	return t.APIKey[:6] + "***" + t.APIKey[len(t.APIKey)-4:]
}

// AgentConfig 描述 AI 评析（LLM）参数（2026-10-16 重构 · D3 / R0.2）。
//
// 安全约定：APIKey 只允许来自环境变量 AIJIAOXUE_AGENT_API_KEY，
// config.yaml 中保持空串；本结构体禁止出现在任何 DTO 中。
type AgentConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// APIKey 只允许来自环境变量 AIJIAOXUE_AGENT_API_KEY；排障只用 MaskedAPIKey()。
	APIKey      string `mapstructure:"apiKey"`
	WorkspaceID string `mapstructure:"workspaceId"`
	// LLMModel 为控制台模型名，必须配置化（模型名带版本且会迭代）。
	LLMModel       string `mapstructure:"llmModel"`
	BaseURL        string `mapstructure:"baseUrl"`
	Timeout        string `mapstructure:"timeout"`
	MaxConcurrency int    `mapstructure:"maxConcurrency"`
	// PromptVersion 写入 evidence.promptVersion，变更提示词时必须递增。
	PromptVersion string `mapstructure:"promptVersion"`
	// ReportDebounce 是教师级报告（S2）的去抖窗口；缺省 10m。
	ReportDebounce string `mapstructure:"reportDebounce"`
	// ReportEnabled 关闭时停止生成 S2，但读接口仍可用。
	ReportEnabled bool `mapstructure:"reportEnabled"`
}

// ReportDebounceDuration 解析 S2 去抖窗口，缺省 10 分钟；非法值同样回退 10 分钟。
func (a AgentConfig) ReportDebounceDuration() time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(a.ReportDebounce))
	if err != nil || d <= 0 {
		return 10 * time.Minute
	}
	return d
}

// TimeoutDuration 解析单次评析超时，缺省 3 分钟。
func (a AgentConfig) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(a.Timeout)
	if err != nil || d <= 0 {
		return 3 * time.Minute
	}
	return d
}

// Concurrency 返回并发闸门容量，缺省 2，且不小于 1。
func (a AgentConfig) Concurrency() int {
	if a.MaxConcurrency < 1 {
		return 2
	}
	return a.MaxConcurrency
}

// MaskedAPIKey 返回可安全写入日志的掩码形式；禁止记录原始 APIKey。
func (a AgentConfig) MaskedAPIKey() string {
	if a.APIKey == "" {
		return "(未配置)"
	}
	if len(a.APIKey) <= 10 {
		return "***"
	}
	return a.APIKey[:6] + "***" + a.APIKey[len(a.APIKey)-4:]
}

// EmbeddingConfig 描述知识库向量化参数（RAG 方案 §5）。
//
// 安全约定：与 Agent 共用同一套百炼凭据（AIJIAOXUE_AGENT_API_KEY /
// AIJIAOXUE_AGENT_WORKSPACE_ID），本结构体不保存 Key，避免出现第二份凭据。
// 模型一旦选定不得随意更换：向量空间不同，历史 chunk 必须 rebuild。
type EmbeddingConfig struct {
	Enabled   bool    `mapstructure:"enabled"`
	Model     string  `mapstructure:"model"`
	Dim       int     `mapstructure:"dim"`
	BatchSize int     `mapstructure:"batchSize"`
	TopK      int     `mapstructure:"topK"`
	MinScore  float64 `mapstructure:"minScore"`
}

// EffectiveDim 返回向量维度，缺省 1024。
func (e EmbeddingConfig) EffectiveDim() int {
	if e.Dim <= 0 {
		return 1024
	}
	return e.Dim
}

// EffectiveBatchSize 返回批量大小，缺省 20，且不超过 qwen3.7-text-embedding-flash 的上限 20。
func (e EmbeddingConfig) EffectiveBatchSize() int {
	if e.BatchSize <= 0 {
		return 20
	}
	if e.BatchSize > 20 {
		return 20
	}
	return e.BatchSize
}

// EffectiveTopK 返回检索 topK，缺省 8。
func (e EmbeddingConfig) EffectiveTopK() int {
	if e.TopK <= 0 {
		return 8
	}
	return e.TopK
}

// EffectiveMinScore 返回相似度阈值，缺省 0.35。
func (e EmbeddingConfig) EffectiveMinScore() float64 {
	if e.MinScore <= 0 || e.MinScore > 1 {
		return 0.35
	}
	return e.MinScore
}

// DemoConfig 描述演示模式开关（B6/N7）。仅本地/dev 环境开启，生产必须关闭。
type DemoConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// CORSConfig 描述跨域白名单。
type CORSConfig struct {
	Origins []string `mapstructure:"origins"`
	// AllowAnyOrigin 放行任意来源，仅供本地/局域网联调。
	//
	// 为什么需要它：前端 dev server 除 localhost 外还会监听内网 IP（Vite 的 Network 地址），
	// 用内网地址打开页面时浏览器会带 `Origin: http://<内网IP>:5173`，
	// 若不在白名单里，登录等 POST 请求会被 CORS 直接拦成 403（curl 不带 Origin，故测不出来）。
	//
	// 🔴 生产必须为 false：本服务 AllowCredentials=true，放行任意来源等于放弃跨域保护。
	AllowAnyOrigin bool `mapstructure:"allowAnyOrigin"`
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
	v.SetDefault("transcription.model", "")
	v.SetDefault("transcription.apiKey", "")
	v.SetDefault("transcription.workspaceId", "")
	v.SetDefault("transcription.baseUrl", "")
	v.SetDefault("transcription.timeout", "30m")
	v.SetDefault("transcription.pollInterval", "5s")
	v.SetDefault("transcription.maxConcurrency", 2)
	v.SetDefault("transcription.diarization", false)
	v.SetDefault("agent.enabled", false)
	v.SetDefault("agent.apiKey", "")
	v.SetDefault("agent.workspaceId", "")
	v.SetDefault("agent.llmModel", "")
	v.SetDefault("agent.baseUrl", "")
	v.SetDefault("agent.timeout", "3m")
	v.SetDefault("agent.maxConcurrency", 2)
	v.SetDefault("agent.promptVersion", "v1")
	v.SetDefault("agent.reportDebounce", "10m")
	v.SetDefault("agent.reportEnabled", true)
	v.SetDefault("embedding.enabled", false)
	v.SetDefault("embedding.model", "")
	v.SetDefault("embedding.dim", 1024)
	v.SetDefault("embedding.batchSize", 20)
	v.SetDefault("embedding.topK", 8)
	v.SetDefault("embedding.minScore", 0.35)
	v.SetDefault("demo.enabled", false)
	v.SetDefault("cors.allowAnyOrigin", false)

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

	// ⚠️ 关键：AutomaticEnv 发现的键不在 AllKeys() 中，viper.Unmarshal 会静默跳过，
	// 导致新键的环境变量永远读不到。新键必须额外 SetDefault（上方已做）或 BindEnv（下方）。
	// 显式绑定同时消除了驼峰歧义（自动推导会得到 ..._APIKEY 而非 ..._API_KEY）。
	_ = v.BindEnv("transcription.apiKey", "AIJIAOXUE_TRANSCRIPTION_API_KEY")
	_ = v.BindEnv("transcription.workspaceId", "AIJIAOXUE_TRANSCRIPTION_WORKSPACE_ID")
	_ = v.BindEnv("transcription.model", "AIJIAOXUE_TRANSCRIPTION_MODEL")
	_ = v.BindEnv("transcription.baseUrl", "AIJIAOXUE_TRANSCRIPTION_BASE_URL")
	_ = v.BindEnv("agent.apiKey", "AIJIAOXUE_AGENT_API_KEY")
	_ = v.BindEnv("agent.workspaceId", "AIJIAOXUE_AGENT_WORKSPACE_ID")
	_ = v.BindEnv("agent.llmModel", "AIJIAOXUE_AGENT_LLM_MODEL")
	_ = v.BindEnv("agent.baseUrl", "AIJIAOXUE_AGENT_BASE_URL")
	_ = v.BindEnv("agent.reportDebounce", "AIJIAOXUE_AGENT_REPORT_DEBOUNCE")
	_ = v.BindEnv("agent.reportEnabled", "AIJIAOXUE_AGENT_REPORT_ENABLED")
	_ = v.BindEnv("embedding.model", "AIJIAOXUE_EMBEDDING_MODEL")
	_ = v.BindEnv("embedding.dim", "AIJIAOXUE_EMBEDDING_DIM")
	_ = v.BindEnv("embedding.batchSize", "AIJIAOXUE_EMBEDDING_BATCH_SIZE")
	_ = v.BindEnv("demo.enabled", "AIJIAOXUE_DEMO_ENABLED")

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
	// 转写开启时做启动期校验：宁可起不来，也不要等督导传完音频后才在转写记录里发现 Key 缺失。
	if cfg.Transcription.Enabled {
		if strings.TrimSpace(cfg.Transcription.APIKey) == "" {
			return nil, fmt.Errorf("config: 已启用 transcription，但环境变量 AIJIAOXUE_TRANSCRIPTION_API_KEY 为空")
		}
		if strings.TrimSpace(cfg.Transcription.Model) == "" {
			return nil, fmt.Errorf("config: 已启用 transcription，但 transcription.model 未配置")
		}
		// engine_version 直接存模型名，列宽见 migrations/6_transcripts_engine_width.up.sql。
		// 在这里拦住远超列宽的配置：一旦写入触发 1406，等于已经为这次 ASR 付过费却存不下来。
		if n := len([]rune(cfg.Transcription.Model)); n > 64 {
			return nil, fmt.Errorf("config: transcription.model 长度 %d 超过 transcripts.engine_version 列宽 64", n)
		}
		if n := len([]rune(cfg.Transcription.Engine)); n > 64 {
			return nil, fmt.Errorf("config: transcription.engine 长度 %d 超过 transcripts.engine 列宽 64", n)
		}
	} else if strings.TrimSpace(cfg.Transcription.APIKey) != "" {
		slog.Warn("transcription 未启用，但检测到 API Key 已注入，本次启动将忽略 ASR")
	}
	// AI 评析开启时做启动期校验：宁可起不来，也不要等转写完成才发现 Key/模型缺失。
	if cfg.Agent.Enabled {
		if strings.TrimSpace(cfg.Agent.APIKey) == "" {
			return nil, fmt.Errorf("config: 已启用 agent，但环境变量 AIJIAOXUE_AGENT_API_KEY 为空")
		}
		if strings.TrimSpace(cfg.Agent.LLMModel) == "" {
			return nil, fmt.Errorf("config: 已启用 agent，但 agent.llmModel 未配置")
		}
	}
	// 知识库向量化与 Agent 共用凭据：启用时必须能拿到 Key 与模型，避免摄取到一半才失败。
	if cfg.Embedding.Enabled {
		if strings.TrimSpace(cfg.Agent.APIKey) == "" {
			return nil, fmt.Errorf("config: 已启用 embedding，但环境变量 AIJIAOXUE_AGENT_API_KEY 为空")
		}
		if strings.TrimSpace(cfg.Embedding.Model) == "" {
			return nil, fmt.Errorf("config: 已启用 embedding，但 embedding.model 未配置")
		}
		if cfg.Embedding.Dim < 1 || cfg.Embedding.Dim > 8192 {
			return nil, fmt.Errorf("config: embedding.dim=%d 非法（应为 1–8192）", cfg.Embedding.Dim)
		}
		if cfg.Embedding.BatchSize < 1 || cfg.Embedding.BatchSize > 20 {
			return nil, fmt.Errorf("config: embedding.batchSize=%d 非法（qwen3.7-text-embedding-flash 上限为 20）", cfg.Embedding.BatchSize)
		}
	}
	if cfg.CORS.AllowAnyOrigin {
		// 与 AllowCredentials=true 组合等于放弃跨域保护，必须让运维在日志里一眼看到。
		slog.Warn("CORS 已放行任意来源（cors.allowAnyOrigin=true），仅可用于本地/局域网联调，生产必须关闭",
			"mode", cfg.Server.Mode)
	}
	return &cfg, nil
}
