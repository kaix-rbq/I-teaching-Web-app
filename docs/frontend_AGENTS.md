# 「爱教学」前端编码智能体指导手册

> **文档用途**：本文件是写给 AI 智能体（编码助手、架构助手，运行于 TRAE / OpenCode）的前端编码工作宪法。
> 智能体在为本项目编写任何前端代码之前，**必须先完整阅读本文件**，并严格遵守其中的技术栈、目录结构、组件设计、美术规范与编码规则。
> **使用方式**：前端仓库 `aijiaoxue-web` 初始化后，本文件应置于仓库根目录。

---

## 1. 项目背景

「爱教学」是面向高校（参考东北大学教学管理场景）的**教学质量全链路数字化管理平台**，产品愿景为"让教学质量持续可测"，走三阶进化路线：

| 进阶 | 主题 | 说明 | 迭代 |
|------|------|------|------|
| 进阶 1 | **查课程** | 全量课程信息透明可查（本轮聚焦） | Sprint 1 |
| 进阶 2 | 看课堂 | AI 听评课与课堂评估（面向督导） | Sprint 2 |
| 进阶 3 | 帮教师 | 质量分析报告与教学改进建议（面向教师） | Sprint 3 |

## 2. Sprint 1 目标与范围

### 2.1 必须完成的故事（来自用户故事地图）

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

### 2.2 范围边界

> **⚠️ 范围更新（Sprint 2/3 已启动）**
> 原 Sprint 1 Won't 清单中的「课堂录音上传 / 语音转写 / AI 评估」与「质量分析报告 / 教学改进建议」**已解除**，正式进入 Sprint 2「看课堂」与 Sprint 3「帮教师」。
> 新的用户故事、页面跳转链路、路由表、组件清单与评分展示规范见 **[`Sprint2-3-教学评价与提优-开发计划.md`](./Sprint2-3-教学评价与提优-开发计划.md)** §7。

**Sprint 2/3 仍然不做（Won't）**：

- ❌ 移动端 App / 移动端深度适配（桌面优先，目标分辨率 ≥1280px）
- ❌ 用户注册、找回密码（账号由管理员预置）
- ❌ 真实教务系统对接（数据用自建种子数据）
- ❌ 微格教学资源库
- ❌ 引入第二套 UI 组件库

**智能体守则：收到超出上述边界的编码请求时，应在回复中明确指出"该需求超出当前 Sprint 范围"，并建议先与产品负责人确认，不得直接实现。**

## 3. 用户角色与权限矩阵

三种角色，英文标识全小写，贯穿路由守卫、菜单渲染、接口鉴权：

| 角色 | 标识 | 核心场景 | 角色主题色 |
|------|------|---------|-----------|
| 教研室主任 | `director` | 查看本室课程、统筹排课、维护课程信息 | 蓝 `#2F54EB` |
| 教师 | `teacher` | 查看本人课程、备课（资源/学生人次）、维护资源 | 青 `#13C2C2` |
| 教学督导 | `supervisor` | 查看全校课程、督导覆盖率、听评课安排 | 紫 `#722ED1` |

### 3.1 页面权限矩阵（✓ 可访问 · ✗ 禁止）

| 页面 | 路由 | director | teacher | supervisor |
|------|------|:---:|:---:|:---:|
| 登录页 | `/login` | ✓ | ✓ | ✓ |
| 工作台 | `/dashboard` | ✓ | ✓ | ✓ |
| 课程列表 | `/courses` | ✓ 本室数据 | ✓ 本人数据 | ✓ 全校数据 |
| 课程详情 | `/courses/:id` | ✓ | ✓ | ✓ |
| 新增课程 | `/courses/new` | ✓ | ✗ | ✗ |
| 编辑课程 | `/courses/:id/edit` | ✓ | ✗ | ✗ |
| 督导总览 | `/supervision` | ✗ | ✗ | ✓ |
| 个人中心 | `/profile` | ✓ | ✓ | ✓ |

### 3.2 页面内操作权限

| 操作 | 位置 | director | teacher | supervisor |
|------|------|:---:|:---:|:---:|
| 新增/编辑课程信息 | 列表页 + 详情页入口 | ✓ | ✗ | ✗ |
| 上传课程资源 | 课程详情·资源 Tab | ✗ | ✓（仅本人课程） | ✗ |
| 删除课程资源 | 课程详情·资源 Tab | ✗ | ✓（仅本人课程） | ✗ |
| 下载课程资源 | 课程详情·资源 Tab | ✓ | ✓ | ✓ |
| 按教师/教研室筛选 | 课程列表筛选栏 | ✓ | ✗（自动限定本人） | ✓ |

