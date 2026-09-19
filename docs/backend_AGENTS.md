#  「爱教学」后端编码智能体指导手册（Go + Gin）

> **文档用途**：本文件是写给 AI 智能体（编码助手 GLM 5.3 + OpenCode、架构助手 DeepSeek V4 Pro）的后端编码工作宪法。
> 智能体在为本项目编写任何后端代码之前，**必须先完整阅读本文件**，并严格遵守其中的技术栈、目录结构、分层架构、接口契约与编码规则。
> **使用方式**：后端仓库 `aijiaoxue-api` 初始化后，本文件应置于仓库根目录。
> **配套文档**：《MySQL数据库创建指导.md》（同目录）——建库、建表与种子数据；前端《AGENTS.md》——页面与组件设计，接口契约以本文第 8 节为准。

---

## 1. 项目背景

「爱教学」是面向高校的教学质量全链路数字化管理平台，走三阶进化路线：**查课程（Sprint 1）→ 看课堂（Sprint 2）→ 帮教师（Sprint 3）**。

当前为 Sprint 1「查课程」：全量课程信息透明可查、用户间数据打通、Web 整体框架可无误运作。后端使命是**保障数据链路连通无误**（成员四职责）：登录鉴权 → 角色化数据裁剪 → 课程/资源/督导数据的可靠读写。

## 2. Sprint 1 目标与范围

### 2.1 必须支撑的故事（来自用户故事地图）

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

### 2.2 范围边界

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
| 认证 | golang-jwt/jwt | v5 | HS256，TTL 24h |
| 密码 | golang.org/x/crypto/bcrypt | latest | cost 10 |
| 配置 | spf13/viper | v1.18 | `config.yaml` + 环境变量覆盖 |
| 日志 | log/slog（标准库） | — | JSON 输出，请求日志中间件 |
| 校验 | gin 内置 validator | — | binding tag |
| CORS | gin-contrib/cors | v1.7 | 白名单 origins |
| 热重载 | air-verse/air | latest | 仅开发环境 |
| 测试 | testing（标准库）+ testify | v2 | service 层单测 |

## 5. 目录结构（强制）

