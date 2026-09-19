# 爱教学 · 教学质量全链路数字化管理平台

面向高校的教学质量全链路数字化管理平台，三阶进化路线：**查课程（Sprint 1）→ 看课堂（Sprint 2）→ 帮教师（Sprint 3）**。

当前处于 **Sprint 2「看课堂」**：Sprint 1（查课程）已交付；Sprint 2 阶段①「评价闭环」后端已落地，阶段②「智能体接入」（录音、异步转写、智能体评分）开发中；Sprint 3「帮教师」待排期。
各阶段范围见 [`docs/backend_AGENTS.md`](./docs/backend_AGENTS.md) §2 与 [`docs/Sprint2-3-教学评价与提优-开发计划.md`](./docs/Sprint2-3-教学评价与提优-开发计划.md)。

## 仓库结构

```
ITmanage/
├── README.md             # 环境、启停、验证、协作入口
├── CONTRIBUTING.md       # 协作与提交流程：分支规范、红线清单、冲突解决、评审要求、PR 模板
├── docs/                 # 团队设计文档（编码前必读）
│   ├── backend_AGENTS.md          # 后端编码宪法：技术栈 / 目录 / 分层 / 接口契约
│   ├── frontend_AGENTS.md         # 前端编码宪法：页面 / 组件 / 设计令牌 / 接口约定
│   ├── MySQL数据库创建指导.md      # Sprint 1 存量表 DDL、种子数据与迁移策略
│   ├── Sprint2-3-教学评价与提优-开发计划.md   # Sprint 2/3 功能与规范（评分体系、建表、接口、权限、路由）
│   └── Sprint2-3-任务清单.md       # 按模块分发的简短任务清单（详细信息指向开发计划）
├── backend/              # Go + Gin + GORM 后端服务（aijiaoxue-api）
└── frontend/             # Vue 3 + TypeScript + Element Plus 前端（aijiaoxue-web）
```

> **新成员请先读 [`CONTRIBUTING.md`](./CONTRIBUTING.md)**——里面的 §1 给出文档阅读顺序，§4 是每次开发的完整动作序列，§6 是绝不能碰的红线。

## 技术栈

| 层 | 选型 |
|----|------|
| 前端 | Vue 3.4 · TypeScript 5.4 · Vite 5 · Pinia · Vue Router 4 · Element Plus 2.7 · Axios |
| 后端 | Go 1.22+ · Gin 1.10 · GORM 1.25 · golang-jwt v5 · bcrypt · viper · log/slog |
| 数据库 | MySQL 8.0（utf8mb4 / utf8mb4_0900_ai_ci） |

---

## For Developers

> 本章是团队日常开发的操作规范：**启动/终止命令 → 验证方式 → 扩展功能的工作流 → 出错处置**。
> 动手前请先完整阅读 `docs/backend_AGENTS.md`（后端）或 `docs/frontend_AGENTS.md`（前端），二者是分层与契约的权威定义。

### 1. 环境准备（每台开发机一次）

| 依赖 | 版本 | 校验命令 |
|------|------|---------|
| Go | 1.22+ | `go version` |
| Node.js | 18+ | `node -v` / `npm -v` |
| MySQL | 8.0+（需 `utf8mb4_0900_ai_ci`，5.7 不可用） | `mysql --version` |
| Git Bash / WSL（运行 `scripts/*.sh` 与 `make`） | 任意 | `bash --version` |
| Python 3（`scripts/verify.sh` 解析 JSON 用） | 3.x | `python3 --version` |
| air（可选） | latest，后端热重载 | `go install github.com/air-verse/air@latest` |

> 🪟 **Windows 成员**：本仓库的初始化/验收脚本是 bash + make，**请先看文末「附录 C：Windows 首次环境搭建（Git Bash + 原生 MySQL）」**，按它走可一次跑通；`.devtools/mysql/` 是 Linux 开发容器专用，Windows 无法使用（见附录 A 说明）。

```bash
# 前端依赖（仅首次 / package.json 变更后）
cd frontend && npm install

# 后端依赖（仅首次 / go.mod 变更后）
cd backend && go mod tidy
```

