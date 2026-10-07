#  「爱教学」后端编码智能体指导手册（Go + Gin）

> **文档用途**：本文件是写给 AI 智能体（编码助手 GLM 5.3 + OpenCode、架构助手 DeepSeek V4 Pro）的后端编码工作宪法。
> 智能体在为本项目编写任何后端代码之前，**必须先完整阅读本文件**，并严格遵守其中的技术栈、目录结构、分层架构、接口契约与编码规则。
> **文档位置**：本文件位于 `docs/backend_AGENTS.md`；后端仓库速查入口为 `backend/AGENTS.md`。
> **配套文档（单一事实源分工）**：接口契约 → 本文 §8；页面/组件/设计 → [`frontend_AGENTS.md`](./frontend_AGENTS.md)；DDL/schema → 代码 `backend/database/schema.sql` + `backend/migrations/`；评分体系/权限隐私/DoD/任务 → [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md)；建库/种子/备份/排查 → [`MySQL数据库创建指导.md`](./MySQL数据库创建指导.md)；启停/初始化命令 → [`README.md`](../README.md)「For Developers」。

> 🔴 **文档同步要求（强制）**：本手册是后端开发的事实源之一。任何改动若涉及**路由 / 接口字段、数据模型或表结构、错误码、权限与数据范围、计分口径、Sprint 范围**，
> 必须在**同一个 PR** 内同步更新本文件，以及 `docs/Sprint2-3-教学评价与提优-开发计划.md` 中对应的契约章节。
> **契约先行：先改文档，再改代码。只改代码不改文档，视为未完成。**

---

## 1. 项目背景

「爱教学」是面向高校的教学质量全链路数字化管理平台，走三阶进化路线：**查课程（Sprint 1）→ 看课堂（Sprint 2）→ 帮教师（Sprint 3）**。

| 阶段 | 主题 | 一句话目标 | 状态 |
|------|------|-----------|------|
| Sprint 1 | 查课程 | 课程信息透明可查、三角色数据打通、Web 框架稳定运行 | ✅ 已交付 |
| Sprint 2 | 看课堂 | 授课记录 + 督导/智能体课堂评价闭环；课堂录音、异步转写与智能体接入 | 🚧 **当前阶段**（阶段① 评价闭环**已实现**；阶段② **部分实现**：录音上传/异步转写/脱敏/播放票据已落地，**智能体评分未接入**） |
| Sprint 3 | 帮教师 | 教学提优：评语与提优建议、趋势折线、智能体对话、申诉复核、质量报告 | ⏳ **未实现**（对应开发计划「阶段三」，待排期） |

后端使命始终是**保障数据链路连通无误**：登录鉴权 → 角色化数据裁剪 → 领域数据（课程 / 资源 / 督导 / 授课 / 评价）的可靠读写。

## 2. 三次 Sprint 的目标与范围

> 三个阶段同等重要，本节均衡说明；**当前处于 Sprint 2**（见 §1 状态表）。详细契约与 DDL 见 [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md)。

### 2.1 Sprint 1「查课程」（已交付）：必须支撑的故事

| 编号 | 角色 | 故事 | 优先级 | 后端落点 |
|------|------|------|--------|---------|
| S1.1 | — | 技术基线：Web 整体框架可无误运作，支撑登录、导航与数据链路 | M | 工程骨架 · 路由注册 · 中间件 · `/healthz` |
| S1.2 | 任意用户 | 统一账号按角色登录，以便用户间数据打通 | M | `/auth/login` · JWT 签发 |
| S2.1 | 主任 | 查看本室教师课程简介与开课情况，以便统筹排课 | M | `GET /courses` 按 `department_id` 裁剪 |
| S2.2 | 教师 | 查看本人课程列表与开课信息 | M | `GET /courses` 按 `teacher_id` 裁剪 |
| S2.3 | 督导 | 查看全校课程开设情况 | M | `GET /courses` 全量 + 筛选参数 |
| S3.1 | 主任 | 新增/修改课程信息 | S | `POST/PUT /courses` + 业务校验 |
| S4.1 | 教师 | 查看课程资源与学生人次 | M | `GET /courses/:id` 聚合 + `GET /courses/:id/resources` |
| S4.2 | 教师 | 上传/维护课程资源 | S | multipart 上传 · 本地存储 · 删除 |
| S5.1 | 督导 | 查看督导覆盖率与听评课安排 | M | `/supervision/coverage` · `/supervision/plans` |

### 2.2 Sprint 2「看课堂」（当前阶段）：目标与范围

Sprint 2 分两阶段推进。**任务编号与完整任务列表见 [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) §7.1 / §7.2 / §7.3**（本文件不重复维护任务列表）。

**阶段① 评价闭环（Sprint 2.1，后端已落地）**

| 能力 | 后端落点 |
|------|---------|
| 授课记录 | `POST /sessions`（督导建课）、`GET /courses/:id/sessions`、`GET /sessions/:id` |
| 督导评分 | `PUT /sessions/:id/supervisor-evaluation`：**五维必填**、幂等覆盖、写 `formula_version` |
| 评分计算 | `pkg/scoring`：维度权重、单次总分、综合均值聚合（纯函数 + 表驱动单测） |
| 评分聚合 | 教师级 / 课程级共用 `scoring.Aggregate`：`GET /teacher-scores`、`GET /teachers/:id/evaluation-summary`、`GET /courses/:id/evaluation-summary` |
| 评价时间线 | `GET /teachers/:id/evaluations`：按课次倒序的历次评价（含督导结构化评语与智能体参考），供教师面板一次拉取渲染 |
| 当堂课评估 | `GET /sessions/:id/evaluation`（场次 + 双侧评分 + 评语；录音/转写位在阶段②填充） |
| 数据迁移 | `golang-migrate`，`migrations/2_teaching_sessions_and_evaluations.*.sql`（§13） |

**阶段② 智能体接入（Sprint 2.2，部分实现）**

已实现（后端落点，详见 §16）：

| 能力 | 后端落点 |
|------|---------|
| 课堂录音上传与流式播放（Range）、播放审计日志 | `internal/handler/recording.go` · `internal/service/recording.go` · `internal/router/router.go:84` |
| 异步转写：`transcripts.status` 状态机（`pending→running→done/failed`）+ 失败重试 + 启动重排 | `internal/asr/client.go` · `internal/service/recording.go`（`RequeueStuck`） |
| 学生姓名脱敏（称谓正则 + 可选词典） | `internal/service/desensitize.go` |
| 短时播放票据（`?ticket=`） | `internal/middleware/auth.go:50`（`AuthOrPlaybackTicket`） |

**未实现（不得按已完成对待）**：

- **智能体评分未接入**：`internal/repository/recording.go:79` 已定义 `UpsertAgentEvaluation`，但**无任何调用方**，`router.go` 亦无 `/agent/*` 路由（契约见开发计划 §4.4）。聚合侧本已能读取 `evaluations` 中的 agent 行（`pkg/scoring/scoring.go:140`、`internal/service/teacherscore.go:330`、`internal/service/session.go:179`），但**没有写入通道**——现有 agent 行只来自种子数据；
- 智能体故障降级未接入：`50003` 目前只覆盖转写链路（ASR 失败写 `failed` + `error_message`，不影响督导评分主流程）；`50002`（`NotImplemented` → HTTP 501）已在 `pkg/errcode` 定义但**当前无任何调用方**——`router.go` 无 `/agent/*` 路由，请求实际落 gin 默认 **404**。智能体评分链路尚未接入，故无其降级可言。

### 2.3 Sprint 3「帮教师」（待排期）：目标与范围

- 督导评语（仅本人可见，时间倒序，标注对应课次）与智能体提优建议；
- 各维度分数历史趋势接口（`GET /teachers/:id/score-trend?semester=&dimension=`）；
- 与智能体就课堂改进流式对话（SSE，断流可重连）；
- 教师申诉 / 督导复核流程（留痕、状态可追溯）、督导间评分校准、质量报告导出。

### 2.4 范围边界（Won't）

