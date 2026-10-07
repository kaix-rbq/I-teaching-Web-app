# 「爱教学」前端编码智能体指导手册

> **文档用途**：本文件是写给 AI 智能体（编码助手、架构助手，运行于 TRAE / OpenCode）的前端编码工作宪法。
> 智能体在为本项目编写任何前端代码之前，**必须先完整阅读本文件**，并严格遵守其中的技术栈、目录结构、组件设计、美术规范与编码规则。
> **使用方式**：本文件位于 `docs/frontend_AGENTS.md`；前端仓库 `aijiaoxue-web` 的编码助手在动手前必须先完整阅读本文件。文档全集与阅读顺序见 [`../CONTRIBUTING.md`](../CONTRIBUTING.md) §1。

---

> ## 🚦 当前阶段：Sprint 2（阶段① 已交付 / 阶段② 前端已接入真实接口）
>
> | 阶段 | 主题 | 状态 |
> |------|------|------|
> | **阶段① （Sprint 2.1）** | 评价闭环：授课记录 · 督导评分 · 结构化评语 · 三级聚合 · 主任教师画像 · 教师质量档案/提优页 | **已交付** —— 路由、页面、组件与接口层均已落地（`frontend/src/router/index.ts`、`frontend/src/views/`、`frontend/src/api/session.ts`、`frontend/src/api/teacher.ts`） |
> | **阶段② （Sprint 2.2）** | 智能体接入：课堂录音 · 异步转写 · 综合分融合 · 督导页「智能体参考」面板 | **前端已接入真实接口** —— 录音上传/播放（`components/session/AudioPlayer.vue`）、转写三态（`components/evaluation/TranscriptViewer.vue` + `GET /sessions/:id/transcript`）、`agentScore` 双源参考（`EvaluationCompare`）均可用；**真实 AI 评分推理管线未实现**（agent 评价行当前由后端 seed 提供） |
> | **阶段③ （Sprint 3）** | 帮教师：智能体提优建议 · 对话（SSE） · 趋势视图 · 申诉复核 · 质量报告导出 | **仅前端演示** —— 提优建议/趋势/对话由 `frontend/src/mocks/teacherImprove.ts` 驱动（`frontend/src/api/agent.ts` 不发网络请求）；申诉复核、报告导出未实现 |
>
> **本手册覆盖全部三个 Sprint，是前端「技术栈 / 目录 / 路由 / 页面规格 / 组件 / 设计令牌 / 编码规范 / DoD」的唯一事实源。** Sprint 1 的基础平台已交付，Sprint 2/3 的功能与设计事实源是
> **[`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md)**（下称「开发计划」）。
> **接口契约以后端 [`backend_AGENTS.md`](./backend_AGENTS.md) §8 为准**；文档冲突时以开发计划与后端契约文档为准（契约先行：先改文档，再改代码）。

---

> ## 🆕 v1.2 角色信息架构收敛与草稿能力（2026-09-28）
>
> 本次更新对既有页面做了**破坏性重排**（以可读性/职责单一为准，规范从宽）。要点：
>
> 1. **登录页 `LoginView`**：品牌雷达五轴标注维度名称；移除雷达下方「教研室主任·director / 教师·teacher / 教学督导·supervisor」角色图例；登录后默认落点由 `defaultRouteName(role)` 决定（教师 → `/me/quality`，其余 → `/dashboard`）。
> 2. **教师端**：**删除「质量驾驶舱」**（与质量档案重复），`/me/quality` 为登录后首页；侧边栏为「我的质量档案 / 我的课程 / 个人中心」。质量档案第三卡由「本学期样本」改为 **「提优分析」**（可提优方向 + 提优建议，AI 预留接口）；删除「全部评价流」「AI 提优建议摘要」；新增 **「授课快照」**（逐次授课卡片，色条对应质量水平，高分标亮点、低分标不足）。
> 3. **提优页 `CourseImproveView`**：重排为「课程信息置顶 → 五维双源横向柱条（增长动画）+ 综合雷达 → 督导评语流 → 智能体提优建议 → 分数趋势 → 课程资源置底」，删除与质量档案重复的罗盘/样本/教师级五维区块。
> 4. **督导端**：「质量驾驶舱」改 **「工作台」**：删除四个统计卡与覆盖率图表；保留 **待评课队列**（今日/本周/本月/**未评**按钮切换）；**评估入口按日期闸门**——`plannedDate <= 今天` 且未评估显示「去评估」（无授课记录先建档），未来课次仅预览不开放评估；「未评」列出已过日期未评课的记录便于补录；工作台右上角新增 **新增授课记录**（替代教务录入），并新增 **待评估授课记录** 区块；新增 **草稿箱**（最近三次 + 更多>>）与 `/drafts` 页面（按课程名查询、编辑/提交/删除）；评估页新增 **保存草稿**。侧边栏精简为「工作台 / 草稿箱 / 个人中心」，删除「全校课程库」「课堂评估」等入口。
> 5. **契约同步**：新增接口 `GET/PUT /sessions/:id/draft`、`GET /drafts`、`DELETE /drafts/:id`、`POST /drafts/:id/submit`；`GET /dashboard` 督导 DTO 改为 `recentPlans + recentDrafts`；`PlanItem` 新增 `sessionId`；seed 扩充为多教师/多课程/多记录且 AI 评分覆盖五维。
>
> 详细页面规格见本文件 §7，设计令牌与视觉规范见 §9；接口契约见 [`backend_AGENTS.md`](./backend_AGENTS.md) §8。

---

> ## 🆕 v1.3 督导课程复盘与教师授课记录（2026-10-06）
>
> 本次更新聚焦「评估完成后去哪看、教师如何复盘」：
>
> 1. **督导端新增「课程列表」**：侧边栏恢复「课程列表」（`course-list`），列表请求带 `mine=1`，后端按 `supervision_plans.supervisor_id` 只返回**本人负责评估的课程**；行点击进入新增的 **`SupervisorCourseView`（`/supervision/courses/:id`）** ——课程当前综合评分 + 历史授课记录「去评估 / 查看·修改评估」+ 逐次课录音上传。
> 2. **待评课队列处理逻辑收敛**：已评估（`evaluated`）的听评课安排**一律从工作台待评课队列清除**（今日/本周/本月/未评四个分档统一过滤），过去未评估课次入口保持开放（「未评」分档 + 日期闸门）；评估提交后授课记录 `status` 置为 `evaluated`，从「待评估授课记录」移除，复盘走课程列表。
> 3. **转写卡片长度控制**：`TranscriptViewer` 完成态默认只预览部分内容（前 3 条 / 前 220 字），超出显示「展开全文」；点击打开**居中悬浮弹窗**（正文可滚动）查看完整转写，解决督导页内容过长的问题。
> 4. **教师「授课快照」跳转修正**：原「查看课程提优」与「去提优」同跳 `/courses/:id/improve`；现改为 **「查看详细记录」→ `/sessions/:id/evaluation`**，展示该次课的督导评分（只读）与 AI 智能体评价，并提供当堂课脱敏转写入口。「我的课程评分明细」栏保留课程列表与「去提优」按钮。
> 5. **契约同步**：`GET /courses` 新增 `mine` 查询参数（仅 supervisor 生效，其他角色忽略）；新增路由 `/supervision/courses/:id`（name `supervisor-course`，权限 supervisor）。**音频隐私不变**：教师端仍不可回放录音，只提供转写文本。
>
> 详细页面规格见 §7.5A、§7.10；接口契约见 [`backend_AGENTS.md`](./backend_AGENTS.md) §8.6 / §8.8。

---

## 1. 项目背景

「爱教学」是面向高校（参考东北大学教学管理场景）的**教学质量全链路数字化管理平台**，产品愿景为"让教学质量持续可测"，走三阶进化路线：

| 进阶 | 主题 | 说明 | 迭代 |
|------|------|------|------|
| 进阶 1 | **查课程** | 全量课程信息透明可查 | Sprint 1（已交付） |
| 进阶 2 | **看课堂** | AI 听评课与课堂质量评估（面向督导，兼顾主任/教师查看） | Sprint 2（阶段① 已交付；阶段② 录音/转写/双源参考已接入，真实 AI 推理管线未实现） |
| 进阶 3 | **帮教师** | 教学改进建议、趋势与对话（面向教师） | Sprint 3（AI 建议/趋势/对话为前端演示数据，真实接口未实现） |

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

### 2.2 Sprint 2（课堂评价闭环 + 智能体接入）—— 阶段①已交付 / 阶段②前端已接入

**目标**：把「课程」下沉到「一次课」，建立**督导评分 → 场次级 → 课程级 → 教师级**的评价数据链；再叠加智能体评分，形成综合分与提优视图。
**它同时服务两个用户**：督导（录入评分）与主任/教师（查看评分）——**主任端与教师端的数字必须逐位一致**，由后端单一聚合函数保证，前端不得各自造口径。

> **实现状态（以代码为准）**：阶段① 全部页面与接口层已落地；阶段② 的课堂录音上传/播放与异步转写三态、`agentScore` 双源参考面板均已接真实接口（`frontend/src/views/SessionEvaluationView.vue:177-206,234`；`frontend/src/api/session.ts:58-69`；`backend/internal/router/router.go:84,102-103,120`）。
> 仍为**演示数据**的只有阶段③的提优建议 / 趋势 / 对话（`frontend/src/api/agent.ts` → `frontend/src/mocks/teacherImprove.ts`）；**真实 AI 评分推理管线未实现**（`GET /sessions/:id/evaluation` 的 `agentScore` 来自 `evaluations` 表 `evaluator_type='agent'` 行，当前由 seed 提供，见 `backend/database/seed.sql:149-167`）。

**用户故事**：

| 编号 | 阶段 | 角色 | 故事 | 优先级 |
|------|------|------|------|--------|
| S6.1 | ① | 督导 | 为一次课建立授课记录（课程/班级/日期/节次/主题），以便评价有落点 | M |
| S6.2 | ① | 任一 | 在课程详情页查看该课程的历史授课记录列表 | M |
| S6.3 | ① | 督导 | 在当堂课评估页按 5 个维度打分并留下结构化评语 | M |
| S6.4 | ① | 主任 | 在教师画像页查看本室教师综合评分列表 | M |
| S6.5 | ① | 主任 | 点击教师查看其评分数据面板（综合分 + 分维度 + 分课程） | M |
| S6.6 | ① | 教师 | 在教学提优页查看本人各维度评分与督导评语 | M |
| S6.7 | ① | 督导 | 工作台与「听评课管理」页分离，完整安排列表可分页筛选 | S |
| S7.1 | ② | 督导 | 上传课堂录音并查看转写文本 | M |
| S7.2 | ② | 督导 | 查看智能体对该堂课的分维度评分作为参考 | M |
| S7.3 | ② | 任一 | 综合分 = 督导评分与智能体评分的加权融合 | M |
| S7.4 | ② | 教师 | 在教学提优页看到综合分随智能体接入而变化 | S |

**前端交付物（阶段①，已交付）**：

- `types/` + `api/` 新增授课记录 / 评价 / 教师评分接口层（`session.ts`、`teacher.ts`）—— T1.9；
- `CourseDetailView` 新增「**历史授课记录**」列表（`SessionTable`），行点击进评估页 —— T1.10；
- `SessionEvaluationView` 当堂课质量评估页：5 维 1–5 分评分表单 + **锚点 tooltip** + 结构化评语（亮点/待改进/建议）+ **智能体参考预留位** —— T1.11；
- `TeacherListView` 教师画像页：本室/全校教师综合分、5 维分、评价次数 n，默认按姓名排序，`n < 3` 标注「样本不足」 —— T1.12；
- `TeacherDetailView` 教师评分面板：综合分 + 分维度条 + 按课程明细 + 历次评价时间线 —— T1.13；
- `CourseImproveView` 教学提优页：课程基本信息 + 资源入口 + 只读督导分与评语 —— T1.14；
- `/supervision` 改造为「听评课管理」完整分页列表；主任工作台改为「本室质量热力 + 重点关注」 —— T1.15。

**前端交付物（阶段②，前端已接入真实接口）**：

- `AudioPlayer` 课堂录音播放（含上传入口）—— 已实现（`components/session/AudioPlayer.vue`；上传 `POST /sessions/:id/recording`）；
- `TranscriptViewer` 转写查看器：**异步任务三态**（`pending/running → done → failed`）+ 状态轮询 + 失败重试 —— 已实现（轮询在 `SessionEvaluationView` 内联 `setTimeout`，见 §7.10）；
- `EvaluationCompare` 「智能体参考」对比面板：标注「AI 参考」、展示低置信度维度与 `evidence` 转写引用 —— 已接入 `GET /sessions/:id/evaluation` 的 `agentScore`（后端 seed 提供数据，无真实 AI 推理管线）；
- 综合分在现有页面上自动生效（前端不改口径，仅展示 `flags` 与 `sample`）—— 已实现。

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
| 工作台 | `/dashboard` | ✓ | ✗（重定向 `/me/quality`） | ✓ | S1 |
| 课程列表 | `/courses` | ✓ 本室数据 | ✓ 本人数据 | ✓ 全校数据 | S1 |
| 课程详情 | `/courses/:id` | ✓ | ✓ | ✓ | S1（S2① 追加历史授课记录） |
| 新增课程 | `/courses/new` | ✓ | ✗ | ✗ | S1 |
| 编辑课程 | `/courses/:id/edit` | ✓ | ✗ | ✗ | S1 |
| 教学提优 | `/courses/:id/improve` | ✗ | ✓ 仅本人课程 | ✗ | S2①（S3③ 增强） |
| 教师画像 | `/teachers` | ✓ 本室 | ✗ | ✓ 全校 | S2① |
| 教师画像详情 | `/teachers/:id` | ✓ 本室 | ✗ | ✓ 全校 | S2① |
| 当堂课质量评估 | `/sessions/:id/evaluation` | 只读 本室 | 只读 本人 | ✓ 唯一可写 | S2①（S2② 增强） |
| 课程综合评分（督导复盘） | `/supervision/courses/:id` | ✗ | ✗ | ✓ 仅本人负责评估的课程 | v1.3 |
| 我的质量档案 | `/me/quality` | ✗ | ✓ 仅本人 | ✗ | v1.2（教师登录首页） |
| 草稿箱 | `/drafts` | ✗ | ✗ | ✓ 仅本人 | v1.2 |
| 听评课管理 | `/supervision` | ✗ | ✗ | ✓ | S1（S2① 改造） |
| 个人中心 | `/profile` | ✓ | ✓ | ✓ | S1 |

> - **路由级角色限制**（`meta.roles`）只在 `/courses/new`、`/courses/:id/edit`（director）、`/courses/:id/improve`（teacher）、`/teachers`、`/teachers/:id`（director/supervisor）、`/supervision`（supervisor）、`/supervision/courses/:id`（supervisor）、`/me/quality`（teacher）、`/drafts`（supervisor）上声明（`frontend/src/router/index.ts`）。
> - `/dashboard`、`/courses`、`/courses/:id`、`/sessions/:id/evaluation`、`/profile` **无 `roles` 限制**（登录即可进）；`/sessions/:id/evaluation` 的「可写/只读」在组件内按角色判定（`frontend/src/views/SessionEvaluationView.vue:42-43`）。
> - 教师访问 `/dashboard` 会被全局守卫重定向到 `/me/quality`（`frontend/src/router/guards.ts:22-24`），教师端**没有**质量驾驶舱。

### 3.2 页面内操作权限

| 操作 | 位置 | director | teacher | supervisor |
|------|------|:---:|:---:|:---:|
| 新增/编辑课程信息 | 列表页 + 详情页入口 | ✓ | ✗ | ✗ |
| 上传课程资源 | 课程详情·资源 Tab | ✗ | ✓（仅本人课程） | ✗ |
| 删除课程资源 | 课程详情·资源 Tab | ✗ | ✓（仅本人课程） | ✗ |
| 下载课程资源 | 课程详情·资源 Tab | ✓ | ✓ | ✓ |
| 按教师/教研室筛选 | 课程列表筛选栏 | ✓ | ✗（自动限定本人） | ✓ |
| 创建授课记录 | 工作台「新增授课记录」/ 待评课队列建档 | ✗ | ✗ | ✓ |
| 提交/覆盖督导评分 | 当堂课评估页 | ✗ | ✗ | ✓ |
| 查看督导评语 | 教师画像 / 质量档案 / 提优页 | 本室 | **仅自己** | ✓ |
| 上传课堂录音 / 触发转写 | 当堂课评估页 · 督导课程综合评分页（阶段②，已实现） | ✗ | ✗ | ✓ |
| 播放课堂音频 | 当堂课评估页 / 督导课程综合评分页（阶段②，已实现） | ✗ | **✗（不可听）** | ✓ |
| 查看转写文本 | 当堂课评估页 / 教师质量档案→授课记录（阶段②，已实现） | ✗ | 本人课程（已脱敏，用于改进） | ✓ |

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
| 框架 | Vue 3 | `^3.4.21` | 一律 Composition API + `<script setup>` |
| 语言 | TypeScript | `^5.4.3` | 严格模式，禁止 `.js` 业务代码 |
| 构建 | Vite | `^5.2.6` | — |
| 路由 | Vue Router | `^4.3.0` | `createWebHistory` 模式 |
| 状态 | Pinia | `^2.1.7` | 仅 auth / dict 两个全局 store |
| UI 库 | Element Plus | `^2.7.0` | 按需自动引入；**唯一 UI 库** |
| 图标 | `@element-plus/icons-vue` | `^2.3.1` | 侧边栏 / 卡片图标 |
| 请求 | Axios | `^1.6.8` | 统一实例封装于 `src/api/http.ts` |
| 日期 | Day.js | `^1.11.10` | — |
| 类型检查 | vue-tsc | `^2.0.6` | `npm run build` / `typecheck` 前置 |
| 样式 | Sass | `^1.72.0` | `<style scoped lang="scss">` |
| Lint | ESLint | `^8.57.0` | 配置 `.eslintrc.cjs` |
| SSE | 原生 `EventSource` / `fetch` 流 | — | 阶段三智能体对话；**不引入第二套 HTTP 库** |
| Mock | 前端本地演示数据（无开关） | — | `src/mocks/teacherImprove.ts` 仅被 `src/api/agent.ts` import，为阶段③ AI 区块提供演示数据；**其余接口全部直连后端** |
| 测试 | Vitest | `^1.5.0` | 编码助手产出的工具函数（含评分换算/排序）需附带测试 |
| 按需引入 | `unplugin-auto-import` | `^0.17.5` | Element Plus API 自动引入 |
| 按需引入 | `unplugin-vue-components` | `^0.26.0` | Element Plus 组件自动注册 |

> **不引入图表库**：`frontend/package.json` 中**没有 ECharts**，代码中也没有任何 echarts import。覆盖率环、五维雷达、质量热力、分数趋势均由自绘 SVG / CSS（`components/evaluation/`）实现。
> 版本以 `frontend/package.json` 为准，不得擅自升级或替换；辅助插件即上表最后两行。

## 5. 目录结构（强制）

> 本结构为**当前真实目录**（以 `ls frontend/src` 为准）。新增文件必须落在既有分层内，禁止自创目录。
> `auto-imports.d.ts` / `components.d.ts` / `vite-env.d.ts` 为工具自动生成，不手工维护，故不列出。

```
frontend/                       # 包名 aijiaoxue-web
├── index.html
├── package.json
├── vite.config.ts
├── tsconfig.json / tsconfig.node.json
├── .eslintrc.cjs
├── .env.development            # VITE_API_BASE_URL / VITE_PROXY_TARGET（无 mock 开关）
├── .env.production
├── public/
└── src/
    ├── main.ts
    ├── App.vue
    ├── api/                  # 接口层：唯一的 axios 调用发生地
    │   ├── http.ts           # 实例、拦截器（token 注入、统一错误处理）
    │   ├── auth.ts           # login / logout / me
    │   ├── course.ts         # 课程 CRUD、列表查询
    │   ├── dashboard.ts      # 工作台聚合（把三角色 DTO 收敛为同一形状）
    │   ├── dict.ts           # 学期/教研室/教师字典
    │   ├── resource.ts       # 资源列表、上传、删除、下载
    │   ├── supervision.ts    # 覆盖率、听评课安排
    │   ├── session.ts        # 授课记录、单场次评估、录音上传、转写
    │   ├── teacher.ts        # 教师评分列表与面板、课程级评分
    │   ├── draft.ts          # 评估草稿（列表/读取/保存/提交/删除）
    │   └── agent.ts          # 阶段③：提优建议 / 趋势 / 对话（当前返回 mocks 演示数据）
    ├── assets/
    ├── components/
    │   ├── common/           # 通用组件（与业务解耦）
    │   │   ├── PageHeader.vue
    │   │   ├── StatCard.vue
    │   │   ├── FilterBar.vue
    │   │   ├── EmptyState.vue
    │   │   ├── RoleTag.vue
    │   │   ├── AiBadge.vue        # AI 徽标（四芒星 + 渐变紫）
    │   │   └── QualityBadge.vue   # 质量等级徽章
    │   ├── course/
    │   │   ├── CourseTable.vue
    │   │   ├── CourseCard.vue
    │   │   ├── CourseInfoForm.vue
    │   │   ├── ResourceList.vue
    │   │   └── ResourceUploader.vue
    │   ├── supervision/
    │   │   ├── CoverageCard.vue
    │   │   ├── ScheduleTable.vue
    │   │   └── EvaluationQueue.vue     # 督导待评课队列
    │   ├── evaluation/
    │   │   ├── ScoreRadar.vue           # 五维雷达/条形（支持督导/AI 双源）
    │   │   ├── ScoreDimensionsCard.vue  # 五维评分卡（雷达 + 维度条）
    │   │   ├── ScoreHeatmap.vue         # 教师 × 五维质量热力
    │   │   ├── QualityCompass.vue       # 双源罗盘（综合分签名图形）
    │   │   ├── AnchorScale.vue          # 锚点刻度条（评分表单控件）
    │   │   ├── EvaluationForm.vue       # 督导评分表单
    │   │   ├── EvaluationCompare.vue    # 督导 vs 智能体对比
    │   │   ├── EvaluationTimeline.vue   # 竖线评价流
    │   │   ├── CommentPanel.vue         # 结构化评语展示
    │   │   ├── TranscriptViewer.vue     # 转写文本（三态）
    │   │   ├── ScoreTrendChart.vue      # 阶段③：趋势
    │   │   ├── AgentSuggestionList.vue  # 阶段③：提优建议
    │   │   └── AgentChat.vue            # 阶段③：流式对话
    │   ├── session/
    │   │   ├── SessionTable.vue         # 历史授课记录列表
    │   │   └── AudioPlayer.vue          # 课堂录音播放
    │   └── teacher/
    │       ├── TeacherScoreTable.vue
    │       ├── TeacherScoreCard.vue
    │       └── TeacherScorePanel.vue
    ├── composables/          # 组合式函数，均以 use 开头
    │   ├── useAuth.ts
    │   ├── useCourseList.ts
    │   ├── usePagination.ts
    │   ├── useSemester.ts
    │   └── useAgentChat.ts
    ├── constants/
    │   ├── index.ts          # 角色/状态枚举、评分维度与锚点、上传白名单
    │   └── index.spec.ts
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
    │   ├── evaluation.ts     # 评分/授课记录/聚合/录音转写类型
    │   ├── draft.ts          # 评估草稿类型
    │   ├── teacher.ts        # 教师评分与趋势类型
    │   ├── agent.ts          # 阶段③智能体类型
    │   ├── router.d.ts       # RouteMeta 增强
    │   └── api.d.ts          # ApiResponse<T> / PageResult<T>
    ├── utils/
    │   ├── format.ts         # 分数色阶映射 scoreTone / scoreToneColor 等
    │   └── format.spec.ts
    ├── views/                # 页面组件，一律以 View 结尾
    │   ├── LoginView.vue
    │   ├── DashboardView.vue
    │   ├── CourseListView.vue
    │   ├── CourseDetailView.vue
    │   ├── CourseFormView.vue          # 新增/编辑复用
    │   ├── SupervisionView.vue         # 听评课管理（后备页）
    │   ├── SupervisorCourseView.vue    # /supervision/courses/:id
    │   ├── TeacherListView.vue         # 教师画像
    │   ├── TeacherDetailView.vue       # 教师画像详情
    │   ├── SessionEvaluationView.vue   # 当堂课质量评估
    │   ├── CourseImproveView.vue       # 教学提优
    │   ├── ProfileQualityView.vue      # 我的质量档案（/me/quality）
    │   ├── DraftBoxView.vue            # 草稿箱（/drafts）
    │   ├── ProfileView.vue
    │   └── error/
    │       ├── ForbiddenView.vue       # /403
    │       └── NotFoundView.vue
    └── mocks/
        └── teacherImprove.ts  # 仅阶段③ AI 区块的演示数据（被 api/agent.ts import）
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
| `/sessions/:id/evaluation` | session-evaluation | SessionEvaluationView | App | 登录（督导可写，他人只读；无 `meta.roles`） | S2① 新增（v1.2 增「保存草稿」；v1.3 教师由授课快照进入，增智能体文字评价与转写入口） |
| `/supervision/courses/:id` | supervisor-course | SupervisorCourseView | App | supervisor | v1.3 新增（督导课程综合评分 + 授课记录评估/录音上传） |
| `/me/quality` | profile-quality | ProfileQualityView | App | teacher | v1.0 新增（v1.2 为教师首页） |
| `/drafts` | draft-box | DraftBoxView | App | supervisor | v1.2 新增（督导草稿箱） |
| `/supervision` | supervision | SupervisionView | App | supervisor | S1（v1.2 起从督导菜单摘除，保留后备） |
| `/profile` | profile | ProfileView | App | 登录 | S1 |
| `/403` | forbidden | ForbiddenView | Blank | 公开 | S1 |
| `/:pathMatch(.*)*` | not-found | NotFoundView | Blank | 公开 | S1 |

> 上表与 `frontend/src/router/index.ts` 一一对应（共 17 个命名路由 + `/` 重定向）。`meta.title` 实际值：`teacher-list` 为「教师画像」、`dashboard` 为「工作台」、`supervision` 为「听评课管理」（面包屑依赖）。
> **登录落点不是固定的 `/dashboard`**：`defaultRouteName(role)` 规定教师 → `profile-quality`（`/me/quality`），其余角色 → `dashboard`（`frontend/src/constants/index.ts:31-34`，用于 `frontend/src/router/guards.ts:18`）。

### 6.2 路由变更表（对照开发计划 §6.1）

| 路由 | name | 现状 | 目标 | 权限 | 变更 |
|------|------|------|------|------|------|
| `/dashboard` | dashboard | 三角色 | 按角色重构（见 §6.3） | 登录 | 改 |
| `/courses` | course-list | — | 不变 | 登录 | — |
| `/courses/new` | course-new | — | 不变 | director | — |
| `/courses/:id` | course-detail | — | **+ 历史授课记录列表** | 登录 | 增强 |
| `/courses/:id/edit` | course-edit | — | 不变 | director | — |
| `/courses/:id/improve` | course-improve | — | **新增** 教学提优 | teacher | 新增 |
| `/teachers` | teacher-list | — | **新增** 教师画像（页面标题「教师画像」） | director/supervisor | 新增 |
| `/teachers/:id` | teacher-detail | — | **新增** 教师画像详情 | director/supervisor | 新增 |
| `/sessions/:id/evaluation` | session-evaluation | — | **新增** 当堂课质量评估页 | 登录（督导可写，他人只读） | 新增 |
| `/supervision` | supervision | 督导总览 | **改造为「听评课管理」** 完整分页列表 | supervisor | 改造（不删路由） |
| `/profile` | profile | — | 不变 | 登录 | — |

> 本节是 Sprint 2.1 的**历史变更记录**；当前真实路由状态以 §6.1 为准。

**三条不可动摇的路由决策**：

1. **`/supervision` 不删除**：v1.2 起从督导侧边栏摘除，但作为听评课计划的**后备完整列表页**保留（承载完整分页与状态/日期筛选），保证历史计划仍可访问。
2. **主任工作台移除重复内容**：「本室教师开课情况」表格与「近期开课」列表与「课程库」重复，改为「**本室质量热力**（`ScoreHeatmap`）+ **重点关注**」＋快捷操作，统计卡只保留主任四项（`frontend/src/views/DashboardView.vue`）。
3. **「教学提优」采用并列路由** `/courses/:id/improve` 而非替换 `/courses/:id`：教师从「我的课程」点击时跳提优页，路径语义清晰，且教师仍可访问原详情页。

### 6.3 三条跳转链路

**① 主任**
```
/dashboard  质量驾驶舱：统计卡带（GET /dashboard）+ 本室质量热力 + 重点关注 + 快捷操作
   ├─▶ /courses（课程库）
   └─▶ /teachers  教师画像（GET /teacher-scores）｜综合分｜5维分｜评价次数 n
         └─▶ /teachers/:id  教师画像详情（TeacherScorePanel：双源罗盘 + 五维卡 + 样本与口径 + 课程明细 + 评价时间线）
               └─▶ /courses/:id ─▶ 历史授课记录 ─▶ /sessions/:id/evaluation（只读）
```

**② 督导**
```
/dashboard  工作台：待评课队列（今日/本周/本月/未评）＋ 草稿箱（最近 3 份）＋ 待评估授课记录 ＋ 新增授课记录
   ├─▶ /drafts  草稿箱（按课程名查询 · 编辑/提交/删除）
   ├─▶ /courses  课程列表（mine=1，仅本人负责评估的课程）
   │      └─▶ /supervision/courses/:id  课程综合评分（历史授课记录「去评估 / 查看·修改评估」+ 上传录音）
   └─▶ /sessions/:id/evaluation  当堂课质量评估页
         ├─ AudioPlayer 课堂录音（仅督导）
         ├─ TranscriptViewer 转写文本（三态轮询 + 失败重试）
         ├─ AnchorScale 五维锚点评分 + 结构化评语 + 保存草稿
         ├─ EvaluationCompare 智能体参考（标「AI 参考」）
         └─ 提交（PUT 幂等覆盖）
```

**③ 教师**
```
/me/quality  我的质量档案（登录后首页；人视角：综合分 + 五维 + 提优分析 + 课程明细 + 授课快照）
   ├─▶ /courses/:id/improve  教学提优（课程视角：课程信息置顶 + 五维双源柱条/雷达 + 评语流 + AI 建议/趋势 + 资源置底）
   └─▶ /sessions/:id/evaluation  授课快照「查看详细记录」：只读督导评分 + AI 评价 + 脱敏转写
```

### 6.4 守卫逻辑（`router/guards.ts`）

1. 无 token 且目标非公开页 → 重定向 `/login?redirect=…`；
2. 有 token 访问 `/login` → 重定向 `/dashboard`；
3. 目标页 `meta.roles` 存在且不含当前角色 → 重定向 `/403`；
4. 登录成功后按 `defaultRouteName(role)` 跳转：**教师 → `/me/quality`（`profile-quality`），其余角色 → `/dashboard`**（`frontend/src/constants/index.ts:31-34`、`frontend/src/router/guards.ts:17-19`）；
5. 教师访问 `/dashboard` 一律重定向到 `/me/quality`（`frontend/src/router/guards.ts:22-24`）——教师端没有质量驾驶舱。

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
- 登录成功：存 token + user 至 `stores/auth`，跳转 `redirect` 查询参数或 `defaultRouteName(role)`（教师 → `/me/quality`，其余 → `/dashboard`；`frontend/src/views/LoginView.vue:43-49`）；
- 提供**演示角色快捷入口**（三个角色按钮），仅开发环境渲染（`import.meta.env.DEV`），便于验收演示；
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

- 侧边栏菜单**按角色渲染**（实现：`frontend/src/layouts/AppLayout.vue:36-64`，单一 `menuItems` 计算属性 + 角色分支）：
  - 教师（`AppLayout.vue:40-45`）：**我的质量档案**（`profile-quality`）/ **我的课程**（`course-list`）/ 个人中心；
  - 督导（`AppLayout.vue:49-56`）：**工作台**（`dashboard`）/ **课程列表**（`course-list`）/ **草稿箱**（`draft-box`）/ 个人中心；
  - 主任（`AppLayout.vue:58-63`）：**质量驾驶舱**（`dashboard`）/ **课程库**（`course-list`）/ **教师画像**（`teacher-list`）/ 个人中心；
  - **禁止为每个角色写一份菜单模板**；
- 菜单项：40px 高，激活态浅蓝底 `--color-primary-bg` + 主色文字 + 左侧 3px 指示条；
- 顶栏右侧：用户姓名 + `RoleTag` + 下拉（个人中心 / 退出登录）；顶栏含全局学期选择器（`useSemester`）；
- 菜单高亮（`AppLayout.vue:66-86`）：`course-detail` / `course-new` / `course-edit` / `course-improve` / `supervisor-course` → 高亮 `course-list`；`teacher-detail` → 高亮 `teacher-list`；`session-evaluation` → 督导高亮 `dashboard`、其他角色高亮 `course-list`。

### 7.3 DashboardView 工作台（三角色差异化，S2① 重构）

页面结构：主任/教师视角 = 问候区 + 统计卡行 + 内容区（两栏 2:1）；督导视角 = 问候区 + 待评课队列/草稿箱两栏 + 整宽「待评估授课记录」。

| 区块 | director | teacher | supervisor |
|------|----------|---------|------------|
| 统计卡（StatCard） | 4 张：本室课程数 / 本室教师数 / 本学期开课班次 / 课程资源总数 | —（教师端不存在工作台，守卫重定向 `/me/quality`） | **无统计卡**（后端 `SupervisorDashboard` 已移除课程数/覆盖率等字段） |
| 主内容（左 2/3） | **本室质量热力** `ScoreHeatmap`（教师 × 五维矩阵，点行下钻教师画像） | — | **待评课队列** `EvaluationQueue`（今日/本周/本月/**未评**四档切换；日期闸门；右上角「课程列表」+「新增授课记录」） |
| 侧内容（右 1/3） | **重点关注**（由 `GET /teacher-scores` 单次响应派生：维度均分最低 + 样本不足/暂无评价人数）＋ **快捷操作**（新增课程 / 进入课程库 / 进入教师画像） | — | **草稿箱**（`recentDrafts` 最近 3 份 + 「更多>>」跳 `/drafts`） |
| 整宽区块 | — | — | **待评估授课记录**表（`pendingSessions`：日期/节次/课程/教师/主题/状态 + 「去评估」） |

> 数据源：`GET /api/v1/dashboard` 按角色返回三种 DTO，由 `frontend/src/api/dashboard.ts` 收敛为同一 `DashboardData` 形状；主任工作台额外拉 `GET /teacher-scores?pageSize=100` 派生质量热力与重点关注（`frontend/src/views/DashboardView.vue`）。
> **督导工作台 DTO 只有 `recentPlans` + `recentDrafts` + `pendingSessions`**（`backend/internal/dto/dashboard.go:26-30`），**不再有 4 张统计卡、`CoverageCard` 或 `byDepartment`**；覆盖率仍由 `GET /supervision/coverage` 单独提供。
> **待评课队列日期闸门**：`plannedDate <= 今天` 且 `evaluated=false` 才显示「去评估」（无授课记录先 `POST /sessions` 建档）；未来课次仅预览；已评估记录从队列清除，复盘走「课程列表 → 课程综合评分」。
> 教师视角的旧「我的课程卡片 + 资源侧栏」已随教师端工作台一并移除（信息与 `/me/quality` 重复）。

### 7.4 CourseListView 课程列表（S2.1 / S2.2 / S2.3）

```
PageHeader（标题随角色：「课程管理」/「我的课程」/「课程列表」 + 主任右侧「新增课程」按钮）
FilterBar：学期下拉 · 教研室下拉(主任/督导) · 教师下拉(主任/督导) · 状态 · 关键词搜索 · 重置
CourseTable：课程编码 | 课程名称 | 授课教师 | 教研室 | 学期 | 班级数 | 学生人次 | 资源数 | 状态 | 操作
分页器（右下，10/20/50 条每页）
```

- 数据范围由后端裁剪（主任→本室、教师→本人、督导→全校），前端列配置三角色一致；
- **v1.3 督导切片**：督导端列表请求带 `mine=1`，后端按 `supervision_plans.supervisor_id` 只返回「本人负责评估的课程」（见 `backend_AGENTS.md` §8.4）；
- 行点击进详情；操作列：查看（全员）、编辑（主任，S3.1）；教师角色行点击改为跳 `/courses/:id/improve`（见 §6.2 决策 3）；**督导角色行点击改为跳 `/supervision/courses/:id`（督导课程综合评分页）**；
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

### 7.5A SupervisorCourseView 督导课程综合评分（v1.3 新增，仅督导）

督导「课程列表 → 具体课程」的落点页，路由 `/supervision/courses/:id`（name `supervisor-course`）。

```
PageHeader：返回 + 课程名称 + 课程编码 Tag + 状态 Tag
信息条：教研室 | 授课教师 | 学期 | 开课班级数 | 学生人次
综合评分区：本课程当前综合评分（督导分 / AI 分 / 综合分）+ ScoreDimensionsCard 五维双源 + 样本卡
历史授课记录区：日期 | 节次 | 主题 | 状态 Tag | 督导分 | 智能体分 | 已评次数 | 操作
分页器（右下，10/20/50 条每页）
```

- 数据源：`GET /courses/:id` + `GET /courses/:id/evaluation-summary`（学期口径取顶栏全局学期）+ `GET /courses/:id/sessions`；
- 操作列两件事：
  - **去评估 / 查看·修改评估**：`status !== 'evaluated'` 显示「去评估」，已评价显示「查看/修改评估」，均跳 `/sessions/:id/evaluation`（督导可在该页覆盖提交，见 §7.10）；
  - **上传录音**：`el-upload` 直传 `POST /sessions/:id/recording`，成功后提示「转写完成后可在评估页查看」，转写与复核在评估页完成；
- 缺失分一律 `—`；样本不足时显式提示，不静默；
- 该页是**已评估记录的复盘入口**：评估提交后授课记录状态变为 `evaluated`，从工作台「待评课队列 / 待评估授课记录」清除，改由本页查看与修改。

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

- **改造要点（T1.15）**：原「督导总览」的统计卡行与 `CoverageCard` **移除**；v1.2 起督导工作台也不再渲染覆盖率与统计卡（覆盖率由 `GET /supervision/coverage` 单独提供），本页只保留**完整分页列表 + 筛选**；
- **本页是听评课计划的后备完整列表**：v1.2 起已从督导侧边栏摘除（督导菜单为「工作台 / 课程列表 / 草稿箱 / 个人中心」），督导的听评课核心动作收口到工作台待评课队列与课程列表；本页保留以保证历史计划仍可访问、可筛选；
- 覆盖率 = 已完成听评课课程数 / 全校开设课程数，由后端计算，前端只展示；
- 覆盖率可视化用自绘 SVG/CSS 进度环（**不引入图表库**，见 §4）。

### 7.8 TeacherListView 教师画像页（S6.4，T1.12；方向：本室/全校）

```
PageHeader：「教师画像」 + 伦理提示 el-alert「仅用于教学支持，不作为考核依据」
FilterBar：学期下拉 · 教研室下拉（督导可按教研室筛选）· 重置
工具栏：视图切换 [质量卡片] [数据表格]（默认质量卡片，选择存 localStorage）
质量卡片视图：TeacherScoreCard 网格 + 卡片排序下拉（按姓名 / 综合分 高→低 / 综合分 低→高）
数据表格视图：TeacherScoreTable：教师姓名 | 工号 | 综合分 | 5 维分 | 评价次数 n | 操作
分页器
```

- 数据源 `GET /teacher-scores?departmentId=&semester=&page=&pageSize=`（**注意不是 `/teachers`**，见 §10.3）；
- **卡片视图为默认**，视图选择持久化在 `localStorage`（key `aijiaoxue_teacher_view`）；表格视图保留为分析模式；
- **默认按姓名/工号排序**，按分排序必须是一次**显式操作**（点击表头或选择排序项），不得默认按分排；
- 每行必须展示**评价次数 n**；`n < 3`（`sampleSufficient === false`）在综合分旁标注「样本不足（n=x）」；
- `compositeScore === null`（无评价）显示「暂无评价」并置底，**不得显示 0，也不得按 0 参与排序**；
- `frontier` 维度渲染为「亮点标记」而非计分维度（`isObservation === true`、`weight === 0`）；
- 页面顶部固定展示「**仅用于教学支持，不作为考核依据**」（伦理要求，见开发计划 §5.3）；
- 行点击跳 `/teachers/:id`（S6.5）。

### 7.9 TeacherDetailView 教师画像详情（S6.5，T1.13）

```
PageHeader：教师姓名 + RoleTag/工号 + 学期选择 + 返回
伦理提示 el-alert：「仅用于教学支持，不作为考核依据」
TeacherScorePanel（summary + timeline 一次传入）：
  三卡横排：① 综合分 · 双源罗盘（QualityCompass，size=lg）
            ② 五维雷达（ScoreDimensionsCard，督导/AI 双源叠加 + 权重徽章）
            ③ 样本与口径（evaluatedCount/sessionCount、督导·AI 计数、alignedCount、融合权重、formulaVersion、flags 提示条）
  按课程明细：课程编码 | 课程名称 | 综合分 | 各维度分 | 评价次数（行可点击）
  历次评价时间线：EvaluationTimeline（竖线 + 课次徽章 + 分数色阶节点；督导评语分色 + AI 评价）
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
       5 个维度，每维 1–5 分，以 AnchorScale 锚点刻度条选择（原生 radio，键盘可达）
       每分档带锚点 tooltip（§7.10.1）
       结构化评语：总体评语 / 亮点 / 待改进 / 建议（textarea）
       「保存草稿」按钮（仅督导，v1.2）+ 提交按钮（PUT 幂等覆盖 → 成功提示 → 回填新分数）
    ② CommentPanel 已提交评语展示（多督导时按时间列出）
  参考区：
    ③ 智能体参考（阶段②）—— EvaluationCompare，标注「AI 参考」；有文字评价时补「总体评语 / 亮点 / 待改进 / 改进建议」（v1.3）
    ④ 音频与转写（阶段②）—— AudioPlayer（仅督导）+ TranscriptViewer
```

- **权限**：仅 `supervisor` 可写；主任（本室）、教师（本人）进入为**只读**——表单禁用、隐藏提交按钮（组件级 `v-if`，非 CSS 隐藏）；
- **五维必填**：任何一维未选即在前端拦截并提示，后端也会返回 `40002`（HTTP **400**）；提交成功后列表/评分需即时反映；
- **幂等覆盖**：同一督导对同一场次重复提交是**覆盖**，前端需在覆盖前给出轻量确认（"将覆盖你上次的评分"）；
- **智能体参考位**：`agentScore` 为 `null` 时展示占位说明「阶段②接入」（`SessionEvaluationView.vue:99-101`），**不得显示 0 分或空图表**；有 `agentScore` 时由 `EvaluationCompare` 与四段文字评价（总体评语/亮点/待改进/改进建议）呈现；
- `frontier` 在本页同样按「亮点标记」处理（不参与加权总分展示）；
- 阶段②的转写区按三态渲染：`pending/running` → 进度/轮询提示；`failed` → 错误信息 + 「重试」按钮；`done` → 转写文本（含说话人）。**轮询当前为组件内联实现**：`SessionEvaluationView.vue` 的 `loadMedia()` 在 `transcript.status` 为 `pending/running` 时用 `setTimeout(..., 2500)` 递归自调用（约 L188-201），并在 `onUnmounted` 清理定时器（L348）。**禁止在组件里裸写 `setInterval`**；`useTranscriptionPolling.ts` composable **当前不存在**（提取为独立 composable 属可选重构，不是现状）；
- **v1.3 教师复盘链路**：教师由「我的质量档案 → 授课快照 → 查看详细记录」进入本页，页面对教师呈现**督导评分（只读）+ AI 智能体评价 + 当堂课脱敏转写入口**，用于回顾授课细节；
- **v1.3 转写卡片长度控制**：`TranscriptViewer` 在 `done` 态默认只预览前 3 条（或前 220 字），超出时显示「展开全文（共 n 条）」；点击后在**居中悬浮弹窗**（`el-dialog align-center`，正文 `max-height: 60vh; overflow-y: auto`）查看完整转写。音频仍仅督导可回放，教师端只提供转写（隐私红线见 §5.1 权限矩阵）。

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
PageHeader：返回 + 课程名称 + 课程编码/状态 + 学期选择
地块（页面级）：伦理提示 el-alert「仅用于教学支持，不作为考核依据」
区块一：课程信息（置顶：名称/编码/状态/学分/学时/学期/班级/学生人次/教研室/教师）
区块二：本课程综合评分（「您在当前课程获得的综合评分」：五维双源横向柱条 + 增长动画；右侧 ScoreRadar 综合分雷达 0-100）
区块三：督导评语流（EvaluationTimeline，按课次倒序）
区块四：智能体提优建议（阶段③，AgentSuggestionList，按时间倒序）
区块五：分数趋势（阶段③，ScoreTrendChart，radio 按维度切换）
区块六：课程资源（ResourceList + ResourceUploader，**置底**）
右下角常驻：「课程提优助手」抽屉（阶段③，AgentChat + useAgentChat，流式对话）
```

- 数据源：课程基本信息 `GET /courses/:id`；课程级评分 `GET /courses/:id/evaluation-summary?semester=`（`fetchCourseEvaluationSummaryApi`）；督导评语/评价流取 `GET /teachers/:id/evaluations`（`id` = 当前登录用户）后按 `courseId` 过滤（`frontend/src/views/CourseImproveView.vue:154-195`）；
- **只读**：教师**不能**修改分数或评语，页面不出现任何编辑态控件；
- 综合分为 `null` 时显示「暂无评价」，**不得显示 0**；
- 突出「**仅用于教学支持**」的语气：文案面向改进而非考核（如「本期共 3 次课被评价，互动维度是提升空间」）；
- 资源上传沿用 Sprint 1 契约（`ResourceUploader`），**不得因为页面改造而丢失该能力**（T1.14 验收项）；
- **阶段③智能体相关区块（四/五 + 抽屉）的真实接口本次留空**：`fetchAgentSuggestionsApi` / `fetchScoreTrendApi` / `streamAgentChatApi`（`frontend/src/api/agent.ts`）暂不发起网络请求，改由前端临时演示数据 `frontend/src/mocks/teacherImprove.ts` 驱动；函数签名与未来后端一致，后端就绪后仅替换实现。趋势区块展示的维度为「综合分 + 5 维」，其中 `frontier` 为观测项。

### 7.12 ProfileView 个人中心（辅助，最低优先级）

- 展示：头像、姓名、角色、所属教研室/工号、账号；
- **不做资料编辑**（超范围）。

### 7.13 ProfileQualityView 我的质量档案（教师首页，`/me/quality`，v1.2 新增）

> 教师的「质量主页」与**登录后首页**（教师端不再有质量驾驶舱）。数据全部来自教师查本人接口，只读。

```
PageHeader：「我的质量档案」+ 副标题「督导与 AI 双源融合的教学质量全景 · 当前学期口径」
伦理提示 el-alert：「仅用于教学支持，不作为考核依据」
三卡横排：
  ① 我的综合分 —— QualityCompass（size=lg）+ flags 提示条
  ② 五维评分 —— ScoreDimensionsCard（雷达双源叠加 + 维度条）
  ③ 提优分析 ——「可提优方向 / 提优建议」两分区 + AiBadge「AI 生成」（当前为示例文本 + `AI · 示例` Tag）
我的课程评分明细：课程编码 | 课程名称 | 综合分（null → 「暂无评价」）| QualityBadge | n=evaluatedCount | 「去提优 →」
授课快照：逐次授课卡片（按时间倒序），左侧色条颜色 = 综合分质量水平；
          高分标「亮点」、低分标「主要不足」（取督导评语，缺省回退 AI 评语），每卡「查看详细记录 →」
```

- 数据源：`GET /teachers/:id/evaluation-summary?semester=`（`id` = 当前登录用户，`fetchTeacherSummaryApi`）+ `GET /teachers/:id/evaluations?semester=&pageSize=50`（授课快照，`fetchTeacherEvaluationsApi`）（`frontend/src/views/ProfileQualityView.vue:66-91`）；
- 「去提优 →」跳 `/courses/:id/improve`；**授课快照「查看详细记录 →」跳 `/sessions/:id/evaluation`**（`ProfileQualityView.vue:93-103`），展示该次课的督导评分（只读）与 AI 智能体评价，并提供当堂课脱敏转写入口；
- 第三卡「提优分析」当前为**页面内示例文本**（`ProfileQualityView.vue:46-55`，注释明确「预留 AI 接口，当前为示例文本」），不是真实 AI 输出；
- 只读，无编辑态控件；综合分为 `null` 显示「暂无评价」，**不得显示 0**；
- 主任/督导无此路由与菜单（他们走 `/teachers/:id`）。

### 7.14 DraftBoxView 草稿箱（督导，`/drafts`，v1.2 新增）

```
PageHeader：「草稿箱」+ 副标题「未提交的评估草稿 · 可编辑、提交或删除」
FilterBar：按课程名称查询（keyword）+ 重置
表格：课程名称(+课程编码 Tag) | 授课教师 | 授课日期 | 节次 | 主题 | 操作[编辑 | 提交 | 删除]
分页器（PAGE_SIZES = 10/20/50）
```

- 数据源 `frontend/src/api/draft.ts`：`GET /drafts?keyword=&page=&pageSize=`（`fetchDraftsApi`）、`POST /drafts/:id/submit`（`submitDraftApi`）、`DELETE /drafts/:id`（`deleteDraftApi`）（`frontend/src/views/DraftBoxView.vue:26-115`）；
- **编辑** → `/sessions/:sessionId/evaluation?draft=1`，评估页优先回填本人草稿；
- **提交** → 先 `ElMessageBox.confirm` 二次确认；后端复用正式提交口径（**五维必填**），成功后写正式评价并删除草稿；维度不全时由拦截器提示且草稿保留；
- **删除** → 先 `ElMessageBox.confirm`（文案「是否确认删除所选草稿？」）再删除；
- 三态完整（`v-loading` 加载态 / EmptyState 空态「暂无草稿」/ 失败态 + 重新加载）；
- 入口：侧边栏「草稿箱」与工作台「草稿箱 → 更多>>」。

## 8. 组件设计清单

> 组件一律「展示与交互」，数据请求放页面或 composable，通过 props 下发。**每张表标注所属阶段：`①`=Sprint 2.1 必做，`②`=Sprint 2.2，`③`=Sprint 3。**

### 8.1 布局组件（`layouts/` + `components/common/`）

| 组件 | Props（概要） | 职责 | 消费方 | 阶段 |
|------|--------------|------|--------|------|
| AppLayout | — | 侧边栏 + 顶栏 + 内容区骨架 | 已登录路由 | S1 |
| BlankLayout | — | 无框架容器 | 登录/403/404 | S1 |
| PageHeader | `title?, subtitle?` + slot#actions | 页面标题 + 操作区，统一页首留白 | 所有内容页 | S1 |
| StatCard | `label, value, unit?, icon?, tone?` | 指标卡（图标 + 大数字 + 标签） | 工作台（主任） | S1 |
| FilterBar | 默认 slot + slot#actions + `@search/@reset` | 筛选栏容器，收拢查询交互 | 课程列表/听评课/教师画像/草稿箱 | S1 |
| EmptyState | `description?`（默认「暂无数据」）+ slot#action | 空数据占位（引导操作） | 所有列表 | S1 |
| RoleTag | `role?` | 角色标签，三角色三色 | 顶栏/表格 | S1 |
| AiBadge | `text?`（默认「AI 参考」） | AI 徽标（四芒星 + `--gradient-ai`），AI 产出内容的准入证 | 一切 AI 区块 | v1.0 |
| QualityBadge | `score?: number\|null, level?: string, showScore?: boolean` | 质量等级徽章（色阶映射；`null` → 灰色「暂无评价」） | 全部分数旁 | v1.0 |

### 8.2 课程域组件（`components/course/`）

| 组件 | Props（概要） | 职责 | 对应故事 | 阶段 |
|------|--------------|------|---------|------|
| CourseTable | `rows, loading?, showEdit?` | 课程数据表格 + 行操作 | S2.1–S2.3 | S1 |
| CourseCard | `course` + `@click` | 课程卡片 | S2.2 | S1 |
| CourseInfoForm | `mode, submitting?, departmentOptions?` | 新增/编辑表单 + 校验 | S3.1 | S1 |
| ResourceList | `resources, canManage?, loading?` | 资源列表 + 下载/删除 | S4.1 / S4.2 | S1 |
| ResourceUploader | `accept?, maxSize?, uploading?, progress?` | 拖拽 + 点击上传，进度与类型校验 | S4.2 | S1 |

### 8.3 督导域组件（`components/supervision/`）

| 组件 | Props（概要） | 职责 | 对应故事 | 阶段 |
|------|--------------|------|---------|------|
| CoverageCard | `overall: CoverageStat\|null, loading?` | 覆盖率环形（+ 分组条形） | S5.1 | S1（v1.2 起督导工作台不再使用） |
| ScheduleTable | `plans, loading?` + `@view` | 听评课安排表 | S5.1 / S6.7 | S1 |
| EvaluationQueue | `plans: SupervisionPlan[], loading?` + `@create/@evaluate` | 待评课队列（今日/本周/本月/未评四档 + 日期闸门 + 状态徽章 + 行内直达动作） | S6.7 | v1.2 |

### 8.4 评价域组件（`components/evaluation/`）

| 组件 | Props（概要） | 职责 | 对应故事 | 阶段 |
|------|--------------|------|---------|------|
| ScoreRadar | `items: RadarItem[], max?, loading?` | 五维条形/雷达；`RadarItem` 支持 `supervisorScore/agentScore` 双源叠加；`isObservation` 维度单独作亮点标记 | S6.5 / S6.6 | ① |
| QualityCompass | `summary: ScoreSummary\|null, size?: 'lg'\|'md'\|'sm', loading?, title?` | 双源罗盘：综合分 + 等级徽章 + 双弧 + 样本 + flags；`null` → 灰环「暂无评价」 | S6.5 / S6.6 | v1.0 |
| ScoreDimensionsCard | `summary: ScoreSummary\|null, title?` | 五维评分卡（雷达双源叠加 + 维度条权重徽章） | S6.5 / S6.6 | v1.0 |
| ScoreHeatmap | `teachers: TeacherScoreItem[], loading?` + `@select` | 教师 × 五维质量热力（主任驾驶舱；点行下钻教师画像） | S6.4 | v1.0 |
| AnchorScale | `dimensionKey, dimensionName, modelValue, readonly?, compact?` + `@update:modelValue` | 锚点刻度条（1–5 五档 + 锚点 tooltip，原生 radio） | S6.3 | v1.0 |
| EvaluationForm | `readonly?, submitting?, showDraft?, draftSaving?` + `@submit/@save-draft` | 5 维锚点评分 + 结构化评语；`readonly` 时禁用 | S6.3 | ① |
| CommentPanel | `items: EvaluationDTO[], loading?` + `@select` | 督导评语展示（亮点/待改进/建议） | S6.6 | ① |
| EvaluationTimeline | `items: TeacherTimelineItem[], loading?, emptyText?` | 竖线评价流（课次徽章 + 分数色阶节点 + 督导评语 + AI 评价） | S6.5 / S6.6 | v1.0 |
| EvaluationCompare | `supervisor?, agent?, loading?` | 督导 vs 智能体分维度对比；标注「AI 参考」与低置信度，展示 `evidence` 引用 | S7.2 | ② |
| TranscriptViewer | `transcript: TranscriptDTO\|null, loading?` + `@retry` | 转写文本三态（pending/running/done/failed）+ 预览/弹窗展开与说话人区分 | S7.1 | ② |
| ScoreTrendChart | `points: ScoreTrendPoint[], dimensionName?, max?, loading?` | 单维度历史趋势折线；`value` 为 `null` 处断开、不补 0 | S8.3 | ③（演示） |
| AgentSuggestionList | `items: AgentSuggestion[], loading?` | 智能体提优建议列表（标「AI 参考」、对应课次、置信度、转写依据） | S8.1 | ③（演示） |
| AgentChat | `messages: AgentChatMessage[], streaming?, disabled?` + `@send/@stop` | 与智能体流式对话（气泡、光标、建议提示词、Enter 发送） | S8.2 | ③（演示） |

### 8.5 授课记录域组件（`components/session/`）

| 组件 | Props（概要） | 职责 | 对应故事 | 阶段 |
|------|--------------|------|---------|------|
| SessionTable | `rows: SessionListItem[], loading?` + `@view` | 历史授课记录表（日期/节次/主题/状态/双侧分/已评次数）+ 行点击 | S6.2 | ① |
| AudioPlayer | `src, name?` | 课堂录音播放（`<audio src>` 使用带短时票据的 `playbackUrl`） | S7.1 | ② |

### 8.6 教师域组件（`components/teacher/`）

| 组件 | Props（概要） | 职责 | 对应故事 | 阶段 |
|------|--------------|------|---------|------|
| TeacherScoreTable | `rows: TeacherScoreItem[], loading?` + `@view` | 教师列表：综合分、5 维分、评价次数 n、样本不足标记；默认按姓名排序 | S6.4 | ① |
| TeacherScoreCard | `item: TeacherScoreItem` + `@view` | 教师质量卡片（罗盘缩略 + 五维迷你条 + 样本徽章），画像列表默认视图 | S6.4 | v1.0 |
| TeacherScorePanel | `summary: TeacherSummary\|null, timeline, loading?, timelineLoading?` + `@select-course` | 教师画像详情面板：三卡（双源罗盘 + 五维卡 + 样本与口径）+ 按课程明细 + 历次评价时间线 | S6.5 | ① |

### 8.7 组件编写规则

- 一律 `<script setup lang="ts">`；Props 用 `defineProps<{…}>()` 类型声明；Emits 用 `defineEmits<{…}>()`；
- 组件只负责**展示与交互**，数据请求放在页面或 composable，通过 props 下发；
- 对外透传 UI 库组件属性时使用 `inheritAttrs: false` + `v-bind="$attrs"`；
- 每个业务组件 props 中必须有 `loading`（或等价）状态，支撑骨架屏/禁用态；
- **评分类组件额外要求**：必须显式处理 `null` 分（渲染 `—` 而非 0）、必须透传并展示 `sample` 与 `flags`，不得在组件内自行"补齐"缺失侧分数。

## 9. 美术与设计规范（Design Tokens）

**风格定位：「学术墨韵 × 评估仪器」** —— 保留「学院蓝 · 清新教育」的专业可信与清爽轻盈，并把「清新教育」校准为**严谨的测量感**：深色结构层承载导航，内容区让给数据；评分数据用仪器级精度呈现（等宽数字、色阶刻度、锚点文案）；AI 用琥珀紫渐变语言统一显形，与「人类督导（靛蓝）」形成一眼可辨的双源对比。层级靠留白与字重，装饰克制（无玻璃、无发光、无大面积渐变）。

> 主操作色保留 `#2F54EB` 不动：它同时是主任角色色与 Element 主题锚点，保留可使全部 `el-*` 组件零回归；辨识度靠下述三件签名而非换主色。

### 9.0 设计叙事与三个视觉签名

**核心叙事：「界面即测量仪器」/「质量评估 × AI 提优」**。产品愿景是「让教学质量持续可测」，界面要让人第一眼理解动作闭环 **测 → 融 → 察 → 提**：

```
   测            融              察              提
（督导锚点评分）（督导×AI 双源融合）（质量全景下钻）（AI 建议 + 对话）
      └────── 场次级 ──────┘        │               │
              └──── 课程级 ────────┤               │
                     └──── 教师级 ─┴── 趋势 ────────┘
```

- **测**：当堂课评估页是系统的「核心动作页」，锚点刻度可视化（§9.0 / §8.4 `AnchorScale`）；
- **融**：综合分永远是「督导 × AI 双源罗盘」图形（`QualityCompass`），而不是一个裸数字；
- **察**：主任工作台以质量热力（`ScoreHeatmap`）/ 待评队列为主角（§7.3）；
- **提**：教师端首屏直达「我的质量档案」与 AI 提优建议，AI 视觉全链路可识别。

**三个视觉签名**（识别度的来源，实现见 `frontend/src/components/`）：

| 签名 | 图形 | 出现位置 | 传达的信息 |
|------|------|---------|-----------|
| **① 双源罗盘** | 内外双弧（督导弧·靛蓝 `#2F54EB` + AI 弧·亮紫 `#8B5CF6`）+ 中心综合分大数字 | 所有综合分展示位（工作台、评估页、教师画像、质量档案、提优页、课程快照） | 「分数 = 人 + AI 双源融合」，一眼看懂可信度结构 |
| **② 质量色阶 + 等级徽章** | 5 级色带（红→琥珀→绿）+「优秀/良好/合格/待改进/需干预」徽章 `QualityBadge` | 全部评分数字、维度条、热力图、状态流 | 分数可读化：不用记数值也知道好坏，色阶即品牌 |
| **③ AI 视觉语言（琥珀紫）** | 四芒星徽标 `AiBadge` + `--gradient-ai` 亮紫渐变 + 浅紫虚线卡 + 流式光标 | AI 参考卡、AI 建议、AI 对话、转写区 | 「这是 AI 的产出，可参考、可追溯」，人机分工可视化 |

**AI 卡片语法（虚线卡约定）**：所有智能体产出内容必须同时满足三条识别规则，缺一不可：

1. **四芒星徽标 `AiBadge`**：`--gradient-ai` 渐变填充 + 「AI 参考」/「AI 建议」/「AI 生成」文字；
2. **浅紫虚线卡片**：统一 class `.ai-panel`（`frontend/src/styles/base.scss:105-120`）= `--color-ai-bg` 浅底 + 1px `rgba(139,92,246,.45)` 虚线描边 + `--radius-ai` / `--shadow-ai` + 顶部 2px 渐变饰条——「虚线」隐喻「参考、未定案」，与人类督导的实线卡片形成语法对比；**禁止页面各自手写**（用法示例：`components/evaluation/EvaluationCompare.vue:50,59`）；
3. **可追溯性露出（部分待后端补字段）**：AI 评分卡当前露出 `aiConfidence`（置信度进度条 + 低置信标签，见 `EvaluationCompare.vue`）；`evidence`（转写引用）**当前未在 `EvaluationDTO` 暴露**（`backend/internal/dto/evaluation.go` 无该字段，仅 DB/model 有 `evidence` 列），故智能体评价对照面板暂无法渲染证据；阶段③ mock 的 AI 建议卡（`AgentSuggestion`）已有 `evidence` 字段。**低置信分当前仍展示并标注警示，不做隐藏**（红线见 §9.6-4）。

> **与督导角色色的区分**：督导紫 `#722ED1` 只出现在 `RoleTag` / 头像（指「人」）；AI 渐变紫只出现在智能体产出内容上（指「机器产出」），两个场景不相交。
> **不做清单**：❌ 玻璃拟态 / 大面积渐变 / 发光特效堆「AI 感」；❌ 第二套 UI 库或图表库；❌ 评分色阶作装饰色。

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

**新增语义分组（`frontend/src/styles/tokens.scss` 已全部定义）**：

| 分组 | Token | 值 | 用途 |
|------|-------|-----|------|
| 角色浅背景 | `--color-director-bg` / `--color-teacher-bg` / `--color-supervisor-bg` | `#F0F5FF` / `#E6FFFB` / `#F9F0FF` | `RoleTag` 与头像底色 |
| 结构色（深色侧栏/页头） | `--color-ink` | `#0E1B2C` | 侧边栏/深色页头基底 |
| | `--color-ink-2` | `#16283E` | 菜单悬停底 |
| | `--color-ink-3` | `#1F3A5F` | 菜单激活底、深色内描边 |
| | `--color-ink-text` | `#DCE7F5` | 侧栏主文字 |
| | `--color-ink-text-dim` | `#8FA3BF` | 侧栏次级文字 |
| AI 视觉色 | `--color-ai` | `#8B5CF6` | AI 锚点色（徽标、AI 弧、AI 文字强调） |
| | `--color-ai-bright` | `#A855F7` | AI 徽标亮紫、悬停、流式光标 |
| | `--color-ai-deep` | `#6D28D9` | AI 描边、按下 |
| | `--color-ai-bg` | `#F6F3FF` | AI 卡片浅底 |
| | `--gradient-ai` | `linear-gradient(135deg, #6D28D9 0%, #A855F7 55%, #D946EF 100%)` | AI 徽标、AI 对话头像、AI 建议卡顶部饰条 |
| 评分色阶 | `--color-score-5` | `#2FB344` | 优秀 · 可作为示范课 |
| | `--color-score-4` | `#84B80C` | 良好 · 骨干教师水平 |
| | `--color-score-3` | `#F5A623` | 合格 · 基本教学要求 |
| | `--color-score-2` | `#F76707` | 待改进 · 存在明显短板 |
| | `--color-score-1` | `#E5484D` | 不合格 · 需要立即干预 |
| | `--color-score-void` | `#C9CDD4` | **无分数（`null` 专用，禁止以 0 充当）** |
| | `--gradient-score` | `linear-gradient(90deg, #E5484D, #F5A623, #2FB344)` | 连续分数色带（趋势图、罗盘弧） |

**色阶映射规则**（`frontend/src/utils/format.ts` 的 `scoreTone` / `scoreToneColor`）：

- 离散分（1–5 原始分）按上表直接映射；
- 连续分（0–100 综合分/维度分）按五段区间映射：`[0,40) → score-1`、`[40,60) → score-2`、`[60,75) → score-3`、`[75,90) → score-4`、`[90,100] → score-5`（与服务端锚点「3=合格」对齐：60 分即李克特 3 分）；
- **色阶只用于「分数及其派生物」**（数字、条形、雷达面积、热力、罗盘弧），禁止作装饰色——这是色阶可信度的前提。

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

// 评分数字专用（tokens.scss「评分数字字体与字号」组）
--font-score: 'Bahnschrift', 'DIN Alternate', 'Segoe UI', sans-serif; // 系统自带 DIN 风，零依赖兜底
--font-size-score-xl: 40px;  // 双源罗盘中心综合分
--font-size-score-lg: 28px;  // StatCard 级评分
--font-size-score-md: 20px;  // 维度条端数字
```

- 行高：正文 1.6，标题 1.3；
- 字重：标题 600，正文 400，StatCard/综合分数字 600 + `font-variant-numeric: tabular-nums`；评分数字走 `--font-score`；
- 评分数字**永远随附证据链**：数字旁必有等级徽章（`QualityBadge`）与样本量标注（红线见 §9.6）。

### 9.3 间距 / 圆角 / 阴影

```scss
--spacing-1: 4px;  --spacing-2: 8px;  --spacing-3: 12px;
--spacing-4: 16px; --spacing-6: 24px; --spacing-8: 32px;  // 4px 基数
--radius-sm: 4px;  // Tag
--radius-md: 6px;  // 按钮、输入框
--radius-lg: 12px; // 卡片、Modal
--radius-ai: 10px; // AI 卡片（虚线卡）
--shadow-card: 0 1px 2px rgba(15,23,42,.04), 0 1px 3px rgba(15,23,42,.06);
--shadow-card-hover: 0 4px 12px rgba(15,23,42,.10);
--shadow-modal: 0 12px 32px rgba(15,23,42,.16);
--shadow-ai: 0 2px 8px rgba(109, 40, 217, 0.08);
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
4. **AI 必须显形**：智能体分数一律带「AI 参考」标签；低置信度维度（`aiConfidence < 0.4`）单独提示，**分数仍展示、不得静默隐藏**；`evidence`（转写引用）**当前未在 `EvaluationDTO` 暴露**（DB/model 有 `evidence` 列），故「无证据不采信」暂无法在前端执行——后端补字段后再启用；
5. **口径提示不隐藏**：`flags` 含 `disjoint` / `formula_mixed` / `no_data` 时，页面需有可见提示条，**不得静默**；
6. **伦理标注常驻**：教师画像列表 / 教师画像详情 / 我的质量档案 / 教学提优页顶部固定展示「仅用于教学支持，不作为考核依据」。

> 评分色阶由设计在 `tokens.scss` 统一补充；**禁止在业务组件里散落写死颜色**（同 §9.1 色彩铁律）。

### 9.7 动效规范（克制、有语义）

| 动效 | 参数 | 场景 | reduced-motion 降级 |
|------|------|------|-------------------|
| 数字进场 count-up | 600ms ease-out（`--duration-count`），一次/进页 | 双源罗盘综合分、驾驶舱关键数字 | 直接显示终值 |
| 雷达展开 | 400ms（`--duration-mid`），从中心放散 | `ScoreRadar` 挂载 | 直接显示 |
| 罗盘弧生长 | 500ms `stroke-dashoffset` | `QualityCompass` 挂载 | 直接显示 |
| 五维双源柱条增长 | `width` 过渡（`--duration-mid`） | 教学提优页评分区 | 直接显示 |
| 转写波形呼吸 | 2s 无限循环，仅 opacity | `TranscriptViewer` `pending/running` | 静态占位 |
| AI 流式光标 | 800ms 闪烁（光标色 `--color-ai-bright`） | `AgentChat` streaming | 常亮下划线 |
| 卡片悬浮 | `--shadow-card-hover`，位移 0 | 全部卡片 | 保留（非动效） |

动效令牌：`--ease-out-soft: cubic-bezier(0.22, 0.61, 0.36, 1)`、`--duration-fast: 200ms`、`--duration-mid: 400ms`、`--duration-count: 600ms`。
动效总量控制：单页同时运动的元素 ≤ 2 处；禁止页面级滚动视差。

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
  sessionId: number | null;  // 关联授课记录；null = 尚未创建，需先建档再评估（v1.2）
  evaluated: boolean;        // 关联授课记录是否已完成督导评价（决定是否暴露评估入口，v1.2）
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
  agentScore: EvaluationDTO | null;   // 后端 evaluator_type='agent' 行；无则为 null
}

/** GET /sessions/:id/transcript 响应（阶段②，已实现） */
interface RecordingDTO {
  id: number;
  originalName: string;
  format: string;            // mp3 / wav / m4a
  size: number;
  durationSec: number;       // ASR 返回后回写；未完成时 0
  streamUrl: string;         // 需 Authorization 头
  playbackUrl: string;       // 带短时票据的流式地址，可直接给 <audio src>；仅督导侧返回（教师侧空串）
}

interface TranscriptSegment {
  start: number;             // 秒
  end: number;               // 秒
  speaker: string;
  text: string;
}

interface TranscriptDTO {
  id: number;
  status: 'pending' | 'running' | 'done' | 'failed';
  content: string;           // 纯文本全文（已脱敏）
  segments: TranscriptSegment[];
  engine: string;
  engineVersion: string;
  errorMessage: string;
}

/** GET /sessions/:id/transcript 的媒体聚合（录音 + 转写） */
interface SessionMedia { recording: RecordingDTO | null; transcript: TranscriptDTO | null }

/** GET /dashboard 督导 DTO 的待评估授课记录行（v1.2） */
interface PendingSessionItem {
  sessionId: number;
  courseId: number;
  courseCode: string;
  courseName: string;
  teacherId: number;
  teacherName: string;
  sessionDate: string;
  period: string;
  topic: string;
  status: SessionStatus;     // scheduled | recorded（status<>evaluated 才会出现在此列表）
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
  agentEvaluation: EvaluationDTO | null;   // 无 agent 评价时为 null
}
```

> 其他前端类型文件（本手册不逐字段重复，以代码为准）：`types/draft.ts`（`DraftDTO` / `DraftUpsertPayload`）、`types/teacher.ts`（`ScoreSummary` / `ScoreDimension` / `TeacherScoreItem` / `TeacherSummary` / `TeacherTimelineItem` / `ScoreTrendPoint` / `ScoreTrendSeries`）、`types/agent.ts`（`AgentSuggestion` / `AgentChatMessage` / `AgentChatPayload` / `AgentChatHandlers`）、`types/supervision.ts`（`DashboardData` / `DashboardStat` / `CoverageStat`）。

### 10.3 接口契约索引（**契约正文见 [`backend_AGENTS.md`](./backend_AGENTS.md) §8，本节不重复**）

> 🔴 前端**不得**依据本手册的转述实现请求：字段名、权限、错误码一律以后端 `backend_AGENTS.md` §8 为唯一事实源，并同步 `frontend/src/types/`。

| 域 | 契约位置 | 前端 api 文件 | 前端在线状态 |
|----|---------|--------------|-------------|
| 认证 auth | `backend_AGENTS.md` §8.1 | `api/auth.ts` | ✅ 已接入 |
| 工作台 dashboard | §8.2 | `api/dashboard.ts` | ✅ 已接入（三角色 DTO 收敛为 `DashboardData`） |
| 字典 dict | §8.3 | `api/dict.ts` | ✅ 已接入（`/departments`、`/teachers` 教师字典） |
| 课程 courses | §8.4 | `api/course.ts` | ✅ 已接入（含 `mine=1` 督导切片） |
| 资源 resources | §8.5 | `api/resource.ts` | ✅ 已接入 |
| 督导 supervision | §8.6 | `api/supervision.ts` | ✅ 已接入 |
| 授课记录与评价聚合 | §8.8 | `api/session.ts`、`api/teacher.ts`、`api/draft.ts` | ✅ 已接入 |
| 课堂录音与转写 | §8.9 | `api/session.ts` | ✅ **已实现**（阶段②；`POST recording` / `GET transcript` / `POST transcript/retry` / `GET recordings/:id/stream`） |
| 智能体对话 / 趋势 | 开发计划 §4.4（**后端未实现**） | `api/agent.ts` | ◐ 由 `mocks/teacherImprove.ts` 演示数据驱动，不发网络请求 |

**前端侧仅有的三条补充约定**（其余全部见后端 §8）：

1. 🔴 **`GET /teachers` 永远是教师字典，教师评分列表是 `GET /teacher-scores`。** 两者并存、用途不同：`/teachers` 返回数组（课程表单下拉依赖），`/teacher-scores` 返回分页评分列表。新增接口不得复用二者，也不得再改路径而不更新本手册与开发计划。
2. **草稿接口共 5 个**（`backend_AGENTS.md` §8.8）：`GET /sessions/:id/draft`（无草稿返回 `null`）、`PUT /sessions/:id/draft`（**允许部分维度为空**）、`GET /drafts?keyword=&page=&pageSize=`、`DELETE /drafts/:id`、`POST /drafts/:id/submit`（五维必填，成功后转正式评价并删除草稿）——对应 `frontend/src/api/draft.ts`。
3. **音频播放票据**：`GET /recordings/:id/stream` 额外允许 `?ticket=` 短时票据；前端直接用后端返回的 `RecordingDTO.playbackUrl`（而非 `streamUrl`）作为 `<audio src>`，因为播放期间的 Range 请求无法携带 `Authorization` 头。教师侧 `playbackUrl` 为空串，页面据此不渲染播放器（`SessionEvaluationView.vue:181-186`）。

### 10.4 前端已接接口清单（阶段① / 阶段②）

> 路径、权限、请求/响应字段以 [`backend_AGENTS.md`](./backend_AGENTS.md) §8.8 / §8.9 为准；下表只标注前端消费点，便于索引。

| 方法 | 路径 | 前端消费点 |
|------|------|-----------|
| GET | `/courses/:id/sessions?semester=&page=&pageSize=` | 课程详情「历史授课记录」/ 督导课程综合评分 |
| POST | `/sessions` | 工作台「新增授课记录」、待评课队列建档 |
| GET | `/sessions/:id` | 单场次基本信息 |
| GET | `/sessions/:id/evaluation` | 当堂课评估页聚合（场次 + 督导评分 + `agentScore`） |
| PUT | `/sessions/:id/supervisor-evaluation` | 督导提交/覆盖评分（幂等；五维必填，缺失 `40002`） |
| GET | `/teacher-scores?departmentId=&semester=&page=&pageSize=` | 教师画像列表、主任工作台质量热力 |
| GET | `/teachers/:id/evaluation-summary?semester=` | 教师画像详情、我的质量档案 |
| GET | `/teachers/:id/evaluations?semester=&page=&pageSize=` | 教师画像详情时间线、质量档案授课快照、提优页评语流 |
| GET | `/courses/:id/evaluation-summary?semester=` | 督导课程综合评分、教学提优页 |
| GET/PUT | `/sessions/:id/draft` · `/drafts` · `/drafts/:id` · `/drafts/:id/submit` | 评估页「保存草稿」、工作台草稿箱、草稿箱页 |
| POST | `/sessions/:id/recording` | 评估页 / 督导课程综合评分「上传录音」 |
| GET | `/sessions/:id/transcript` | 评估页录音与转写区（三态轮询） |
| POST | `/sessions/:id/transcript/retry` | 转写失败「重试」 |
| GET | `/recordings/:id/stream` | `AudioPlayer`（经 `playbackUrl`，仅督导） |

**阶段③后端未实现、前端不发请求的接口**（`api/agent.ts` 返回演示数据）：`GET /teachers/:id/score-trend?semester=&dimension=`、`POST /agent/chat`（SSE）。后端就绪前**不返回 `501`**——请求根本不会发出；后端就绪后仅替换 `api/agent.ts` 实现，页面与组件无需改动。

### 10.5 错误码（新增部分）

| code | HTTP | 含义 | 前端处置 |
|------|------|------|---------|
| 40001 | 400 | 参数校验失败 | 表单内联提示 |
| **40002** | **400** | 业务规则校验失败（授课记录日期晚于今天、**督导评分五维未录全**、草稿提交时维度不全） | 就地提示具体缺失维度；**注意 HTTP 是 400，不是 500** |
| 40301 | 403 | 角色无权访问该功能 | 提示并回退 |
| 40302 | 403 | 无权操作该数据（越权访问他室/他人数据） | 提示并回退；不得当作"数据为空" |
| 40901 | 409 | 数据已存在（同课程同班级同日同节次重复建课、重复上传录音） | 提示重复并引导查看已有记录 |
| 40902 | 409 | 重复提交 | **保留码：`PUT` 幂等覆盖，后端不返回该码** |
| 50002 | 501 | 功能未实现（阶段③智能体接口等后端未实现功能） | 展示"功能开发中"占位，不弹全局错误 |
| 50003 | 503 | 依赖服务不可用（ASR 引擎未配置或转写失败） | 展示失败态 + 重试按钮 |

> **录音转写（阶段②）不是 `50002`**：相关路由已实现（`backend/internal/router/router.go:84,102-103,120`），失败走 `50003` + `transcripts.status='failed'`。

### 10.6 请求层约定

- 统一响应 `{ code, message, data }`：`code === 0` 成功；**HTTP 401**（后端返回 `40101`）清 token 跳登录；其余弹 `ElMessage.error(message)`；
- 鉴权头：`Authorization: Bearer <token>`，token 存 `localStorage`（key：`aijiaoxue_token`，用户信息 key：`aijiaoxue_user`）；
- 聚合接口**必须透传 `semester`**（开发计划 §2.5.6：跨学期平均会抹平改进），前端默认当前学期；
- **没有 mock/real 接口切换开关**：`frontend/.env.development` 只有 `VITE_API_BASE_URL` 与 `VITE_PROXY_TARGET`，代码中不存在 `VITE_USE_MOCK`。
  > `frontend/src/mocks/` 当前仅含 `teacherImprove.ts`，被 `frontend/src/api/agent.ts` 直接 import，为**阶段③ AI 区块**（提优建议 / 趋势 / 对话）提供演示数据（后端 `GET /teachers/:id/score-trend`、`POST /agent/chat` 未实现）。其余接口全部直连真实后端。后端就绪后删除该 mock 并令 `api/agent.ts` 改走真实请求即可，页面与组件无需改动。

## 11. 状态管理与数据流

```
views ──调用──> api/*（唯一请求出口）
  │                 │
  ├── composables/useCourseList（列表查询状态内聚：rows/loading/query/page）
  ├── composables/usePagination｜useSemester（分页与全局学期）｜useAgentChat（阶段③流式对话）
  ├── composables/useAuth（登录态）
  └── stores/auth（token/user/hasRole）  stores/dict（学期、教研室字典）
```

- `stores/auth.ts`：`login()` / `logout()` / `hasRole(role)` / `state: { token, user }`，持久化 token；
- 列表类数据**不进 Pinia**，用 composable 内聚（分页、筛选、加载态），避免全局状态膨胀——**评分列表、聚合结果同理进 composable，不进 store**；
- 字典（学期、教研室）启动时拉取一次缓存于 `stores/dict.ts`；
- 轮询类副作用（转写）必须在 `onUnmounted` 清理定时器；**当前实现是 `SessionEvaluationView.vue` 内联 `setTimeout` 递归（L188-201，L348 清理），没有 `useTranscriptionPolling.ts` composable**（见 §7.10）；
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

**当前确认保留的文档集**（前端相关）：[`README.md`](../README.md)、[`CONTRIBUTING.md`](../CONTRIBUTING.md)、[`backend/AGENTS.md`](../backend/AGENTS.md)、[`docs/backend_AGENTS.md`](./backend_AGENTS.md)、[`docs/frontend_AGENTS.md`](./frontend_AGENTS.md)（本文件）、[`docs/Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md)、[`docs/MySQL数据库创建指导.md`](./MySQL数据库创建指导.md)。

**前端智能体推荐阅读顺序**：

| # | 文档 | 读什么 |
|---|------|--------|
| 1 | [`CONTRIBUTING.md`](../CONTRIBUTING.md) | 协作流程、红线清单、PR 要求 |
| 2 | [`docs/Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) | 当前迭代的功能范围、评分体系（§2）、权限（§5）、路由（§6） |
| 3 | [`docs/backend_AGENTS.md`](./backend_AGENTS.md) §8 | **接口契约唯一事实源**（字段名、权限、错误码） |
| 4 | 本文件 | 技术栈 / 目录 / 路由 / 页面规格 / 组件 / 设计令牌 / 编码规范 / DoD |

必须在同一 PR 同步本文件的情形：

1. **路由**：新增/删除/改名路由、改变 `name`、改变 `meta.roles` 或页面权限 → 更新 §6.1 路由总表、§6.2 路由变更表（如涉及决策变更还需在 §13 说明理由）；
2. **接口与类型字段**：新增或修改接口路径/查询参数/响应字段、DTO 增删字段、错误码或 HTTP 映射变化 → 更新 §10；
3. **权限**：页面权限矩阵或页面内操作权限的变化（含"教师可见范围"这类敏感边界）→ 更新 §3；
4. **设计令牌与美术规范**：新增/修改 `tokens.scss` 令牌、评分展示规范 → 更新 §9；
5. **页面规格**：页面结构、区块、交互流程、三态要求的实质性变化 → 更新 §7；组件 Props/职责变化 → 更新 §8；
6. **阶段范围**：某功能从"待开发"进入"进行中/已交付"，或阶段归属调整 → 更新顶部阶段横幅与 §2。

**一致性要求**：

- 本手册**必须与 [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) 及 [`backend_AGENTS.md`](./backend_AGENTS.md) §8 保持一致**；冲突时以开发计划与后端契约文档为准，并**立即修正本文件**；
- 接口字段与路径以 `backend_AGENTS.md` §8 + 后端 `internal/dto/` 为唯一事实源，不得凭记忆书写；
- 违反同步要求的 PR 视为未完成（见 §14 通用 DoD）。

## 14. 验收标准（DoD）

### 14.1 通用 DoD（所有故事适用）

功能可用 + 三态完整 + 权限正确 + 构建通过；`vue-tsc` 零错误；`npm run build` 通过；涉及本手册 §13 所列内容的改动**已同步文档**。

### 14.2 Sprint 1

| 故事 | 前端验收要点 |
|------|-------------|
| S1.1 | `npm run dev` 启动无报错；全部路由可导航；`npm run typecheck` / `build` / `lint` 全绿；构建产物无错 |
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
| T1.12 / S6.4 | 教师画像页展示评价次数 n；`n<3` 有「样本不足」标记；**默认按姓名排序**；显示「不作为考核依据」 |
| T1.13 / S6.5 | 教师画像详情含综合分 + 5 维 + 按课程明细 + 历次时间线；`flags` 可见 |
| T1.14 / S6.6 | 教学提优页保留课程基本信息与资源上传；只读展示督导分与评语 |
| T1.15 / S6.7 | `/supervision` 完整分页列表可用，且不再重复统计卡/覆盖率卡；主任工作台改为「本室质量热力 + 重点关注」（不再有教师开课表格） |
| 一致性 | 同一教师，主任端与教师端综合分、各维度分**逐位相同** |
| 空数据 | 无评价教师综合分显示「暂无评价」并置底，**不得显示 0** |

### 14.4 Sprint 2 阶段②（智能体接入）

| 任务/故事 | 前端验收要点 |
|-----------|-------------|
| T2.7 / S7.1 | ✅ **已实现**：音频播放器 + 转写查看器三态（pending/running/done/failed）可用；失败可重试；轮询在离开页面后停止（内联 `setTimeout` + `onUnmounted` 清理） |
| T2.8 / S7.2 | 「智能体参考」面板标注「AI 参考」；低置信度维度有提示；`objective` 为 `null` 时显示空而非 0。**前端已接真实 `GET /sessions/:id/evaluation` 的 `agentScore`；真实 AI 评分推理管线未实现，agent 评价行当前由后端 seed 提供** |
| T2.9 / S7.3 | 综合分随智能体评价接入更新（后端聚合已支持双源）；`disjoint` 场景有显式提示，不静默 |
| 隐私 | ✅ **已实现**：教师端看不到音频播放器（`playbackUrl` 为空即不渲染）；转写文本已脱敏 |

### 14.5 Sprint 3（帮教师）

| 任务/故事 | 前端验收要点 |
|-----------|-------------|
| T3.1 / S8.1 | 教师端可见智能体提优建议，且与课堂记录对应、按时间倒序（**当前由 `mocks/teacherImprove.ts` 演示数据驱动，真实接口未实现**） |
| T3.2 / S8.3 | 趋势折线可按维度切换（**当前演示数据驱动，真实接口 `GET /teachers/:id/score-trend` 未实现**） |
| T3.3 / S8.2 | SSE 流式输出；中断可恢复；失败不阻塞页面（**当前用 `setTimeout` 分包模拟流式，真实 SSE `POST /agent/chat` 未实现**） |
| T3.4 / S8.4 | 教师可提交申诉、督导可复核，状态可追溯（**未实现**：无对应路由/页面/接口） |

---

## 更新记录

| 版本 | 日期 | 变更 |
|------|------|------|
| v1.0 | Sprint 1 | 初版：技术栈、目录结构、设计令牌、编码规范、Sprint 1 页面规格与验收 |
| v2.0 | 2026-09 | 覆盖三个 Sprint：新增当前阶段横幅与三 Sprint 对等章节；补全 Sprint 2/3 页面、路由变更表、组件与目录（§5–§8）；接口层修正（`/teacher-scores` 与 `/teachers` 分离、`GET /courses/:id/sessions`、`/sessions/...`、两个 `evaluation-summary`、`evaluatedCount`、`40002` = HTTP 400）；新增评分展示规范（§9.6）；新增「文档同步要求（强制）」（§13）；补充阶段①②③验收标准（§14） |
| v2.1 | 2026-09-22 | 交付「教学提优」页（任务五、六）：§7.11 补全七大区块与数据源；新增 `ScoreSummaryPanel` / `AgentSuggestionList` / `AgentChat` 组件规格，`ScoreTrendChart` Props 更新（§8.4）；更新 mock 约定（§10）——阶段③智能体区块真实接口留空，由 `src/mocks/teacherImprove.ts` 演示数据驱动；「我的课程」入口改跳 `/courses/:id/improve` |
| v1.3 | 2026-10-06 | 督导课程复盘与教师授课记录：督导侧边栏恢复「课程列表」（`mine=1` 只列本人负责评估的课程）并新增 `/supervision/courses/:id`（§7.5A）；待评课队列清除已评估记录（§7.3）；`TranscriptViewer` 预览 + 居中弹窗展开（§7.10）；教师「授课快照」改跳当堂课评价页并展示督导 + AI 评价与转写入口（§7.10）；`GET /courses` 增 `mine` 参数 |
| v1.4 | 2026-10-07 | 文档整合：并入原《前端设计-new》（该文件已删除），修正阶段状态/技术栈/权限矩阵/目录/组件/契约，补齐 ProfileQualityView 与 DraftBoxView 规格。要点：① 阶段横幅与 §2 改为「阶段① 已交付 / 阶段② 前端已接入 / 阶段③ 演示数据」；② §4 移除 ECharts、取消 `VITE_USE_MOCK`、版本对齐 `frontend/package.json`；③ §3.1 补 `/me/quality`、`/drafts`、`/supervision/courses/:id` 与路由级权限真值；④ §5 目录树、§8 组件清单按代码全量校正（移除不存在的 `ScoreSummaryPanel` 与 `useTranscriptionPolling.ts`，补齐 `AiBadge`/`QualityBadge`/`AnchorScale`/`ScoreDimensionsCard`/`ScoreHeatmap`/`QualityCompass`/`EvaluationQueue`/`TeacherScoreCard`/`EvaluationTimeline`）；⑤ §7.2/7.3/7.7-7.11 页面规格校正，新增 §7.13 我的质量档案、§7.14 草稿箱；⑥ §9 折叠原《前端设计-new》仍有效内容（核心叙事、三视觉签名、AI 虚线卡约定）并按 `tokens.scss` 全量对齐令牌；⑦ §10 类型补 `RecordingDTO`/`TranscriptDTO`/`PendingSessionItem`，契约折叠为指向 `backend_AGENTS.md` §8 的索引并修正「录音转写已实现」；⑧ §11/§13/§14 校正轮询实现、文档集与 DoD，删除已废弃文档引用 |

> **版本维护约定**：本手册的版本号随任一强制同步项（§13）的变更递增，并在上表登记。修改本文件时，请一并核对 [`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md) 是否需同步更新。