> **国内网络必读：配置 Go 模块代理**
> WSL / 国内网络通常**无法直连** `proxy.golang.org`，表现为首次 `go mod tidy`、`make run` 卡住或报 `dial tcp ... i/o timeout`。
> 首次使用前执行一次（写入 `~/.config/go/env`，持久生效）：
>
> ```bash
> go env -w GOPROXY=https://goproxy.cn,direct
> ```
>
> 若本机已运行代理（如 Clash 监听 `127.0.0.1:7897`），也可走代理直连官方源：
>
> ```bash
> export HTTP_PROXY=http://127.0.0.1:7897
> export HTTPS_PROXY=http://127.0.0.1:7897
> go env -w GOPROXY=https://proxy.golang.org,direct
> ```
>
> 校验：`go env GOPROXY`，再在 `backend/` 下执行 `go mod download` 应能正常完成（约 14s / 首次）。

### 2. 首次初始化（只做一次）

```bash
cd backend

# ① 复制配置模板并按本机修改（config.yaml 不入库）
cp config.example.yaml config.yaml
#    至少确认 mysql.dsn 指向本机 MySQL、端口与账号正确

# ② 建库 + 建表 + 迁移 + 种子数据 + 写入 bcrypt 密码哈希（一条命令走完）
bash scripts/init_db.sh
#    等价的五步手工方式（顺序不可调换）：
#    make db-init                                  # schema.sql：建库 / 建用户 / Sprint 1 六张表
#    make migrate                                  # golang-migrate：V1 基线幂等 + V2 建 teaching_sessions / evaluations
#    make db-seed                                  # seed.sql：⚠️ 先 TRUNCATE 全部 8 张表再写入，只对本地库执行
#    make seed                                     # cmd/seed：bcrypt 覆写演示账号密码（密码 123456）
#    mysql -u aijiaoxue -paijiaoxue_dev aijiaoxue -e "SELECT ..."   # 见下方 ③ 自查

# ③ 自查（期望 departments=3 users=6 courses=8 classes=14 resources=6 plans=5 sessions=3 evaluations=6）
mysql -u aijiaoxue -paijiaoxue_dev aijiaoxue -e "
SELECT (SELECT COUNT(*) FROM departments) departments,
       (SELECT COUNT(*) FROM users) users,
       (SELECT COUNT(*) FROM courses) courses,
       (SELECT COUNT(*) FROM course_classes) classes,
       (SELECT COUNT(*) FROM resources) resources,
       (SELECT COUNT(*) FROM supervision_plans) plans,
       (SELECT COUNT(*) FROM teaching_sessions) sessions,
       (SELECT COUNT(*) FROM evaluations) evaluations;"
```

演示账号（密码统一 `123456`）：`director`（王建国·主任）、`teacher`（李明·教师）、`supervisor`（陈静·督导）。

### 3. 每次开发的启动 / 终止流程

三个进程相互独立，**启动顺序：MySQL → 后端 → 前端**；**终止顺序相反**。

#### 3.1 MySQL

```bash
# macOS (Homebrew)            # Ubuntu / Debian (systemd)      # 本仓库沙箱容器（无需 root，仅 Linux 容器）
brew services start mysql     sudo systemctl start mysql        bash .devtools/mysql/start.sh
brew services stop  mysql     sudo systemctl stop  mysql        bash .devtools/mysql/stop.sh
brew services restart mysql   sudo systemctl restart mysql      bash .devtools/mysql/stop.sh && bash .devtools/mysql/start.sh

# Windows（原生 MySQL Installer 安装，服务名默认 MySQL80）—— 详见附录 C
# PowerShell（管理员）：Start-Service MySQL80   /   Stop-Service MySQL80   /   Restart-Service MySQL80
# 或：net start MySQL80
```

> 沙箱容器内的 MySQL 8.0 实例说明见文末「附录 A」（**仅限本 Linux 开发容器；非容器环境请用本机 MySQL**）。
> Windows 首次搭建见文末「附录 C」。

#### 3.2 后端（默认 `http://127.0.0.1:8080`）

```bash
cd backend

# 启动（推荐：air 热重载，保存 .go 文件自动重编译重启）
make run
#   air 未安装时 make run 会自动回退为 go run ./cmd/server

# 启动（不用热重载 / 排查启动期问题时）
go run ./cmd/server -config config.yaml

# 编译为可执行文件后启动（生产/演示）
make build && ./bin/aijiaoxue-api -config config.yaml

# 终止
#   前台运行：Ctrl+C（服务已实现优雅关闭，会等待在途请求结束）
#   后台运行：pkill -f aijiaoxue-api   或   kill $(pgrep -f 'aijiaoxue-api|exe/server')
```