**权限实现三原则**：
1. **路由级**：路由 `meta: { roles: ['director'] }` + 全局前置守卫 `router.beforeEach` 校验，未授权跳转 403 页；
2. **组件级**：操作按钮用 `v-if="auth.hasRole('director')"` 控制显隐（禁止仅用 CSS 隐藏）；
3. **数据级**：数据范围由后端按 token 裁剪，前端筛选参数只是查询条件，**不得依赖前端做数据越权防护**。

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
| 图表 | ECharts | ^5.5 | 仅督导覆盖率可选用；可用 CSS 进度条替代 |
| Mock | 环境变量切换 | — | `VITE_USE_MOCK=true` 时走 `src/mocks/` |
| 测试 | Vitest | ^1.5 | 编码助手产出的工具函数需附带测试 |

辅助插件：`unplugin-auto-import`、`unplugin-vue-components`（Element Plus 按需引入）。

## 5. 目录结构（强制）

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
    │   ├── resource.ts       # 资源列表、上传、删除
    │   └── supervision.ts    # 覆盖率、听评课安排
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
    │   └── supervision/      # 督导域业务组件
    │       ├── CoverageCard.vue
    │       └── ScheduleTable.vue
    ├── composables/          # 组合式函数，均以 use 开头
    │   ├── useAuth.ts
    │   ├── useCourseList.ts
    │   └── usePagination.ts
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
    │   └── api.d.ts          # ApiResponse<T>
    ├── views/                # 页面组件，一律以 View 结尾
    │   ├── LoginView.vue
    │   ├── DashboardView.vue
    │   ├── CourseListView.vue
    │   ├── CourseDetailView.vue
    │   ├── CourseFormView.vue     # 新增/编辑复用
    │   ├── SupervisionView.vue
    │   ├── ProfileView.vue
    │   └── error/NotFoundView.vue
    └── mocks/                # 种子数据与 mock 拦截
        ├── index.ts
        ├── courses.ts
        └── supervision.ts
