#  「爱教学」后端编码智能体指导手册（Go + Gin）

> **文档用途**：本文件是写给 AI 智能体（编码助手 GLM 5.3 + OpenCode、架构助手 DeepSeek V4 Pro）的后端编码工作宪法。
> 智能体在为本项目编写任何后端代码之前，**必须先完整阅读本文件**，并严格遵守其中的技术栈、目录结构、分层架构、接口契约与编码规则。
> **使用方式**：后端仓库 `aijiaoxue-api` 初始化后，本文件应置于仓库根目录。
> **配套文档**：《MySQL数据库创建指导.md》（同目录）——建库、建表与种子数据；[`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md)——Sprint 2/3 的算法、DDL 与接口契约；前端《AGENTS.md》——页面与组件设计。接口契约以本文第 8 节为准。

> 🔴 **文档同步要求（强制）**：本手册是后端开发的事实源之一。任何改动若涉及**路由 / 接口字段、数据模型或表结构、错误码、权限与数据范围、计分口径、Sprint 范围**，
> 必须在**同一个 PR** 内同步更新本文件，以及 `docs/Sprint2-3-教学评价与提优-开发计划.md` 中对应的契约章节。
> **契约先行：先改文档，再改代码。只改代码不改文档，视为未完成。**

---

## 1. 项目背景

「爱教学」是面向高校的教学质量全链路数字化管理平台，走三阶进化路线：**查课程（Sprint 1）→ 看课堂（Sprint 2）→ 帮教师（Sprint 3）**。

| 阶段 | 主题 | 一句话目标 | 状态 |
|------|------|-----------|------|
| Sprint 1 | 查课程 | 课程信息透明可查、三角色数据打通、Web 框架稳定运行 | ✅ 已交付 |
| Sprint 2 | 看课堂 | 授课记录 + 督导/智能体课堂评价闭环；课堂录音、异步转写与智能体接入 | 🚧 **当前阶段**（阶段① 评价闭环已落地，阶段② 智能体接入待开发） |
| Sprint 3 | 帮教师 | 教学提优：评语与提优建议、趋势折线、智能体对话、申诉复核、质量报告 | ⏳ 待排期 |

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

Sprint 2 分两阶段推进，任务编号见开发计划 §7.1 / §7.2，简短清单见 [`Sprint2-3-任务清单.md`](./Sprint2-3-任务清单.md)。

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

**阶段② 智能体接入（Sprint 2.2，待开发）**

- 课堂录音上传与流式播放（Range）、播放审计日志；
- 异步语音转写任务：`transcripts.status` 状态机（`pending→running→done/failed`）+ 失败重试 + 学生姓名脱敏；
- 智能体评分写入 `evaluations` 的 agent 行（无法观测的维度写 `NULL`，附 `evidence` 与 `ai_confidence`）；
- 聚合纳入 agent 侧并**全量返回 `flags`**（`no_data/sup_only/ai_only/disjoint/sample_insufficient/agent_not_calibrated/formula_mixed`）；
- 智能体故障不得影响督导评分主流程（降级返回 50003）。

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
aijiaoxue-api/
├── AGENTS.md                # 本文件
├── go.mod / go.sum
├── Makefile                 # run / seed / test / lint / build
├── config.yaml              # 本地配置（config.example.yaml 提交模板，config.yaml 不入库）
├── cmd/
│   ├── server/main.go       # 唯一服务入口
│   ├── migrate/main.go      # golang-migrate 执行器（up / down [N] / version）
│   └── seed/main.go         # 种子数据工具（bcrypt 写入演示账号）
├── migrations/              # 版本化迁移（embed 打包）；命名 <版本>_<名称>.up/down.sql
│   ├── 1_baseline_sprint1.{up,down}.sql
│   └── 2_teaching_sessions_and_evaluations.{up,down}.sql
├── internal/
│   ├── config/config.go     # viper 加载与结构体（含 evaluation 计分口径启动校验）
│   ├── router/router.go     # 路由表 + 分组（唯一路由注册地）
│   ├── middleware/          # recovery / logger / cors / auth / rbac
│   ├── handler/             # HTTP 层：参数绑定、调用 service、组装响应
│   │   ├── auth.go course.go resource.go supervision.go dict.go
│   │   └── session.go teacherscore.go     # Sprint 2.1
│   ├── service/             # 业务层：权限判断、业务校验、事务边界、聚合
│   │   ├── auth.go course.go resource.go supervision.go dashboard.go
│   │   └── session.go teacherscore.go     # Sprint 2.1
│   ├── repository/          # 数据层：GORM 查询，无业务逻辑
│   │   ├── user.go course.go resource.go supervision.go
│   │   └── session.go evaluation.go       # Sprint 2.1
│   ├── model/               # GORM 模型，与表一一对应
│   └── dto/                 # 请求/响应结构体（json tag 与前端 types 对齐）
├── pkg/
│   ├── response/            # OK / Fail 统一响应
│   ├── errcode/             # 错误码常量 + HTTP 映射（含单测）
│   ├── scoring/             # 评分纯函数：权重 / 单次总分 / 综合均值聚合
│   └── jwtutil/             # 签发与解析
├── database/
│   ├── schema.sql           # Sprint 1 存量表建表脚本（新表以 migrations/ 为准）
│   └── seed.sql             # 种子数据（含 §3.4 授课/评价）；密码哈希由 cmd/seed 覆写
├── uploads/                 # 课程资源文件存储（.gitignore）
└── scripts/
    ├── init_db.sh           # 建库 → migrate up → seed.sql → cmd/seed
    └── verify.sh            # 端到端验收（Sprint 1 + 2.1 断言）
```

**目录铁律**：
- 依赖方向单向：`handler → service → repository → model`，禁止反向 import、禁止跨层跳调（handler 直查库 = 违规）；
- `router/` 之外**禁止注册路由**；`handler/` 之外**禁止读写 `*gin.Context`**；
- `dto/` 的 json tag 必须与前端 `types/` 字段名逐字对齐（camelCase）。

## 6. 配置管理（config.yaml）

```yaml
server:
  port: 8080
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
cors:
  origins: ["http://localhost:5173"]  # 前端 dev 地址
```

- 环境变量优先级高于 yaml，键名前缀 `AIJIAOXUE_`（viper `AutomaticEnv` + `SetEnvKeyReplacer`）；
- **密钥不得提交仓库**：仓库内只放 `config.example.yaml`，`config.yaml` 进 `.gitignore`；
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

## 8. 接口契约（Sprint 1 的 15 个业务接口 + Sprint 2.1 新增 9 个 + 1 个运维接口）

前缀 `/api/v1`。**字段名与前端 `src/types/` 逐字对齐，本节是前后端联调的唯一事实源。**

> **Sprint 2.1 已实现接口**见本文 §8.8；Sprint 2.2 / Sprint 3 的接口（录音转写、智能体、趋势）见
> [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) §4.3 / §4.4，本文件随实现进度同步。

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
type SupervisorDashboard struct { // 督导
    CourseCount   int       `json:"courseCount"`
    PlanCount     int       `json:"planCount"`
    CompletedCount int      `json:"completedCount"`
    CoverageRate  float64   `json:"coverageRate"`   // 0-1
    RecentPlans   []PlanItem `json:"recentPlans"`   // 前 8 条
}
```

### 8.3 字典 dict（新增，供前端筛选下拉）

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| GET `/departments` | 登录 | 教研室列表 `[{id, name}]` |
| GET `/teachers?departmentId=` | 登录 | 教师列表 `[{id, name}]`，按教研室过滤 |

> 注：此二接口为前端 AGENTS.md §10.2 的补充项（前端文档已同步更新）。学期选项由前端常量维护，不设接口。

### 8.4 课程 courses

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| GET `/courses` | 登录 | 分页列表，数据范围按角色裁剪 |
| GET `/courses/:id` | 登录 | 详情（含班级） |
| POST `/courses` | director | 新增 |
| PUT `/courses/:id` | director | 修改（code 不可改） |

```go
// GET /courses 查询参数（全部可选）
//   semester, departmentId, teacherId, status, keyword, page(默认1), pageSize(默认10, max50)
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
    ID             uint64 `json:"id"`
    CourseID       uint64 `json:"courseId"`
    CourseName     string `json:"courseName"`
    TeacherName    string `json:"teacherName"`
    SupervisorName string `json:"supervisorName"`
    PlannedDate    string `json:"plannedDate"` // YYYY-MM-DD
    Status         string `json:"status"`      // planned|completed
}
```

**覆盖率口径（唯一）**：分母 = 当前学期 `status='open'` 的课程数；分子 = 这些课程中已有 `status='completed'` 听评课记录（DISTINCT course_id）的数量。分部门同理。当前学期取 `semesters` 常量中最新一项（Sprint 1 固定 `2026-2027-1`，定义在 `internal/service/consts.go`）。

### 8.7 运维 healthz

`GET /healthz`（无鉴权）返回 `{"code":0,"data":{"status":"up"}}`，供部署探活与 S1.1 验收。

### 8.8 授课记录与评价聚合（Sprint 2.1，已实现）

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
```