启动成功的标志（stdout JSON 日志）：

```
{"level":"INFO","msg":"mysql connected"}
{"level":"INFO","msg":"server started","addr":":8080","mode":"debug"}
```

#### 3.3 前端（默认 `http://127.0.0.1:5173`）

```bash
cd frontend

npm run dev          # 启动 Vite 开发服务器（已配置 /api 代理到 VITE_PROXY_TARGET，默认 127.0.0.1:8080）
npm run build        # 类型检查 + 生产构建到 dist/
npm run preview      # 本地预览构建产物

# 终止：前台 Ctrl+C；后台：pkill -f "node.*vite"
```

#### 3.4 一页速查

| 动作 | 命令 |
|------|------|
| 启动全部 | ① `bash .devtools/mysql/start.sh` ② `cd backend && make run` ③ `cd frontend && npm run dev` |
| 终止全部 | ① 前端 `Ctrl+C` ② 后端 `Ctrl+C` ③ `bash .devtools/mysql/stop.sh` |
| 只重启后端 | 后端窗口 `Ctrl+C` → `make run`（改用 air 时改 service/repo 会自动重启） |
| 只重启前端 | 一般无需重启（HMR）；**改 `.env.development` / `vite.config.ts` 必须重启** |
| 重新灌种子数据 ⚠️ | `cd backend && make migrate && make db-seed && make seed` —— **`seed.sql` 会先 `TRUNCATE` 全部 8 张表**，仅用于本地开发库 |

> ⚠️ **`make db-seed` 是破坏性操作**：`backend/database/seed.sql` 开头执行 `SET FOREIGN_KEY_CHECKS=0` + 对全部 8 张表（含 `teaching_sessions` / `evaluations`）`TRUNCATE`，会清空既有数据后重新写入种子。
> 新库必须先 `make migrate` 建出 V2 的两张表，否则 `seed.sql` 会因表不存在而报错。
> 只对**本地开发库**执行；**绝不要对共享库或他人正在使用的库执行**。只想重置密码哈希时用 `make seed`（只 UPDATE `users.password_hash`，不清数据）。

### 4. 验证方式（按改动范围选择）

```bash
cd backend

make lint            # go vet + gofmt -l，零告警才算过
make test            # service 层表驱动单测（数据裁剪 / 覆盖率 / 资源归属 / 登录）
make build           # 编译通过

# 端到端验收：143 项断言，覆盖 Sprint 1 三故事线 + Sprint 2.1 评价闭环 + 越权负例 + 覆盖率/评分对账
BASE=http://127.0.0.1:8080/api/v1 MYSQL_PORT=3306 bash scripts/verify.sh

cd ../frontend
npm run lint         # ESLint
npm run test         # Vitest
npm run typecheck    # vue-tsc 类型检查
npm run build        # 构建通过
```

`scripts/verify.sh` 的期望值全部来自 `docs/MySQL数据库创建指导.md` §6 速查表，**改数据口径时同步改脚本与文档**。
注意：脚本会新建一门课程并在结束时清理，因此请勿在业务高峰期对共享库执行。

### 5. 协作与提交流程（PR 模式）

> **完整规范见 [`CONTRIBUTING.md`](./CONTRIBUTING.md)**——标准动作序列、契约先行对照表、红线清单、冲突解决、事故补救、评审要求、PR 描述模板。本节只列最小必读。

**核心规则：任何改动都不得直接推到 `main`**，必须走「建分支 → 自检 → 提 PR → 评审 → 合入」。

```bash
# 开工：从最新 main 切分支（命名 feature/<故事编号>-<短描述>）
git switch main && git pull --ff-only origin main
git switch -c feature/S6.3-session-evaluation

# 开发：契约先行——改接口/表结构/设计约定，先改文档再改代码（见 §6.2 / §6.3）

# 自检全绿后再提交
cd backend      && make lint && make test && make build
cd ../frontend  && npm run lint && npm run typecheck && npm run test && npm run build

git status && git diff --staged      # 先看清改了什么
git add <具体文件>                    # 不要用 git add -A
git commit -m "feat(session): 新增授课记录列表接口"
git push -u origin feature/S6.3-session-evaluation
# 然后在 GitHub 开 PR，目标分支选 main
```

**四条最容易踩的红线**：