```

**目录铁律**：
- 页面组件只出现在 `views/`，业务组件只出现在 `components/<domain>/`；
- `api/` 之外的任何文件**禁止 import axios**；
- 颜色、字号、间距**只允许**在 `styles/tokens.scss` 定义，业务代码引用 CSS 变量。

## 6. 路由与页面清单

| 路由 | name | 页面 | 布局 | 权限 | 对应故事 |
|------|------|------|------|------|---------|
| `/login` | login | LoginView | Blank | 公开 | S1.1 / S1.2 |
| `/` | — | 重定向 `/dashboard` | — | 登录 | S1.1 |
| `/dashboard` | dashboard | DashboardView | App | 登录 | S2.1 / S2.2 / S2.3 / S5.1 概览 |
| `/courses` | course-list | CourseListView | App | 登录 | S2.1 / S2.2 / S2.3 |
| `/courses/new` | course-new | CourseFormView | App | director | S3.1 |
| `/courses/:id` | course-detail | CourseDetailView | App | 登录 | S2.x / S4.1 / S4.2 |
| `/courses/:id/edit` | course-edit | CourseFormView | App | director | S3.1 |
| `/supervision` | supervision | SupervisionView | App | supervisor | S5.1 / S6.7 |
| `/teachers` | teacher-list | TeacherListView | App | director / supervisor | S6.4 |
| `/teachers/:id` | teacher-detail | TeacherDetailView | App | director / supervisor | S6.5 |
| `/courses/:id/improve` | course-improve | CourseImproveView | App | teacher | S6.6 / S8.1 |
| `/sessions/:id/evaluation` | session-evaluation | SessionEvaluationView | App | supervisor（他人只读） | S6.3 / S7.2 |
| `/profile` | profile | ProfileView | App | 登录 | 辅助（低优先级） |
| `/:pathMatch(.*)*` | not-found | NotFoundView | Blank | 公开 | — |

> **Sprint 2/3 路由变更说明**（详见开发计划 §7）：
> - `/supervision` **不删除**，由「督导总览」改造为「听评课管理」完整分页列表——工作台只展示前 8 条，删页会导致第 9 条以后无法访问；
> - 主任工作台移除「本室教师开课情况」表格与「近期开课」列表（与课程管理重复），改为「课程管理」「教师管理」两个入口卡；
> - 「教学提优」采用**并列路由** `/courses/:id/improve` 而非替换 `/courses/:id`：教师从「我的课程」点击时跳提优页，路径语义清晰且可继续访问原详情页。

守卫逻辑（`router/guards.ts`）：
1. 无 token 且目标非公开页 → 重定向 `/login?redirect=…`；
2. 有 token 访问 `/login` → 重定向 `/dashboard`；
3. 目标页 `meta.roles` 存在且不含当前角色 → 重定向 403 提示页；
4. 登录成功后按角色跳转：一律进入 `/dashboard`（工作台内容按角色渲染）。

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

- 侧边栏菜单**按角色渲染**：
  - 主任：工作台 / 课程管理 / 个人中心
  - 教师：工作台 / 我的课程 / 个人中心
  - 督导：工作台 / 全校课程 / 督导总览 / 个人中心
  - 实现：单一菜单配置数组 + `roles` 过滤，**禁止为每个角色写一份菜单**；
- 菜单项：40px 高，激活态浅蓝底 `--color-primary-bg` + 主色文字 + 左侧 3px 指示条；
- 顶栏右侧：用户姓名 + `RoleTag` + 下拉（个人中心 / 退出登录）。

### 7.3 DashboardView 工作台（三角色差异化）

页面结构 = 问候区 + 统计卡行 + 内容区（两栏 2:1）。

| 区块 | director | teacher | supervisor |
|------|----------|---------|------------|
| 统计卡（4 张 StatCard） | 本室课程数 / 本室教师数 / 本学期开课班次 / 课程资源总数 | 我的课程数 / 授课班级数 / 学生总人次 / 资源总数 | 全校课程数 / 本学期听评课计划数 / 已完成听评课 / 督导覆盖率 |
| 主内容（左 2/3） | 本室教师开课情况 CourseTable（前 10 条，点击进详情） | 我的课程 CourseCard 网格（点击进详情） | 听评课安排 ScheduleTable（前 8 条 + "查看全部"跳 `/supervision`） |
| 侧内容（右 1/3） | 快捷操作：新增课程（S3.1）/ 近期开课列表 | 近期上传资源 + 资源快捷上传入口（跳详情页资源 Tab） | CoverageCard 覆盖率环形进度 + 按教研室的覆盖率排行 |

> 工作台数据由后端按角色返回（`GET /api/v1/dashboard`），前端一套模板 + 角色配置渲染。

### 7.4 CourseListView 课程列表（S2.1 / S2.2 / S2.3）

```
PageHeader（标题随角色：「课程管理」/「我的课程」/「全校课程」 + 主任右侧「新增课程」按钮）
FilterBar：学期下拉 · 教研室下拉(主任/督导) · 教师下拉(主任/督导) · 状态 · 关键词搜索 · 重置
CourseTable：课程编码 | 课程名称 | 授课教师 | 教研室 | 学期 | 班级数 | 学生人次 | 资源数 | 状态 | 操作
分页器（右下，10/20/50 条每页）
```

- 数据范围由后端裁剪（主任→本室、教师→本人、督导→全校），前端列配置三角色一致；
- 行点击进详情；操作列：查看（全员）、编辑（主任，S3.1）；
- 列表三态：加载中（骨架屏）/ 空数据（EmptyState + 引导文案）/ 加载失败（重试按钮）。

### 7.5 CourseDetailView 课程详情（S2.x / S4.1 / S4.2）

```
PageHeader：返回 + 课程名称 + 课程编码 Tag + 状态 Tag + 操作（主任：编辑；教师：上传资源）
概要卡：学分 / 学时 / 学期 / 教研室 / 授课教师 / 学生人次（大数字 StatCard 风格）
Tabs：
  ① 基本信息 —— 课程简介（富文本只读）、培养目标、适用专业
  ② 开课信息 —— 班级表格：班级名 | 上课时间 | 地点 | 学生数（S2.1 排课依据）
  ③ 课程资源 —— ResourceList + ResourceUploader（教师可见，S4.1 / S4.2）
