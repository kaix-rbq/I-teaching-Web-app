# 「爱教学」前端编码智能体指导手册

> **文档用途**：本文件是写给 AI 智能体（编码助手、架构助手，运行于 TRAE / OpenCode）的前端编码工作宪法。
> 智能体在为本项目编写任何前端代码之前，**必须先完整阅读本文件**，并严格遵守其中的技术栈、目录结构、组件设计、美术规范与编码规则。
> **使用方式**：前端仓库 `aijiaoxue-web` 初始化后，本文件应置于仓库根目录。

---

> ## 🚦 当前阶段：Sprint 2（阶段① 评价闭环进行中）
>
> | 阶段 | 主题 | 状态 |
> |------|------|------|
> | **阶段① （Sprint 2.1）** | 评价闭环：授课记录 · 督导评分 · 结构化评语 · 三级聚合 · 主任教师管理 · 教师提优页（只读督导分） | **进行中** —— 后端接口已就绪，前端 T1.9–T1.15 待开发/在开发 |
> | **阶段② （Sprint 2.2）** | 智能体接入：课堂录音 · 异步转写 · 智能体评分 · 综合分融合 · 督导页「智能体参考」面板 | **待开发** |
> | **阶段③ （Sprint 3）** | 帮教师：智能体提优建议 · 对话（SSE） · 趋势视图 · 申诉复核 · 质量报告导出 | **后续排期** |
>
> **本手册覆盖全部三个 Sprint。** 项目当前**不再只是 Sprint 1**：Sprint 1 的基础平台已交付，Sprint 2/3 的功能与设计**事实源**是
> **[`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md)**（下称「开发计划」），任务分发见 [`Sprint2-3-任务清单.md`](./Sprint2-3-任务清单.md)。
> **两者冲突时，以开发计划为准**（契约先行：先改文档，再改代码）。

---

## 1. 项目背景

「爱教学」是面向高校（参考东北大学教学管理场景）的**教学质量全链路数字化管理平台**，产品愿景为"让教学质量持续可测"，走三阶进化路线：

| 进阶 | 主题 | 说明 | 迭代 |
|------|------|------|------|
| 进阶 1 | **查课程** | 全量课程信息透明可查 | Sprint 1（已交付） |
| 进阶 2 | **看课堂** | AI 听评课与课堂质量评估（面向督导，兼顾主任/教师查看） | Sprint 2（进行中） |
| 进阶 3 | **帮教师** | 教学改进建议、趋势与对话（面向教师） | Sprint 3（后续排期） |

**核心架构：评分源可插拔。** `supervisor`（督导）与 `agent`（智能体）只是 `evaluations.evaluator_type` 的两个取值，
向同一张表写同样的结构，聚合层只认该表、不关心分数从哪来。这决定了前端的两个基本事实：

① 督导评分表单与智能体参考面板**结构同构**（同一套 5 维度、同一套 DTO），因此可复用同一批组件；
② **智能体延期不阻塞阶段①交付**——「智能体参考」位先在评估页预留，阶段②再填真实数据。

## 2. 三次 Sprint 的目标与范围

> 三次 Sprint 是**递进关系**，不是替换关系：Sprint 2/3 的所有页面都建立在 Sprint 1 的角色体系、课程数据与设计令牌之上。

### 2.1 Sprint 1（基础平台）—— 已交付

**目标**：打通「登录 → 按角色看课程 → 看课程详情/资源 → 看督导覆盖」的最小可用数据链路，为后续评价体系提供课程与用户底座。

**用户故事（来自用户故事地图）**：

| 编号 | 角色 | 故事 | 优先级 |
|------|------|------|--------|
| S1.1 | — | 技术基线：Web 整体框架可无误运作，支撑登录、导航与数据链路 | M |
| S1.2 | 任意用户 | 统一账号按角色登录，以便用户间数据打通 | M |
| S2.1 | 教研室主任 | 查看本室教师课程简介与开课情况，以便统筹排课 | M |
| S2.2 | 教师 | 查看本人课程列表与开课信息，以便掌握教学安排 | M |
| S2.3 | 教学督导 | 查看全校课程开设情况，以便安排听评课 | M |
| S3.1 | 教研室主任 | 新增/修改课程信息，以便课程数据保持最新 | S |
| S4.1 | 教师 | 查看课程资源与学生人次，以便高效备课 | M |
| S4.2 | 教师 | 上传/维护课程资源，以便支撑课后学习 | S |
| S5.1 | 教学督导 | 查看督导覆盖率与听评课安排，以便统筹听评课 | M |

> `[M]` Must 必须实现；`[S]` Should 应该实现；`[C]` Could 有余力实现。

**前端交付物**：

- `LoginView` 三角色登录 + 演示快速入口（S1.1 / S1.2）；
- `AppLayout` 侧边栏按角色渲染 + 顶栏面包屑/用户下拉（S1.1）；
- `DashboardView` 三角色差异化工作台：统计卡 + 主内容 + 侧内容（S2.1 / S2.2 / S2.3 / S5.1）；
- `CourseListView` 单套模板 + 后端数据裁剪（主任本室 / 教师本人 / 督导全校）+ `CourseTable`（S2.1–S2.3）；
- `CourseDetailView` 概要卡 + 三个 Tab（基本信息 / 开课信息 / 课程资源）（S2.x / S4.1 / S4.2）；
- `CourseFormView` 新增/编辑复用 + 校验（S3.1）；
- `ResourceList` / `ResourceUploader` 资源列表、上传（进度/类型/大小校验）、下载、删除（S4.1 / S4.2）；
- `SupervisionView` 督导覆盖率 + 听评课安排（S5.1）；
- `ProfileView` 个人中心（只读，低优先级）。

### 2.2 Sprint 2（课堂评价闭环 + 智能体接入）—— 阶段①进行中 / 阶段②待开发

**目标**：把「课程」下沉到「一次课」，建立**督导评分 → 场次级 → 课程级 → 教师级**的评价数据链；再叠加智能体评分，形成综合分与提优视图。
**它同时服务两个用户**：督导（录入评分）与主任/教师（查看评分）——**主任端与教师端的数字必须逐位一致**，由后端单一聚合函数保证，前端不得各自造口径。

**用户故事**：

| 编号 | 阶段 | 角色 | 故事 | 优先级 |
|------|------|------|------|--------|
| S6.1 | ① | 督导 | 为一次课建立授课记录（课程/班级/日期/节次/主题），以便评价有落点 | M |
| S6.2 | ① | 任一 | 在课程详情页查看该课程的历史授课记录列表 | M |
| S6.3 | ① | 督导 | 在当堂课评估页按 5 个维度打分并留下结构化评语 | M |
| S6.4 | ① | 主任 | 在教师管理页查看本室教师综合评分列表 | M |
| S6.5 | ① | 主任 | 点击教师查看其评分数据面板（综合分 + 分维度 + 分课程） | M |
| S6.6 | ① | 教师 | 在教学提优页查看本人各维度评分与督导评语 | M |
| S6.7 | ① | 督导 | 工作台与「听评课管理」页分离，完整安排列表可分页筛选 | S |
| S7.1 | ② | 督导 | 上传课堂录音并查看转写文本 | M |
| S7.2 | ② | 督导 | 查看智能体对该堂课的分维度评分作为参考 | M |
| S7.3 | ② | 任一 | 综合分 = 督导评分与智能体评分的加权融合 | M |
| S7.4 | ② | 教师 | 在教学提优页看到综合分随智能体接入而变化 | S |

**前端交付物（阶段①，本轮前端主战场）**：

- `types/` + `api/` 新增授课记录 / 评价 / 教师评分接口层（`session.ts`、`teacher.ts`）—— T1.9；
- `CourseDetailView` 新增「**历史授课记录**」列表（`SessionTable`），行点击进评估页 —— T1.10；
- `SessionEvaluationView` 当堂课质量评估页：5 维 1–5 分评分表单 + **锚点 tooltip** + 结构化评语（亮点/待改进/建议）+ **智能体参考预留位** —— T1.11；
- `TeacherListView` 教师管理页：本室/全校教师综合分、5 维分、评价次数 n，默认按姓名排序，`n < 3` 标注「样本不足」 —— T1.12；
- `TeacherDetailView` 教师评分面板：综合分 + 分维度条 + 按课程明细 + 历次评价时间线 —— T1.13；
- `CourseImproveView` 教学提优页：课程基本信息 + 资源入口 + 只读督导分与评语 —— T1.14；
- `/supervision` 改造为「听评课管理」完整分页列表；主任工作台改为「课程管理」「教师管理」两个入口卡 —— T1.15。

**前端交付物（阶段②）**：

- `AudioPlayer` 课堂录音流式播放（含上传入口）—— T2.7；
- `TranscriptViewer` 转写查看器：**异步任务三态**（`pending/running → done → failed`）+ 状态轮询 + 失败重试 —— T2.7；
- `EvaluationCompare` 「智能体参考」对比面板：标注「AI 参考」、展示低置信度维度与 `evidence` 转写引用 —— T2.8；
- 综合分在现有页面上自动生效（前端不改口径，仅展示 `flags` 与 `sample`）。

### 2.3 Sprint 3（帮教师 / 提优）—— 后续排期

**目标**：把「分数」变成「可行动的改进」：教师能看到智能体建议、能看到自己是否在进步、能就课堂改进与智能体对话。

**用户故事**：

| 编号 | 角色 | 故事 | 优先级 |
|------|------|------|--------|
| S8.1 | 教师 | 查看智能体针对课堂记录给出的提优建议 | M |
| S8.2 | 教师 | 与智能体就课堂改进对话 | C |
| S8.3 | 教师 | 查看本人各维度分数的历史趋势 | S |
| S8.4 | 教师 | 对评分提出申诉，督导复核 | S |

**前端交付物**：

- `ScoreTrendChart` 趋势折线组件：按维度切换（含 `frontier`），数据源 `GET /teachers/:id/score-trend`（T3.2）；
- 智能体提优建议列表：在 `CourseImproveView` 按时间倒序展示并标注对应课次（T3.1）；
- 智能体对话（SSE 流式）：`POST /agent/chat`，断流可重连、失败不阻塞页面（T3.3）；
- 教师申诉 / 督导复核界面：申诉记录留痕、状态可追溯（T3.4）；
- 质量报告导出入口（T3.6）。

### 2.4 范围边界（本轮 Won't）

**智能体守则：收到超出下述边界的编码请求时，应在回复中明确指出"该需求超出当前 Sprint 范围"，并建议先与产品负责人确认，不得直接实现。**

- ❌ 移动端 App / 移动端深度适配（桌面优先，目标分辨率 ≥1280px）
- ❌ 用户注册、找回密码、第三方登录（账号由种子数据预置）
- ❌ 真实教务系统对接（授课记录由督导人工建立，不做课表自动同步）
- ❌ 实时课堂直播 / 实时转写（只做**课后**上传与转写）
- ❌ 多模态视频画面分析（只做**音频**链路）
- ❌ 微格教学资源库
- ❌ 引入第二套 UI 组件库、第二个 Web 框架

> **已解除的范围锁定**：原 Sprint 1 Won't 清单中的「课堂录音上传 / 语音转写 / AI 评估」与「质量分析报告 / 教学改进建议」**已正式进入 Sprint 2 / Sprint 3**。
> 开发计划中暂不实现、但**不视为违规**的项：时间衰减、跨学期平均、非对齐数据的通用聚合算法（见开发计划 §2.5.6 / §10.1）。

## 3. 用户角色与权限矩阵

三种角色，英文标识全小写，贯穿路由守卫、菜单渲染、接口鉴权：

| 角色 | 标识 | 核心场景 | 角色主题色 |
|------|------|---------|-----------|
| 教研室主任 | `director` | 查看本室课程与教师评分、统筹排课、维护课程信息 | 蓝 `#2F54EB` |
| 教师 | `teacher` | 查看本人课程、备课（资源/学生人次）、查看本人评分与提优建议 | 青 `#13C2C2` |
| 教学督导 | `supervisor` | 查看全校课程、听评课管理、录入授课记录与评分 | 紫 `#722ED1` |

### 3.1 页面访问权限矩阵（✓ 可访问 · ✗ 禁止 · 只读 = 可看不可写）

| 页面 | 路由 | director | teacher | supervisor | 阶段 |
|------|------|:---:|:---:|:---:|------|
| 登录页 | `/login` | ✓ | ✓ | ✓ | S1 |
| 工作台 | `/dashboard` | ✓ | ✓ | ✓ | S1 |
| 课程列表 | `/courses` | ✓ 本室数据 | ✓ 本人数据 | ✓ 全校数据 | S1 |
| 课程详情 | `/courses/:id` | ✓ | ✓ | ✓ | S1（S2① 追加历史授课记录） |
| 新增课程 | `/courses/new` | ✓ | ✗ | ✗ | S1 |
| 编辑课程 | `/courses/:id/edit` | ✓ | ✗ | ✗ | S1 |
| 教学提优 | `/courses/:id/improve` | ✗ | ✓ 仅本人课程 | ✗ | S2①（S3③ 增强） |
| 教师管理 | `/teachers` | ✓ 本室 | ✗ | ✓ 全校 | S2① |
| 教师评分面板 | `/teachers/:id` | ✓ 本室 | ✗ | ✓ 全校 | S2① |
| 当堂课质量评估 | `/sessions/:id/evaluation` | 只读 本室 | 只读 本人 | ✓ 唯一可写 | S2①（S2② 增强） |
| 听评课管理 | `/supervision` | ✗ | ✗ | ✓ | S1（S2① 改造） |
| 个人中心 | `/profile` | ✓ | ✓ | ✓ | S1 |

### 3.2 页面内操作权限

| 操作 | 位置 | director | teacher | supervisor |
|------|------|:---:|:---:|:---:|
| 新增/编辑课程信息 | 列表页 + 详情页入口 | ✓ | ✗ | ✗ |
| 上传课程资源 | 课程详情·资源 Tab | ✗ | ✓（仅本人课程） | ✗ |
| 删除课程资源 | 课程详情·资源 Tab | ✗ | ✓（仅本人课程） | ✗ |
| 下载课程资源 | 课程详情·资源 Tab | ✓ | ✓ | ✓ |
| 按教师/教研室筛选 | 课程列表筛选栏 | ✓ | ✗（自动限定本人） | ✓ |
| 创建授课记录 | 课程详情·历史授课记录 | ✗ | ✗ | ✓ |
| 提交/覆盖督导评分 | 当堂课评估页 | ✗ | ✗ | ✓ |
| 查看督导评语 | 教师面板 / 提优页 | 本室 | **仅自己** | ✓ |
| 上传课堂录音 / 触发转写 | 当堂课评估页（阶段②） | ✗ | ✗ | ✓ |
| 播放课堂音频 | 当堂课评估页（阶段②） | ✗ | **✗（不可听）** | ✓ |
| 查看转写文本 | 当堂课评估页（阶段②） | ✗ | 本人课程（已脱敏，用于改进） | ✓ |

> 🔴 **教师绝不能看到同事的评分与评语。** 教师端可见范围严格限定为**仅本人**——这是组织敏感信息，越权等于事故。
> **音频对教师不可见**：音频里的学生人声与教师人声不可分割，教师端只开放**已脱敏的转写文本**。

**权限实现三原则**：
1. **路由级**：路由 `meta: { roles: ['director'] }` + 全局前置守卫 `router.beforeEach` 校验，未授权跳转 403 页；
2. **组件级**：操作按钮用 `auth.hasRole('director')` 控制显隐（禁止仅用 CSS 隐藏）；
3. **数据级**：数据范围由后端按 token 裁剪，前端筛选参数只是查询条件，**不得依赖前端做数据越权防护**。

> **评分场景的补充铁律**：「当前角色能看到哪些数据」永远由后端 `service` 层裁剪决定；
> 前端即使拿到越权 URL，也只能得到 40302，页面必须能优雅展示该错误而不是白屏。

## 4. 技术栈（版本锁定，不得擅自升级或替换）

| 类别 | 选型 | 版本 | 说明 |
|------|------|------|------|
| 框架 | Vue 3 | ^3.4 | 一律 Composition API + `<script setup>` |
| 语言 | TypeScript | ^5.4 | 严格模式，禁止 `.js` 业务代码 |
| 构建 | Vite | ^5.2 | — |
| 路由 | Vue Router | ^4.3 | `createWebHistory` 模式 |
| 状态 | Pinia | ^2.1 | 仅 auth / dict 两个全局 store |
| UI 库 | Element Plus | ^2.7 | 按需自动引入；**唯一 UI 库** |
| 请求 | Axios | ^1.6 | 统一实例封装于 `src/api/http.ts` |
| 日期 | Day.js | ^1.11 | — |
| 图表 | ECharts | ^5.5 | 督导覆盖率、评分趋势折线可用；简单可视化优先 CSS/SVG |
| SSE | 原生 `EventSource` / `fetch` 流 | — | 阶段三智能体对话；**不引入第二套 HTTP 库** |
| Mock | 环境变量切换 | — | 预留：`VITE_USE_MOCK=true` 时走 `src/mocks/`（当前前后端直连，mock 目录待建） |
| 测试 | Vitest | ^1.5 | 编码助手产出的工具函数（含评分换算/排序）需附带测试 |

辅助插件：`unplugin-auto-import`、`unplugin-vue-components`（Element Plus 按需引入）。

## 5. 目录结构（强制）

> 本结构同时标注了**现状**（Sprint 1 已存在）与 **Sprint 2/3 待新增**（`← 阶段X` 标记）。新增文件必须落在既有分层内，禁止自创目录。

```
aijiaoxue-web/
├── AGENTS.md                 # 本文件
├── index.html
├── vite.config.ts
├── tsconfig.json
├── .env.development          # VITE_API_BASE_URL / VITE_USE_MOCK
├── .env.production
└── src/
    ├── main.ts
    ├── App.vue
    ├── api/                  # 接口层：唯一的 axios 调用发生地
    │   ├── http.ts           # 实例、拦截器（token 注入、统一错误处理、mock 切换）
    │   ├── auth.ts           # login / logout / me
    │   ├── course.ts         # 课程 CRUD、列表查询
    │   ├── dashboard.ts      # 工作台聚合（把三角色 DTO 收敛为同一形状）
    │   ├── dict.ts           # 学期/教研室/教师字典
    │   ├── resource.ts       # 资源列表、上传、删除、下载
    │   ├── supervision.ts    # 覆盖率、听评课安排
    │   ├── session.ts        # ← 阶段①：授课记录、单场次评估聚合
    │   ├── teacher.ts        # ← 阶段①：教师评分列表与面板、课程级评分
    │   └── agent.ts          # ← 阶段③：智能体对话（SSE）
    ├── assets/               # 图片、logo 等
    ├── components/
    │   ├── common/           # 通用组件（与业务解耦）
    │   │   ├── PageHeader.vue
    │   │   ├── StatCard.vue
    │   │   ├── FilterBar.vue
    │   │   ├── EmptyState.vue
    │   │   └── RoleTag.vue
    │   ├── course/           # 课程域业务组件
    │   │   ├── CourseTable.vue
    │   │   ├── CourseCard.vue
    │   │   ├── CourseInfoForm.vue
    │   │   ├── ResourceList.vue
    │   │   └── ResourceUploader.vue
    │   ├── supervision/      # 督导域业务组件
    │   │   ├── CoverageCard.vue
    │   │   └── ScheduleTable.vue
    │   ├── evaluation/       # ← 阶段①评分域组件
    │   │   ├── ScoreRadar.vue          # 五维雷达/条形
    │   │   ├── ScoreTrendChart.vue     # ← 阶段③：趋势
    │   │   ├── EvaluationForm.vue      # 督导评分表单（含锚点 tooltip）
    │   │   ├── EvaluationCompare.vue   # ← 阶段②：督导 vs 智能体对比
    │   │   ├── CommentPanel.vue        # 结构化评语展示
    │   │   └── TranscriptViewer.vue    # ← 阶段②：转写文本
    │   ├── session/          # ← 阶段①授课记录域
    │   │   ├── SessionTable.vue        # 历史授课记录列表
    │   │   └── AudioPlayer.vue         # ← 阶段②
    │   └── teacher/          # ← 阶段①教师域
    │       ├── TeacherScoreTable.vue
    │       └── TeacherScorePanel.vue
    ├── composables/          # 组合式函数，均以 use 开头
    │   ├── useAuth.ts
    │   ├── useCourseList.ts
    │   ├── usePagination.ts
    │   └── useTranscriptionPolling.ts  # ← 阶段②：转写状态轮询
    ├── constants/            # 角色/状态枚举、上传白名单、学期字典
    │   └── index.ts
    ├── layouts/
    │   ├── AppLayout.vue     # 主布局（侧边栏+顶栏+内容区）
    │   └── BlankLayout.vue   # 登录页等无框架页面
    ├── router/
    │   ├── index.ts          # 路由表
    │   └── guards.ts         # 登录态 + 角色守卫
    ├── stores/
    │   ├── auth.ts           # token / user / hasRole()
    │   └── dict.ts           # 学期、教研室等字典
    ├── styles/
    │   ├── tokens.scss       # 设计令牌（唯一颜色/字号/间距定义处）
    │   ├── element-theme.scss# Element Plus 主题变量覆盖
    │   └── base.scss         # 全局重置与基础样式
    ├── types/
    │   ├── user.ts
    │   ├── course.ts
    │   ├── resource.ts
    │   ├── supervision.ts
    │   ├── dict.ts
    │   ├── evaluation.ts     # ← 阶段①：评分/授课记录/聚合类型
    │   ├── router.d.ts       # RouteMeta 增强
    │   └── api.d.ts          # ApiResponse<T> / PageResult<T>
    ├── utils/
    │   ├── format.ts
    │   └── format.spec.ts
    ├── views/                # 页面组件，一律以 View 结尾
    │   ├── LoginView.vue
    │   ├── DashboardView.vue
    │   ├── CourseListView.vue
    │   ├── CourseDetailView.vue
    │   ├── CourseFormView.vue     # 新增/编辑复用
    │   ├── SupervisionView.vue    # S2① 改造为「听评课管理」
    │   ├── TeacherListView.vue    # ← 阶段①
    │   ├── TeacherDetailView.vue  # ← 阶段①
    │   ├── SessionEvaluationView.vue # ← 阶段①（阶段②增强）
    │   ├── CourseImproveView.vue  # ← 阶段①（阶段③增强）
    │   ├── ProfileView.vue
    │   └── error/
    │       ├── ForbiddenView.vue  # /403
    │       └── NotFoundView.vue
    └── mocks/                # 预留：VITE_USE_MOCK=true 时启用的 mock 拦截（当前仓库尚未创建）
        ├── index.ts
        ├── courses.ts
        └── supervision.ts
```

**目录铁律**：
- 页面组件只出现在 `views/`，业务组件只出现在 `components/<domain>/`；**新增评分相关组件只放 `components/evaluation/`、`components/session/`、`components/teacher/`，不得塞进 `course/`**；
- `api/` 之外的任何文件**禁止 import axios**；
- 颜色、字号、间距**只允许**在 `styles/tokens.scss` 定义，业务代码引用 CSS 变量。

## 6. 路由与页面清单

### 6.1 路由总表

| 路由 | name | 页面 | 布局 | 权限 | 阶段 |
|------|------|------|------|------|------|
| `/login` | login | LoginView | Blank | 公开 | S1 |
| `/` | — | 重定向 `/dashboard` | — | 登录 | S1 |
| `/dashboard` | dashboard | DashboardView | App | 登录 | S1（S2① 按角色重构） |
| `/courses` | course-list | CourseListView | App | 登录 | S1 |
| `/courses/new` | course-new | CourseFormView | App | director | S1 |
| `/courses/:id` | course-detail | CourseDetailView | App | 登录 | S1（S2① 追加历史授课记录） |
| `/courses/:id/edit` | course-edit | CourseFormView | App | director | S1 |
| `/courses/:id/improve` | course-improve | CourseImproveView | App | teacher | S2① 新增（S3③ 增强） |
| `/teachers` | teacher-list | TeacherListView | App | director / supervisor | S2① 新增 |
| `/teachers/:id` | teacher-detail | TeacherDetailView | App | director / supervisor | S2① 新增 |
| `/sessions/:id/evaluation` | session-evaluation | SessionEvaluationView | App | supervisor（他人只读） | S2① 新增（S2② 增强） |
| `/supervision` | supervision | SupervisionView | App | supervisor | S1（S2① 改造为「听评课管理」） |
| `/profile` | profile | ProfileView | App | 登录 | S1 |
| `/403` | forbidden | ForbiddenView | Blank | 公开 | S1 |
| `/:pathMatch(.*)*` | not-found | NotFoundView | Blank | 公开 | S1 |

### 6.2 路由变更表（对照开发计划 §6.1）

| 路由 | name | 现状 | 目标 | 权限 | 变更 |
|------|------|------|------|------|------|
| `/dashboard` | dashboard | 三角色 | 按角色重构（见 §6.3） | 登录 | 改 |
| `/courses` | course-list | — | 不变 | 登录 | — |
| `/courses/new` | course-new | — | 不变 | director | — |
| `/courses/:id` | course-detail | — | **+ 历史授课记录列表** | 登录 | 增强 |
| `/courses/:id/edit` | course-edit | — | 不变 | director | — |
| `/courses/:id/improve` | course-improve | — | **新增** 教学提优 | teacher | 新增 |
| `/teachers` | teacher-list | — | **新增** 教师管理 | director/supervisor | 新增 |
| `/teachers/:id` | teacher-detail | — | **新增** 教师评分面板 | director/supervisor | 新增 |
| `/sessions/:id/evaluation` | session-evaluation | — | **新增** 当堂课质量评估页 | supervisor（他人只读） | 新增 |
| `/supervision` | supervision | 督导总览 | **改造为「听评课管理」** 完整分页列表 | supervisor | 改造（不删路由） |
| `/profile` | profile | — | 不变 | 登录 | — |

**三条不可动摇的路由决策**：

1. **`/supervision` 不删除**：工作台只展示听评课安排前 8 条，删页会导致第 9 条以后无法访问；改造后承载完整分页与状态/日期筛选。
2. **主任工作台移除重复内容**：「本室教师开课情况」表格与「近期开课」列表与「课程管理」重复，改为「课程管理」「教师管理」两个入口卡，**只保留统计卡**。
3. **「教学提优」采用并列路由** `/courses/:id/improve` 而非替换 `/courses/:id`：教师从「我的课程」点击时跳提优页，路径语义清晰，且教师仍可访问原详情页。

### 6.3 三条跳转链路

**① 主任**
```
/dashboard  统计卡 + 「课程管理」「教师管理」入口卡
   ├─▶ /courses（课程管理）
   └─▶ /teachers  本室教师｜综合分｜5维分｜评价次数 n
         └─▶ /teachers/:id  教师评分面板
               ├─ 综合分大数字 + 分维度条形图
               ├─ 按课程分组的分数明细
               ├─ 历次评价时间线（含督导评语）
               └─▶ /courses/:id ─▶ 历史授课记录 ─▶ /sessions/:id/evaluation（只读）
```

**② 督导**
```
/dashboard  统计卡 + 覆盖率环 + 听评课安排（前 8 条）＋「查看全部」
   ├─▶ /supervision  听评课管理（完整分页 + 状态/日期筛选）
   └─▶ /courses/:id  课程详情
         └─ 历史授课记录列表
               └─▶ /sessions/:id/evaluation  当堂课质量评估页
                     ├─ 音频播放器（阶段②）
                     ├─ 转写文本（阶段②，含说话人分离）
                     ├─ 督导评分表单：5 维 1-5 分 + 结构化评语
                     ├─ 智能体评分参考（可折叠，标"AI 参考"）
                     └─ 提交
```

**③ 教师**
```
/dashboard  我的课程卡片
   └─▶ /courses/:id/improve  教学提优
         ├─ 课程基本信息 + 资源上传接口（保留）
         ├─ 评分区：综合分 + 5 维分 + 与上学期对比 +（阶段三）趋势折线
         ├─ 督导评语列表（按时间倒序，标注是哪一次课）
         └─ 智能体提优建议（按时间倒序，阶段三）
```

### 6.4 守卫逻辑（`router/guards.ts`）

1. 无 token 且目标非公开页 → 重定向 `/login?redirect=…`；
2. 有 token 访问 `/login` → 重定向 `/dashboard`；
3. 目标页 `meta.roles` 存在且不含当前角色 → 重定向 `/403`；
4. 登录成功后按角色跳转：一律进入 `/dashboard`（工作台内容按角色渲染）。

> 守卫只做**页面级**拦截；**数据级鉴权在服务端**。前端不得因为"页面能进"就假设"数据能拿"。

## 7. 页面设计规格

### 7.1 LoginView 登录页（S1.1 / S1.2）

```
┌────────────────────────────────────────────┐
│                                            │
│         左侧品牌区(60%) │ 右侧表单区(40%)     │
│   Logo + 「爱教学」     │  欢迎登录          │
│   愿景标语 + 三角色      │  账号输入框        │
│   插画/渐变装饰         │  密码输入框        │
│                       │  登录按钮(主色,全宽) │
│                       │  演示账号提示(灰色)  │
└────────────────────────────────────────────┘
```

- 表单校验：账号/密码非空；错误用 `ElMessage.error` 提示"账号或密码错误"；
- 登录成功：存 token + user 至 `stores/auth`，跳转 `redirect` 或 `/dashboard`；
- 提供**演示角色快捷入口**（三个按钮：以主任/教师/督导身份体验），仅开发环境渲染，便于验收演示；
- 禁止记住密码、注册、第三方登录（超范围）。

### 7.2 AppLayout 主布局（S1.1）

```
┌────────┬──────────────────────────────────┐
│ 224px  │ 顶栏 56px：面包屑 ···· 用户头像▾   │
│ 侧边栏  ├──────────────────────────────────┤
│        │                                  │
│ Logo   │   内容区 padding 24px             │
│ 菜单组  │   max-width 1440px 居中           │
│ (按角色) │   <router-view />                │
│        │                                  │
│ 折叠键  │                                  │
└────────┴──────────────────────────────────┘
```

- 侧边栏菜单**按角色渲染**（Sprint 2① 目标态）：
  - 主任：工作台 / 课程管理 / **教师管理** / 个人中心
  - 教师：工作台 / 我的课程 / 个人中心
  - 督导：工作台 / 全校课程 / **听评课管理** / 个人中心
  - 实现：单一菜单配置数组 + `roles` 过滤，**禁止为每个角色写一份菜单**；
- 菜单项：40px 高，激活态浅蓝底 `--color-primary-bg` + 主色文字 + 左侧 3px 指示条；
- 顶栏右侧：用户姓名 + `RoleTag` + 下拉（个人中心 / 退出登录）；
- 当路由为 `course-detail` / `course-new` / `course-edit` / `course-improve` 时，侧边栏高亮 `course-list`。

### 7.3 DashboardView 工作台（三角色差异化，S2① 重构）

页面结构 = 问候区 + 统计卡行 + 内容区（两栏 2:1）。

| 区块 | director | teacher | supervisor |
|------|----------|---------|------------|
| 统计卡（4 张 StatCard） | 本室课程数 / 本室教师数 / 本学期开课班次 / 课程资源总数 | 我的课程数 / 授课班级数 / 学生总人次 / 资源总数 | 全校课程数 / 本学期听评课计划数 / 已完成听评课 / 督导覆盖率 |
| 主内容（左 2/3） | **两个入口卡：「课程管理」「教师管理」**（S2①：移除原教师开课表格） | 我的课程 CourseCard 网格（点击跳 `/courses/:id/improve`） | 听评课安排 ScheduleTable（前 8 条 + "查看全部"跳 `/supervision`） |
| 侧内容（右 1/3） | **移除「近期开课」列表**，保留快捷入口 | 近期上传资源 + 资源快捷上传入口（跳详情页资源 Tab） | CoverageCard 覆盖率环形进度 + 按教研室的覆盖率排行 |

> 工作台数据由后端按角色返回（`GET /api/v1/dashboard`），前端一套模板 + 角色配置渲染。
> **督导总览与工作台合并**：重复的统计卡与覆盖率卡**只保留一处**（在工作台），`/supervision` 不再重复渲染这两块。

### 7.4 CourseListView 课程列表（S2.1 / S2.2 / S2.3）

```
PageHeader（标题随角色：「课程管理」/「我的课程」/「全校课程」 + 主任右侧「新增课程」按钮）
FilterBar：学期下拉 · 教研室下拉(主任/督导) · 教师下拉(主任/督导) · 状态 · 关键词搜索 · 重置
CourseTable：课程编码 | 课程名称 | 授课教师 | 教研室 | 学期 | 班级数 | 学生人次 | 资源数 | 状态 | 操作
分页器（右下，10/20/50 条每页）
```

- 数据范围由后端裁剪（主任→本室、教师→本人、督导→全校），前端列配置三角色一致；
- 行点击进详情；操作列：查看（全员）、编辑（主任，S3.1）；教师角色行点击改为跳 `/courses/:id/improve`（见 §6.2 决策 3）；
- 列表三态：加载中（骨架屏）/ 空数据（EmptyState + 引导文案）/ 加载失败（重试按钮）。

### 7.5 CourseDetailView 课程详情（S2.x / S4.1 / S4.2 / S6.2）

```
PageHeader：返回 + 课程名称 + 课程编码 Tag + 状态 Tag + 操作（主任：编辑；教师：上传资源）
概要卡：学分 / 学时 / 学期 / 教研室 / 授课教师 / 学生人次（大数字 StatCard 风格）
Tabs：
  ① 基本信息 —— 课程简介（富文本只读）、培养目标、适用专业
  ② 开课信息 —— 班级表格：班级名 | 上课时间 | 地点 | 学生数（S2.1 排课依据）
  ③ 课程资源 —— ResourceList + ResourceUploader（教师可见，S4.1 / S4.2）
  ④ 历史授课记录 —— ← S2① 新增：SessionTable（SessionTable 见 §8.5）
```

**④ 历史授课记录（T1.10）**：

- 列：日期 | 节次 | 主题 | 状态 Tag | 督导分 | 智能体分 | 操作（查看评估）；
- 数据源 `GET /courses/:id/sessions?semester=&page=&pageSize=`，**支持分页与学期切片**；
- 三态完整（加载/空/失败）；空数据引导文案区分角色：督导显示「还没有授课记录，去创建」（S6.1 入口），其他角色显示「暂无授课记录」；
- 行点击跳 `/sessions/:id/evaluation`（S6.3）；**教师/主任进入该页为只读**；
- 无评价侧分数显示 `—`（**不得显示 0**）；`evaluationCount` 用于展示"已评 n 次"。

### 7.6 CourseFormView 课程新增/编辑（S3.1，仅主任）

- 路由 `/courses/new` 与 `/courses/:id/edit` 复用同一组件，依据 `route.params.id` 区分模式；
- 表单字段：课程编码（编辑态只读）、课程名称、所属教研室（下拉）、授课教师（下拉，限本室）、学期（下拉）、学分（1–6）、学时、课程简介（textarea，≤500 字）、状态；
- 校验规则：必填项、学分/学时数字范围、编码格式 `^[A-Z]{2,4}\d{4}$`；
- 提交：主按钮"保存"→ 成功 `ElMessage.success` → 跳转详情页；次按钮"取消"→ 返回上一页，未保存变更需 `ElMessageBox.confirm` 二次确认；
- 编辑模式进入时拉取详情回填，加载中显示骨架屏。

### 7.7 SupervisionView 听评课管理（S5.1，仅督导；S2① 改造）

```
PageHeader：「听评课管理」
FilterBar：状态（计划中/已完成）· 日期起 · 重置
ScheduleTable 听评课安排：课程名称 | 授课教师 | 督导人 | 计划日期 | 状态 | 操作
分页器（完整分页，10/20/50 条每页）
```

- **改造要点（T1.15）**：原「督导总览」的统计卡行与 `CoverageCard` **移除**（已合并到工作台，§7.3），本页只保留**完整分页列表 + 筛选**；
- 这是工作台「查看全部」的落点：工作台只展示前 8 条，本页保证第 9 条以后可访问；
- 覆盖率 = 已完成听评课课程数 / 全校开设课程数，由后端计算，前端只展示；
- 覆盖率可视化优先用 CSS/Canvas 进度环（SVG circle），ECharts 作为备选。

### 7.8 TeacherListView 教师管理页（S6.4，T1.12；方向：本室/全校）

```
PageHeader：「教师管理」 + 显著标注「仅用于教学支持，不作为考核依据」
FilterBar：学期下拉 · 教研室下拉（督导可按教研室筛选）· 重置
TeacherScoreTable：教师姓名 | 工号 | 综合分 | 5 维分（可横向滚动或简列） | 评价次数 n | 操作
分页器
```

- 数据源 `GET /teacher-scores?departmentId=&semester=&page=&pageSize=`（**注意不是 `/teachers`**，见 §10.5）；
- **默认按姓名/工号排序**，按分排序必须是一次**显式操作**（点击表头），不得默认按分排；
- 每行必须展示**评价次数 n**；`n < 3`（`sampleSufficient === false`）在综合分旁标注「样本不足（n=x）」；
- `compositeScore === null`（无评价）显示「暂无评价」并置底，**不得显示 0，也不得按 0 参与排序**；
- `frontier` 维度渲染为「亮点标记」而非计分维度（`isObservation === true`、`weight === 0`）；
- 页面顶部固定展示「**仅用于教学支持，不作为考核依据**」（伦理要求，见开发计划 §5.3）；
- 行点击跳 `/teachers/:id`（S6.5）。

### 7.9 TeacherDetailView 教师评分面板（S6.5，T1.13）

```
PageHeader：教师姓名 + RoleTag/工号 + 学期选择 + 返回
评分总览：综合分（大数字） + 督导分/智能体分（双侧） + 权重标注（α=0.5）
ScoreRadar：5 维条形/雷达（frontier 单独作为亮点标记）
按课程明细：课程编码 | 课程名称 | 综合分 | 各维度分 | 评价次数
历次评价时间线：时间（倒序） | 课程/课次 | 督导评语（亮点/待改进/建议） | 分数
```

- 数据源：面板汇总 `GET /teachers/:id/evaluation-summary?semester=`；**历次评价时间线 `GET /teachers/:id/evaluations?semester=&page=&pageSize=`**（一次拉取即含督导评语与智能体参考，**禁止**逐场调 `/sessions/:id/evaluation` 拼时间线）；
- **数字一致性**：本页综合分与教师端 `/courses/:id/improve` 的综合分必须逐位相同（同一聚合函数），前端不得做任何四舍五入差异处理，直接展示后端返回值；
- 必须展示 `sample`：`sessionCount / evaluatedCount / supervisorCount / agentCount / alignedCount`，其中 `sampleSufficient` 以 **`evaluatedCount`** 为准；
- `flags` 必须显式渲染为提示条（如 `no_data` → 「暂无评价」、`sample_insufficient` → 「样本不足」、`disjoint` → 「督导与智能体评价尚未对齐，当前分数代表性有限」、`formula_mixed` → 「口径版本混杂」）；
- 「按课程明细」行可跳到对应课程详情（再进历史授课记录）。

### 7.10 SessionEvaluationView 当堂课质量评估页（S6.3 / S7.1 / S7.2，T1.11 / T2.7 / T2.8）

```
PageHeader：返回 + 课程名 + 课次信息（日期/节次/主题/班级/教师） + 状态 Tag
场次信息条：授课教师 / 班级 / 授课日期 / 节次 / 主题 / 学期
评分数据面板（整宽，直观可视且信息全面）：
   ScoreRadar 五维雷达（督导评分时实时反映表单，其余角色反映所选评价）
   五维分条：维度名 + 生效权重 + 1–5 分 + 等级；frontier 以「观测项」样式单列（不计入加权）
   摘要：督导总分（后端 totalScore）、已评督导人数、智能体参考状态、口径版本
布局（左 2/3 主区 + 右 1/3 参考区）：
  主区：
    ① EvaluationForm 督导评分表单
       5 个维度，每维 1–5 分（Radio/Step），每分档带锚点 tooltip（§7.10.1）
       结构化评语：总体评语 / 亮点 / 待改进 / 建议（textarea）
       提交按钮（PUT 幂等覆盖 → 成功提示 → 回填新分数）
    ② CommentPanel 已提交评语展示（多督导时按时间列出）
  参考区：
    ③ 智能体参考（阶段②）—— EvaluationCompare，标注「AI 参考」
    ④ 音频与转写（阶段②）—— AudioPlayer + TranscriptViewer
```

- **权限**：仅 `supervisor` 可写；主任（本室）、教师（本人）进入为**只读**——表单禁用、隐藏提交按钮（组件级 `v-if`，非 CSS 隐藏）；
- **五维必填**：任何一维未选即在前端拦截并提示，后端也会返回 `40002`（HTTP **400**）；提交成功后列表/评分需即时反映；
- **幂等覆盖**：同一督导对同一场次重复提交是**覆盖**，前端需在覆盖前给出轻量确认（"将覆盖你上次的评分"）；
- **智能体参考位（阶段①）**：`agentScore` 为 `null` 时展示占位说明「智能体评价将在阶段②接入」，**不得显示 0 分或空图表**；
- `frontier` 在本页同样按「亮点标记」处理（不参与加权总分展示）；
- 阶段②的转写区按三态渲染：`pending/running` → 进度/轮询提示；`failed` → 错误信息 + 「重试」按钮；`done` → 转写文本（含说话人）。轮询用 composable（`useTranscriptionPolling.ts`），**不得在组件里裸写 `setInterval`**。

#### 7.10.1 评分锚点 tooltip（强制）

督导之间打分尺度不一致是本系统最大的可比性风险。**每个维度每个分值必须有可操作的行为描述**，以 tooltip 呈现，文案取自开发计划 §2.3。

| 分 | 等级 | 通用描述 |
|----|------|---------|
| 5 | 优秀 | 可作为示范课 |
| 4 | 良好 | 达到骨干教师水平 |
| 3 | 合格 | 达到基本教学要求 |
| 2 | 待改进 | 存在明显短板 |
| 1 | 不合格 | 需要立即干预 |

维度专属锚点（完整版随 UI 交付，节选）：

| 维度 | 5 分 | 3 分 | 1 分 |
|------|------|------|------|
| `objective` | 目标明确，内容准确无错误，重难点突出 | 目标基本清晰，无实质性知识错误 | 目标缺失或存在知识性错误 |
| `content` | 内容充实有深度，理论联系实际，无水分 | 内容完整但以照本宣科为主 | 内容空洞或明显偏离课程大纲 |
| `interaction` | 有效提问与讨论充分，学生参与度高 | 偶有提问但以自问自答为主 | 全程单向讲授，无任何互动 |
| `organization` | 环节清晰，时间分配合理，节奏张弛有度 | 环节完整但时间分配略显失衡 | 结构混乱或严重拖堂/提前下课 |
| `frontier` | 自然融入学科前沿或交叉应用，与主线结合紧密 | 提及前沿但较生硬 | 无前沿内容（**不单独作为扣分依据**） |

> ⚠️ 两条硬性 UI 约束：`organization` **不得以"学生安静/音量低"作为正向证据**；`frontier` **不得因基础课性质而扣分**，且权重为 0、仅作亮点展示。

### 7.11 CourseImproveView 教学提优页（S6.6 / S7.4 / S8.1 / S8.3，T1.14）

> **独立路由** `/courses/:id/improve`，与 `/courses/:id` 并列（见 §6.2 决策 3）。教师从「我的课程」点击课程条目直接进入本页。

```
PageHeader：返回 + 课程名称 + 学期选择
区块一：课程基本信息（学分/学时/学期/班级/学生人次）—— 保留
区块二：资源区（ResourceList + ResourceUploader）—— 保留教师上传/删除能力
区块三：评分区（阶段①）
        综合分（大数字）+ 5 维分（ScoreRadar）+ 与上学期对比
        样本量标注：已评价场次 evaluatedCount / 授课场次 sessionCount
区块四：督导评语列表（阶段①，按时间倒序，标注对应课次）
区块五：智能体提优建议（阶段③，按时间倒序）
区块六：趋势折线（阶段③，ScoreTrendChart，可按维度切换）
区块七：与智能体对话（阶段③，SSE 流式）
```

- 数据源：课程级评分 `GET /courses/:id/evaluation-summary?semester=`；本人教师级评分 `GET /teachers/:id/evaluation-summary?semester=`（`id` = 当前登录用户）；
- **只读**：教师**不能**修改分数或评语，页面不出现任何编辑态控件；
- 综合分为 `null` 时显示「暂无评价」，**不得显示 0**；
- 突出「**仅用于教学支持**」的语气：文案面向改进而非考核（如「本期共 3 次课被评价，互动维度是提升空间」）；
- 资源上传沿用 Sprint 1 契约（`ResourceUploader`），**不得因为页面改造而丢失该能力**（T1.14 验收项）。

### 7.12 ProfileView 个人中心（辅助，最低优先级）

- 展示：头像、姓名、角色、所属教研室/工号、账号；
- **不做资料编辑**（超范围）。

## 8. 组件设计清单

> 组件一律「展示与交互」，数据请求放页面或 composable，通过 props 下发。**每张表标注所属阶段：`①`=Sprint 2.1 必做，`②`=Sprint 2.2，`③`=Sprint 3。**

### 8.1 布局组件（`layouts/` + `components/common/`）

| 组件 | Props（概要） | 职责 | 消费方 | 阶段 |
|------|--------------|------|--------|------|
| AppLayout | — | 侧边栏 + 顶栏 + 内容区骨架 | 已登录路由 | S1 |
| BlankLayout | — | 无框架容器 | 登录/403/404 | S1 |
| PageHeader | `title, subtitle` + slot#actions | 页面标题 + 操作区，统一页首留白 | 所有内容页 | S1 |
| StatCard | `label, value, unit, icon, tone` | 指标卡（图标 + 大数字 + 标签） | 工作台/详情/督导/评分 | S1 |
| FilterBar | slot + `@search/@reset` | 筛选栏容器，收拢查询交互 | 课程列表/听评课/教师管理 | S1 |
| EmptyState | `description` + slot#action | 空数据占位（引导操作） | 所有列表 | S1 |
| RoleTag | `role` | 角色标签，三角色三色 | 顶栏/表格 | S1 |

### 8.2 课程域组件（`components/course/`）

| 组件 | Props（概要） | 职责 | 对应故事 | 阶段 |
|------|--------------|------|---------|------|
| CourseTable | `rows, loading, showEdit` | 课程数据表格 + 行操作 | S2.1–S2.3 | S1 |
| CourseCard | `course` | 课程卡片（工作台教师视图） | S2.2 | S1 |
| CourseInfoForm | `modelValue, mode` | 新增/编辑表单 + 校验 | S3.1 | S1 |
| ResourceList | `resources, canManage` | 资源列表 + 下载/删除 | S4.1 / S4.2 | S1 |
| ResourceUploader | `courseId, accept, maxSize` | 拖拽 + 点击上传，进度与类型校验 | S4.2 | S1 |

### 8.3 督导域组件（`components/supervision/`）

| 组件 | Props（概要） | 职责 | 对应故事 | 阶段 |
|------|--------------|------|---------|------|
| CoverageCard | `overall, byDepartment[]` | 覆盖率环形 + 分组条形 | S5.1 | S1 |
| ScheduleTable | `plans, loading` | 听评课安排表 | S5.1 / S6.7 | S1 |

### 8.4 评价域组件（`components/evaluation/`，新增）

| 组件 | Props（概要） | 职责 | 对应故事 | 阶段 |
|------|--------------|------|---------|------|
| ScoreRadar | `dimensions, loading` | 五维条形/雷达；`isObservation` 维度单独作亮点标记 | S6.5 / S6.6 | ① |
| EvaluationForm | `sessionId, modelValue, readonly, loading` | 5 维 1–5 分 + 锚点 tooltip + 三条结构化评语；`readonly` 时禁用 | S6.3 | ① |
| CommentPanel | `items, loading` | 督导评语展示（亮点/待改进/建议） | S6.6 | ① |
| EvaluationCompare | `supervisor, agent, loading` | 督导 vs 智能体分维度对比；标注「AI 参考」与低置信度，展示 `evidence` 引用 | S7.2 | ② |
| TranscriptViewer | `status, segments, text, loading` + `@retry` | 转写文本三态（pending/running/done/failed）与说话人区分 | S7.1 | ② |
| ScoreTrendChart | `series, dimension, loading` | 按维度切换的历史趋势折线 | S8.3 | ③ |

### 8.5 授课记录域组件（`components/session/`，新增）

| 组件 | Props（概要） | 职责 | 对应故事 | 阶段 |
|------|--------------|------|---------|------|
| SessionTable | `rows, loading, readonly, canCreate` | 历史授课记录表（日期/节次/主题/状态/双侧分/已评次数）+ 行点击 | S6.2 | ① |
| AudioPlayer | `src, duration, loading` | 课堂录音流式播放（Range 请求，服务端支持） | S7.1 | ② |

### 8.6 教师域组件（`components/teacher/`，新增）

| 组件 | Props（概要） | 职责 | 对应故事 | 阶段 |
|------|--------------|------|---------|------|
| TeacherScoreTable | `rows, loading, semester` | 教师列表：综合分、5 维分、评价次数 n、样本不足标记；默认按姓名排序 | S6.4 | ① |
| TeacherScorePanel | `summary, loading` | 教师评分面板：综合分 + 维度条 + 按课程明细 + 历次评价时间线 | S6.5 | ① |

### 8.7 组件编写规则

- 一律 `<script setup lang="ts">`；Props 用 `defineProps<{…}>()` 类型声明；Emits 用 `defineEmits<{…}>()`；
- 组件只负责**展示与交互**，数据请求放在页面或 composable，通过 props 下发；
- 对外透传 UI 库组件属性时使用 `inheritAttrs: false` + `v-bind="$attrs"`；
- 每个业务组件 props 中必须有 `loading`（或等价）状态，支撑骨架屏/禁用态；
- **评分类组件额外要求**：必须显式处理 `null` 分（渲染 `—` 而非 0）、必须透传并展示 `sample` 与 `flags`，不得在组件内自行"补齐"缺失侧分数。

## 9. 美术与设计规范（Design Tokens）

**风格定位：「学院蓝 · 清新教育」** —— 专业可信、清爽轻盈、卡片化、留白充分。面向高校教师/主任/督导用户，信息密度中等，强调数据的可读性与层级。

### 9.1 色彩系统

| Token | 值 | 用途 |
|-------|-----|------|
| `--color-primary` | `#2F54EB` | 主色：主按钮、激活态、链接 |
| `--color-primary-hover` | `#597EF7` | 主色悬停 |
| `--color-primary-active` | `#1D39C4` | 主色按下 |
| `--color-primary-bg` | `#F0F5FF` | 激活菜单底、选中行、主色浅背景 |
| `--color-success` | `#52C41A` | 成功、开课中 |
| `--color-warning` | `#FAAD14` | 提醒、草稿态 |
| `--color-error` | `#FF4D4F` | 错误、删除 |
| `--color-info` | `#13C2C2` | 信息、教师角色色 |
| `--color-director` | `#2F54EB` | 主任角色色（头像底/RoleTag） |
| `--color-teacher` | `#13C2C2` | 教师角色色 |
| `--color-supervisor` | `#722ED1` | 督导角色色 |
| `--color-text-primary` | `#1F2329` | 一级文字（标题/数值） |
| `--color-text-secondary` | `#4E5969` | 二级文字（正文） |
| `--color-text-tertiary` | `#86909C` | 三级文字（辅助说明） |
| `--color-text-disabled` | `#C9CDD4` | 禁用文字 |
| `--color-border` | `#E5E6EB` | 输入框/卡片描边 |
| `--color-divider` | `#F2F3F5` | 分割线 |
| `--color-bg-page` | `#F7F8FA` | 页面背景 |
| `--color-bg-card` | `#FFFFFF` | 卡片背景 |

**色彩铁律**：业务代码**禁止出现任何硬编码色值**，必须 `var(--color-*)`；新增颜色必须先在 `tokens.scss` 定义并说明用途；状态色仅用于语义（成功/警告/错误），不得作装饰。

### 9.2 字体与排版

```scss
--font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
  "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans SC", sans-serif;
--font-size-xs: 12px;   // 辅助标签
--font-size-sm: 13px;   // 次要说明
--font-size-base: 14px; // 正文、表格（全局基准）
--font-size-lg: 16px;   // 区块标题
--font-size-xl: 18px;   // 卡片标题
--font-size-2xl: 20px;  // 页面副标题
--font-size-3xl: 24px;  // PageHeader 标题
--font-size-stat: 28px; // StatCard / 综合分大数字（tabular-nums）
```

- 行高：正文 1.6，标题 1.3；
- 字重：标题 600，正文 400，StatCard/综合分数字 600 + `font-variant-numeric: tabular-nums`。

### 9.3 间距 / 圆角 / 阴影

```scss
--spacing-1: 4px;  --spacing-2: 8px;  --spacing-3: 12px;
--spacing-4: 16px; --spacing-6: 24px; --spacing-8: 32px;  // 4px 基数
--radius-sm: 4px;  // Tag
--radius-md: 6px;  // 按钮、输入框
--radius-lg: 12px; // 卡片、Modal
--shadow-card: 0 1px 2px rgba(15,23,42,.04), 0 1px 3px rgba(15,23,42,.06);
--shadow-card-hover: 0 4px 12px rgba(15,23,42,.10);
--shadow-modal: 0 12px 32px rgba(15,23,42,.16);
```

### 9.4 布局尺寸

| 项 | 值 |
|----|-----|
| 侧边栏宽 | 224px（可折叠至 64px，仅图标） |
| 顶栏高 | 56px |
| 内容区内边距 | 24px，max-width 1440px 居中 |
| 表格行高 | 48px（紧凑场景 40px），表头底色 `--color-bg-page` |
| 卡片内边距 | 24px（紧凑 16px） |
| 适配目标 | 桌面优先 ≥1280px；1024–1280px 侧边栏默认折叠；<1024px 不承诺 |

### 9.5 交互与反馈规范

- 列表页**三态必备**：Loading（骨架屏，禁止白屏转圈）/ Empty（EmptyState + 引导按钮）/ Error（错误图 + 重试）；
- 表单校验：失焦即时校验，提交时整体校验，错误文案具体（"请输入课程名称"而非"输入有误"）；
- 破坏性操作（删除资源、覆盖评分）：必须 `ElMessageBox.confirm` 二次确认，并写明对象名称；
- 成功反馈统一 `ElMessage.success`；失败提示由 `api/http.ts` 拦截器统一弹出，**页面层不得重复弹错**；
- 上传约束：类型白名单 `pdf/doc/docx/ppt/pptx/mp4/zip`，单文件 ≤100MB，显示进度条，失败可重试。

### 9.6 评分展示规范（Sprint 2/3 补充）

评分是本产品最容易产生误导的地方，以下为**展示层硬性要求**（口径由后端决定，前端不得重算）：

1. **缺失 ≠ 0**：`null` 分一律渲染 `—`（或灰底占位），**禁止显示 0、禁止用 0 参与前端排序**；
2. **frontier 不是计分维度**：`isObservation: true` / `weight: 0` 的维度单独以「亮点标记」样式展示，不得混入综合分的加权说明；
3. **必须暴露样本量**：任何展示综合分的位置都要能看到 `evaluatedCount`；`sampleSufficient === false` 时紧邻综合分标注「样本不足（n=x）」；
4. **AI 必须显形**：智能体分数一律带「AI 参考」标签；低置信度维度（`aiConfidence < 0.4`）单独提示；没有 `evidence` 引用的智能体分数不予展示（防误导）；
5. **口径提示不隐藏**：`flags` 含 `disjoint` / `formula_mixed` / `no_data` 时，页面需有可见提示条，**不得静默**；
6. **伦理标注常驻**：教师管理列表与教师评分面板顶部固定展示「仅用于教学支持，不作为考核依据」。

> 评分色阶由设计在 `tokens.scss` 统一补充；**禁止在业务组件里散落写死颜色**（同 §9.1 色彩铁律）。

## 10. 数据模型与接口约定

### 10.1 核心类型（`src/types/`，Sprint 1 现状）

```ts
type Role = 'director' | 'teacher' | 'supervisor';

interface User {
  id: number;
  name: string;
  role: Role;
  departmentId?: number;
  department?: string;   // 教研室
  jobNo?: string;        // 工号
  avatar?: string;
}

interface ClassInfo {
  id: number;
  className: string;     // 如 "软件 2201"
  schedule: string;      // 如 "周一 3-4 节"
  location: string;
  studentCount: number;
}

interface Course {
  id: number;
  code: string;              // 如 "SE3101"
  name: string;
  credit: number;
  hours: number;
  semester: string;          // 如 "2026-2027-1"
  departmentId: number;
  department: string;        // 教研室
  teacherId: number;
  teacherName: string;
  description: string;
  objective?: string;        // 培养目标（后端当前不返回，渲染按空值降级）
  major?: string;            // 适用专业（同上）
  classes?: ClassInfo[];     // 仅详情接口返回
  classCount: number;
  studentCount: number;      // 各班学生数之和（后端汇总）
  resourceCount: number;
  status: 'open' | 'draft' | 'closed';
}

interface Resource {
  id: number;
  courseId: number;
  name: string;
  type: 'pdf' | 'doc' | 'ppt' | 'video' | 'zip' | 'other';
  size: number;              // 字节
  uploader: string;
  uploadedAt: string;        // ISO 8601
}

interface SupervisionPlan {
  id: number;
  courseId: number;
  courseName: string;
  teacherName: string;
  supervisorName: string;
  plannedDate: string;
  status: 'planned' | 'completed';
}

interface CoverageStat {
  totalCourses: number;
  supervisedCourses: number;
  rate: number;              // 0-1，后端计算
  byDepartment: { department: string; rate: number }[];
}

interface ApiResponse<T> { code: number; message: string; data: T }
interface PageResult<T> { list: T[]; total: number; page: number; pageSize: number }
```

### 10.2 Sprint 2 新增类型（`src/types/evaluation.ts`，T1.9）

> **逐字对齐后端 `internal/dto/evaluation.go`**，字段名即后端 JSON tag；缺失侧一律 `null`。

```ts
type SessionStatus = 'scheduled' | 'recorded' | 'evaluated';
type EvaluatorType = 'supervisor' | 'agent';

/** 课程历史授课记录行（GET /courses/:id/sessions） */
interface SessionListItem {
  id: number;
  sessionDate: string;               // YYYY-MM-DD
  period: string;                    // 如 "3-4 节"
  topic: string;
  status: SessionStatus;
  supervisorScore: number | null;    // 单次总分（仅展示/排序）
  agentScore: number | null;
  evaluationCount: number;
}

/** 单场次基本信息 */
interface SessionDetail {
  id: number;
  courseId: number;
  courseCode: string;
  courseName: string;
  classId: number;                   // 0 = 未指定
  className: string;
  teacherId: number;
  teacherName: string;
  semester: string;
  sessionDate: string;
  period: string;
  topic: string;
  status: SessionStatus;
}

/** 一条评价的完整展示（督导表单回填 / 智能体参考共用） */
interface EvaluationDTO {
  evaluatorId: number;
  evaluatorName: string;
  evaluatorType: EvaluatorType;
  aiModelVersion: string;
  aiConfidence: number | null;
  formulaVersion: string;
  objective: number | null;          // 1-5；智能体侧 objective 恒为 null
  content: number | null;
  interaction: number | null;
  organization: number | null;
  frontier: number | null;
  totalScore: number | null;
  comment: string;
  highlights: string;                // 亮点
  improvements: string;              // 待改进
  suggestions: string;               // 建议 / 智能体提优建议
  createdAt: string;
  updatedAt: string;
}

/** GET /sessions/:id/evaluation 聚合响应（当堂课评估页） */
interface SessionEvaluation {
  session: SessionDetail;
  supervisorScores: EvaluationDTO[];  // 一次课可多督导
  agentScore: EvaluationDTO | null;   // 阶段一恒为 null
}

/** 督导评分提交体（PUT /sessions/:id/supervisor-evaluation，五维必填） */
interface SupervisorEvaluationPayload {
  objective: number;
  content: number;
  interaction: number;
  organization: number;
  frontier: number;
  comment?: string;
  highlights?: string;
  improvements?: string;
  suggestions?: string;
}

type DimensionKey = 'objective' | 'content' | 'interaction' | 'organization' | 'frontier';

type ScoreFlag =
  | 'no_data' | 'sup_only' | 'ai_only' | 'disjoint'
  | 'sample_insufficient' | 'agent_not_calibrated' | 'formula_mixed';

interface DimensionScore {
  key: DimensionKey;
  name: string;
  weight: number;                    // frontier 为 0
  isObservation: boolean;            // frontier: true
  score: number | null;              // 0-100
  supervisorScore: number | null;
  agentScore: number | null;
}

interface ScoreSample {
  sessionCount: number;              // 授课场次
  evaluatedCount: number;            // 已评价场次 ← 样本充足性以此为准
  supervisorCount: number;
  agentCount: number;
  alignedCount: number;              // 双侧都有评价的场次，必须暴露
  sampleSufficient: boolean;         // = evaluatedCount >= 3
}

interface ScoreSummary {
  compositeScore: number | null;     // 综合分；无评价为 null，禁止显示 0
  supervisorScore: number | null;
  agentScore: number | null;
  dimensions: DimensionScore[];
  sample: ScoreSample;
  flags: ScoreFlag[];
  weights: { supervisor: number; agent: number };
  formulaVersion: string;
}

/** GET /teacher-scores 行（继承 ScoreSummary） */
interface TeacherScoreItem extends ScoreSummary {
  teacherId: number;
  teacherName: string;
  jobNo: string;
}

/** GET /teachers/:id/evaluation-summary 的「按课程明细」行 */
interface CourseScoreItem extends ScoreSummary {
  courseId: number;
  courseCode: string;
  courseName: string;
}

interface TeacherEvaluationSummary extends ScoreSummary {
  teacherId: number;
  teacherName: string;
  semester: string;
  courses: CourseScoreItem[];
}

interface CourseEvaluationSummary extends ScoreSummary {
  courseId: number;
  courseCode: string;
  courseName: string;
  teacherId: number;
  teacherName: string;
  semester: string;
}

/** GET /teachers/:id/evaluations 的历次评价时间线行（T1.13） */
interface TeacherEvaluationTimelineItem {
  sessionId: number;
  sessionDate: string;       // YYYY-MM-DD，时间线倒序依据
  period: string;
  topic: string;
  courseId: number;
  courseCode: string;
  courseName: string;
  status: SessionStatus;
  compositeScore: number | null;   // 该场次综合分（与教师级聚合同源）
  supervisorScore: number | null;  // 同场多督导总分均值
  agentScore: number | null;
  supervisorEvaluations: EvaluationDTO[];  // 含 comment/highlights/improvements/suggestions
  agentEvaluation: EvaluationDTO | null;   // 阶段②前通常为 null
}
```

### 10.3 接口清单（RESTful，前缀 `/api/v1`）—— Sprint 1 基线

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | `/auth/login` | 公开 | 入参 `{username, password}`，返回 `{token, user}` |
| GET | `/auth/me` | 登录 | 当前用户信息 |
| POST | `/auth/logout` | 登录 | 退出 |
| GET | `/dashboard` | 登录 | 按角色返回工作台聚合数据 |
| GET | `/courses` | 登录 | 查询参数：`semester/departmentId/teacherId/status/keyword/page/pageSize`；数据范围按角色裁剪 |
| GET | `/courses/:id` | 登录 | 课程详情（含 classes） |
| POST | `/courses` | director | 新增课程 |
| PUT | `/courses/:id` | director | 修改课程 |
| GET | `/courses/:id/resources` | 登录 | 资源列表 |
| POST | `/courses/:id/resources` | teacher | 上传资源（multipart/form-data） |
| DELETE | `/resources/:id` | teacher | 删除资源 |
| GET | `/resources/:id/download` | 登录 | 下载资源（需 Authorization，前端以 blob 取回） |
| GET | `/supervision/coverage` | supervisor | 覆盖率统计 |
| GET | `/supervision/plans` | supervisor | 听评课安排（分页 + 状态筛选；日期参数为 `dateFrom`） |
| GET | `/departments` | 登录 | 教研室列表（课程列表筛选下拉数据源，`[{id, name}]`） |
| GET | `/teachers` | 登录 | **教师字典**（课程表单下拉数据源，`[{id, name, departmentId?}]`） |

### 10.4 Sprint 2.1 新增接口（阶段① ✅ 已实现）

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/courses/:id/sessions?semester=&page=&pageSize=` | 登录（数据裁剪） | 课程历史授课记录列表，含每场双侧评分摘要 |
| POST | `/sessions` | supervisor | 创建授课记录 |
| GET | `/sessions/:id` | 登录（数据裁剪） | 单场次基本信息 |
| GET | `/sessions/:id/evaluation` | 登录（数据裁剪） | **当堂课评估页聚合**：场次 + 督导评分 + 智能体参考 + 评语 |
| PUT | `/sessions/:id/supervisor-evaluation` | supervisor | 提交/覆盖督导评分与评语（幂等）；**5 个维度全部必填**，缺失返回 `40002` |
| GET | `/teacher-scores?departmentId=&semester=&page=&pageSize=` | director（本室）/ supervisor（全校） | **教师评分列表**：综合分、双侧分、分维度、样本量 |
| GET | `/teachers/:id/evaluation-summary?semester=` | director（本室）/ teacher（仅自己）/ supervisor | 教师级评分面板（含按课程明细） |
| GET | `/teachers/:id/evaluations?semester=&page=&pageSize=` | director（本室）/ teacher（仅自己）/ supervisor | **教师历次评价时间线**：按课次倒序，含督导结构化评语 + 智能体参考 + 场次综合分 |
| GET | `/courses/:id/evaluation-summary?semester=` | 登录（数据裁剪） | **课程级**评分（教学提优页用），与教师级共用聚合函数 |

> 🔴 **`GET /teachers` 永远是教师字典，教师评分列表是 `GET /teacher-scores`。**
> 两者并存、用途不同：`/teachers` 返回数组（课程表单下拉依赖），`/teacher-scores` 返回分页评分列表。
> **新增接口不得复用二者，也不得再改路径而不更新本手册与开发计划 §4.2。**

### 10.5 阶段二 / 阶段三接口（未实现，未实现前统一返回 `501 / 50002`）

| 方法 | 路径 | 权限 | 阶段 | 说明 |
|------|------|------|------|------|
| POST | `/sessions/:id/recording` | supervisor | ② | multipart 上传音频（扩展名与大小白名单校验） |
| GET | `/recordings/:id/stream` | supervisor（教师不可见） | ② | 音频流式播放，支持 Range |
| GET | `/sessions/:id/transcript` | 登录（数据裁剪） | ② | 转写文本；未完成时返回 `status` 供轮询 |
| POST | `/sessions/:id/transcript/retry` | supervisor | ② | 转写失败后重试 |
| POST | `/agent/chat` | 登录 | ③ | 智能体对话（SSE 流式） |
| GET | `/teachers/:id/score-trend?semester=&dimension=` | director（本室）/ teacher（仅自己）/ supervisor | ③ | 趋势序列 |

> 智能体接口**必须与主链路解耦**：调用失败只记录日志并保留 `transcripts.status='failed'`，**不得影响督导评分与主流程**。前端同理：AI 区块失败只能局部降级，不得让整页报错。

### 10.6 错误码（新增部分）

| code | HTTP | 含义 | 前端处置 |
|------|------|------|---------|
| 40001 | 400 | 参数校验失败 | 表单内联提示 |
| **40002** | **400** | 业务规则校验失败（授课记录日期晚于今天、**督导评分五维未录全**） | 就地提示具体缺失维度；**注意 HTTP 是 400，不是 500** |
| 40301 | 403 | 角色无权访问该功能 | 提示并回退 |
| 40302 | 403 | 无权操作该数据（越权访问他室/他人数据） | 提示并回退；不得当作"数据为空" |
| 40901 | 409 | 数据已存在（同课程同班级同日同节次重复建课） | 提示重复并引导查看已有记录 |
| 40902 | 409 | 重复提交 | **保留码：`PUT` 幂等覆盖，后端不返回该码** |
| 50002 | 501 | 功能未实现（阶段二的智能体接口在阶段一返回） | 展示"功能开发中"占位，不弹全局错误 |
| 50003 | 503 | 依赖服务不可用（ASR 引擎未配置或转写失败） | 展示失败态 + 重试按钮 |

### 10.7 请求层约定

- 统一响应 `{ code, message, data }`：`code === 0` 成功；**HTTP 401**（后端返回 `40101`）清 token 跳登录；其余弹 `ElMessage.error(message)`；
- 鉴权头：`Authorization: Bearer <token>`，token 存 `localStorage`（key：`aijiaoxue_token`，用户信息 key：`aijiaoxue_user`）；
- 聚合接口**必须透传 `semester`**（开发计划 §2.5.6：跨学期平均会抹平改进），前端默认当前学期；
- Mock 策略（预留）：`.env.development` 中 `VITE_USE_MOCK=true` 时由 `src/mocks/` 拦截，**接口函数签名与真实后端完全一致**，保证无缝切换联调。
  > 当前仓库**尚未创建 `src/mocks/`**，前端直连真实后端；如需启用 mock，请新建该目录并保持本约定。

## 11. 状态管理与数据流

```
views ──调用──> api/*（唯一请求出口）
  │                 │
  ├── composables/useCourseList（列表查询状态内聚：rows/loading/query/page）
  ├── composables/useTranscriptionPolling（阶段②：转写状态轮询）
  └── stores/auth（token/user/hasRole）  stores/dict（学期、教研室字典）
```

- `stores/auth.ts`：`login()` / `logout()` / `hasRole(role)` / `state: { token, user }`，持久化 token；
- 列表类数据**不进 Pinia**，用 composable 内聚（分页、筛选、加载态），避免全局状态膨胀——**评分列表、聚合结果同理进 composable，不进 store**；
- 字典（学期、教研室）启动时拉取一次缓存于 `stores/dict.ts`；
- 轮询类副作用（转写）必须封装在 composable 内并在 `onUnmounted` 清理定时器；
- 组件间通信：父子 props/emits 为主，跨层级少量场景用 `provide/inject`，**禁止滥用全局事件总线**。

## 12. 编码规范（强制）

1. **Composition API only**：全部 `<script setup lang="ts">`，禁止 Options API；
2. **命名**：组件多词 PascalCase（文件名与组件名一致）；页面组件以 `View` 结尾；composable 以 `use` 开头；类型以大驼峰；常量全大写下划线；
3. **禁止 `any`**：类型不明确时用 `unknown` + 收窄，或补充类型定义；
4. **禁止硬编码**：颜色/字号/间距用 CSS 变量；魔法字符串用 `const` 枚举（角色、状态、评分维度、flags）；
5. **禁止在组件内直接调用 axios**：请求只出现在 `src/api/`；
6. **单向数据流**：子组件不修改 props，通过 emit 通知父级；
7. **样式**：`<style scoped lang="scss">`；公共样式进 `styles/`；类名用 BEM（`course-table__row--active`）；
8. **路由**：`name` 全局唯一；`meta.title` 必填（面包屑依赖）；`meta.roles` 仅用于页面级角色限制；
9. **评分口径前端零计算**：综合分、维度分、样本量、flags **一律使用后端返回值**；前端只允许做展示格式化（如 `null → '—'`），不得自行加权、平均或四舍五入到不同精度；
10. **提交前自检**：`npm run build` 零错误、`npm run lint` 零 error；`vue-tsc` 零错误；新增工具函数附带 Vitest 单测；
11. **Git 提交**：Conventional Commits —— `feat(course): 新增课程列表筛选` / `fix(login): 修复 token 失效未跳转`；分支命名 `feature/S6.3-session-evaluation`（故事编号 + 短描述）。

## 13. 文档同步要求（强制）

**本手册与开发计划是前端的事实源。任何前端变更，只要触及下列任一项，必须在同一个 PR 内更新本文件；契约先行——先改文档，再改代码。**

必须在同一 PR 同步本文件的情形：

1. **路由**：新增/删除/改名路由、改变 `name`、改变 `meta.roles` 或页面权限 → 更新 §6.1 路由总表、§6.2 路由变更表（如涉及决策变更还需在 §13 说明理由）；
2. **接口与类型字段**：新增或修改接口路径/查询参数/响应字段、DTO 增删字段、错误码或 HTTP 映射变化 → 更新 §10；
3. **权限**：页面权限矩阵或页面内操作权限的变化（含"教师可见范围"这类敏感边界）→ 更新 §3；
4. **设计令牌与美术规范**：新增/修改 `tokens.scss` 令牌、评分展示规范 → 更新 §9；
5. **页面规格**：页面结构、区块、交互流程、三态要求的实质性变化 → 更新 §7；组件 Props/职责变化 → 更新 §8；
6. **阶段范围**：某功能从"待开发"进入"进行中/已交付"，或阶段归属调整 → 更新顶部阶段横幅与 §2。

**一致性要求**：

- 本手册**必须与 [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) 保持一致**；两者冲突时以开发计划为准，并**立即修正本文件**；
- 接口字段与路径以开发计划 §4 + 后端 `internal/dto/` 为唯一事实源，不得凭记忆书写；
- 违反同步要求的 PR 视为未完成（见 §14 通用 DoD）。

## 14. 验收标准（DoD）

### 14.1 通用 DoD（所有故事适用）

功能可用 + 三态完整 + 权限正确 + 构建通过；`vue-tsc` 零错误；`npm run build` 通过；涉及本手册 §13 所列内容的改动**已同步文档**。

### 14.2 Sprint 1

| 故事 | 前端验收要点 |
|------|-------------|
| S1.1 | `npm run dev` 启动无报错；全部路由可导航；mock/real 接口可切换；构建产物无错 |
| S1.2 | 三角色账号分别登录成功并进入各自工作台；菜单按角色渲染；token 失效自动跳回登录页 |
| S2.1 | 主任登录 → 课程列表默认为本室课程，含教师/班级数/学生人次列，可按教师筛选 |
| S2.2 | 教师登录 → 课程列表仅显示本人课程，含开课班级与时间 |
| S2.3 | 督导登录 → 课程列表显示全校课程，可按教研室/学期筛选 |
| S3.1 | 主任可新增课程（校验生效）；可从列表/详情进入编辑，保存后数据刷新 |
| S4.1 | 教师在课程详情可见资源列表与各班学生人次、汇总人次 |
| S4.2 | 教师可上传资源（类型/大小校验 + 进度），可删除（二次确认），列表实时更新 |
| S5.1 | 督导在工作台/督导总览可见覆盖率（总体 + 分教研室）与听评课安排列表 |

### 14.3 Sprint 2 阶段①（评价闭环）

| 任务/故事 | 前端验收要点 |
|-----------|-------------|
| T1.9 / S6.x | `types/` + `api/` 覆盖 §10.4 全部接口；`vue-tsc` 零错误 |
| T1.10 / S6.2 | 课程详情「历史授课记录」三态完整；分页可用；无评分侧显示 `—` |
| T1.11 / S6.3 | 5 维 1–5 分必填校验生效；锚点 tooltip 齐全；结构化评语可提交；提交后列表即时反映；重复提交为覆盖且有确认 |
| T1.12 / S6.4 | 教师管理页展示评价次数 n；`n<3` 有「样本不足」标记；**默认按姓名排序**；显示「不作为考核依据」 |
| T1.13 / S6.5 | 教师面板含综合分 + 5 维 + 按课程明细 + 历次时间线；`flags` 可见 |
| T1.14 / S6.6 | 教学提优页保留课程基本信息与资源上传；只读展示督导分与评语 |
| T1.15 / S6.7 | `/supervision` 完整分页列表可用，且不再重复统计卡/覆盖率卡；主任工作台改为两个入口卡 |
| 一致性 | 同一教师，主任端与教师端综合分、各维度分**逐位相同** |
| 空数据 | 无评价教师综合分显示「暂无评价」并置底，**不得显示 0** |

### 14.4 Sprint 2 阶段②（智能体接入）

| 任务/故事 | 前端验收要点 |
|-----------|-------------|
| T2.7 / S7.1 | 音频播放器 + 转写查看器三态（pending/running/done/failed）可用；失败可重试；轮询在离开页面后停止 |
| T2.8 / S7.2 | 「智能体参考」面板标注「AI 参考」；低置信度维度有提示；`objective` 为 `null` 时显示空而非 0 |
| T2.9 / S7.3 | 综合分随智能体接入更新；`disjoint` 场景有显式提示，不静默 |
| 隐私 | 教师端看不到音频入口；转写文本已脱敏 |

### 14.5 Sprint 3（帮教师）

| 任务/故事 | 前端验收要点 |
|-----------|-------------|
| T3.1 / S8.1 | 教师端可见智能体提优建议，且与课堂记录对应、按时间倒序 |
| T3.2 / S8.3 | 趋势折线可按维度切换；种子数据呈上升趋势 |
| T3.3 / S8.2 | SSE 流式输出；中断可恢复；失败不阻塞页面 |
| T3.4 / S8.4 | 教师可提交申诉、督导可复核，状态可追溯 |

---

## 更新记录

| 版本 | 日期 | 变更 |
|------|------|------|
| v1.0 | Sprint 1 | 初版：技术栈、目录结构、设计令牌、编码规范、Sprint 1 页面规格与验收 |
| v2.0 | 2026-09 | 覆盖三个 Sprint：新增当前阶段横幅与三 Sprint 对等章节；补全 Sprint 2/3 页面、路由变更表、组件与目录（§5–§8）；接口层修正（`/teacher-scores` 与 `/teachers` 分离、`GET /courses/:id/sessions`、`/sessions/...`、两个 `evaluation-summary`、`evaluatedCount`、`40002` = HTTP 400）；新增评分展示规范（§9.6）；新增「文档同步要求（强制）」（§13）；补充阶段①②③验收标准（§14） |

> **版本维护约定**：本手册的版本号随任一强制同步项（§13）的变更递增，并在上表登记。修改本文件时，请一并核对 [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) 是否需同步更新。