- 🔴 **路由命名**：教师评分列表是 `GET /teacher-scores`；`GET /teachers` 永远是 Sprint 1 的**教师字典**（前端课程表单依赖）。二者不得混用。
- 唯一聚合出口：教师级与课程级都调用 `pkg/scoring.Aggregate`，禁止"先算课程级再对课程取平均"。
- 综合分取**综合均值**口径（各场次综合分的算术平均）；默认数据双侧对齐时与维度加权求和自洽。详见开发计划 §2.5.2。

## 9. 中间件规范（`internal/middleware/`）

注册顺序（`router.go` 中唯一生效顺序）：

```
Recovery → Logger → CORS → [Auth → RBAC]（按分组挂载）
```

| 中间件 | 职责 | 要点 |
|--------|------|------|
| Recovery | panic 兜底 | 返回 500 + code 50001，日志含堆栈 |
| Logger | 请求日志（slog JSON） | method/path/status/耗时/user_id（Auth 之后才有 user_id，可将用户注入 context 由 Logger 在响应后输出） |
| CORS | 白名单放行 | origins 来自配置；允许 `Authorization` 头 |
| Auth | 解析 Bearer token | claims 注入 `c.Set("userID"/"role"/"deptID")`；失败返回 40101 |
| RequireRoles(roles...) | 角色守卫 | 不匹配返回 40301；`router` 分组挂载，禁止散落在 handler 里判断角色 |