```

- 学生人次 = 各班级学生数之和，在概要卡以 StatCard 呈现（S4.1）；
- 资源 Tab：教师角色且 `course.teacherId === user.id` 时显示上传与删除；其他角色只读列表 + 下载；
- 资源列表列：文件名 | 类型图标 | 大小 | 上传人 | 上传时间 | 操作（下载 / 删除）。

### 7.6 CourseFormView 课程新增/编辑（S3.1，仅主任）

- 路由 `/courses/new` 与 `/courses/:id/edit` 复用同一组件，依据 `route.params.id` 区分模式；
- 表单字段：课程编码（编辑态只读）、课程名称、所属教研室（下拉）、授课教师（下拉，限本室）、学期（下拉）、学分（1–6）、学时、课程简介（textarea，≤500 字）、状态；
- 校验规则：必填项、学分/学时数字范围、编码格式 `^[A-Z]{2,4}\d{4}$`；
- 提交：主按钮"保存"→ 成功 `ElMessage.success` → 跳转详情页；次按钮"取消"→ 返回上一页，未保存变更需 `ElMessageBox.confirm` 二次确认；
- 编辑模式进入时拉取详情回填，加载中显示骨架屏。

### 7.7 SupervisionView 督导总览（S5.1，仅督导）

```
统计卡行：全校课程数 / 本学期听评课计划 / 已完成 / 覆盖率(%)
CoverageCard：总体覆盖率环形进度 + 按教研室覆盖率水平条形列表
ScheduleTable 听评课安排：课程名称 | 授课教师 | 督导人 | 计划日期 | 状态(计划中/已完成) | 筛选(状态/日期)
```

- 覆盖率 = 已完成听评课课程数 / 全校开设课程数，由后端计算，前端只展示；
- 覆盖率可视化优先用 CSS/Canvas 进度环（SVG circle），ECharts 作为备选。

### 7.8 ProfileView 个人中心（辅助，最低优先级）

- 展示：头像、姓名、角色、所属教研室/工号、账号；
- Sprint 1 只读展示 + 退出登录，**不做资料编辑**（超范围）。

## 8. 组件设计清单

### 8.1 布局组件（`layouts/` + `components/common/`）

| 组件 | Props（概要） | 职责 | 消费方 |
|------|--------------|------|--------|
| AppLayout | — | 侧边栏 + 顶栏 + 内容区骨架 | 已登录路由 |
| BlankLayout | — | 无框架容器 | 登录/404 |
| PageHeader | `title, subtitle` + slot#actions | 页面标题 + 操作区，统一页首留白 | 所有内容页 |
| StatCard | `label, value, unit, icon, tone` | 指标卡（图标 + 大数字 + 标签） | 工作台/详情/督导 |
| FilterBar | slot + `@search/@reset` | 筛选栏容器，收拢查询交互 | 课程列表/督导 |
| EmptyState | `description` + slot#action | 空数据占位（引导操作） | 所有列表 |
| RoleTag | `role` | 角色标签，三角色三色 | 顶栏/表格 |

### 8.2 课程域组件（`components/course/`）

| 组件 | Props（概要） | 职责 | 对应故事 |
|------|--------------|------|---------|
| CourseTable | `rows, loading, showEdit` | 课程数据表格 + 行操作 | S2.1–S2.3 |
| CourseCard | `course` | 课程卡片（工作台教师视图） | S2.2 |
| CourseInfoForm | `modelValue, mode` | 新增/编辑表单 + 校验 | S3.1 |
| ResourceList | `resources, canManage` | 资源列表 + 下载/删除 | S4.1 / S4.2 |
| ResourceUploader | `courseId, accept, maxSize` | 拖拽 + 点击上传，进度与类型校验 | S4.2 |

### 8.3 督导域组件（`components/supervision/`）

| 组件 | Props（概要） | 职责 | 对应故事 |
|------|--------------|------|---------|
| CoverageCard | `overall, byDepartment[]` | 覆盖率环形 + 分组条形 | S5.1 |
| ScheduleTable | `plans, loading` | 听评课安排表 | S5.1 |

### 8.4 组件编写规则

- 一律 `<script setup lang="ts">`；Props 用 `defineProps<{…}>()` 类型声明；Emits 用 `defineEmits<{…}>()`；
- 组件只负责**展示与交互**，数据请求放在页面或 composable，通过 props 下发；
- 对外透传 UI 库组件属性时使用 `inheritAttrs: false` + `v-bind="$attrs"`；
- 每个业务组件 props 中必须有 `loading`（或等价）状态，支撑骨架屏/禁用态。

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
--font-size-stat: 28px; // StatCard 大数字（tabular-nums）
```

- 行高：正文 1.6，标题 1.3；
- 字重：标题 600，正文 400，StatCard 数字 600 + `font-variant-numeric: tabular-nums`。

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
- 破坏性操作（删除资源）：必须 `ElMessageBox.confirm` 二次确认，并写明对象名称；
- 成功反馈统一 `ElMessage.success`；失败提示由 `api/http.ts` 拦截器统一弹出，**页面层不得重复弹错**；
- 上传约束：类型白名单 `pdf/doc/docx/ppt/pptx/mp4/zip`，单文件 ≤100MB，显示进度条，失败可重试。