| 禁止 | 后果 | 正确做法 |
|------|------|---------|
| 直接 `git push` 到 `main` | 绕过评审，覆盖团队代码 | 开分支提 PR |
| `git push --force` | 重写已共享历史，抹掉他人提交 | 用 `git revert`；只有自己的分支且已 rebase 时才用 `--force-with-lease` |
| `git add -A` 不看 diff 就提交 | 把 `config.yaml`、`.devtools/` 等带进仓库 | `git status` + `git diff --staged` 确认后再逐个 `git add` |
| 对共享数据库跑 `make db-seed` | **`seed.sql` 会 TRUNCATE 全部 6 张表，数据清空** | 只对本地库执行（见 §3.4） |

**评审要求**：至少 1 人 approve 才能合入，**作者不得自审自合**。后端由成员四初审 + 成员一/三复审；前端由成员二初审 + 成员一/三复审。

### 6. 扩展功能的标准工作流

#### 6.1 分层铁律（改代码前先定位层）

```
HTTP → middleware（鉴权/角色守卫）
        → handler   绑定参数 + 校验 + response.OK/Fail
          → service   权限判断、业务规则、事务、聚合
            → repository  GORM 查询，只做数据操作
              → MySQL
```

依赖方向**单向**：`handler → service → repository → model`。禁止反向 import、禁止跨层跳调、禁止在 `router/` 之外注册路由、禁止在 `handler/` 之外读写 `*gin.Context`。

| 你想做的事 | 该改哪个目录 |
|-----------|-------------|
| 新增/修改接口路径、挂中间件、改角色守卫 | `backend/internal/router/router.go` |
| 新增请求/响应字段、加校验规则 | `backend/internal/dto/` + `frontend/src/types/` |
| 新增业务规则（归属校验、唯一性、聚合口径） | `backend/internal/service/` |
| 新增 SQL / 联表 / 聚合查询 | `backend/internal/repository/` |
| 新增表 / 改表结构 | `backend/database/schema.sql` + `backend/internal/model/` |
| 新增错误码 | `backend/pkg/errcode/errcode.go` |
| 新增页面 | `frontend/src/views/` + `frontend/src/router/index.ts` |
| 新增业务组件 | `frontend/src/components/<domain>/` |
| 新增接口调用 | `frontend/src/api/`（**唯一允许 import axios 的地方**） |
| 新增全局状态 | `frontend/src/stores/`（仅 auth / dict 两个 store） |
| 新增颜色、字号、间距 | `frontend/src/styles/tokens.scss`（业务代码禁止硬编码） |

#### 6.2 场景 A：新增一个查询接口（最常见）

**改动顺序固定，不要跳步：**

1. **契约先行** —— 先在 `docs/backend_AGENTS.md` §8 写出接口的方法/路径/权限/DTO 字段，通知前端同步 `frontend/src/types/`。任何 DTO 字段增删改都必须先落文档。
2. **数据层** —— 在 `internal/repository/` 加方法：第一个参数 `ctx context.Context`，用 `WithContext(ctx)`，返回 `model`/行结构体 + `error`，不写业务分支。
3. **业务层** —— 在 `internal/service/` 加方法：先回答「**这个角色看多大范围**」再写代码；用 `ScopeFor(role, deptID, userID)` 拿数据范围并拼进 `CourseFilter`；错误统一返回 `errcode.*`。
4. **HTTP 层** —— 在 `internal/handler/` 加方法：只做 `bindQuery/bindJSON` → 调 service → `response.OK/Fail`，禁止写业务 if 和 SQL。
5. **注册路由** —— 在 `internal/router/router.go` 的对应分组挂上（需要角色限制就用 `middleware.RequireRoles(...)`）。
6. **前端** —— `src/api/` 加请求函数 → `src/types/` 对齐字段 → `src/views/` 或 `components/` 消费；列表页必须实现**加载中 / 空数据 / 加载失败**三态。
7. **验证** —— `make lint && make test && make build`；把断言补进 `scripts/verify.sh`；前端 `npm run typecheck && npm run build`。

#### 6.3 场景 B：新增表 / 修改表结构

> **事实源分工**：`backend/database/schema.sql` 只负责 Sprint 1 存量表；**Sprint 2 起的新表以 `backend/migrations/` 下的迁移脚本为唯一事实源**（内容须与 `docs/Sprint2-3-教学评价与提优-开发计划.md` §3 的 DDL 一致）。