```
aijiaoxue-api/
├── AGENTS.md                # 本文件
├── go.mod / go.sum
├── Makefile                 # run / seed / test / lint / build
├── config.yaml              # 本地配置（config.example.yaml 提交模板，config.yaml 不入库）
├── cmd/
│   ├── server/main.go       # 唯一服务入口
│   └── seed/main.go         # 种子数据工具（bcrypt 写入演示账号）
├── internal/
│   ├── config/config.go     # viper 加载与结构体
│   ├── router/router.go     # 路由表 + 分组（唯一路由注册地）
│   ├── middleware/          # recovery / logger / cors / auth / rbac
│   ├── handler/             # HTTP 层：参数绑定、调用 service、组装响应
│   │   ├── auth.go
│   │   ├── course.go
│   │   ├── resource.go
│   │   ├── supervision.go
│   │   └── dict.go
│   ├── service/             # 业务层：权限判断、业务校验、事务边界
│   │   ├── auth.go
│   │   ├── course.go
│   │   ├── resource.go
│   │   ├── supervision.go
│   │   └── dashboard.go
│   ├── repository/          # 数据层：GORM 查询，无业务逻辑
│   │   ├── user.go
│   │   ├── course.go
│   │   ├── resource.go
│   │   └── supervision.go
│   ├── model/               # GORM 模型，与表一一对应
│   └── dto/                 # 请求/响应结构体（json tag 与前端 types 对齐）
├── pkg/
│   ├── response/            # OK / Fail 统一响应
│   ├── errcode/             # 错误码常量
│   └── jwtutil/             # 签发与解析
├── database/
│   ├── schema.sql           # 建表脚本（与 MySQL 文档一致）
│   └── seed.sql             # 种子数据（密码哈希占位，由 cmd/seed 覆写）
├── uploads/                 # 课程资源文件存储（.gitignore）
└── scripts/
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
| 40101 | 401 | 未登录 / token 失效 | 无 Authorization、token 过期 |
| 40301 | 403 | 无角色权限 | 教师调用 `POST /courses` |
| 40302 | 403 | 无数据操作权限 | 教师上传他人课程资源 |
| 40401 | 404 | 资源不存在 | 课程 id 无效 |
| 40901 | 409 | 数据冲突 | 课程编码+学期重复 |
| 50001 | 500 | 服务器内部错误 | 数据库异常 |

- 鉴权失败返回 **HTTP 401 + code 40101**，前端 axios 拦截器据此清 token 跳登录页（与前端 AGENTS.md §10.3 对齐）；
- `message` 面向用户可读（中文），内部细节写日志不外泄；
- 实现：`pkg/response.OK(c, data)` / `pkg/response.Fail(c, errcode.Params)`，handler 中**禁止手写 `c.JSON` 拼信封**。

## 8. 接口契约（15 个业务接口 + 1 个运维接口）

前缀 `/api/v1`。**字段名与前端 `src/types/` 逐字对齐，本节是前后端联调的唯一事实源。**

> **Sprint 2/3 新增接口**（授课记录、当堂课评估、评价聚合、录音转写、智能体）见
> [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) §5，
> 新增错误码 40002 / 40902 / 50002 / 50003 见该文件同节。

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
authed.GET("/teachers", h.Dict.Teachers)

authed.Group("", mw.RequireRoles("director")).
    POST("/courses", h.Course.Create).
    PUT("/courses/:id", h.Course.Update)

authed.Group("", mw.RequireRoles("teacher")).
    POST("/courses/:id/resources", h.Resource.Upload).
    DELETE("/resources/:id", h.Resource.Delete)

authed.Group("", mw.RequireRoles("supervisor")).
    GET("/supervision/coverage", h.Super.Coverage).
    GET("/supervision/plans", h.Super.Plans)
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
> [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) §3、§4。
> 新增表**只新增、不改存量表**（`supervision_plans` 不动，由 `teaching_sessions.plan_id` 反向关联）。

- 迁移策略：开发期以 `database/schema.sql` 为唯一事实源手工执行；GORM **不使用 AutoMigrate**（避免双源漂移）；表结构变更 = 修改 schema.sql + 更新 model + 在 PR 说明列明变更；
- 连接池：`SetMaxOpenConns/SetMaxIdleConns` 来自配置；启动时 `db.Ping()` 失败直接 fatal 退出。

## 14. 编码规范（Go）

1. 遵循 [Go 官方 Code Review Comments](https://go.dev/wiki/CodeReviewComments) 与 `gofmt`/`go vet` 零告警；
2. **命名**：导出用注释开头（`// Foo does ...`）；文件名 snake_case；错误变量 `errXxx`；DTO 后缀 `Req/Resp/DTO`；
3. **错误**：`if err != nil` 立即处理，禁止 `_` 吞错；wrap 用 `fmt.Errorf("...: %w", err)`；
4. **context**：全链路传递，禁止 `context.Background()` 出现在请求路径中；
5. **常量**：角色、状态、类型枚举一律 `internal/service/consts.go` 定义（`RoleDirector = "director"`...），禁止散落字符串字面量；
6. **注释**：默认不写；仅在非显而易见的业务口径（如覆盖率计算）处写一行说明；
7. **测试**：service 层关键函数（覆盖率计算、数据范围裁剪、资源归属校验）须有表驱动单测，表驱动用例覆盖三角色；
8. **Makefile**：`make run`（air 热重载）、`make seed`、`make test`、`make lint`（go vet + gofmt -l）、`make build`；
9. **Git**：Conventional Commits（`feat(course): 新增课程列表接口`）；分支 `feature/S2.1-course-list`（与前端同名故事对齐）。

## 15. 验收标准（DoD）与数据链路打通脚本

每个故事 DoD = 接口可用 + 权限正确 + 数据范围正确 + `go build`/`go vet` 零错误。

**数据链路一键验收**（`scripts/verify.sh`，Sprint 1 评审演示用）：

```bash
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