## 10. 数据模型与接口约定

### 10.1 核心类型（`src/types/`）

```ts
type Role = 'director' | 'teacher' | 'supervisor';

interface User {
  id: string;
  name: string;
  role: Role;
  department?: string;   // 教研室
  jobNo?: string;        // 工号
}

interface ClassInfo {
  id: string;
  className: string;     // 如 "软件 2201"
  schedule: string;      // 如 "周一 3-4 节"
  location: string;
  studentCount: number;
}

interface Course {
  id: string;
  code: string;              // 如 "SE3101"
  name: string;
  credit: number;
  hours: number;
  semester: string;          // 如 "2026-2027-1"
  department: string;        // 教研室
  teacherId: string;
  teacherName: string;
  description: string;
  classes: ClassInfo[];
  studentCount: number;      // 各班学生数之和（后端汇总）
  resourceCount: number;
  status: 'open' | 'draft' | 'closed';
}

interface Resource {
  id: string;
  courseId: string;
  name: string;
  type: 'pdf' | 'doc' | 'ppt' | 'video' | 'zip' | 'other';
  size: number;              // 字节
  uploader: string;
  uploadedAt: string;        // ISO 8601
}

interface SupervisionPlan {
  id: string;
  courseId: string;
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

### 10.2 接口清单（RESTful，前缀 `/api/v1`）

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
| GET | `/supervision/coverage` | supervisor | 覆盖率统计 |
| GET | `/supervision/plans` | supervisor | 听评课安排（分页 + 状态筛选） |
| GET | `/departments` | 登录 | 教研室列表（课程列表筛选下拉数据源，`[{id, name}]`） |
| GET | `/teachers` | 登录 | 教师列表，查询参数 `departmentId`（筛选下拉数据源，`[{id, name}]`） |

### 10.3 请求层约定

- 统一响应 `{ code, message, data }`：`code === 0` 成功；`401` 清 token 跳登录；其余弹 `ElMessage.error(message)`；
- 鉴权头：`Authorization: Bearer <token>`，token 存 `localStorage`（key：`aijiaoxue_token`）；
- Mock 策略：`.env.development` 中 `VITE_USE_MOCK=true` 时由 `src/mocks/` 拦截，**接口函数签名与真实后端完全一致**，保证无缝切换联调。

## 11. 状态管理与数据流

```
views ──调用──> api/*（唯一请求出口）
  │                 │
  ├── composables/useCourseList（列表查询状态内聚：rows/loading/query/page）
  └── stores/auth（token/user/hasRole）  stores/dict（学期、教研室字典）
```

- `stores/auth.ts`：`login()` / `logout()` / `hasRole(role)` / `state: { token, user }`，持久化 token；
- 列表类数据**不进 Pinia**，用 composable 内聚（分页、筛选、加载态），避免全局状态膨胀；
- 字典（学期、教研室）启动时拉取一次缓存于 `stores/dict.ts`；
- 组件间通信：父子 props/emits 为主，跨层级少量场景用 `provide/inject`，**禁止滥用全局事件总线**。

## 12. 编码规范（强制）

1. **Composition API only**：全部 `<script setup lang="ts">`，禁止 Options API；
2. **命名**：组件多词 PascalCase（文件名与组件名一致）；页面组件以 `View` 结尾；composable 以 `use` 开头；类型以大驼峰；常量全大写下划线；
3. **禁止 `any`**：类型不明确时用 `unknown` + 收窄，或补充类型定义；
4. **禁止硬编码**：颜色/字号/间距用 CSS 变量；魔法字符串用 `const` 枚举（角色、状态）；
5. **禁止在组件内直接调用 axios**：请求只出现在 `src/api/`；
6. **单向数据流**：子组件不修改 props，通过 emit 通知父级；
7. **样式**：`<style scoped lang="scss">`；公共样式进 `styles/`；类名用 BEM（`course-table__row--active`）；
8. **路由**：`name` 全局唯一；`meta.title` 必填（面包屑依赖）；
9. **提交前自检**：`npm run build` 零错误、`npm run lint` 零 error；新增工具函数附带 Vitest 单测；
10. **Git 提交**：Conventional Commits —— `feat(course): 新增课程列表筛选` / `fix(login): 修复 token 失效未跳转`；分支命名 `feature/S2.1-course-list`（故事编号 + 短描述）。

## 13. 验收标准（DoD）

每个故事完成的定义——功能可用 + 三态完整 + 权限正确 + 构建通过：

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