1. 新增表：在 `backend/migrations/` 新建 `<版本>_<名称>.up.sql` / `.down.sql`。**必须用 golang-migrate 命名**（如 `2_teaching_sessions_and_evaluations.up.sql`）；Flyway 风格 `V2__xxx.up.sql` **不被识别**，会让 `make migrate` 报 `first .: file does not exist`。
2. 改 `backend/internal/model/<table>.go`：字段与列一一对应，写 `TableName()`。**禁止使用 GORM `AutoMigrate`**（避免代码与数据库双源漂移）。
3. 执行迁移：
   ```bash
   cd backend
   make migrate           # go run ./cmd/migrate up（V1 基线幂等 + 增量）
   make migrate-version   # 查看当前版本
   make migrate-down      # 回滚一个版本
   ```
4. 存量表（Sprint 1）加列：同步更新 `database/schema.sql` 与 `docs/MySQL数据库创建指导.md`，并新增一条 `ALTER TABLE` 迁移脚本，在 PR 说明中列明。
5. 新表**只新增、不动存量结构**，外键以 `course_id` 挂接。
6. 在 PR 说明中列明「表结构变更清单」，并由成员四 + 成员一复核。

#### 6.4 场景 C：接口契约变更（前后端联调返工的高发区）

```text
改 docs/backend_AGENTS.md §8
  → 通知前端改 frontend/src/types/
    → 后端改 internal/dto/ + service + handler
      → 前端改 src/api/ 与相关 view/component
        → 双方跑 scripts/verify.sh + npm run typecheck
```

**契约变更必须一次性改完四端**（文档 / dto / types / api），否则表现是「接口 200 但页面空白」。

#### 6.5 场景 D：只改前端页面

1. 类型先行：`src/types/` → `src/api/` → `src/components/` → `src/views/`。
2. 颜色/字号/间距只用 `styles/tokens.scss` 里的 CSS 变量。
3. 破坏性操作（删除资源等）必须 `ElMessageBox.confirm` 二次确认。
4. 请求失败提示由 `src/api/http.ts` 拦截器统一弹出，**页面层不要重复弹错**。
5. 验证：`npm run lint && npm run typecheck && npm run build`，再手工过一遍三态与权限显隐。

### 7. 出错的解决措施（排查手册）

先用「断点定位法」判断故障在哪一段：

```
① MySQL（库/表/数据） → ② GORM 连接 & 后端 API → ③ Vite 代理 → ④ 前端页面
```

| 现象 | 定位点 | 处置 |
|------|--------|------|
| 后端启动即退出，日志 `db ping` / `gorm open` 报错 | ① | DSN 错误、MySQL 未启动、账号无权限；核对 `config.yaml` 的 `mysql.dsn` |
| `sql: Scan error` 扫描时间字段失败 | ① | DSN 缺 `parseTime=True` |
| `Unknown collation: 'utf8mb4_0900_ai_ci'` | ① | MySQL 版本 < 8.0；升级，或全量替换为 `utf8mb4_general_ci`（同时改 `schema.sql` 与 DSN） |
| 中文乱码 | ① | 连接串缺 `charset=utf8mb4` |
| 时间差 8 小时 | ① | DSN 缺 `loc=Local` |
| 登录恒返回 `40101`（密码正确） | ① | 种子密码仍是占位串 → `make seed` 覆写 |
| 接口 `40401` | ② | 课程/资源 id 不存在，或超出数据范围（教师看他人课程返回 `40302`） |
| 新增课程返回 `40901` | ② | 同学期同编码已存在（`uk_course_code_semester`），属预期行为 |
| 教师建课返回 `40301` | ② | 路由角色守卫拦截，属预期；主任接口不得由教师调用 |
| 资源上传 `40001` | ② | 扩展名不在 `upload.allowExt`，或超过 `upload.maxSize`，或嗅探出可执行/脚本类型 |
| 资源上传 `50001` 且日志含 `Incorrect datetime value` | ② | 模型里有非 GORM 约定的时间列（如 `uploaded_at`）未赋值，需在 service 显式 `time.Now()` |
| 接口返回非 JSON 或 `404` | ② | 路径前缀错误（业务接口都在 `/api/v1`），或 `router.go` 未注册 |
| 前端所有请求 `Network Error` | ③ | 后端未启动；或 `.env.development` 的 `VITE_PROXY_TARGET` 指错 |
| 前端 401 未跳登录页 | ④ | 后端鉴权失败必须返回 **HTTP 401**（`errcode.Code.HTTPStatus()` 已保证），否则 axios 拦截器走不到清 token 分支 |
| 页面 `code === 0` 但字段是 `undefined` | ④ | 前后端字段名不一致（camelCase 必须逐字对齐），先改文档再改代码 |
| 页面样式色值不对 | ④ | 硬编码色值；应使用 `var(--color-*)`，新增色值先进 `tokens.scss` |
| 上传成功但 `uploads/` 为空 | ① | `upload.dir` 是相对路径，文件落在**后端进程的工作目录**下 |
| `make run` 提示 `air 未安装` | — | 属正常回退；需要热重载则 `go install github.com/air-verse/air@latest` |