路由分组示意（唯一注册地 `router/router.go`）：

```go
v1 := r.Group("/api/v1")
v1.POST("/auth/login", h.Auth.Login)

authed := v1.Group("", mw.Auth())
authed.GET("/auth/me", h.Auth.Me)
authed.POST("/auth/logout", h.Auth.Logout)
authed.GET("/dashboard", h.Dash.Board)
authed.GET("/courses", h.Course.List)
authed.GET("/courses/:id", h.Course.Detail)
authed.GET("/courses/:id/resources", h.Resource.List)
authed.GET("/departments", h.Dict.Departments)
authed.GET("/teachers", h.Dict.Teachers)          // 教师字典（Sprint 1，勿改语义）

// —— Sprint 2.1：评价闭环（数据裁剪仍在 service 层）——
authed.GET("/courses/:id/sessions", h.Session.ListByCourse)
authed.GET("/courses/:id/evaluation-summary", h.TeacherScore.CourseSummary)
authed.GET("/sessions/:id", h.Session.Detail)
authed.GET("/sessions/:id/evaluation", h.Session.Evaluation)
authed.GET("/teachers/:id/evaluation-summary", h.TeacherScore.TeacherSummary)
authed.GET("/teachers/:id/evaluations", h.TeacherScore.TeacherEvaluations)

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
    PUT("/sessions/:id/supervisor-evaluation", h.Session.Submit)

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

- 建库、建表、索引、种子数据：**严格按《MySQL数据库创建指导.md》执行**，两文档表结构以该文档 DDL 为准；
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
| Recording | recordings | **Sprint 2.2 新增**：课堂录音 |
| Transcript | transcripts | **Sprint 2.2 新增**：课堂转写（异步任务产物） |

> Sprint 2/3 的建表 DDL、迁移脚本与评分算法见
> [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) §3、§2。
> 新增表**只新增、不改存量表**（`supervision_plans` 不动，由 `teaching_sessions.plan_id` 反向关联）。

- **事实源分工**：`database/schema.sql` 只管 Sprint 1 存量表；**Sprint 2 起的新表以 `migrations/` 下的迁移脚本为唯一事实源**，二者不重复维护；
- **迁移策略**：从 Sprint 2 起引入 `golang-migrate`（`cmd/migrate` + `migrations/` embed）。GORM **不使用 AutoMigrate**（避免双源漂移）；
  新增表 = 新建迁移脚本 + 更新 `internal/model` + 在 PR 说明列明变更。初始化流程见 `scripts/init_db.sh`：`schema.sql → migrate up → seed.sql → cmd/seed`；
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

**数据链路一键验收**（`scripts/verify.sh`，覆盖 **Sprint 1 + Sprint 2.1**，共 140+ 条断言）：

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

## 更新记录

| 版本 | 日期 | 变更 |
|------|------|------|
| v2.0 | 2026-09 | 均衡覆盖三次 Sprint；注明当前处于 Sprint 2；新增 Sprint 2.1 接口契约（§8.8）、错误码 HTTP 映射与回归说明、`golang-migrate` 迁移规范、文档同步要求（页首）|

*本文档与 [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md)、[`MySQL数据库创建指导.md`](./MySQL数据库创建指导.md)、[`frontend_AGENTS.md`](./frontend_AGENTS.md) 互为配套，任何契约变更须四者同步。*