> **⚠️ 范围更新（Sprint 2/3 已启动）**
> 原 Sprint 1 Won't 清单中的「课堂录音上传 / ASR 语音转写接入」与「质量评估报告 / 教学优化建议」**已解除**，正式进入 Sprint 2「看课堂」与 Sprint 3「帮教师」。
> 新的开发目标、用户故事、评分体系、数据模型与接口契约见 **[`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md)**；本文件 §8 契约与 §13 模型表随该计划同步扩充。

**Sprint 2/3 仍然不做（Won't）**：

- ❌ 用户注册、找回密码、第三方登录（账号仍由种子数据预置）
- ❌ 消息通知、全文搜索引擎、读写分离、微服务拆分
- ❌ 引入第二个 Web 框架或 ORM
- ❌ 移动端 App / 真实教务系统对接 / 实时课堂直播 / 视频画面分析

**智能体守则：收到超出上述边界的编码请求时，必须在回复中明确指出"该需求超出当前 Sprint 范围"，不得直接实现。**

## 3. 角色与数据范围规则（本手册最核心的一节）

三角色英文标识全小写，贯穿 JWT claims、RBAC 中间件与数据裁剪：

| 角色 | 标识 | 课程列表数据范围 | SQL 条件（伪） |
|------|------|----------------|---------------|
| 教研室主任 | `director` | 本教研室全部课程 | `courses.department_id = :user.department_id` |
| 教师 | `teacher` | 本人授课课程 | `courses.teacher_id = :user.id` |
| 教学督导 | `supervisor` | 全校课程 | 无附加条件 |

**铁律**：
1. **数据范围裁剪只能发生在后端**（service 层拼接查询条件），前端传来的筛选参数只是追加条件，不构成权限依据；
2. 每个查询接口实现时，先回答"这个角色看多大范围"，再写 SQL；
3. 写操作额外校验归属（见 §11），越权返回 `40302`。

## 4. 技术栈（版本锁定，不得擅自升级或替换）

| 类别 | 选型 | 版本 | 说明 |
|------|------|------|------|
| 语言 | Go | 1.22+ | 启用 `GOEXPERIMENT` 无需，标准库即可 |
| Web 框架 | Gin | v1.10 | 唯一框架 |
| ORM | GORM | v1.25 + `gorm.io/driver/mysql` | 参数化查询，防注入 |
| 数据库 | MySQL | 8.0 | utf8mb4，见配套 MySQL 文档 |
| 迁移 | golang-migrate/migrate | v4 | Sprint 2 起引入；文件命名 `<版本>_<名称>.up/down.sql` |
| 认证 | golang-jwt/jwt | v5 | HS256，TTL 24h |
| 密码 | golang.org/x/crypto/bcrypt | latest | cost 10 |
| 配置 | spf13/viper | v1.18 | `config.yaml` + 环境变量覆盖 |
| 日志 | log/slog（标准库） | — | JSON 输出，请求日志中间件 |
| 校验 | gin 内置 validator | — | binding tag |
| CORS | gin-contrib/cors | v1.7 | 白名单 origins |
| 热重载 | air-verse/air | latest | 仅开发环境 |
| 测试 | testing（标准库）+ testify | v2 | service / scoring / errcode 单测 |

## 5. 目录结构（强制）

```
aijiaoxue-api/                # 本仓库中的 backend/
├── AGENTS.md                # 后端速查入口（指向 ../docs/backend_AGENTS.md，即本文件）
├── go.mod / go.sum
├── Makefile                 # run / seed / test / lint / build
├── config.yaml              # 本地配置（config.example.yaml 提交模板，config.yaml 不入库）
├── cmd/
│   ├── server/main.go       # 唯一服务入口（启动时驱动 RequeueStuck）
│   ├── migrate/main.go      # golang-migrate 执行器（up / down [N] / version）
│   └── seed/main.go         # 种子数据工具（bcrypt 写入演示账号）
├── migrations/              # 版本化迁移（embed 打包）；命名 <版本>_<名称>.up/down.sql
│   ├── 1_baseline_sprint1.{up,down}.sql
│   ├── 2_teaching_sessions_and_evaluations.{up,down}.sql
│   ├── 3_recordings_and_transcripts.{up,down}.sql   # 录音 / 转写 / 播放审计
│   ├── 4_transcripts_content_default.{up,down}.sql  # transcripts.content 默认值
│   ├── 5_evaluation_drafts.{up,down}.sql            # 督导评估草稿
│   └── 6_transcripts_engine_width.{up,down}.sql     # engine / engine_version 放宽到 VARCHAR(64)
├── internal/
│   ├── asr/client.go        # ASR 适配层（百炼 filetrans：上传→提交→轮询→解析）
│   ├── config/config.go     # viper 加载与结构体（含 evaluation 计分口径、transcription 启动校验）
│   ├── router/router.go     # 路由表 + 分组（唯一路由注册地）
│   ├── middleware/          # recovery / logger / cors / auth / rbac
│   ├── handler/             # HTTP 层：参数绑定、调用 service、组装响应
│   │   ├── auth.go course.go resource.go supervision.go dict.go health.go bind.go
│   │   ├── session.go teacherscore.go     # Sprint 2.1
│   │   └── recording.go draft.go          # Sprint 2.2 阶段② / 草稿箱
│   ├── service/             # 业务层：权限判断、业务校验、事务边界、聚合
│   │   ├── auth.go course.go resource.go supervision.go dashboard.go dict.go
│   │   ├── session.go teacherscore.go     # Sprint 2.1
│   │   └── recording.go draft.go mapper.go desensitize.go   # Sprint 2.2 阶段②
│   ├── repository/          # 数据层：GORM 查询，无业务逻辑
│   │   ├── user.go course.go resource.go supervision.go
│   │   ├── session.go evaluation.go       # Sprint 2.1
│   │   └── recording.go draft.go          # Sprint 2.2 阶段②
│   ├── model/               # GORM 模型，与表一一对应（recording.go = Recording / Transcript）
│   └── dto/                 # 请求/响应结构体（json tag 与前端 types 对齐）
│       ├── auth.go course.go dashboard.go dict.go page.go resource.go supervision.go validate.go
│       └── draft.go evaluation.go         # 草稿 / 评价聚合 / 录音转写 DTO
├── pkg/
│   ├── response/            # OK / Fail 统一响应
│   ├── errcode/             # 错误码常量 + HTTP 映射（含单测）
│   ├── scoring/             # 评分纯函数：权重 / 单次总分 / 综合均值聚合
│   └── jwtutil/             # 签发与解析
├── database/
│   ├── schema.sql           # Sprint 1 存量表建表脚本（新表以 migrations/ 为准）
│   └── seed.sql             # 种子数据（含 §3.4 授课/评价）；密码哈希由 cmd/seed 覆写
├── uploads/                 # 文件存储（.gitignore）：课程资源 + recordings/{sessionId}/{uuid}.{ext}
└── scripts/
    ├── init_db.sh           # 建库 → migrate up → seed.sql → cmd/seed
    └── verify.sh            # 端到端验收（Sprint 1 + 2.1，144 条 want 断言；无录音/转写断言，见 §15）
```

**目录铁律**：
- 依赖方向单向：`handler → service → repository → model`，禁止反向 import、禁止跨层跳调（handler 直查库 = 违规）；
- `router/` 之外**禁止注册路由**；`handler/` 之外**禁止读写 `*gin.Context`**；
- `dto/` 的 json tag 必须与前端 `types/` 字段名逐字对齐（camelCase）。

## 6. 配置管理（config.yaml）

> 模板即事实源：`backend/config.example.yaml`。以下片段与其保持同步，字段值以该文件为准。

```yaml
server:
  port: 8080
  mode: debug                          # debug | release
mysql:
  dsn: "aijiaoxue:aijiaoxue_dev@tcp(127.0.0.1:3306)/aijiaoxue?charset=utf8mb4&parseTime=True&loc=Local"
  maxOpenConns: 20
  maxIdleConns: 5
jwt:
  secret: "change-me-in-production"   # 生产环境必须用环境变量 AIJIAOXUE_JWT_SECRET 覆盖
  ttl: "24h"
upload:
  dir: "./uploads"
  maxSize: 104857600                  # 100MB
  allowExt: [".pdf", ".doc", ".docx", ".ppt", ".pptx", ".mp4", ".zip"]

# 课堂音频转写（ASR，Sprint 2.2 阶段②）。详见 §16。
# 🔴 API Key 只走环境变量 AIJIAOXUE_TRANSCRIPTION_API_KEY（显式 BindEnv），
#    本文件 apiKey 永远留空；enabled=true 且 Key 为空时启动即 Fail-Fast。
# ⚠️ 变量名带下划线：自动推导会得到 ..._APIKEY（连写）而失效，必须显式绑定。
transcription:
  enabled: false
  engine: "dashscope-qwen-audio-asr"            # 写入 transcripts.engine
  engineVersion: "qwen-audio-3.1-asr-flash-filetrans"
  model: "qwen-audio-3.1-asr-flash-filetrans"
  apiKey: ""                                    # 永远留空，由环境变量覆盖
  workspaceId: ""                               # 百炼业务空间 ID，如 ws-xxxxxxxx
  baseUrl: ""                                   # 留空则按 workspaceId 推导
  timeout: "30m"
  pollInterval: "5s"
  maxConcurrency: 2
  diarization: false                            # 说话人分离：仅支持单声道音频
  studentNames: []                              # 脱敏词典；为空时仅用称谓正则

cors:
  origins: ["http://localhost:5173", "http://127.0.0.1:5173"]
  # 放行任意来源，仅限本地/局域网联调（Vite Network 地址）；生产必须 false。
  # 🔴 因 AllowCredentials=true，禁止写 AllowOrigins: ["*"]（gin-contrib/cors 会 panic）。
  allowAnyOrigin: false

# 课堂评价计分口径（开发计划 §2.2）。权重合计必须为 1.00，服务启动时校验。
# 变更口径时必须递增 formulaVersion。
evaluation:
  weights:                            # frontier 为观测项，权重 0，不计入加权总分
    objective: 0.30
    content: 0.30
    interaction: 0.20
    organization: 0.20
    frontier: 0.00
  supervisorWeight: 0.5               # α：综合分中督导评分权重
  agentWeight: 0.5                    # 1−α：综合分中智能体评分权重
  formulaVersion: "v1"
  minSampleSize: 3                    # 低于该值标记"样本不足"
```

- 环境变量优先级高于 yaml，键名前缀 `AIJIAOXUE_`（viper `AutomaticEnv` + `SetEnvKeyReplacer`）；
  **新增配置键必须在 `config.Load` 中 `SetDefault` 或 `BindEnv`**，否则 `AutomaticEnv` 发现的键不在 `AllKeys()` 中，`Unmarshal` 会静默跳过（`internal/config/config.go:207-213` 有注释说明）；
- **密钥不得提交仓库**：仓库内只放 `config.example.yaml`，`config.yaml` 进 `.gitignore`；
- 启动期强制校验：`mysql.dsn` / `jwt.secret` 非空、`evaluation.weights` 非零项合计为 1.00、`transcription.enabled` 时 Key/Model 非空且 `engine`/`model` 长度 ≤64（`internal/config/config.go:223-260`）；
- 前端开发期推荐 Vite proxy 将 `/api` 代理到 `127.0.0.1:8080`，CORS 作为兜底配置。

## 7. 统一响应与错误码

所有业务接口返回 HTTP 200 + 信封：

```json
{ "code": 0, "message": "ok", "data": { } }
```

| code | HTTP | 含义 | 触发示例 |
|------|------|------|---------|
| 0 | 200 | 成功 | — |
| 40001 | 400 | 参数校验失败 | 缺少必填字段、格式错误 |
| 40002 | 400 | 业务规则校验失败 | 授课日期晚于今天、督导评分五维未录全 |
| 40101 | 401 | 未登录 / token 失效 | 无 Authorization、token 过期 |
| 40301 | 403 | 无角色权限 | 教师调用 `POST /courses` |
| 40302 | 403 | 无数据操作权限 | 教师上传他人课程资源 |
| 40401 | 404 | 资源不存在 | 课程 id 无效 |
| 40901 | 409 | 数据已存在 | 课程编码+学期重复、同课程同日同节次重复建课 |
| 50001 | 500 | 服务器内部错误 | 数据库异常 |
| 50002 | 501 | 功能未实现（Sprint 2.2 起） | 阶段一调用智能体接口 |
| 50003 | 503 | 依赖服务不可用（Sprint 2.2 起） | ASR 引擎未配置或转写失败 |

> 🔴 **新增错误码必须三处同步**：`Code` 常量、`messages` 文案、`HTTPStatus()` 映射，并补 `pkg/errcode/errcode_test.go` 的表驱动用例。
> 历史回归：`40002` 曾漏配 `HTTPStatus()`，落 `default → 500`——响应体 code 正确但 HTTP 500，日志被记为服务端错误。
> `40902` 为「同一督导重复评分」保留码；因 `PUT` 幂等覆盖，**不会实际返回**。

- 鉴权失败返回 **HTTP 401 + code 40101**，前端 axios 拦截器据此清 token 跳登录页（与前端 AGENTS.md §10.3 对齐）；
- `message` 面向用户可读（中文），内部细节写日志不外泄；
- 实现：`pkg/response.OK(c, data)` / `pkg/response.Fail(c, errcode.Params)`，handler 中**禁止手写 `c.JSON` 拼信封**。

## 8. 接口契约（已注册路由共 36 条）

> 计数口径 = `backend/internal/router/router.go` 实际 `r.GET/POST/PUT/DELETE` 注册数，按鉴权分组：
> 未挂 `Auth` 组 4（`GET /healthz` 根路径 + `GET /api/v1/healthz` + `POST /auth/login` + `GET /recordings/:id/stream`，
> 最后一条自带 `AuthOrPlaybackTicket`）
> \+ 登录 17 + director 2 + teacher 2 + supervisor 10 + director/supervisor 1。
> 下文 §8.1–§8.7 为 Sprint 1 存量，§8.8 为 Sprint 2.1（14 条），§8.9 为 Sprint 2.2 阶段②（4 条）。

前缀 `/api/v1`。**字段名与前端 `src/types/` 逐字对齐，本节是前后端联调的唯一事实源。**

> **已实现接口**：Sprint 2.1 见本文 §8.8，Sprint 2.2 阶段②（录音与转写）见 §8.9。
> **未实现接口**：智能体评分/对话（`/agent/*`，**未注册路由**，请求落 404）与阶段三趋势等见
> [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) §4.2 / §4.4。**未实现即不得在联调中当作可用**。

### 8.1 认证 auth

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| POST `/auth/login` | 公开 | 登录 |
| GET `/auth/me` | 登录 | 当前用户 |
| POST `/auth/logout` | 登录 | 退出（Sprint 1 无服务端黑名单，返回成功即可） |

```go
// POST /auth/login
type LoginReq struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}
type UserDTO struct {
    ID         uint64 `json:"id"`
    Name       string `json:"name"`
    Role       string `json:"role"`               // director|teacher|supervisor
    Department string `json:"department,omitempty"` // 教研室名称（督导为空）
    JobNo      string `json:"jobNo,omitempty"`
}
type LoginResp struct {
    Token string  `json:"token"`
    User  UserDTO `json:"user"`
}
```

- 逻辑：查用户 → `bcrypt.CompareHashAndPassword` → 签发 JWT（claims：`sub`=user_id、`role`、`exp`）；
- 失败一律返回 `40101` + "账号或密码错误"（不区分账号不存在/密码错误，防枚举）。

### 8.2 工作台 dashboard

| GET `/dashboard` | 登录 | 按角色返回聚合数据，一套接口三种 DTO |
|---|---|---|

```go
type DirectorDashboard struct { // 主任
    CourseCount    int              `json:"courseCount"`    // 本室课程数
    TeacherCount   int              `json:"teacherCount"`   // 本室教师数
    ClassCount     int              `json:"classCount"`     // 本学期开课班次
    ResourceCount  int              `json:"resourceCount"`  // 本室课程资源总数
    RecentCourses  []CourseListItem `json:"recentCourses"`  // 前 10 条
}
type TeacherDashboard struct { // 教师
    CourseCount   int              `json:"courseCount"`
    ClassCount    int              `json:"classCount"`
    StudentCount  int              `json:"studentCount"`   // 汇总人次
    ResourceCount int              `json:"resourceCount"`
    MyCourses     []CourseListItem `json:"myCourses"`
}
type SupervisorDashboard struct { // 督导（工作台：聚焦「记录课程并评估」）
    RecentPlans     []PlanItem          `json:"recentPlans"`     // 全部听评课安排（前端按今日/本周/本月分档）
    RecentDrafts    []DraftDTO          `json:"recentDrafts"`    // 最近 3 份草稿（草稿箱区块）
    PendingSessions []PendingSessionItem `json:"pendingSessions"` // 待评估授课记录（含手动新增，最新在前）
}
type PendingSessionItem struct { // 手动新增/尚未评价的授课记录（status=scheduled|recorded）
    SessionID   uint64 `json:"sessionId"`
    CourseID    uint64 `json:"courseId"`
    CourseCode  string `json:"courseCode"`
    CourseName  string `json:"courseName"`
    TeacherID   uint64 `json:"teacherId"`
    TeacherName string `json:"teacherName"`
    SessionDate string `json:"sessionDate"`
    Period      string `json:"period"`
    Topic       string `json:"topic"`
    Status      string `json:"status"`
}
```

> **变更说明（工作台精简）**：督导工作台已移除 `courseCount/planCount/completedCount/coverageRate/byDepartment`
> 等与听评课核心任务无关的统计；覆盖率仍由 `GET /supervision/coverage` 提供。

### 8.3 字典 dict（新增，供前端筛选下拉）

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| GET `/departments` | 登录 | 教研室列表 `[{id, name}]` |
| GET `/teachers?departmentId=` | 登录 | 教师列表 `[{id, name}]`，按教研室过滤 |

> 注：此二接口为前端 AGENTS.md §10.2 的补充项（前端文档已同步更新）。学期选项由前端常量维护，不设接口。

### 8.4 课程 courses

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| GET `/courses` | 登录 | 分页列表，数据范围按角色裁剪；`mine=1` 时**督导**只返回本人听评课计划覆盖的课程 |
| GET `/courses/:id` | 登录 | 详情（含班级） |
| POST `/courses` | director | 新增 |
| PUT `/courses/:id` | director | 修改（code 不可改） |

```go
// GET /courses 查询参数（全部可选）
//   semester, departmentId, teacherId, status, keyword, mine, page(默认1), pageSize(默认10, max50)
//   mine=true 仅对 supervisor 生效：EXISTS(supervision_plans WHERE course_id=c.id AND supervisor_id=本人)；
//   其他角色忽略该参数（数据范围由 service 层角色裁剪决定，前端参数不构成权限依据）。
type CourseListItem struct {
    ID            uint64 `json:"id"`
    Code          string `json:"code"`
    Name          string `json:"name"`
    TeacherName   string `json:"teacherName"`
    Department    string `json:"department"`
    Semester      string `json:"semester"`
    ClassCount    int    `json:"classCount"`
    StudentCount  int    `json:"studentCount"`   // SUM(班级学生数)，查询时聚合
    ResourceCount int    `json:"resourceCount"`
    Status        string `json:"status"`          // open|draft|closed
}
type ClassInfoDTO struct {
    ID           uint64 `json:"id"`
    ClassName    string `json:"className"`
    Schedule     string `json:"schedule"`
    Location     string `json:"location"`
    StudentCount int    `json:"studentCount"`
}
type CourseDetail struct {
    CourseListItem
    Credit      int             `json:"credit"`
    Hours       int             `json:"hours"`
    Description string          `json:"description"`
    TeacherID   uint64          `json:"teacherId"`
    Classes     []ClassInfoDTO  `json:"classes"`
}
// POST/PUT /courses
type CourseUpsertReq struct {
    Code         string `json:"code" binding:"required,max=12"` // 服务端正则 ^[A-Z]{2,4}\d{4}$；编辑态忽略
    Name         string `json:"name" binding:"required,max=128"`
    DepartmentID uint64 `json:"departmentId" binding:"required"`
    TeacherID    uint64 `json:"teacherId" binding:"required"`
    Semester     string `json:"semester" binding:"required,max=16"`
    Credit       int    `json:"credit" binding:"required,min=1,max=6"`
    Hours        int    `json:"hours" binding:"required,min=16,max=128"`
    Description  string `json:"description" binding:"max=500"`
    Status       string `json:"status" binding:"required,oneof=open draft closed"`
}
```

- 分页响应：`{ list, total, page, pageSize }`（与前端 `PageResult<T>` 对齐）；
- `studentCount` / `resourceCount` **不落库**，列表查询用子查询聚合，保证一致性。

### 8.5 资源 resources

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| GET `/courses/:id/resources` | 登录 | 资源列表 |
| POST `/courses/:id/resources` | teacher | 上传（multipart/form-data，字段名 `file`） |
| DELETE `/resources/:id` | teacher | 删除（校验本人课程） |

```go
type ResourceDTO struct {
    ID         uint64 `json:"id"`
    Name       string `json:"name"`
    Type       string `json:"type"`   // pdf|doc|ppt|video|zip|other（按扩展名映射）
    Size       int64  `json:"size"`   // 字节
    Uploader   string `json:"uploader"`
    UploadedAt string `json:"uploadedAt"` // ISO 8601
}
```

- 上传流程：校验课程存在且 `course.teacher_id == user.id`（否则 40302）→ 校验扩展名白名单与大小 → 存储路径 `uploads/{courseID}/{uuid}{ext}` → 落库相对路径；
- 删除流程：校验归属 → 删库记录（事务内）→ 删磁盘文件（失败仅记日志，不回滚事务）；
- 下载（Sprint 1 可选）：`GET /resources/:id/download` 以 `c.FileAttachment` 输出，文件名做 `filepath.Base` 清洗。

### 8.6 督导 supervision

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| GET `/supervision/coverage` | supervisor | 覆盖率统计 |
| GET `/supervision/plans?page&pageSize&status&dateFrom&dateTo` | supervisor | 听评课安排分页 |

```go
type CoverageDTO struct {
    TotalCourses      int          `json:"totalCourses"`
    SupervisedCourses int          `json:"supervisedCourses"`
    Rate              float64      `json:"rate"`   // 0-1，保留两位
    ByDepartment      []DeptRate  `json:"byDepartment"`
}
type DeptRate struct {
    Department string  `json:"department"`
    Rate       float64 `json:"rate"`
}
type PlanItem struct {
    ID             uint64  `json:"id"`
    CourseID       uint64  `json:"courseId"`
    CourseName     string  `json:"courseName"`
    TeacherName    string  `json:"teacherName"`
    SupervisorName string  `json:"supervisorName"`
    PlannedDate    string  `json:"plannedDate"` // YYYY-MM-DD
    Status         string  `json:"status"`      // planned|completed
    SessionID      *uint64 `json:"sessionId"`   // 关联授课记录；null=尚无，需先创建再评估
    Evaluated      bool    `json:"evaluated"`   // 关联授课记录是否已督导评价（前端据此暴露/隐藏评估入口）
}
```

> **评估入口日期闸门（前端约束）**：`evaluated=false` 且 `plannedDate <= 今天` 才显示「去评估」（无授课记录时先 `POST /sessions` 建档再跳转）；
> `plannedDate > 今天` 仅预览、不开放评估。工作台「未评」筛选列出 `plannedDate < 今天 && !evaluated`（不含今日，专用于补录）。

**覆盖率口径（唯一）**：分母 = 当前学期 `status='open'` 的课程数；分子 = 这些课程中已有 `status='completed'` 听评课记录（DISTINCT course_id）的数量。分部门同理。当前学期取 `semesters` 常量中最新一项（Sprint 1 固定 `2026-2027-1`，定义在 `internal/service/consts.go`）。

### 8.7 运维 healthz

`GET /healthz` 与 `GET /api/v1/healthz`（均无鉴权，同一 handler）返回 `{"code":0,"data":{"status":"up"}}`，
供部署探活与 S1.1 验收（`router.go:74,77`）。

### 8.8 授课记录与评价聚合（Sprint 2.1，已实现，14 条）

> 契约细节与示例响应见 [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) §4.1 / §4.2。

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| GET `/courses/:id/sessions?semester=&page=&pageSize=` | 登录（数据裁剪） | 课程历史授课记录，含每场双侧评分摘要 |
| POST `/sessions` | supervisor | 创建授课记录（日期不得晚于今天，`uk_session` 唯一） |
| GET `/sessions/:id` | 登录（数据裁剪） | 单场次基本信息 |
| GET `/sessions/:id/evaluation` | 登录（数据裁剪） | 当堂课评估页聚合：场次 + 督导评分 + 智能体参考 + 评语 |
| PUT `/sessions/:id/supervisor-evaluation` | supervisor | 提交/覆盖督导评分（幂等）；**五维必填**，缺失 40002 |
| GET `/teacher-scores?departmentId=&semester=&page=&pageSize=` | director（本室）/ supervisor（全校） | 教师评分列表 |
| GET `/teachers/:id/evaluation-summary?semester=` | director（本室）/ teacher（仅自己）/ supervisor | 教师级评分面板 |
| GET `/teachers/:id/evaluations?semester=&page=&pageSize=` | director（本室）/ teacher（仅自己）/ supervisor | 教师历次评价时间线：按课次倒序，含督导结构化评语 + 智能体参考 + 场次综合分 |
| GET `/courses/:id/evaluation-summary?semester=` | 登录（数据裁剪） | 课程级评分，与教师级共用同一聚合函数 |
| GET `/sessions/:id/draft` | supervisor | 读取本人该场次草稿（无草稿返回 `null`） |
| PUT `/sessions/:id/draft` | supervisor | 保存/覆盖本人草稿（**允许部分维度为空**） |
| GET `/drafts?keyword=&page=&pageSize=` | supervisor | 本人草稿分页列表（按课程名查询） |
| DELETE `/drafts/:id` | supervisor | 删除本人草稿（越权/不存在 40401） |
| POST `/drafts/:id/submit` | supervisor | 草稿转正式评价：复用正式提交口径，**五维必填**；成功后删除草稿 |

```go
// 五维必填：objective/content/interaction/organization/frontier，均为 1-5 整数
type SupervisorEvaluationReq struct {
    Objective    *int   `json:"objective" binding:"omitempty,min=1,max=5"`
    Content      *int   `json:"content" binding:"omitempty,min=1,max=5"`
    Interaction  *int   `json:"interaction" binding:"omitempty,min=1,max=5"`
    Organization *int   `json:"organization" binding:"omitempty,min=1,max=5"`
    Frontier     *int   `json:"frontier" binding:"omitempty,min=1,max=5"`
    Comment      string `json:"comment" binding:"max=2000"`
    Highlights   string `json:"highlights" binding:"max=1000"`
    Improvements string `json:"improvements" binding:"max=1000"`
    Suggestions  string `json:"suggestions" binding:"max=1000"`
}
type SampleDTO struct { // 响应内嵌，聚合接口共用
    SessionCount     int  `json:"sessionCount"`
    EvaluatedCount   int  `json:"evaluatedCount"`   // 已评价场次；sampleSufficient 以此为准
    SupervisorCount  int  `json:"supervisorCount"`
    AgentCount       int  `json:"agentCount"`
    AlignedCount     int  `json:"alignedCount"`
    SampleSufficient bool `json:"sampleSufficient"`
}

// EvaluationDTO：单条评价（督导行与 agent 行同结构，靠 evaluatorType 区分）。
// 用于 GET /sessions/:id/evaluation 的 supervisorScores[] / agentScore，
// 以及 GET /teachers/:id/evaluations 时间线的 supervisorEvaluations[] / agentEvaluation。
type EvaluationDTO struct {
    EvaluatorID    uint64  `json:"evaluatorId"`
    EvaluatorName  string  `json:"evaluatorName"` // agent 行为空串
    EvaluatorType  string  `json:"evaluatorType"` // supervisor | agent
    AIModelVersion string  `json:"aiModelVersion"`
    AIConfidence   *float64 `json:"aiConfidence"` // 整条置信度 0-1，仅 agent
    FormulaVersion string  `json:"formulaVersion"`
    // 五维 1-5；仅 agent 侧允许 null（无法观测该维度）
    Objective    *int `json:"objective"`
    Content      *int `json:"content"`
    Interaction  *int `json:"interaction"`
    Organization *int `json:"organization"`
    Frontier     *int `json:"frontier"`
    TotalScore   *float64 `json:"totalScore"`
    Comment      string `json:"comment"`
    Highlights   string `json:"highlights"`
    Improvements string `json:"improvements"`
    Suggestions  string `json:"suggestions"`
    // Evidence：仅 agent 行非空，督导行为 null。DB 列是 JSON，此处做结构化投影，
    // 供评估页渲染「分数→转写原文」的可追溯引用（无证据不采信）。
    Evidence  *EvidenceDTO `json:"evidence"`
    CreatedAt string       `json:"createdAt"`
    UpdatedAt string       `json:"updatedAt"`
}

// EvidenceDTO 是 evaluations.evidence 的结构化投影（schemaVersion 1）。
type EvidenceDTO struct {
    SchemaVersion int                          `json:"schemaVersion"`
    CitedChunks   []string                     `json:"citedChunks"`   // 引用的知识库片段，如 kb-rubric#3
    Dimensions    map[string]EvidenceDimension `json:"dimensions"`    // key ∈ 五维
    NotObservable []string                     `json:"notObservable"` // 显式声明无法评价的维度
    PromptVersion string                       `json:"promptVersion"`
}
type EvidenceDimension struct {
    Confidence float64         `json:"confidence"`
    Quotes     []EvidenceQuote `json:"quotes"`
}
type EvidenceQuote struct {
    Start float64 `json:"start"` // 秒，与 transcripts.segments 对齐
    End   float64 `json:"end"`
    Quote string  `json:"quote"` // 必须能在脱敏转写中定位到
}

// 评估草稿（evaluation_drafts 表；与正式评价分表，不进入聚合口径）
type DraftUpsertReq struct {
    Objective    *int   `json:"objective" binding:"omitempty,min=1,max=5"` // 草稿允许为空
    Content      *int   `json:"content"`
    Interaction  *int   `json:"interaction"`
    Organization *int   `json:"organization"`
    Frontier     *int   `json:"frontier"`
    Comment      string `json:"comment"`
    Highlights   string `json:"highlights"`
    Improvements string `json:"improvements"`
    Suggestions  string `json:"suggestions"`
}
type DraftDTO struct {
    ID           uint64 `json:"id"`
    SessionID    uint64 `json:"sessionId"`
    CourseID     uint64 `json:"courseId"`
    CourseCode   string `json:"courseCode"`
    CourseName   string `json:"courseName"`
    TeacherName  string `json:"teacherName"`
    SessionDate  string `json:"sessionDate"`
    Period       string `json:"period"`
    Topic        string `json:"topic"`
    // 以下五维与评语可为 null（草稿可只写一半）
    Objective    *int   `json:"objective"`
    Content      *int   `json:"content"`
    Interaction  *int   `json:"interaction"`
    Organization *int   `json:"organization"`
    Frontier     *int   `json:"frontier"`
    Comment      string `json:"comment"`
    Highlights   string `json:"highlights"`
    Improvements string `json:"improvements"`
    Suggestions  string `json:"suggestions"`
    CreatedAt    string `json:"createdAt"`
    UpdatedAt    string `json:"updatedAt"`
}
```

- 🔴 **路由命名**：教师评分列表是 `GET /teacher-scores`；`GET /teachers` 永远是 Sprint 1 的**教师字典**（前端课程表单依赖）。二者不得混用。
- 🔴 **evidence 只属于 agent 行**：督导行必须为 `NULL`（JSON 列禁写空串，见 §13 实现陷阱）。读取时从 JSON 列结构化为 `EvidenceDTO`，不得原样透传字符串。每个非 `null` 维度至少 1 条能在脱敏转写中定位的引用；智能体无法观测的维度写 `null` 并在 `notObservable` 中声明——**禁止对无法观测的维度强行给分**。
- 唯一聚合出口：教师级与课程级都调用 `pkg/scoring.Aggregate`，禁止"先算课程级再对课程取平均"。
- 综合分取**综合均值**口径（各场次综合分的算术平均）；默认数据双侧对齐时与维度加权求和自洽。详见开发计划 §2.5.2。

### 8.9 课堂录音与转写（Sprint 2.2 阶段②，已实现，4 条）

> 权限依据见开发计划 §5.1：**音频对教师不可见**（音频中人声不可分割），教师端只开放已脱敏的转写文本。
> DTO 定义在 `backend/internal/dto/evaluation.go`；配置、实现记录与遗留项见 §16。

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| POST `/sessions/:id/recording` | supervisor | multipart 上传音频（`mp3/wav/m4a`，≤ `upload.maxSize`）；一节课一条主录音，重复上传 40901 |
| GET `/sessions/:id/transcript` | 登录（数据裁剪） | 返回 `{recording, transcript}`；转写未完成时由 `transcript.status` 供前端轮询 |
| POST `/sessions/:id/transcript/retry` | supervisor | 转写失败后重新入队（`status` 回到 `pending`） |
| GET `/recordings/:id/stream` | supervisor | 音频流式播放，支持 Range；**额外允许 `?ticket=` 短时票据**（见下） |

```go
// RecordingDTO —— 阶段②新增 playbackUrl
type RecordingDTO struct {
    ID           uint64 `json:"id"`
    OriginalName string `json:"originalName"`
    Format       string `json:"format"`      // mp3 / wav / m4a
    Size         int64  `json:"size"`
    DurationSec  int    `json:"durationSec"` // ASR 返回后回写；未完成时为 0
    StreamURL    string `json:"streamUrl"`   // 需 Authorization 头
    // PlaybackURL 是带短时票据的流式播放地址，可直接交给 <audio src>：
    // 浏览器播放期间会持续发出 Range 请求，无法携带 Authorization 头，只能靠查询串鉴权。
    // 仅对 supervisor 返回（教师侧恒为空串）；票据绑定录音 id，TTL 4 小时。
    PlaybackURL  string `json:"playbackUrl"`
}

type TranscriptDTO struct {
    ID            uint64              `json:"id"`
    Status        string              `json:"status"` // pending|running|done|failed
    Content       string              `json:"content"` // 纯文本全文（已脱敏）
    Segments      []TranscriptSegment `json:"segments"`
    Engine        string              `json:"engine"`
    EngineVersion string              `json:"engineVersion"`
    ErrorMessage  string              `json:"errorMessage"`
}
type TranscriptSegment struct {
    Start   float64 `json:"start"` // 秒
    End     float64 `json:"end"`   // 秒
    Speaker string  `json:"speaker"`
    Text    string  `json:"text"`
}
```

- 🔴 **播放票据**：`?ticket=` 只对 `AuthOrPlaybackTicket` 中间件生效（仅挂在流式路由），
  `scope=playback` 且 `rid` 必须等于路径 `:id`，防止拿 A 的票据听 B 的录音；
  播放票据**不能**用作会话令牌——`Auth` 中间件显式拒绝 `scope=playback` 的令牌。
- 🔴 **日志脱敏**：`Logger` 中间件必须把 `ticket`/`token` 等查询参数掩码后再落盘，
  否则票据进日志等于鉴权形同虚设（见 §9）。
- **转写异步且解耦**：`transcripts.status` 状态机 `pending→running→done/failed`；
  ASR 失败只写 `failed` + `error_message`，映射 50003，**绝不影响督导评分主流程**。
- **进程重启恢复**：启动时 `RequeueStuck` 把遗留的 `pending/running` 转写重新入队（幂等）。
- **脱敏强制**：写入 `transcripts.content` / `segments` 前必须过 `Scrubber`；
  当前实现为「称谓正则 + 可选词典」，学生名册建立后可替换为 NER 实现，调用点不变。

## 9. 中间件规范（`internal/middleware/`）

注册顺序（`router.go` 中唯一生效顺序）：

```
Recovery → Logger → CORS → [Auth → RBAC]（按分组挂载）
```

| 中间件 | 职责 | 要点 |
|--------|------|------|
| Recovery | panic 兜底 | 返回 500 + code 50001，日志含堆栈 |
| Logger | 请求日志（slog JSON） | method/path/status/耗时/user_id（Auth 之后才有 user_id，可将用户注入 context 由 Logger 在响应后输出）；**查询串脱敏**：`ticket`/`token`/`api_key`/`apikey`/`access_token` 掩码后再落盘（`middleware/logger.go:47`） |
| CORS | 白名单放行 | origins 来自配置；允许 `Authorization` 头。`cors.allowAnyOrigin=true` 时回显任意来源（**仅本地联调，生产必须 false**）；因 `AllowCredentials=true`，禁用 `AllowOrigins:["*"]`（会 panic），必须用 `AllowOriginFunc` |
| Auth | 解析 Bearer token | claims 注入 `c.Set("userID"/"role"/"deptID")`；失败返回 40101；**显式拒绝 `scope=playback` 票据**（票据不得当会话令牌） |
| AuthOrPlaybackTicket | 流式路由专用鉴权 | 先取 `Authorization`，否则取 `?ticket=`；票据须 `scope=playback` 且 `rid` == 路径 `:id`，否则 40101。仅挂 `GET /recordings/:id/stream`（`middleware/auth.go:50`） |
| RequireRoles(roles...) | 角色守卫 | 不匹配返回 40301；`router` 分组挂载，禁止散落在 handler 里判断角色 |

路由分组示意（唯一注册地 `router/router.go`）：

```go
r.GET("/healthz", h.Health)                  // 根路径探活（无需鉴权）

v1 := r.Group("/api/v1")
v1.GET("/healthz", h.Health)
v1.POST("/auth/login", h.Auth.Login)

// 🔴 音频流只能挂在 v1 上：<audio> 带不了 Authorization，只能靠 ?ticket=；
// 若放进下面的 authed 组，组上的 Auth 会先执行并 401，票据中间件根本没机会运行。
v1.GET("/recordings/:id/stream", mw.AuthOrPlaybackTicket(jwt), h.Recording.Stream)

authed := v1.Group("", mw.Auth())
authed.GET("/auth/me", h.Auth.Me)
authed.POST("/auth/logout", h.Auth.Logout)
authed.GET("/dashboard", h.Dash.Board)
authed.GET("/courses", h.Course.List)
authed.GET("/courses/:id", h.Course.Detail)
authed.GET("/courses/:id/resources", h.Resource.List)
authed.GET("/resources/:id/download", h.Resource.Download)
authed.GET("/departments", h.Dict.Departments)
authed.GET("/teachers", h.Dict.Teachers)          // 教师字典（Sprint 1，勿改语义）

// —— Sprint 2.1：评价闭环（数据裁剪仍在 service 层）——
authed.GET("/courses/:id/sessions", h.Session.ListByCourse)
authed.GET("/courses/:id/evaluation-summary", h.TeacherScore.CourseSummary)
authed.GET("/sessions/:id", h.Session.Detail)
authed.GET("/sessions/:id/evaluation", h.Session.Evaluation)
authed.GET("/teachers/:id/evaluation-summary", h.TeacherScore.TeacherSummary)
authed.GET("/teachers/:id/evaluations", h.TeacherScore.TeacherEvaluations)

// —— Sprint 2.2 阶段②：转写读取与重试（播放流在 v1 上，见上）——
authed.GET("/sessions/:id/transcript", h.Recording.Transcript)
authed.POST("/sessions/:id/transcript/retry", h.Recording.Retry)

authed.Group("", mw.RequireRoles("director")).
    POST("/courses", h.Course.Create).
    PUT("/courses/:id", h.Course.Update)

authed.Group("", mw.RequireRoles("teacher")).
    POST("/courses/:id/resources", h.Resource.Upload).
    DELETE("/resources/:id", h.Resource.Delete)

authed.Group("", mw.RequireRoles("supervisor")).
    GET("/supervision/coverage", h.Super.Coverage).
    GET("/supervision/plans", h.Super.Plans).
    POST("/sessions", h.Session.Create).
    PUT("/sessions/:id/supervisor-evaluation", h.Session.Submit).
    POST("/sessions/:id/recording", h.Recording.Upload).
    GET("/sessions/:id/draft", h.Draft.GetBySession).
    PUT("/sessions/:id/draft", h.Draft.Save).
    GET("/drafts", h.Draft.List).
    DELETE("/drafts/:id", h.Draft.Delete).
    POST("/drafts/:id/submit", h.Draft.Submit)

authed.Group("", mw.RequireRoles("director", "supervisor")).
    GET("/teacher-scores", h.TeacherScore.List)
```

## 10. 分层架构与数据流

```
HTTP 请求
  → middleware（鉴权 / 角色守卫）
    → handler：绑定参数 + 校验（binding tag）→ 调 service → 组装响应
      → service：业务校验（角色归属/唯一性）→ 事务边界 → 调 repository
        → repository：GORM 查询/持久化（纯数据操作）
          → MySQL
```

各层纪律：

| 层 | 允许 | 禁止 |
|----|------|------|
| handler | 绑定/校验参数、调 service、`response.OK/Fail` | 直接调 repository / 写 SQL / 写业务分支 |
| service | 权限判断、业务规则、事务（`db.Transaction`）、聚合计算 | 读写 `gin.Context`、拼 SQL 字符串 |
| repository | GORM 链式查询、`context.Context` 传递、返回 model/DTO | 业务 if/else、调用 service |

- 所有 repository 方法第一个参数为 `ctx context.Context`，GORM 用 `WithContext(ctx)`；
- 事务只在 service 层开启（目前仅"删除资源 + 删文件"外的多表写场景，Sprint 1 极少）；
- 错误处理：repository 返回 `error` 原样上抛；service 判断后转 `errcode`；日志只在 handler/middleware 层记。

## 11. 业务规则要点（写操作的权限与校验清单）

| 场景 | 规则 | 违反返回 |
|------|------|---------|
| 主任新增课程 | `departmentId` 必须等于主任本室；`teacherId` 必须属于该教研室且 role=teacher | 40302 |
| 新增课程 | `code + semester` 唯一；code 匹配 `^[A-Z]{2,4}\d{4}$` | 40901 / 40001 |
| 主任修改课程 | 课程必须属于本室；`code` 字段忽略不可改 | 40302 |
| 教师上传资源 | `course.teacher_id == user.id` 且课程 status=open | 40302 |
| 教师删除资源 | 资源所属课程 `teacher_id == user.id` | 40302 |
| 上传文件 | 扩展名 ∈ 白名单；大小 ≤ 100MB | 40001 |
| 督导接口 | 仅 supervisor（路由组已守卫） | 40301 |
| 督导建课 | 日期不得晚于今天；同课程+班级+日期+节次唯一（`uk_session`；日期按 `YYYY-MM-DD` 比较） | 40002 / 40901 |
| 督导评分 | **五维全部必填**（1-5 整数）；`PUT` 幂等覆盖；写 `formula_version`；`evidence` 为 NULL | 40002 / 40001 |
| 评分读取 | teacher 仅本人 / director 本室 / supervisor 全校 | 40302 |
| 评分维度 | 仅智能体侧允许 `NULL`；督导侧不得落 NULL 维度 | — |

课程编码正则在 validator 中注册自定义规则 `coursecode`（`RegisterValidation`），禁止在 handler 里手写正则 if。

## 12. 安全规范

1. **密码**：只存 bcrypt 哈希（cost 10）；明文密码禁止落日志、落 DTO；
2. **JWT**：HS256；secret 从配置/环境变量读取；claims 最小化（不放部门名等可变信息，部门以 `deptID` 查库）；TTL 24h（Sprint 1 不做 refresh）；
3. **SQL 注入**：一律 GORM 参数化（`?` / 命名参数），禁止 `fmt.Sprintf` 拼 SQL；
4. **路径穿越**：上传文件名用 `filepath.Ext` 取扩展名 + UUID 重命名，**绝不使用用户原始文件名拼路径**；下载时 `filepath.Base` 清洗；
5. **上传白名单**：双重校验（扩展名 + `http.DetectContentType` 嗅探 MIME 前缀），拒绝可执行文件；
6. **错误信息**：对外只返回 `errcode.message`，堆栈与 SQL 错误只进日志；
7. **CORS**：生产环境 origins 收紧为部署域名；
8. **依赖**：`go mod tidy` 后提交 lock；引入新依赖需在 PR 说明中给出理由（经人工审查）。

## 13. 数据库对接

- 建库、种子数据、备份与排查：**按 [`MySQL数据库创建指导.md`](./MySQL数据库创建指导.md) 执行**；
  **表结构（DDL）的事实源是代码**——`database/schema.sql`（Sprint 1 基线）+ `migrations/`（增量），本文档不复述 DDL；
- GORM 模型与表的映射（`internal/model/`）：

| model | 表 | 说明 |
|-------|-----|------|
| Department | departments | 教研室 |
| User | users | 三角色账号 |
| Course | courses | 课程主表 |
| ClassInfo | course_classes | 开课班级 |
| Resource | resources | 课程资源 |
| SupervisionPlan | supervision_plans | 听评课安排 |
| TeachingSession | teaching_sessions | **Sprint 2 新增**：授课记录（某次具体的课，一切评价的落点） |
| Evaluation | evaluations | **Sprint 2 新增**：评价（`evaluator_type` 区分督导/智能体，可插拔 scorer） |
| EvaluationDraft | evaluation_drafts | **评估草稿**：督导未提交的私人工作副本（维度可空，不进入聚合） |
| Recording | recordings | **Sprint 2.2 新增**：课堂录音 |
| Transcript | transcripts | **Sprint 2.2 新增**：课堂转写（异步任务产物） |

> Sprint 2/3 的建表 DDL、迁移脚本与评分算法见
> [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) §3、§2。
> 新增表**只新增、不改存量表**（`supervision_plans` 不动，由 `teaching_sessions.plan_id` 反向关联）。

- **事实源分工**：`database/schema.sql` 只管 Sprint 1 存量表；**Sprint 2 起的新表以 `migrations/` 下的迁移脚本为唯一事实源**，二者不重复维护；
- **迁移策略**：从 Sprint 2 起引入 `golang-migrate`（`cmd/migrate` + `migrations/` embed）。GORM **不使用 AutoMigrate**（避免双源漂移）；
  新增表 = 新建迁移脚本 + 更新 `internal/model` + 在 PR 说明列明变更。初始化流程见 `scripts/init_db.sh`：`schema.sql → migrate up → seed.sql → cmd/seed`；
- **迁移清单（V1–V6，逐个对应 `backend/migrations/` 下同名文件）**：

| 版本 | 文件（`.up/down.sql`） | 内容 |
|------|------------------------|------|
| V1 | `1_baseline_sprint1` | Sprint 1 基线表：`departments` / `users` / `courses` / `course_classes` / `resources` / `supervision_plans` |
| V2 | `2_teaching_sessions_and_evaluations` | `teaching_sessions` / `evaluations`（阶段① 评价闭环） |
| V3 | `3_recordings_and_transcripts` | `recordings` / `transcripts` / `recording_playback_logs`（阶段② 录音与转写） |
| V4 | `4_transcripts_content_default` | `transcripts.content` 补 `DEFAULT ('')`，避开 `STRICT_TRANS_TABLES` 下的 1364 |
| V5 | `5_evaluation_drafts` | `evaluation_drafts`（督导草稿箱，`uk_draft` 一人一场次一份） |
| V6 | `6_transcripts_engine_width` | `transcripts.engine` / `engine_version` 放宽到 `VARCHAR(64)`（真实模型名 34 字符触发 1406，见 §16） |

- 🔴 **迁移文件命名**：`<版本>_<名称>.up.sql` / `.down.sql`（如 `2_teaching_sessions_and_evaluations.up.sql`）。
  Flyway 风格 `V2__xxx.up.sql` **不被 golang-migrate 识别**，会让 `migrate up` 报 `first .: file does not exist`——禁止使用；
- **实现陷阱**：JSON 列（`evaluations.evidence`）的 Go 字段必须用指针，禁止用零值 `''` 写入（MySQL 8 报 3140）；DATE 列比较必须按 `YYYY-MM-DD` 绑定，否则唯一性预检漏判；
- 连接池：`SetMaxOpenConns/SetMaxIdleConns` 来自配置；启动时 `db.Ping()` 失败直接 fatal 退出。

## 14. 编码规范（Go）

1. 遵循 [Go 官方 Code Review Comments](https://go.dev/wiki/CodeReviewComments) 与 `gofmt`/`go vet` 零告警；
2. **命名**：导出用注释开头（`// Foo does ...`）；文件名 snake_case；错误变量 `errXxx`；DTO 后缀 `Req/Resp/DTO`；
3. **错误**：`if err != nil` 立即处理，禁止 `_` 吞错；wrap 用 `fmt.Errorf("...: %w", err)`；
4. **context**：全链路传递，禁止 `context.Background()` 出现在请求路径中；
5. **常量**：角色、状态、类型枚举一律 `internal/service/consts.go` 定义（`RoleDirector = "director"`...），禁止散落字符串字面量；
6. **注释**：默认不写；仅在非显而易见的业务口径（如覆盖率计算）处写一行说明；
7. **测试**：关键逻辑须有表驱动单测——`pkg/scoring`（权重/单次总分/缺失矩阵/恒等式）、`pkg/errcode`（错误码→HTTP 映射）、service 层（数据范围裁剪、资源归属、评分五维必填、幂等覆盖）；表驱动用例覆盖三角色；
8. **Makefile**：`make run`（air 热重载）、`make migrate`/`migrate-down`/`migrate-version`、`make seed`、`make test`、`make lint`（go vet + gofmt -l）、`make build`、`make verify`；
9. **Git**：Conventional Commits（`feat(course): 新增课程列表接口`）；分支 `feature/S2.1-course-list`（与前端同名故事对齐）。

## 15. 验收标准（DoD）与数据链路打通脚本

每个故事 DoD = 接口可用 + 权限正确 + 数据范围正确 + `go build` / `go vet` / `gofmt` / `go test` 零错误。

**数据链路一键验收**（`scripts/verify.sh`，覆盖 **Sprint 1 + Sprint 2.1**）：

> ⚠️ **覆盖边界（事实陈述，勿当已覆盖）**：当前脚本共 **144 条 `want` 断言**（`grep -c 'want "' backend/scripts/verify.sh`）。
> 其中**没有任何录音 / 转写断言**——脚本里与阶段②相关的只有两处 `aiModelVersion="qwen-audio-v1"` 的种子数据断言
> （`verify.sh:185,246`）。因此**阶段②（录音上传 / 异步转写 / 播放票据）目前没有自动化端到端断言**，
> 临时验收口径见 §16 与开发计划 §8.2。

```bash
# 一键执行（需已启动 MySQL 与后端服务）
cd backend && MYSQL_PORT=3307 bash scripts/verify.sh

# 也可手工逐条核对：
BASE=http://localhost:8080/api/v1

# 0. 技术基线（S1.1）
curl -s $BASE/healthz

# 1. 三角色登录（S1.2）
TOK_D=$(curl -s -X POST $BASE/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"director","password":"123456"}' | jq -r .data.token)
TOK_T=$(curl -s -X POST $BASE/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"teacher","password":"123456"}' | jq -r .data.token)
TOK_S=$(curl -s -X POST $BASE/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"supervisor","password":"123456"}' | jq -r .data.token)

# 2. 课程列表数据范围（S2.1/S2.2/S2.3）——人工核对 list 内容与 total
curl -s "$BASE/courses?page=1&pageSize=10" -H "Authorization: Bearer $TOK_D"   # 期望：仅软件工程教研室
curl -s "$BASE/courses?page=1&pageSize=10" -H "Authorization: Bearer $TOK_T"   # 期望：仅李明本人课程
curl -s "$BASE/courses?page=1&pageSize=10" -H "Authorization: Bearer $TOK_S"   # 期望：全量

# 3. 主任新增课程（S3.1）
curl -s -X POST $BASE/courses -H "Authorization: Bearer $TOK_D" -H 'Content-Type: application/json' \
  -d '{"code":"SE3206","name":"软件质量保证","departmentId":1,"teacherId":2,"semester":"2026-2027-1","credit":2,"hours":32,"description":"Sprint 1 验收用","status":"open"}'

# 4. 教师上传资源（S4.2）→ 列表可见（S4.1）
curl -s -X POST $BASE/courses/1/resources -H "Authorization: Bearer $TOK_T" -F "file=@./test.pdf"
curl -s "$BASE/courses/1/resources" -H "Authorization: Bearer $TOK_T"

# 5. 越权负向用例（必须返回 403xx）
curl -s -X POST $BASE/courses -H "Authorization: Bearer $TOK_T" -H 'Content-Type: application/json' -d '{...}'   # 教师建课 → 40301
curl -s "$BASE/supervision/coverage" -H "Authorization: Bearer $TOK_D"                                          # 主任访问督导接口 → 40301

# 6. 督导覆盖率（S5.1）——期望与种子数据推算一致（见 MySQL 文档 §7）
curl -s $BASE/supervision/coverage -H "Authorization: Bearer $TOK_S"
```

| 故事 | 后端验收要点 |
|------|-------------|
| S1.1 | `go build` 零错误；`/healthz` 200；CORS 放行前端 dev 源；结构化日志输出 |
| S1.2 | 三演示账号登录成功返回 token+user；错误密码 40101；token 过期/伪造 40101 |
| S2.1–S2.3 | 同一接口三角色返回的数据范围与 §3 表一致；筛选参数叠加生效 |
| S3.1 | 建课校验全过（编码格式/唯一/教研室归属）；修改后详情即时可见 |
| S4.1 | 详情聚合 classes/studentCount/resourceCount 数值正确（对种子数据可核算） |
| S4.2 | 上传白名单与大小校验生效；教师仅能操作本人课程；删除同步清文件 |
| S5.1 | 覆盖率数值与 MySQL 文档 §7 推算一致；plans 分页与状态筛选可用 |

**Sprint 2.1 后端 DoD**（详见开发计划 §8.1）：

| 项 | 后端验收要点 |
|----|-------------|
| 迁移 | 空库 `make migrate` 建成 2 张新表；`migrate down 1` 可回滚；文件命名符合工具约定 |
| 评分算法 | §2.4 验算例 82.50；§2.5.5 缺失矩阵 5 行；教师级综合分 ≡ 各次课综合分均值（非对齐数据亦成立） |
| 一致性 | 主任端与教师端综合分 / 各维度分逐位相同（单一 `scoring.Aggregate`） |
| 数据范围 | 教师访问他人面板 40302；主任访问他室 40302 |
| 写路径 | 督导评分五维必填（缺失 40002）；重复提交幂等覆盖；`formula_version` 落库；同课同日同节次 40901 |
| 样本量 | `sampleSufficient` 以 `evaluatedCount` 为准；无评价综合分 `null` 且不显示 0 |
| 端到端 | `verify.sh` 全绿；请求日志无 HTTP 5xx |

---

## 16. 课堂录音与转写：配置、实现记录与遗留

> 本节内容合并自本轮文档整合中删除的原「录音转写接入方案」，只保留其中**仍然有效**的部分：
> 实现事实、真实缺陷与遗留项；不再保留「待做 / 缺口」式的方案推演。接口契约见 §8.9，权限依据见开发计划 §5.1。

### 16.1 配置与密钥

| 项 | 规则 | 证据 |
|----|------|------|
| API Key 来源 | **只允许环境变量 `AIJIAOXUE_TRANSCRIPTION_API_KEY`**（`config.Load` 显式 `BindEnv`）；`config.example.yaml` 中 `apiKey: ""` 永远留空 | `internal/config/config.go:210` · `config.example.yaml:37` |
| 启动校验 | `transcription.enabled=true` 但 Key 为空 → **启动即报错退出**（Fail-Fast）；`model`/`engine` 为空或长度 >64 同样拒绝启动（在向厂商付费之前拦住） | `internal/config/config.go:234-249` |
| 日志 | 只打印掩码 Key（如 `sk-459***ad68`），禁止记录原始值 | `internal/config/config.go:103`（`MaskedAPIKey`） |
| 命名坑 | `AutomaticEnv` 自动推导会得到 `..._APIKEY`（驼峰连写）而失效，必须显式 `BindEnv`；新配置键同理，否则 `Unmarshal` 静默跳过 | `internal/config/config.go:207-213` |

### 16.2 存储与播放

| 项 | 实现 | 证据 |
|----|------|------|
| 存储路径 | `uploads/recordings/{sessionId}/{uuid}.{ext}`；扩展名白名单 `mp3/wav/m4a`；一节课一条主录音（`uk_rec_session`），重复上传 **40901** | `internal/service/recording.go:93-118` |
| 文件权限 | `0o640` + 显式 `Sync()`（音频含学生人声，不做全局可读；避免崩溃留下半截文件被当完整音频） | `internal/service/recording.go:362-381` |
| 流式播放 | `GET /recordings/:id/stream` 走 `http.ServeFile`，原生支持 Range（实测 `206 Partial Content` + `Content-Range`） | `internal/handler/recording.go:80` |
| 播放票据 | `<audio>` 无法发送 `Authorization`，故接受 `?ticket=`：`scope=playback`、绑定录音 `rid`、TTL **4h**、**仅 supervisor 签发**（教师侧 `playbackUrl` 恒为空串） | `internal/service/recording.go:34,152-157` |
| 票据校验 | `AuthOrPlaybackTicket` 要求 `scope=playback` 时 `rid == 路径 :id`；`Auth` 中间件**显式拒绝** playback 票据（不得当会话令牌） | `internal/middleware/auth.go:34,50-77` |
| 🔴 路由位置 | 该路由**必须注册在 `v1` 上，不能放进 `Auth` 组**：否则组上的 `Auth` 先 401，票据中间件根本没机会运行 | `internal/router/router.go:80-84` |
| 日志脱敏 | `Logger` 把 `ticket`/`token`/`api_key` 等查询参数掩码后落盘（实测 `?ticket=%2A%2A%2A`） | `internal/middleware/logger.go:47` |

### 16.3 ASR 引擎与状态机

| 项 | 实现 | 证据 |
|----|------|------|
| 引擎 / 模型 | `dashscope-qwen-audio-asr` / `qwen-audio-3.1-asr-flash-filetrans`（百炼 filetrans：上传凭证 → OSS → 提交 → 轮询 → 下载解析） | `internal/asr/client.go` · `config.example.yaml:34-36` |
| 状态机 | `pending→running→done/failed`；失败写 `failed` + `error_message`（截断到 255 字符），映射 **50003**，**绝不阻断督导评分主流程** | `internal/service/recording.go:239-332` · `pkg/errcode/errcode.go:26,71` |
| 失败必落地 | 写失败状态用**独立短上下文**（主上下文可能已超时/取消），否则前端永远停在 `running` | `internal/service/recording.go:243-253` |
| 重启恢复 | 启动时 `RequeueStuck` 把遗留的 `pending/running` 重新入队（幂等，重复调用安全） | `internal/service/recording.go:214-233` · `cmd/server/main.go:63` |
| 并发闸门 | 容量 = `transcription.maxConcurrency`，避免同时把多个大文件推进内存 | `internal/service/recording.go:57,274-281` |
| 脱敏 | `Scrubber` = 称谓正则（`X同学/小朋友们` + 否定列表）+ 可选词典（`transcription.studentNames`）；**学生名册/NER 属未来工作，调用点不变** | `internal/service/desensitize.go` · `internal/service/recording.go:296` |

### 16.4 联调发现的三个真实缺陷（长期教训）

| # | 缺陷 | 根因 | 修复 |
|---|------|------|------|
| 1 | `transcripts.engine_version` 列宽不足 →「ASR 已计费但结果存不下来」 | 原 `VARCHAR(32)`，真实模型名 `qwen-audio-3.1-asr-flash-filetrans` 有 **34** 字符；MySQL 8 严格模式报 `1406 Data too long`，**整条 UPDATE 回滚** → `content`/`segments` 全丢、`status` 停在 `running`、需重新识别（再付一次费） | 迁移 6 把 `engine`/`engine_version` 放宽到 `VARCHAR(64)`；启动期校验长度 ≤64；最终写入失败时置 `failed` 而非留 `running` |
| 2 | 流式路由挂在 `Auth` 组内，票据机制完全失效 | gin 先执行**组上**的 `Auth`，而 `<audio>` 带不了 `Authorization`，请求在 `AuthOrPlaybackTicket` 运行前就被 401（表现为「票据签发正常、URL 正确，但音频播不出来」） | 路由移到 `v1` 单独注册；回归用例 `internal/router/router_test.go`：无凭据 401 / 有效票据不得 401 / 会话令牌仍可用（单挂中间件的单测发现不了，必须测真实路由表） |
| 3 | CORS 白名单漏内网 IP →「页面能开、一点登录就 403」 | Vite dev server 除 localhost 外还监听内网 IP（Network 地址），`Origin: http://<内网IP>:5173` 不在 `cors.origins` 即被 CORS 拦；`curl` 不带 `Origin`，命令行自测发现不了 | 新增 `cors.allowAnyOrigin`（默认 `false`，用 `AllowOriginFunc` 回显来源）；开启时启动打 WARN；**因 `AllowCredentials=true`，绝不能写 `AllowOrigins:["*"]`（gin-contrib/cors 直接 panic）**；生产必须关闭（回归：`internal/middleware/cors_test.go`） |

### 16.5 遗留项（未做，需要时再排期）

| 项 | 说明 | 现状核实 |
|----|------|---------|
| `recordings.file_path` 存相对路径 | 当前存 `uploads/recordings/1/{uuid}.mp3`，依赖进程 cwd 为 `backend/`；**生产建议**只存相对录音根的 key，读取时与 `upload.dir` 拼接 | `internal/service/recording.go:103-111`（`filepath.Join(upload.dir, …)` 直接落库） |
| 音频保留期限 | 本轮定为「不限期」，故无清理任务；合规审查若提出期限需补定时任务 | 无相关代码 |
| 学生名册 | 未建立，脱敏仅靠称谓正则；音近字误识别、直呼全名（无「同学」后缀）可能漏网 | `internal/service/desensitize.go`（注释自述该局限） |
| 超长音频 | 仅校验文件大小（`upload.maxSize`，默认 100MB），未按时长拒绝；开启说话人分离时官方建议 ≤2 小时 | `internal/service/recording.go:97-99` |
| 429 退避重试 | 未对上游限流做专门退避 | `internal/asr/client.go` 无退避/重试逻辑 |
| 生产音频卸载 | 仍由 Go 进程 `http.ServeFile` 转发，生产建议改 Nginx `X-Accel-Redirect` | `internal/handler/recording.go:80` |

---

## 更新记录

| 版本 | 日期 | 变更 |
|------|------|------|
| v2.0 | 2026-09 | 均衡覆盖三次 Sprint；注明当前处于 Sprint 2；新增 Sprint 2.1 接口契约（§8.8）、错误码 HTTP 映射与回归说明、`golang-migrate` 迁移规范、文档同步要求（页首）|
| v3.0 | 2026-10-07 | 文档整合：修正阶段状态（阶段② 部分实现、智能体评分未接入）；补齐 §5 目录树、§6 配置（transcription/evaluation/cors.allowAnyOrigin）、§13 迁移 V1–V6；§8 接口计数按 `router.go` 校正为 36 条并同步路由示意与中间件表；并入录音转写实现记录与遗留（§16）；删除对已删文档的引用并去重接口契约 |

*文档分工见页首「单一事实源」表：接口契约以本文 §8 为准；页面/路由/组件见 [`frontend_AGENTS.md`](./frontend_AGENTS.md)；DDL 见 `backend/database/schema.sql` + `backend/migrations/`；评分体系/权限/DoD 见 [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md)；建库/种子/备份见 [`MySQL数据库创建指导.md`](./MySQL数据库创建指导.md)。接口契约变更须先改本文 §8，再同步 `internal/dto/` 与前端 `src/types/`。*