**通用处置原则**

1. **先看后端日志**：请求日志是 slog JSON（`method` / `path` / `status` / `latency_ms` / `user_id`），SQL 错误也会打印，多数问题一眼可定位。
2. **错误信息不外泄**：对外只返回 `errcode` 的中文文案，堆栈与 SQL 错误只进日志；排查时以日志为准。
3. **不要绕过权限**：任何「先在前端放开、后端后面补」的写法都违反 `docs/backend_AGENTS.md` §3 数据裁剪铁律，一律拒绝。
4. **不要手改库来兼容代码**：表结构以 `database/schema.sql` 为准，禁止 GORM `AutoMigrate` 造成双源漂移。
5. **范围守卫**：涉及 `docs/*_AGENTS.md` §2.2 Won't 清单的需求（课堂录音、ASR、质量报告、用户注册、第二套 UI 库等）必须先亮红灯说明，不得直接实现。

### 8. 附录 A：容器内沙箱 MySQL（仅本开发容器）

本容器的 3306 端口已被一个凭据未知的 MySQL 占用，因此仓库内另行放置了一个**无需 root** 的本地实例：

| 项 | 值 |
|----|----|
| 位置 | `.devtools/mysql/`（已加入 `.gitignore`） |
| 版本 | MySQL 8.0.46 |
| 端口 | `127.0.0.1:3307` |
| 账号 | `aijiaoxue` / `aijiaoxue_dev`（仅授权 `aijiaoxue` 库） |
| 启动 / 停止 | `bash .devtools/mysql/start.sh` / `bash .devtools/mysql/stop.sh` |
| 配置文件 | `.devtools/mysql/my.cnf` |

`backend/config.yaml`（不入库）中的 DSN 已指向 3307。**在有正式 MySQL 的机器上，请改回 3306**：

```yaml
mysql:
  dsn: "aijiaoxue:aijiaoxue_dev@tcp(127.0.0.1:3306)/aijiaoxue?charset=utf8mb4&parseTime=True&loc=Local"
```

### 9. 附录 B：实现相对设计文档的差异（需团队确认）

后端严格按 `docs/backend_AGENTS.md` §8 契约实现，以下为**为满足现有前端页面与联调所需而做的纯增量补充**（不删改原有字段，向后兼容）：

| 位置 | 补充内容 | 原因 |
|------|---------|------|
| `CourseListItem` | `credit` `hours` `description` `teacherId` `departmentId` | 前端列表列与编辑回填需要；均取自 `courses` 主表，无冗余 |
| `UserDTO` | `departmentId` | 主任新增课程时需默认回填本室 |
| `TeacherOption` | `departmentId` | 前端「按教研室过滤教师下拉」 |
| `TeacherDashboard` | `recentResources` | 前端 §7.3 教师工作台「近期上传资源」侧栏 |
| `SupervisorDashboard` | `byDepartment` | 前端 §7.3「按教研室覆盖率排行」 |
| `GET /healthz` | 同时挂载 `/healthz` 与 `/api/v1/healthz` | 兼容设计文档 §8.7 与 §15 验收脚本两种写法 |
| `GET /resources/:id/download` | 已实现（文档标注「可选」） | 前端资源下载为真实功能，非占位 |
| `courses.objective` / `courses.major` | **未实现** | 前端详情页「培养目标 / 适用专业」在 Sprint 1 DDL 中无对应列，页面已做空值降级；若需启用须按 §6.3 新增列 |

> 前端原 `src/mocks/` 临时数据与 `VITE_USE_MOCK` 开关已删除，前端全部数据来自后端 API（开发期经 Vite proxy）。接口 id 统一使用后端 `uint64` 对应的 `number` 类型。

### 10. 附录 C：Windows 首次环境搭建（Git Bash + 原生 MySQL）

> **适用**：Windows 10/11，本机**从未配置过 MySQL**。目标：把「库 + 种子数据 + 后端 + 前端」跑通，并能执行 143 项验收。
> **路线**：**Git for Windows（Git Bash） + MySQL Installer 原生安装**（服务名默认 `MySQL80`，端口 `3306`）。
> ⚠️ 本仓库 `.devtools/mysql/`（附录 A）是 **Linux 开发容器专用**（ELF 二进制 + bash + `LD_LIBRARY_PATH`），且已 `.gitignore`，Windows 上既拿不到也跑不了，**请忽略它**。

#### C.1 安装四件套

| 依赖 | 下载 | 安装要点 | 校验（Git Bash，装完请重开） |
|------|------|---------|------------------------------|
| Git for Windows | git-scm.com | 默认选项，自带 **Git Bash** | `git --version` |
| Go 1.22+ | go.dev/dl | msi 默认安装 | `go version` |
| Node.js 18+ | nodejs.org（LTS） | msi 默认安装 | `node -v && npm -v` |
| MySQL 8.0 | dev.mysql.com/downloads/installer | 选 **Server only**；**记住 root 密码**；端口保持 **3306**；把 `C:\Program Files\MySQL\MySQL Server 8.0\bin` 加入 PATH | `mysql --version` |
| Python 3 | python.org | 勾选 **Add python.exe to PATH** | `python3 --version`（兜底见 C.7） |

> 版本红线：MySQL 必须是 **8.0+**（排序规则 `utf8mb4_0900_ai_ci`，5.7 不可用）。

#### C.2 配置 Go 模块代理（国内网络必需）

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env GOPROXY
```

> `go env -w` 跨 shell 生效；仅走本机代理时才需要 `$env:HTTP_PROXY="http://127.0.0.1:7897"`（PowerShell）。

#### C.3 启动 MySQL 服务

PowerShell（**管理员**）：

```powershell
Get-Service MySQL80        # 确认服务存在
Start-Service MySQL80      # 启动；停止用 Stop-Service MySQL80
mysql -u root -p -e "SELECT VERSION();"
```

或 `net start MySQL80`。MySQL 8 安装后通常已设为开机自启。

#### C.4 拉取依赖

```bash
cd frontend && npm install
cd ../backend && go mod tidy
```

#### C.5 初始化数据库（Git Bash）

```bash
cd backend
cp config.example.yaml config.yaml     # 复制模板
# 打开 config.yaml，确认 DSN = 127.0.0.1:3306、账号 aijiaoxue / aijiaoxue_dev

# 一条命令走完：建库建账号 → 迁移 → 种子 → 密码哈希 → 自查
MYSQL_ADMIN_PASSWORD='你的root密码' bash scripts/init_db.sh
```

> **为什么用 `MYSQL_ADMIN_PASSWORD`**：Git Bash（mintty）里 MySQL 的交互式密码输入可能卡住；用环境变量传入可绕开。想先验证客户端，可执行 `winpty mysql -u root -p`。

**不想用脚本时的手工五步**（顺序不可调换；请在 Git Bash 中执行）：

```bash
cd backend

# 1) 建库 / 建账号 / 建 Sprint 1 六张表（用 root 执行 schema.sql）
MYSQL_PWD='你的root密码' mysql -h 127.0.0.1 -P 3306 -u root < database/schema.sql

# 2) 版本化迁移：V1 基线幂等 + V2 建 teaching_sessions / evaluations
go run ./cmd/migrate -config config.yaml up

# 3) 种子数据（⚠️ 破坏性：先 TRUNCATE 全部 8 张表，仅本地库）
MYSQL_PWD=aijiaoxue_dev mysql -h 127.0.0.1 -P 3306 -u aijiaoxue aijiaoxue < database/seed.sql

# 4) 覆写 bcrypt 密码哈希（不执行则登录一律 40101）
go run ./cmd/seed -config config.yaml

# 5) 自查（期望 3 / 6 / 8 / 14 / 6 / 5 / 3 / 6）
MYSQL_PWD=aijiaoxue_dev mysql -h 127.0.0.1 -P 3306 -u aijiaoxue aijiaoxue -e "
SELECT (SELECT COUNT(*) FROM departments) departments,
       (SELECT COUNT(*) FROM users) users,
       (SELECT COUNT(*) FROM courses) courses,
       (SELECT COUNT(*) FROM course_classes) classes,
       (SELECT COUNT(*) FROM resources) resources,
       (SELECT COUNT(*) FROM supervision_plans) plans,
       (SELECT COUNT(*) FROM teaching_sessions) sessions,
       (SELECT COUNT(*) FROM evaluations) evaluations;"
```

> 第 2 步不能省：`teaching_sessions` / `evaluations` 由迁移创建，缺失时第 3 步会因表不存在报错。
> `MYSQL_PWD` 只为避开 Git Bash 交互提示，它会在进程环境中短暂可见；生产环境请改用 `mysql_config_editor` 或 `[client]` 配置段。

#### C.6 启动服务（两个 Git Bash 窗口）

```bash
# 窗口 1：后端（不依赖 make）
cd backend && go run ./cmd/server -config config.yaml

# 窗口 2：前端
cd frontend && npm run dev
```

看到 `{"level":"INFO","msg":"mysql connected"}` 与 `server started` 即成功，浏览器打开 http://127.0.0.1:5173 。

#### C.7 跑验收

```bash
cd backend
BASE=http://127.0.0.1:8080/api/v1 MYSQL_PORT=3306 bash scripts/verify.sh
# 期望：143 通过 / 0 失败
```

> **`verify.sh` 依赖 `python3`**。若 Git Bash 中 `python3 --version` 不可用（Windows 版 Python 常只提供 `python`），在当前 Git Bash 会话执行 `alias python3=python` 即可（写进 `~/.bashrc` 可持久）。
> 脚本尾部的清理命令还依赖 `mysql` 在 PATH 中。

#### C.8 `make` 不是必需的

Git Bash 默认不带 `make`；需要可 `choco install make`（或 MSYS2）。不做也行，等价命令如下（均在 `backend/` 下）：

| make 目标 | 等价命令 |
|-----------|---------|
| `make run` | `go run ./cmd/server -config config.yaml` |
| `make test` | `go test ./... -count=1` |
| `make lint` | `go vet ./... && gofmt -l .` |
| `make build` | `go build -o bin/aijiaoxue-api ./cmd/server` |
| `make migrate` | `go run ./cmd/migrate -config config.yaml up` |
| `make migrate-down` | `go run ./cmd/migrate -config config.yaml down 1` |
| `make migrate-version` | `go run ./cmd/migrate -config config.yaml version` |
| `make seed` | `go run ./cmd/seed -config config.yaml` |
| `make db-init` | `mysql -u root -p < database/schema.sql`（Git Bash 可加 `winpty`） |
| `make db-seed` | `mysql -u aijiaoxue -paijiaoxue_dev aijiaoxue < database/seed.sql` |
| `make verify` | `MYSQL_PORT=3306 bash scripts/verify.sh` |

#### C.9 Windows 常见坑

| 现象 | 原因 / 处置 |
|------|------------|
| `mysql: command not found` | MySQL `bin` 未加入 PATH；重开 Git Bash |
| 输入 root 密码后无响应 | Git Bash(mintty) TTY 问题；改用 `MYSQL_ADMIN_PASSWORD=... bash scripts/init_db.sh` 或 `winpty mysql -u root -p` |
| 时间差 8 小时 / GORM 扫描 DATETIME 报错 | DSN 缺 `loc=Local` 或 `parseTime=True`（见 MySQL 文档 §11） |
| `python3: command not found` | 见 C.7 |
| `Error 1045 Access denied` | 账号/密码与 `config.yaml` 不一致；重跑 C.5 第 1 步建账号 |
| 端口 3306 被占用 | `netstat -ano \| findstr :3306`（PowerShell 用 `Get-NetTCPConnection -LocalPort 3306`）；改 MySQL 端口后同步 `config.yaml` 的 DSN 与 `MYSQL_PORT` |
| `make: command not found` | 见 C.8（用等价命令或 `choco install make`） |

---

*文档维护：随接口契约与工作流变更同步更新；后端分层规则见 `docs/backend_AGENTS.md`，前端页面与设计规范见 `docs/frontend_AGENTS.md`。*
