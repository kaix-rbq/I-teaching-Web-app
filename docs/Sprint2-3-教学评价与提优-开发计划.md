# 「爱教学」Sprint 2 / Sprint 3 开发计划 —— 课堂评价与教学提优

> **文档定位**：本文件是 Sprint 2「看课堂」与 Sprint 3「帮教师」的 **迭代范围 / 评分体系 / 权限隐私合规 / 任务与状态 / DoD / 后续方向** 的唯一事实源。
> 其他主题各有归属，本文件不再复制其内容，只给索引：
>
> - **接口契约** → [`docs/backend_AGENTS.md`](./backend_AGENTS.md) §8（§8.8 授课记录与评价聚合 · §8.9 课堂录音与转写）
> - **建表 DDL / 迁移** → [`backend/database/schema.sql`](../backend/database/schema.sql) + [`backend/migrations/`](../backend/migrations/)
> - **页面 / 路由 / 组件** → [`docs/frontend_AGENTS.md`](./frontend_AGENTS.md) §6 / §7
> - **建库 / 种子 / 备份** → [`docs/MySQL数据库创建指导.md`](./MySQL数据库创建指导.md)
> - **种子数据** → [`backend/database/seed.sql`](../backend/database/seed.sql)
>
> **版本**：v1.4（2026-10-07 合并任务清单、DDL/接口/页面规格改为指向单一事实源，见文末「更新记录」）
> **使用方式**：本文件只描述**要开发什么、按什么规范做**。§10 汇总了必须遵守的硬性约定，变更任一条需先更新本文档再改代码（契约先行）。

---

## 1. 范围与目标

### 1.1 三阶段推进（正式排期与当前状态）

| 阶段 | 主题 | 核心内容 | 是否依赖智能体 | 当前状态 |
|------|------|---------|--------------|---------|
| **阶段一**（Sprint 2.1） | 评价闭环 | 授课记录 · 督导评分 · 结构化评语 · 三级聚合 · 教师画像 · 教师提优页（只读督导分） | ❌ 不依赖 | ✅ **已实现** |
| **阶段二**（Sprint 2.2） | 智能体接入 | 课堂录音 · 异步转写 · 智能体评分 · 综合分（双侧融合）· 督导页智能体参考面板 | ✅ | 🟡 **部分实现** |
| **阶段三**（Sprint 3） | 帮教师 | 智能体提优建议 · 对话（SSE）· 趋势视图 · 申诉复核 · 质量报告导出 | ✅ | ❌ **未实现** |

**阶段二的实现边界（逐项对照代码，不得按"已完成"整体对待）**：

| 子项 | 状态 | 证据（路径:行） |
|------|------|----------------|
| 录音上传 / 流式播放（短时票据） | ✅ | `backend/internal/service/recording.go:81,166`；票据路由 `backend/internal/router/router.go:84` |
| 异步转写（状态机 + 失败重试 + 启动重排） | ✅ | `service/recording.go:188,214,239`；`repository/recording.go:71` |
| ASR 适配（Qwen-Audio / 百炼） | ✅ | `backend/internal/asr/client.go:70,99` |
| 转写脱敏（学生姓名替换） | ✅ | `backend/internal/service/desensitize.go:147`；调用点 `service/recording.go:296` |
| 播放审计（谁听了谁的课） | ✅ | `backend/internal/repository/recording.go:82`；调用点 `service/recording.go:184` |
| 前端播放器 / 转写查看器 / 双源对照面板 | ✅ | `frontend/src/views/SessionEvaluationView.vue:499,517,523` |
| **智能体评分写入 `evaluations` 的 agent 行** | ❌ **未接入** | `repository/recording.go:79` `UpsertAgentEvaluation` **无任何调用方** |

> 🔴 **阶段二的真实缺口**：智能体评分无写入通道 → 运行期除种子数据外 `evaluations` 只有 `supervisor` 行。聚合侧已支持 `agent` 侧（§2.5.4），但线上综合分实际仍为督导单侧。
> 🔴 `router.go` 中**没有任何 `/agent/*` 路由**，`50002`（`NotImplemented`）虽在 `pkg/errcode` 定义并映射 HTTP 501，但**无调用方**（详见 §4.4）。

> **核心架构：评分源可插拔**
> `supervisor` 与 `agent` 只是 `evaluations.evaluator_type` 的两个取值，向同一张表写同样的结构；聚合层只认该表，不关心分数从哪来。
> 收益：① 智能体延期不阻塞阶段一交付；② 智能体效果不佳时可降权甚至停用；③ 未来加「同行评议」「学生评教」只需新增一个枚举值，不改表、不改聚合。

### 1.2 用户故事

| 编号 | 阶段 | 角色 | 故事 | 优先级 | 落点 |
|------|------|------|------|--------|------|
| S6.1 | 一 | 督导 | 为一次课建立授课记录（课程/班级/日期/节次/主题），以便评价有落点 | M | `POST /sessions` |
| S6.2 | 一 | 任一 | 在课程详情页查看该课程的历史授课记录列表 | M | `GET /courses/:id/sessions` |
| S6.3 | 一 | 督导 | 在当堂课评估页按 5 个维度打分并留下结构化评语 | M | `PUT /sessions/:id/supervisor-evaluation` |
| S6.4 | 一 | 主任 | 在教师画像页查看本室教师综合评分列表 | M | `GET /teacher-scores` |
| S6.5 | 一 | 主任 | 点击教师查看其评分数据面板（综合分 + 分维度 + 分课程） | M | `GET /teachers/:id/evaluation-summary` |
| S6.6 | 一 | 教师 | 在教学提优页查看本人各维度评分与督导评语 | M | `GET /courses/:id/evaluation-summary` |
| S6.7 | 一 | 督导 | 工作台与「听评课管理」页分离，完整安排列表可分页筛选 | S | `GET /supervision/plans` |
| S7.1 | 二 | 督导 | 上传课堂录音并查看转写文本 | M | `POST /sessions/:id/recording` · `GET /sessions/:id/transcript` |
| S7.2 | 二 | 督导 | 查看智能体对该堂课的分维度评分作为参考 | M | 评估页聚合接口 |
| S7.3 | 二 | 任一 | 综合分 = 督导评分与智能体评分的加权融合 | M | 聚合服务 |
| S7.4 | 二 | 教师 | 在教学提优页看到综合分随智能体接入而变化 | S | 同 S6.6 |
| S8.1 | 三 | 教师 | 查看智能体针对课堂记录给出的提优建议 | M | 评估聚合 |
| S8.2 | 三 | 教师 | 与智能体就课堂改进对话 | C | 新增 `POST /agent/chat`（SSE，**未实现**，见 §4.4） |
| S8.3 | 三 | 教师 | 查看本人各维度分数的历史趋势 | S | 新增趋势接口（**未实现**） |
| S8.4 | 三 | 教师 | 对评分提出申诉，督导复核 | S | 新增申诉表（**未实现**） |

> `[M]` Must；`[S]` Should；`[C]` Could。
> `S6.4` 的落点是 `GET /teacher-scores`（**不是** `GET /teachers`，后者永远是教师字典，见 §4.2）。

### 1.3 范围边界（本轮 Won't）

- ❌ 移动端 App / 移动端深度适配（桌面优先不变）
- ❌ 用户注册、找回密码、第三方登录（账号仍由种子数据预置）
- ❌ 真实教务系统对接（授课记录由人工建立，不做课表自动同步）
- ❌ 实时课堂直播 / 实时转写（只做**课后**上传与转写）
- ❌ 多模态视频画面分析（只做**音频**链路）
- ❌ 引入第二个 Web 框架、ORM 或 UI 组件库
- ❌ 微服务拆分、读写分离、全文搜索引擎

---

## 2. 评分体系

> 本节是本文件最核心、不可外移的内容：维度、权重、锚点、单次总分、三级聚合与缺失处理，其他文档只引用不复制。

### 2.1 评分维度（冻结版 v1）

**维度一旦上线不得随意增删**——改了历史分数就不可比。共 5 个维度：

| # | key | 维度名 | 生效权重 | 智能体可评 | 说明 |
|---|-----|--------|---------|-----------|------|
| 1 | `objective` | 教学目标与内容准确性 | **30%** | ⚠️ 部分 | 讲对了没有是底线维度 |
| 2 | `content` | 内容质量与深度（有干货不水课） | **30%** | ⚠️ 部分 | — |
| 3 | `interaction` | 学生互动与参与 | **20%** | ✅ 可 | — |
| 4 | `organization` | 课堂组织与节奏 | **20%** | ✅ 可 | — |
| 5 | `frontier` | 前沿与交叉学科 | **0%（观测项）** | ⚠️ 部分 | 不计入加权总分，仅单独展示为"亮点标记" |

**两条命名与权重的设计依据**：

- `organization` 替代「课堂纪律」：**安静的课堂不等于优质课堂**。若以音量低/静音占比高作为正向证据，会奖励满堂灌、惩罚翻转课堂，并与 `interaction` 维度互相抵消。锚点定义为"教学环节清晰、时间分配合理、纪律问题处理得当"。
- `frontier` 设为观测项：基础课（操作系统、计算机组成原理等）天然不具备前沿内容，若计入加权会使其任课教师**结构性低分**，并倒逼教师往基础课里硬塞前沿内容。故仅作亮点展示，不参与评分。

**权重依据**：内容本身（1+2）合计 **60%** 是教学质量主体，互动 20%，组织 20%。**生效权重合计必须为 1.00，服务启动时校验。**

> **智能体使用同一套维度，但允许 `null`。** 智能体没有教材与教学大纲，无法可靠判断"内容准确性"，只能观测语速、停顿、提问频次、师生话轮比等**声学/语言特征**。`null` 表示"该源无法评价此维度"，该维度在融合时自动只取另一侧的值（§2.5.2）。**禁止让智能体对无法观测的维度强行给分。**

### 2.2 权重配置

```yaml
evaluation:
  weights:                       # 生效权重，非零项合计必须为 1.00，服务启动时校验
    objective: 0.30
    content: 0.30
    interaction: 0.20
    organization: 0.20
    frontier: 0.00               # 观测项，不计入加权
  supervisorWeight: 0.5          # α：综合分中督导评分权重
  agentWeight: 0.5               # 1−α：综合分中智能体评分权重
  formulaVersion: "v1"           # 口径版本，变更时递增
  minSampleSize: 3               # 低于此值标记"样本不足"
```

### 2.3 评分锚点（必须写入 UI tooltip）

督导之间打分尺度不一致是本系统最大的可比性风险。**每个维度的每个分值必须有可操作的行为描述**，否则分数不可比。

**通用锚点**：

| 分 | 等级 | 通用描述 |
|----|------|---------|
| 5 | 优秀 | 可作为示范课 |
| 4 | 良好 | 达到骨干教师水平 |
| 3 | 合格 | 达到基本教学要求 |
| 2 | 待改进 | 存在明显短板 |
| 1 | 不合格 | 需要立即干预 |

**维度专属锚点（节选，完整版随 UI 交付）**：

| 维度 | 5 分 | 3 分 | 1 分 |
|------|------|------|------|
| `objective` | 目标明确，内容准确无错误，重难点突出 | 目标基本清晰，无实质性知识错误 | 目标缺失或存在知识性错误 |
| `content` | 内容充实有深度，理论联系实际，无水分 | 内容完整但以照本宣科为主 | 内容空洞或明显偏离课程大纲 |
| `interaction` | 有效提问与讨论充分，学生参与度高 | 偶有提问但以自问自答为主 | 全程单向讲授，无任何互动 |
| `organization` | 环节清晰，时间分配合理，节奏张弛有度 | 环节完整但时间分配略显失衡 | 结构混乱或严重拖堂/提前下课 |
| `frontier` | 自然融入学科前沿或交叉应用，与主线结合紧密 | 提及前沿但较生硬 | 无前沿内容（**不单独作为扣分依据**） |

> ⚠️ 两条硬性约束：`organization` **不得以"学生安静/音量低"作为正向证据**；`frontier` **不得因基础课性质而扣分**。

### 2.4 单次评价得分

评分录入用 **1–5 分整数**（李克特量表），不用 0–100——督导在课堂上边听边打分，5 档可操作，100 档是假精度。

```
dim_i = (v_i − 1) / 4 × 100                维度分，v_i ∈ {1,2,3,4,5}
total = Σ w_i × dim_i                        w 取 §2.1 生效权重（frontier = 0）
```

使用 `(vᵢ−1)/4` 而非 `vᵢ/5`，是为避免"最低分永远是 20 分"造成的**分数压缩**（否则无法区分"很差"与"极差"）。值域完整覆盖 `[0, 100]`。

**验算**（`objective=5, content=4, interaction=3, organization=5, frontier=2`）：

```
dim  = (100, 75, 50, 100, 25)
total = 0.30×100 + 0.30×75 + 0.20×50 + 0.20×100 + 0×25
      = 30.00 + 22.50 + 10.00 + 20.00 + 0
      = 82.50 分          ← frontier 的 25 分不参与计算
```

**智能体侧的置信度加权**：智能体对每个维度输出 `{score, confidence, evidence}`，其中：

- `confidence ∈ [0,1]`：模型自评把握度
- `evidence`：支撑该分数的转写片段引用 `[{start, end, quote}]`

维度级置信度加权：`v̄ = Σ(cᵢ·vᵢ) / Σcᵢ`；若某维度 `c < 0.4`，该维度置 `null` 并标记"低置信度"。

> **没有 `evidence` 的智能体分数不予采信**——督导看到"互动 2 分"却不知道为什么，无法复核，界面就失去参考价值。

### 2.5 三层聚合与综合分

```
场次级 score_j  ──聚合──▶  课程级  ──聚合──▶  教师级
 （单次评价）              （教学提优页）       （教师画像面板）
```

#### 2.5.1 聚合函数（服务层纯函数，两级共用）

```go
// DimensionScores 是单次评价的 5 个维度原始分（1-5），nil 表示该评分源无法评价该维度。
type DimensionScores struct {
    Objective    *int
    Content      *int
    Interaction  *int
    Organization *int
    Frontier     *int
}

// SessionScore 是一个场次的双侧评价，nil 表示该侧无评价。
type SessionScore struct {
    SessionID  uint64
    CourseID   uint64
    Supervisor *DimensionScores
    Agent      *DimensionScores
}

// Aggregate 是课程级与教师级共用的唯一聚合入口。
// 传教师的全部场次即得教师级，传某课程的场次即得课程级。
func Aggregate(items []SessionScore, weights Weights, alpha float64) Summary
```

> 🔴 **教师级不得"先算课程级、再对课程取平均"**——会引入二次偏差（带 1 门课×10 次 与 带 5 门课×2 次 不可比）。**同一个函数，喂不同数据集**，这是保证「主任页与教师页数字完全一致」的唯一可靠做法。

#### 2.5.2 聚合口径：综合均值（当前实现）

```
① 单次评价的维度分（0-100 百分制）
   dim_i = (v_i − 1) / 4 × 100                       v_i ∈ {1..5}

② 单次总分（仅用于列表排序与快速展示，非聚合来源）
   total = Σ w_i × dim_i                             某侧缺维度时，权重在其余维度上重新归一化

③ 场次级双侧融合（逐维度）
   dim_i^session = α × dim_i^sup + (1−α) × dim_i^agent
   某一侧该维度为 NULL 时，该维度直接取另一侧的值

④ 课程级 / 教师级综合分（综合均值口径，当前实现）
   composite = mean over sessions( composite(场次级) )
   仅统计有评价、可算出综合分的场次；未评价场次不计入分母（不得当作 0 分）

⑤ 课程级 / 教师级维度展示分
   dim_i^level = α × mean(dim_i^sup over sessions) + (1−α) × mean(dim_i^agent over sessions)
   单侧缺失时该维度直接取另一侧（§2.5.4 SQL 的 COALESCE 三级回退）
```

> 🔴 **口径取舍（2026-09 决议）**：**教师级 / 课程级综合分取「综合均值」**——各场次综合分的算术平均。
> 这保证**教师面板的总分恒等于其各次课总分的平均值**（§8.1 恒等式在任意数据下恒定成立）。用 §3.4 种子数据验证：c1 三次课综合分 58.75 / 70.00 / 78.75，教师级为 **69.17**。
> **默认前提是数据双侧对齐**（每节课督导与智能体都有评价，或缺失模式在各场次一致）：此时维度展示分的加权求和 `Σ w_i × dim_i^level` 与综合均值逐位相等，页面数字自洽。
> 非对齐数据（部分场次仅单侧评价）下的通用算法本轮**不实现**，后续单独立项；届时必须同时更新本节与 `pkg/scoring`，不得只改代码。

#### 2.5.3 综合分的样本对齐

```
共同场次集合 J = { j | 该场次督导与智能体都有评价 }

综合分 = mean_{j∈J} ( α × 督导分_j + (1−α) × 智能体分_j )      α 默认 0.5
```

**必须按场次对齐的理由**（督导 1 次 90 分、智能体 10 次均值 60 分）：

| 口径 | 结果 | 问题 |
|------|------|------|
| 各自历史平均再平均 | `(90+60)/2 = 75` | 1 次督导意见占 50% 权重，与 10 次智能体意见等权，失真 |
| **场次对齐后融合（采用）** | `0.5×90 + 0.5×60 = 75`（仅 `J` 内 1 个场次） | 数值巧合相同，但**同时返回 `alignedCount=1 / agentCount=10`**，使用者知道这个 75 只代表 1 次课 |

两种口径的差别不在数值，而在**是否暴露真实覆盖度**。实现时必须返回 `alignedCount`。

#### 2.5.4 聚合 SQL（MySQL 8.0 CTE，维度层）

SQL 只负责把**原始 1-5 维度分**按侧取均分；`(v−1)/4×100` 换算与权重加权在 Go service 层完成（线性等价，放哪层结果相同）。

> **当前实现说明**：`evaluation.go` 的 `ListSessionDimRows` 按 §2.5.4 取「每场次每侧维度均分」（1-5 原始标度），
> 再交给 `pkg/scoring`：维度展示分按 ⑤ 的 COALESCE 回退融合；**综合分按 ④ 的综合均值口径**，即对 `SessionComposite` 求算术平均。
> 下面 SQL 中 `dim_*` 列即 ⑤ 的维度展示分；综合分不在 SQL 内计算（避免与 Go 口径不一致）。

**教师级**（课程级把 `ts.teacher_id = :teacher_id` 换成 `ts.course_id = :course_id`、`GROUP BY course_id` 即可）：

```sql
WITH session_dims AS (
  SELECT
    ts.id AS session_id,
    ts.teacher_id,
    MAX(CASE WHEN e.evaluator_type='supervisor' THEN 1 ELSE 0 END) AS has_sup,
    MAX(CASE WHEN e.evaluator_type='agent'      THEN 1 ELSE 0 END) AS has_ai,
    AVG(CASE WHEN e.evaluator_type='supervisor' THEN e.objective_score    END) AS sup_objective,
    AVG(CASE WHEN e.evaluator_type='supervisor' THEN e.content_score      END) AS sup_content,
    AVG(CASE WHEN e.evaluator_type='supervisor' THEN e.interaction_score  END) AS sup_interaction,
    AVG(CASE WHEN e.evaluator_type='supervisor' THEN e.organization_score END) AS sup_organization,
    AVG(CASE WHEN e.evaluator_type='supervisor' THEN e.frontier_score     END) AS sup_frontier,
    AVG(CASE WHEN e.evaluator_type='agent'      THEN e.objective_score    END) AS ai_objective,
    AVG(CASE WHEN e.evaluator_type='agent'      THEN e.content_score      END) AS ai_content,
    AVG(CASE WHEN e.evaluator_type='agent'      THEN e.interaction_score  END) AS ai_interaction,
    AVG(CASE WHEN e.evaluator_type='agent'      THEN e.organization_score END) AS ai_organization,
    AVG(CASE WHEN e.evaluator_type='agent'      THEN e.frontier_score     END) AS ai_frontier
  FROM teaching_sessions ts
  LEFT JOIN evaluations e
         ON e.session_id = ts.id
        AND e.formula_version = 'v1'          -- 口径版本隔离
  WHERE ts.teacher_id = :teacher_id
    AND ts.session_date BETWEEN :semester_start AND :semester_end
  GROUP BY ts.id, ts.teacher_id
)
SELECT
  teacher_id,
  COUNT(*)                    AS session_count,
  SUM(has_sup)                AS supervisor_count,
  SUM(has_ai)                 AS agent_count,
  SUM(has_sup AND has_ai)     AS aligned_count,
  -- 维度分：α 融合；单侧缺失时自动取另一侧（COALESCE 三级回退）
  COALESCE(:alpha*AVG(sup_objective)    + (1-:alpha)*AVG(ai_objective),
           AVG(sup_objective),    AVG(ai_objective))    AS dim_objective,
  COALESCE(:alpha*AVG(sup_content)      + (1-:alpha)*AVG(ai_content),
           AVG(sup_content),      AVG(ai_content))      AS dim_content,
  COALESCE(:alpha*AVG(sup_interaction)  + (1-:alpha)*AVG(ai_interaction),
           AVG(sup_interaction),  AVG(ai_interaction))  AS dim_interaction,
  COALESCE(:alpha*AVG(sup_organization) + (1-:alpha)*AVG(ai_organization),
           AVG(sup_organization), AVG(ai_organization)) AS dim_organization,
  COALESCE(:alpha*AVG(sup_frontier)     + (1-:alpha)*AVG(ai_frontier),
           AVG(sup_frontier),     AVG(ai_frontier))     AS dim_frontier
FROM session_dims
GROUP BY teacher_id;
```

> **三处易错点**：
> ① `COALESCE(a + b, x, y)` 的三级回退正好表达"双侧齐全 → 融合；仅一侧 → 取该侧；都无 → NULL"。注意 `α*x + (1−α)*NULL` 在 SQL 中求值为 NULL，回退才生效。
> ② 侧别计数必须用 `MAX(CASE ... THEN 1 ELSE 0 END)` 再 `SUM`，**不能用 `COUNT(ai_objective)`**——智能体的 `objective_score` 恒为 NULL，用它计数会得到 0。
> ③ 同一场次多督导时取 `AVG`，**不是 MAX**。

#### 2.5.5 缺失与样本不足处理矩阵（必须写死，否则前端出现 NaN）

| 督导场次 | 智能体场次 | 共同场次 | 综合分 | 返回标记 |
|---------|-----------|---------|--------|---------|
| 0 | 0 | 0 | `null` | `no_data` — 列表置底并显示"暂无评价"，**不得按 0 分排序** |
| n>0 | 0 | 0 | 督导均值 | `sup_only` |
| 0 | m>0 | 0 | 智能体均值 | `ai_only` |
| n>0 | m>0 | k>0 | 共同场次融合后取均值 | `ok`，附 `coverage: k/n` |
| n>0 | m>0 | **0** | 退化为各自均值加权 | `disjoint: true`，**必须显式提示** |

**样本量规则**（样本充足性以「已评价场次」`evaluatedCount` 为准，避免"有课但没评"被误判为样本充足）：

- `sessionCount = 0`（无授课记录）→ 综合分 `null`，不参与排序
- `evaluatedCount = 0`（有课但一次都没评）→ 综合分 `null`、`flags` 含 `no_data`，**不得显示 0**
- `evaluatedCount < minSampleSize(默认3)` → 置 `sampleSufficient: false`，前端标注"样本不足（n=x）"
- 响应同时返回 `sessionCount / evaluatedCount / supervisorCount / agentCount / alignedCount`，覆盖度一目了然

#### 2.5.6 口径版本与学期切片

- **口径版本**：维度权重可配 → 改权重会让历史分数同时变化且无法解释。因此**单次评价的 `total_score` 落库时必须同时写 `formula_version`**；改口径只对新评价生效，历史分按旧口径保留；聚合 SQL 按版本隔离，版本混杂时返回提示。
  > 这是「汇总不落库」原则的**必要例外**——该原则针对跨表聚合，单行内确定性函数不在此列。**跨表跨行的教师级/课程级聚合仍然查询时计算，不落库。**
- **时间衰减**：本轮 **不做**。它与「教学提优」诉求冲突——教师改进后应看到分数上升，用**趋势折线**体现进步比用衰减更直观、更好解释。
- **学期切片**：所有聚合接口必须支持 `?semester=`，默认当前学期。跨学期平均会抹平改进。

---

## 3. 数据模型（DDL 事实源索引）

> **事实源**：Sprint 1 存量表 → [`backend/database/schema.sql`](../backend/database/schema.sql)；Sprint 2 起新增表 → [`backend/migrations/`](../backend/migrations/)。本文件**不再复制 DDL**，只保留 SQL 里看不见的建表顺序、命名约定与非显然陷阱。
>
> **迁移方式**：按 `docs/MySQL数据库创建指导.md` §9，从 Sprint 2 起使用 `golang-migrate`，脚本以 `embed` 打包（`backend/migrations/migrations.go:12`）。
>
> 🔴 **文件命名必须遵循 golang-migrate 默认约定**：`<版本>_<名称>.up.sql` / `.down.sql`（正则 `^([0-9]+)_(.*)\.(up|down)\.(.*)$`），例如 `1_baseline_sprint1.up.sql`、`2_teaching_sessions_and_evaluations.up.sql`。
> **Flyway 风格的 `V2__xxx.up.sql` 不被识别为迁移**，会让 `migrate up` 直接报 `first .: file does not exist`（工具能加载目录但没有可用迁移）。历史上本项目曾用该命名，修复时已改名——**不得改回**。

### 3.1 阶段一迁移：`migrations/2_teaching_sessions_and_evaluations.up.sql`

| 表 | 作用 | 关键约定（详见迁移脚本） |
|----|------|------------------------|
| `teaching_sessions` | 授课记录（一切评价的落点） | `class_id BIGINT UNSIGNED NOT NULL DEFAULT 0`；`uk_session(course_id,class_id,session_date,period)`；`plan_id` 反向指回 `supervision_plans` |
| `evaluations` | 课堂评价：督导与智能体同表，靠 `evaluator_type` 区分 | `uk_eval(session_id,evaluator_type,evaluator_id)`；五维 `TINYINT UNSIGNED NULL`；`evidence JSON NULL`（仅 agent）；`total_score DECIMAL(5,2)`；`formula_version` |

- **可插拔评分源的落地**：`supervisor` 与 `agent` 只是 `evaluator_type` 的两个取值，写同样的结构；聚合层只认该表（§1.1）。
- **正式评价无草稿态**：`evaluations` 不设 `status` / `submitted_at`——写入即生效，`created_at` 即提交时间。督导评分通过 `PUT /sessions/:id/supervisor-evaluation` 幂等覆盖，不产生中间态。
- **草稿独立成表**：督导草稿存于 `migrations/5_evaluation_drafts.up.sql` 的 `evaluation_drafts`（`uk_draft(session_id,supervisor_id)`，维度分允许 `NULL`），**不进聚合口径**；提交时由 service 强制五维齐全。
- **为什么用列式而非 EAV**（`evaluation_scores(evaluation_id, dimension_key, score)`）：维度已冻结 → 列式可直接 `AVG(content_score)`、类型安全、索引友好；EAV 每次聚合都要 PIVOT，SQL 复杂且极易出错。代价是新增维度需 `ALTER TABLE`，在"维度冻结"前提下可接受。

### 3.2 阶段二迁移：`migrations/3_recordings_and_transcripts.up.sql`（+4 补默认值 / 6 放宽列宽）

| 表 | 作用 | 关键约定（详见迁移脚本） |
|----|------|------------------------|
| `recordings` | 课堂录音 | `uk_rec_session(session_id)`：一次课一条主录音；FK → `teaching_sessions` / `users` |
| `transcripts` | 异步转写产物 | `uk_tr_session`；`status ENUM(pending,running,done,failed)`；`content MEDIUMTEXT`（V4 补 `DEFAULT ('')`，防绕过 GORM 的写入触发 1364）；`engine` / `engine_version` `VARCHAR(64)`（V6 放宽，见 §3.3-6） |
| `recording_playback_logs` | **播放审计**（谁在何时听了谁的课） | `recording_id` + `user_id` + `started_at`；由 `RecordingService.Stream` 落库 |

- **转写必须异步**：45 分钟音频的 ASR 需数分钟，同步 HTTP 必然超时。`transcripts.status` 即为此设计，前端轮询；`failed` 必须可重试（`POST /sessions/:id/transcript/retry`），进程重启由 `RequeueStuck` 重排 `pending/running`。

### 3.3 必须写进迁移说明的实现陷阱（每条均已在迁移/代码中核对）

1. **建表顺序 = 外键依赖顺序。** 迁移按版本号顺序执行：V1 建 `departments` → `users` → `courses` → `course_classes` → `resources` → `supervision_plans`，V2 才能建引用 `courses` / `supervision_plans` 的 `teaching_sessions`，以及引用 `teaching_sessions` 的 `evaluations`；V3 引用 V2。被引用表必须先存在，倒序建表直接失败。
2. **`class_id` 必须 `NOT NULL DEFAULT 0`。** MySQL 唯一索引**对 NULL 不去重**——若 `class_id` 可空，`uk_session` 形同虚设，同一节课能建出无数条记录。此坑只会在线上暴露。（`migrations/2_teaching_sessions_and_evaluations.up.sql:11,21`）
3. **JSON 列禁止写入空串。** `evaluations.evidence` 是 `JSON` 列，MySQL 8 严格模式对 `''` 抛 `ERROR 3140 Invalid JSON text: "The document is empty"`。Go 模型字段必须用指针（`*string`），督导行写 `NULL`，不能依赖零值 `""`。（`backend/internal/model/evaluation.go:21-24`）
4. **种子 `INSERT` 的列数与值数必须逐行核对。** 督导评价段曾漏 `suggestions` 列，导致 `ERROR 1136 Column count doesn't match value count`，种子数据整体加载失败（现 `backend/database/seed.sql` 已逐行对齐）。
5. **DATE 列比较必须按 `YYYY-MM-DD` 绑定。** 直接把 `time.Time` 交给驱动会被带上时区换算后的时分秒（`2026-09-12 08:00:00`），与 `DATE` 值不相等 → 唯一性预检漏判，最后由数据库 1062 兜底（用户看到 50001 而不是 40901）。仓储层比较 `session_date` 时先 `Format("2006-01-02")`。（`backend/internal/repository/session.go:119-137`、`repository/errors.go:16-23`）
6. **`engine_version` 必须 ≥ 模型名长度（V6 已放宽到 64）。** 原设计 `VARCHAR(32)`，而真实模型名 `qwen-audio-3.1-asr-flash-filetrans` 有 34 字符，触发 `ERROR 1406 Data too long`——**整条 UPDATE 回滚**，表现为「识别成功、文本已拿到，但 content/segments 全没落库且 status 卡在 running，前端无限轮询」，而这次 ASR 已经计费。**必须在库侧放宽**（MySQL 8 严格模式超长即报错而非截断）。（`migrations/6_transcripts_engine_width.up.sql:3-11`）

### 3.4 种子数据与对账基准（事实源 `backend/database/seed.sql`）

覆盖 **4 位教师**（李明 / 张华 / 刘洋 / 赵磊）、**6 门课程**、**14 次授课**，每门课程配同一位督导（陈静）；每次课同时具备督导与智能体两侧、且**两侧均覆盖五维**（智能体不再缺 `objective`，雷达图无缺角）。

**c1 对账基准（软件项目管理 · 李明，3 次课，督导 + 智能体双侧五维）**：

| 指标 | 第 1 次课 | 第 2 次课 | 第 3 次课 | 教师级汇总 |
|------|----------|----------|----------|-----------|
| 督导单次总分 | 52.50 | 70.00 | 82.50 | **68.33** |
| 智能体单次总分 | 65.00 | 70.00 | 75.00 | **70.00** |
| **综合分** | **58.75** | **70.00** | **78.75** | **69.17** |

- 教师级综合分 69.17 **恰好等于三次课综合分的算术平均**（§2.5.2 的恒等式），若实现后两者不等即为聚合层次写错；
- 算法**仍保留**「某侧维度为 NULL 时该维度只取另一侧 / 单侧总分权重再归一化」的容错分支（由 `pkg/scoring` 单测覆盖，种子数据不再出现缺维）；
- 全量 14 次课的逐场综合分与教师级均值由 `pkg/scoring` 单测 `TestSeedDataset*` 锁定（`backend/pkg/scoring/scoring_test.go:352,365`）；算法变更须同步重算 seed 注释；
- `total_score` 仅为展示与排序用，**聚合以维度分为准**。

---

## 4. 接口索引（契约事实源：[`backend_AGENTS.md`](./backend_AGENTS.md) §8）

> 前缀 `/api/v1`，沿用 Sprint 1 统一响应信封与错误码。**请求/响应字段、权限、错误码的权威定义在 `backend_AGENTS.md` §7 / §8**（§8.8 授课记录与评价聚合、§8.9 课堂录音与转写）；字段名与 `backend/internal/dto/` 逐字对齐。本节只做索引，并保留评分/产品口径上独有的约定。

**本节仅保留的新增错误码（完整表见 `backend_AGENTS.md` §7）**：

| code | HTTP | 含义 | 触发示例 | 实现状态 |
|------|------|------|---------|---------|
| 40002 | 400 | 业务规则校验失败 | 授课记录日期晚于今天、**督导评分五维未录全** | ✅ 阶段一 |
| 40901 | 409 | 数据已存在 | 同课程同班级同日同节次重复建课（`uk_session`） | ✅ 阶段一 |
| 40902 | 409 | 重复提交 | 同一督导对同一场次重复评分 | 保留：`PUT` 幂等覆盖，**不返回**该码 |
| 50002 | 501 | 功能未实现 | — | ⚠️ 仅 `pkg/errcode` 定义，**当前无调用方**（§4.4） |
| 50003 | 503 | 依赖服务不可用 | ASR 引擎未配置或转写任务失败 | ✅ 阶段二转写链路 |

> ⚠️ `40002` 的 HTTP 状态必须是 **400**（`pkg/errcode` 的 `HTTPStatus()` 已覆盖 `BizRule`）。
> 该映射曾漏配而落 `default → 500`：响应体 code 正确但 HTTP 500，日志被记为服务端错误——回归测试见 `backend/pkg/errcode/errcode_test.go`。

### 4.1 授课记录与督导评分 → `backend_AGENTS.md` §8.8

- 路由、请求字段、权限、错误码全见 §8.8，本节不复制。
- **评分专属约定**：
  - `PUT /sessions/:id/supervisor-evaluation` **5 个维度全部必填**（1–5 整数），缺失返回 40002；缺维度不得落库；
  - 该接口**幂等覆盖**（同一督导同一场次至多一行，无草稿态）；落库同时写 `formula_version`；
  - **仅智能体侧**允许维度为 `NULL`（§2.1）；
  - 草稿接口（`/sessions/:id/draft`、`/drafts*`）见 §8.8；草稿允许部分维度为空，提交时强制五维齐全，**草稿永不进入聚合口径**。

### 4.2 评价聚合（主任页与教师页的唯一事实源） → `backend_AGENTS.md` §8.8

- 🔴 **路由命名**：教师评分列表为 **`GET /teacher-scores`**，不是 `GET /teachers`。Sprint 1 的 `GET /teachers` 已用于「教师字典」（前端课程表单依赖，返回数组），不能改成带分页的评分列表。两者并存，**新增接口不得复用二者，也不得再改路径而不更新本节与 §8.8。**
- **综合分口径**（§2.5.2 综合均值）：教师级 / 课程级 `compositeScore` = 各场次综合分的算术平均；教师级恒等于其各次课综合分的平均。维度展示分 = α（默认 0.5）双侧融合，单侧缺失取另一侧。
- **必须返回的样本与状态字段**（字段名见 `backend/internal/dto/evaluation.go`，`SampleDTO` / `ScoreSummary`）：
  - `sample`：`sessionCount / evaluatedCount / supervisorCount / agentCount / alignedCount / sampleSufficient`；`sampleSufficient` 以 `evaluatedCount` 为准；
  - `flags`：后端显式回传，不让前端猜。**已实现**：`no_data` / `sup_only` / `ai_only` / `disjoint`（`pkg/scoring/scoring.go:305` `missingFlags`）+ `sample_insufficient`（`service/teacherscore.go:381`）。文档早期列出的 `agent_not_calibrated` / `formula_mixed` **后端尚未产生**（`formula_mixed` 仅前端有文案占位），不得当作已实现；
  - 未评价场次不计入分母，综合分为 `null` 且不参与排序，**不得按 0 分处理**（§2.5.5）。
- 未实现：`GET /teachers/:id/score-trend`（阶段三趋势，前端现为 mock，见 §4.4）。

### 4.3 录音与转写 → `backend_AGENTS.md` §8.9

- 路由、字段与权限全见 §8.9。产品级红线（§5.1 / §5.2）：
  - **音频对教师不可见**：`GET /recordings/:id/stream` 仅督导，服务层强制 403；`<audio>` 无法携带 Authorization，故额外签发 `?ticket=` 短时票据（路由挂载见 `backend/internal/router/router.go:80-84`）；
  - 播放必须写审计 `recording_playback_logs`（§5.2-2）；
  - **转写异步**：`GET /sessions/:id/transcript` 返回 `{recording, transcript}`，用 `transcript.status` 轮询；`failed` 可 `POST /sessions/:id/transcript/retry`；
  - **脱敏**：转写文本中的学生姓名按称谓正则 + `transcription.studentNames` 词典替换（§5.2-4）。

### 4.4 智能体接口（阶段二起，**未实现**）

| 方法 路径 | 权限 | 说明 | 状态 |
|-----------|------|------|------|
| `POST /agent/transcribe` | 内部 | 触发转写任务（异步） | ❌ 未实现；实际由上传录音后入队（§8.9） |
| `POST /agent/evaluate` | 内部 | 触发智能体评分（写 `evaluations` 的 agent 行） | ❌ 未实现；`repository/recording.go:79` `UpsertAgentEvaluation` 已就绪但**无调用方** |
| `POST /agent/chat` | 登录 | 阶段三：SSE 流式对话 | ❌ 未实现；前端 `frontend/src/api/agent.ts` 返回 `src/mocks/teacherImprove.ts` 的演示数据 |

> 🔴 **诚实说明（修正早期版本的错误描述）**：`backend/internal/router/router.go` 中**没有任何 `/agent/*` 路由**，因此请求会落到 gin 默认 **404**；`50002` 虽在 `pkg/errcode` 定义并映射 HTTP 501，但**无任何调用方**。"未实现前统一返回 501 / 50002"的说法不成立——**不得在联调中把它们当作可用接口**。
> 🔴 **硬约束（不变，`service/recording.go:237` 已实现）**：智能体接口**必须与主链路解耦**：调用失败只记录日志并保留 `transcripts.status='failed'`，**不得影响督导评分与主流程**。

---

## 5. 权限、隐私与合规

### 5.1 权限矩阵（沿用 Sprint 1「数据裁剪只发生在后端」铁律）

| 资源 | director | teacher | supervisor |
|------|:--------:|:-------:|:----------:|
| 教师评分列表 | 仅本室 | ✗ | 全校 |
| 教师评分面板 | 仅本室 | **仅自己** | 全校 |
| 课程级评分 | 本室课程 | 本人课程 | 全部 |
| 课程历史授课记录 | 本室课程 | 本人课程 | 全部 |
| 当堂课评估页 | 本室只读 | 本人只读 | 全部，**唯一可写** |
| 提交督导评分 | ✗ | ✗ | ✓ |
| 查看督导评语 | 本室 | **仅自己** | ✓ |
| 音频播放 | ✗ | **✗（不可听）** | ✓ |
| 转写文本 | ✗ | **本人课程（可看，用于改进）** | ✓ |

> 🔴 **教师绝不能看到同事的评分与评语。** 教师端可见范围严格限定为**仅本人**——这是组织敏感信息，越权等于事故。
> **音频对教师不可见**：音频中的学生人声与教师人声不可分割，教师端只开放**已脱敏的转写文本**（§5.2-4）。

### 5.2 隐私与合规（采集音频前必须落地）

课堂录音含**学生声音**，属个人信息。一旦开始采集即产生合规义务：

1. **告知**：录音前需课堂口头告知或课程公告
2. **访问控制 + 审计**：谁在何时听了谁的课，必须留痕——**已落地**为 `recording_playback_logs` 表（`backend/migrations/3_recordings_and_transcripts.up.sql:37`），由 `RecordingService.Stream` 在放行播放时写入（`backend/internal/service/recording.go:184`）。早期文档中的 `audit_logs` 表**不存在**，勿再引用。
3. **保留期限**：如 1 学期后归档或删除，不得无限期保存
4. **转写脱敏**：转写文本中的学生姓名做 NER 替换（"学生A"），避免评语出现可识别个人的内容
5. **最小可见**：教师可见**自己的转写文本**用于改进；音频默认仅督导（音频里的人声不只是教师的）

### 5.3 评分伦理

- 教师评分列表**默认按姓名/工号排序**，按分排序作为显式操作
- 每行必须展示**评价次数 n**，`n < 3` 标注样本不足
- UI 显著标注「**仅用于教学支持，不作为考核依据**」
- 督导间评分校准（rater bias）列为已知局限（§10.1），阶段三引入

---

## 6. 页面与路由

> 页面规格、组件清单与路由总表的事实源为 [`docs/frontend_AGENTS.md`](./frontend_AGENTS.md) §6 / §7。本节只保留**路由现状**与产品级决策（并列路由、三条跳转链路）。

### 6.1 路由现状（对照 `frontend/src/router/index.ts`）

| 路由 | name | meta.roles | 说明 |
|------|------|-----------|------|
| `/dashboard` | dashboard | 登录 | 三角色差异化工作台；教师登录后重定向到 `/me/quality`（`frontend/src/router/guards.ts:23`） |
| `/courses` | course-list | 登录 | 课程列表（主任「课程库」/ 教师「我的课程」/ 督导「课程列表」） |
| `/courses/new` | course-new | director | 新增课程 |
| `/courses/:id` | course-detail | 登录 | 课程详情（含历史授课记录列表） |
| `/courses/:id/edit` | course-edit | director | 编辑课程 |
| `/courses/:id/improve` | course-improve | teacher | 教学提优（并列路由决策见 §6.2） |
| `/sessions/:id/evaluation` | session-evaluation | 登录 | 当堂课质量评估页；**唯一可写角色为 supervisor**（教师由「授课快照 → 查看详细记录」进入复盘） |
| `/supervision` | supervision | supervisor | 听评课管理（完整分页 + 状态/日期筛选）；**不删除** |
| `/supervision/courses/:id` | supervisor-course | supervisor | 督导课程综合评分页（综合分 + 历史授课记录 + 录音上传） |
| `/teachers` | teacher-list | director/supervisor | **教师画像**（侧边栏文案见 `frontend/src/layouts/AppLayout.vue:61`；不是「教师管理」） |
| `/teachers/:id` | teacher-detail | director/supervisor | 教师评分面板 |
| `/me/quality` | profile-quality | teacher | 我的质量档案 |
| `/drafts` | draft-box | supervisor | 草稿箱 |
| `/profile` | profile | 登录 | 个人中心 |
| `/login` · `/403` · `/:pathMatch(.*)*` | login / forbidden / not-found | public | 登录、无权、404 |

> 与早期版本的差异：补入 `/me/quality`、`/drafts`（旧表遗漏）；`/teachers` 页面标题为「教师画像」。
> `/supervision` **不删除**：工作台只展示听评课安排前 8 条，删页会导致第 9 条以后无法访问。

### 6.2 「教学提优」与课程详情的关系

**采用并列路由**：保留 `/courses/:id`；教师从**「我的课程」点击时跳 `/courses/:id/improve`**（`frontend/src/views/CourseListView.vue:49`）。

| 方案 | 做法 | 评价 |
|------|------|------|
| A 真替换 | `/courses/:id` 对 teacher 渲染提优页 | 路由语义随角色漂移；教师把链接发给督导，对方看到完全不同的页面 |
| B 加 Tab | 详情页内加「教学提优」Tab | 改动最小，但教师每次要多点一次 |
| **C 并列路由（采用）** | 教师从「我的课程」跳 `/courses/:id/improve` | 同时满足"点课程直接进提优"与"路径语义清晰"，教师仍可访问原详情 |

教学提优页保留课程基本信息（学分/学时/学期/班级/学生人次）与资源上传接口。

### 6.3 三条跳转链路（产品级目标）

**① 主任**
```
/dashboard  统计卡 + 「课程管理」「教师画像」入口卡
   ├─▶ /courses（课程库）
   └─▶ /teachers  本室教师｜综合分｜5维分｜评价次数 n
         └─▶ /teachers/:id  教师评分面板
               ├─ 综合分大数字 + 分维度条形图
               ├─ 按课程分组的分数明细
               ├─ 历次评价时间线（含督导评语）
               └─▶ /courses/:id ─▶ 历史授课记录 ─▶ /sessions/:id/evaluation（只读）
```
> 工作台**移除**「本室教师开课情况」表格与「近期开课」列表（与课程管理重复），仅保留统计卡与两个入口。

**② 督导**
```
/dashboard  待评课队列（今日/本周/本月/未评，已评估自动清除）＋ 待评估授课记录
   ├─▶ /supervision  听评课管理（完整分页 + 状态/日期筛选）
   └─▶ /courses  课程列表（mine=1：本人负责评估的课程）
         └─▶ /supervision/courses/:id  课程综合评分页
               ├─ 本课程当前综合评分（督导 × AI 双源）
               └─ 历史授课记录列表
                     ├─「去评估 / 查看·修改评估」→ /sessions/:id/evaluation
                     │     ├─ 音频播放器（阶段二，仅督导）
                     │     ├─ 转写文本（阶段二，含说话人分离；卡片限长 + 弹窗展开）
                     │     ├─ 督导评分表单：5 维 1-5 分 + 结构化评语
                     │     ├─ 智能体评分参考（可折叠，标"AI 参考"）
                     │     └─ 提交（提交后授课记录置 evaluated，从队列清除）
                     └─ 上传课堂录音（POST /sessions/:id/recording）
```

**③ 教师**
```
/me/quality  我的质量档案
   ├─ 我的课程评分明细：课程列表 +「去提优」→ /courses/:id/improve
   └─ 授课快照：逐次授课卡片 +「查看详细记录」→ /sessions/:id/evaluation（只读）
         ├─ 督导评分与结构化评语（只读）
         ├─ AI 智能体评价（维度对照 + 文字评语）
         └─ 当堂课脱敏转写（教师不可回放录音）
/courses/:id/improve  教学提优
   ├─ 课程基本信息 + 资源上传接口（保留）
   ├─ 评分区：综合分 + 5 维分 + 与上学期对比 +（阶段三）趋势折线
   ├─ 督导评语列表（按时间倒序，标注是哪一次课）
   └─ 智能体提优建议（按时间倒序，当前为 mock）
```

> **挂链现状核对（2026-10-07）**：`/courses → /courses/:id/improve`（`CourseListView.vue:49`）、`/me/quality → /sessions/:id/evaluation`（`ProfileQualityView.vue:102`）、`/dashboard → /sessions/:id/evaluation`（`DashboardView.vue:162,178,183`）、`/dashboard → /teachers`（`DashboardView.vue:144,353`）、`/supervision/courses/:id → /sessions/:id/evaluation`（`SupervisorCourseView.vue:108`）均已挂链；**`/supervision` 与 `/supervision/courses/:id` 目前没有入口链接**（侧边栏 `AppLayout.vue:49-55` 无对应项，工作台也无跳转），只能直接输入 URL 访问——入口待补。

### 6.4 前端目录与组件

> 目录结构（`src/api` / `src/components` / `src/views`）与组件清单以 [`frontend_AGENTS.md`](./frontend_AGENTS.md) §5 / §8 为事实源，本文件不复制。本文件关心的口径只两条：评估页组件必须覆盖「评分表单 + 锚点 tooltip + 双源对照 + 结构化评语 + 脱敏转写」，提优页必须覆盖「督导评语流 + 智能体建议 + 趋势」。

---

## 7. 任务列表（含状态）

> 分工沿用 4+n 模式：**成员一**（徐仕杰，DRI/产品）、**成员二**（刘子杰，前端）、**成员三**（后端）、**成员四**（胡凯翔，后端架构）。
> 依赖列中的编号表示必须先完成的任务。
> **状态口径**：✅ 已完成 · 🟡 部分完成 · ❌ 未完成。状态由代码核对得出（证据见 §1.1 与各表备注），**不是排期占位**。
> 原按模块分发的《Sprint2-3 任务清单》已并入本表（该文件随本次合并删除）：模块分组不提供超出本表的额外信息（同一批任务、同一批负责人），为避免双份清单漂移，**任务分发的唯一入口是本表**。

### 7.1 阶段一（Sprint 2.1）—— 评价闭环，不依赖智能体

| 编号 | 任务 | 负责 | 依赖 | 验收 | 状态 |
|------|------|------|------|------|------|
| T1.1 | 引入 `golang-migrate`，编写 `2_teaching_sessions_and_evaluations` 迁移脚本（§3.1） | 成员四 | — | 空库执行 `migrate up` 可建成 2 张表；文件命名符合同工具约定 | ✅ |
| T1.2 | 扩展 `model`：`TeachingSession`、`Evaluation` | 成员四 | T1.1 | `go build` 通过，字段与 DDL 一一对应 | ✅ |
| T1.3 | `pkg/scoring`：维度权重、单次总分、`Aggregate()` 纯函数 | 成员四 | — | 表驱动单测覆盖 §2.4 验算例与 §2.5.5 缺失矩阵全部 5 行；恒等式（综合均值）在非对齐数据下亦成立 | ✅ |
| T1.4 | repository：`session.go`、`evaluation.go`（含 §2.5.4 聚合 SQL） | 成员三 | T1.2 | `go test` 通过；聚合结果与手算一致 | ✅ |
| T1.5 | service：`session.go`（授课记录 CRUD + 归属校验） | 成员三 | T1.4 | 越权用例返回 40302 | ✅ |
| T1.6 | service：`session.go`（提交评分、幂等覆盖、`formula_version` 写入） | 成员三 | T1.5 | 五维必填（缺失 40002）；重复提交覆盖而非报错 | ✅ |
| T1.7 | service：`teacherscore.go`（教师级/课程级聚合，单一事实源） | 成员四 | T1.3 T1.4 | 主任视角与教师视角数字**完全一致** | ✅ |
| T1.8 | handler + router：§4.1 / §4.2 全部接口 | 成员三 | T1.5–T1.7 | `/healthz` 与 Sprint 1 接口无回归 | ✅ |
| T1.9 | 前端 `types/` + `api/`：session / teacher 接口层 | 成员二 | T1.8 | `vue-tsc` 零错误 | ✅ |
| T1.10 | 前端：课程详情页新增「历史授课记录」列表 | 成员二 | T1.9 | 三态完整（加载/空/失败） | ✅ |
| T1.11 | 前端：当堂课质量评估页（评分表单 + 锚点 tooltip + 结构化评语） | 成员二 | T1.9 | 5 维 1-5 分必填校验；提交后列表即时反映 | ✅ |
| T1.12 | 前端：教师画像页 `TeacherListView` | 成员二 | T1.9 | 展示评价次数 n；n<3 有样本不足标记；默认按姓名排序 | ✅ |
| T1.13 | 前端：教师评分面板 `TeacherDetailView` | 成员二 | T1.12 | 综合分 + 5 维 + 按课程明细 + 历次时间线 | ✅ |
| T1.14 | 前端：教学提优页 `CourseImproveView`（只读督导分 + 评语） | 成员二 | T1.9 | 保留基本信息与资源上传 | ✅ |
| T1.15 | 前端：`/supervision` 改造为听评课管理；主任工作台改为入口卡 | 成员二 | — | 完整分页列表可用；工作台不再重复课程表格 | ✅ |
| T1.16 | 种子数据扩展（§3.4）+ `verify.sh` 新增断言 | 成员三 | T1.8 | 新增断言全绿（当前 `backend/scripts/verify.sh` 共 137 条 `want` 断言） | ✅ |
| T1.17 | 文档同步：三份 AGENTS / MySQL 文档的范围与契约章节 | 成员四 | — | 与本文档无矛盾 | ✅ |

### 7.2 阶段二（Sprint 2.2）—— 智能体接入

| 编号 | 任务 | 负责 | 依赖 | 验收 | 状态 |
|------|------|------|------|------|------|
| T2.1 | `3_recordings_and_transcripts` 迁移脚本（§3.2） | 成员四 | T1.1 | 迁移可重复执行 | ✅ |
| T2.2 | 音频上传（白名单 + 大小限制 + 时长解析） | 成员三 | T2.1 | 非法扩展名返回 40001 | ✅ |
| T2.3 | 异步转写任务框架（worker + 状态机 + 重试） | 成员四 | T2.2 | 任务失败不影响主流程；`failed` 可重试 | ✅ |
| T2.4 | ASR 适配层（Qwen-Audio 接入，接口先行、实现可后补） | 成员四 | T2.3 | 未配置引擎时返回 50003 而非崩溃 | ✅ |
| T2.5 | 转写脱敏（学生姓名 NER 替换） | 成员四 | T2.3 | 单测覆盖姓名替换 | ✅ |
| T2.6 | 智能体评分服务（写 `evaluations` 的 agent 行） | 成员四 | T2.4 | 无法观测的维度写 `NULL` | ❌ **未完成**：`repository/recording.go:79` `UpsertAgentEvaluation` 无调用方；无 `/agent/*` 路由 |
| T2.7 | 前端：音频播放器 + 转写查看器（含轮询） | 成员二 | T2.3 | 转写中/失败/完成三态 | ✅ |
| T2.8 | 前端：评估页「智能体参考」对比面板 | 成员二 | T2.6 | 标注"AI 参考"；低置信度维度有提示 | ✅ 面板已交付（`EvaluationCompare.vue`）；真实数据待 T2.6 |
| T2.9 | 综合分生效：聚合 SQL 纳入 agent 侧 + `flags` 全量返回 | 成员四 | T2.6 | §2.5.5 缺失矩阵全部有单测 | ✅ 聚合侧已实现；无 agent 写入通道（取决于 T2.6） |
| T2.10 | 审计日志（谁听了谁的课） | 成员三 | T2.2 | 播放接口写入审计 | ✅ `recording_playback_logs`（`service/recording.go:184`） |

### 7.3 阶段三（Sprint 3）—— 帮教师

| 编号 | 任务 | 负责 | 依赖 | 验收 | 状态 |
|------|------|------|------|------|------|
| T3.1 | 智能体提优建议生成与展示 | 成员四/二 | T2.6 | 教师端可见，按时间倒序 | ❌ 未完成：督导评语展示已有（`EvaluationTimeline` / `CommentPanel`），**AI 建议为 mock**（`frontend/src/mocks/teacherImprove.ts`） |
| T3.2 | 趋势接口 + 趋势折线组件 | 成员三/二 | T1.7 | §3.4 种子数据应呈上升趋势 | ❌ 未完成：组件与 mock 已就绪（`ScoreTrendChart.vue`），接口未实现 |
| T3.3 | 智能体对话（SSE 流式） | 成员四/二 | T2.6 | 断流可重连；失败不阻塞页面 | ❌ 未完成：`api/agent.ts` 用定时分包模拟流式 |
| T3.4 | 教师申诉 / 督导复核流程 | 成员三/二 | T1.6 | 申诉记录留痕，状态可追溯 | ❌ 未完成 |
| T3.5 | 督导间评分校准（示范课基线偏移） | 成员四 | T2.9 | 校准前后分数可对比 | ❌ 未完成 |
| T3.6 | 质量报告导出 | 成员三 | T1.7 | 导出内容与页面数字一致 | ❌ 未完成 |

---

## 8. 验收标准（DoD）

### 8.1 阶段一

| 项 | 标准 |
|----|------|
| 数据链路 | 督导建授课记录 → 打分 → 教师端 10 秒内看到分数与评语 |
| **一致性** | 同一教师，主任端与教师端综合分、各维度分**逐位相同**（由单一聚合函数保证） |
| 数据范围 | 教师访问 `/teachers/3/evaluation-summary` 返回 40302；主任访问他室教师同样 40302 |
| 算法 | §2.4 验算例返回 **82.50**；§2.5.5 五种缺失组合全部有断言 |
| 恒等式 | 教师级综合分 ≡ 其各次课综合分的算术平均（§2.5.2 综合均值口径，**任意数据下恒定成立**）；用 §3.4 种子数据对账 c1 应为 **69.17** |
| 维度必填 | 督导 `PUT /sessions/:id/supervisor-evaluation` 五维缺一返回 **40002**；缺维度不得落库 |
| 样本量 | `sampleSufficient` 以 `evaluatedCount`（已评价场次）为准；有课未评时综合分为 `null` 且 `no_data` |
| 错误码 | `40002` 的 HTTP 状态必须是 **400**（不得落 500）；`pkg/errcode` 表驱动单测锁定 |
| 空数据 | 无评价教师综合分为 `null`、列表置底，**不得显示 0** |
| 构建 | `go build` / `go vet` / `gofmt` / `go test` 零告警；`vue-tsc` 零错误；`npm run build` 通过 |
| 端到端 | `scripts/verify.sh` 全绿；响应日志无 HTTP 5xx |

### 8.2 阶段二

> ⚠️ **前置缺口**：T2.6 智能体评分未接入。下表「维度可空」「综合分」两条当前**只在种子数据（`seed.sql` 的 agent 行）下成立**，不得据此判定阶段二完成——阶段二完成的标志是 T2.6 落地（有真实写入通道）。

| 项 | 标准 |
|----|------|
| 转写异步 | 上传 45 分钟音频不阻塞接口；轮询可见 `pending→running→done` |
| 降级 | ASR 引擎不可用时，督导评分与聚合**完全不受影响**；接口返回 50003 |
| 维度可空 | 智能体 `objective_score` 为 `NULL` 时，该维度自动退化为"只取督导侧"，不出现 NaN |
| 综合分 | 双侧齐备时按 α 融合；`disjoint` 场景返回显式提示 |
| 隐私 | 转写文本中学生姓名已脱敏；音频播放写入 `recording_playback_logs` |

### 8.3 阶段三

| 项 | 标准 |
|----|------|
| 趋势 | 种子数据呈现上升折线；可按维度切换 |
| 建议 | 教师端可见智能体提优建议且与课堂记录对应 |
| 对话 | SSE 流式输出，中断可恢复 |
| 申诉 | 教师可提交异议，督导可复核，全过程留痕 |

---

## 9. 迭代方向（本计划外，供后续评估）

按「价值 ÷ 成本」排序，供阶段三之后排期参考：

| # | 方向 | 价值/成本 | 说明 |
|---|------|----------|------|
| 9.1 | 评语常用标签库 | 高/低 | 降低督导书写成本，便于教师端检索 |
| 9.2 | 待评价清单 | 高/低 | 主任页显示"本室未评价课程/教师"，把覆盖率从统计变成行动 |
| 9.3 | 教学质量象限图 | 中/低 | X 覆盖率、Y 平均分 → 重点帮扶 / 保持 / 补覆盖 / 标杆 |
| 9.4 | 资源与评分相关性 | 中/中 | 有资源的课是否评分更高，数据积累后是可对外讲的结论 |
| 9.5 | 同行评议 | 中/中 | 新增 `evaluator_type='peer'`，架构已支持（§1.1） |
| 9.6 | 说话人分离（diarization） | 中/中 | **智能体评「学生互动」的前提**，不分说话人就算不出师生话轮比 |
| 9.7 | 智能体模型版本管理 | 中/低 | 换模型后分数不可比，`ai_model_version` 已预留 |
| 9.8 | 跨学期对比 | 低/低 | 依赖 9.3 之外的 T3.2 |
| 9.9 | 评语通知教师 | 中/中 | 需重新评估「消息通知」的开禁 |
| 9.10 | 细粒度权限（ABAC） | 低/高 | 出现"院级督导"等新角色时再考虑 |
| 9.11 | 补齐 `/supervision` 与 `/supervision/courses/:id` 的导航入口 | 高/低 | 两页已实现但无入口链接（§6.3 现状核对），属可用性缺口 |

---

## 10. 关键设计约定速查

下表汇总开发中必须遵守的硬性约定。**变更任一条必须先更新本文档再改代码**（契约先行）。

| # | 约定 | 出处 |
|---|------|------|
| 1 | 授课记录由督导在听课后创建；无教务对接，不自动生成 | S6.1 · T1.5 |
| 2 | 5 个维度冻结；`frontier` 为观测项，权重 0，不计入加权总分 | §2.1 · §2.2 |
| 3 | 维度权重非零项合计必须为 1.00，服务启动时校验 | §2.2 |
| 4 | 综合分 α 默认 0.5（督导 : 智能体），可配置 | §2.2 |
| 5 | 综合分取**综合均值**口径（各场次综合分算术平均）；教师级综合分 ≡ 各次课综合分的算术平均，任意数据下恒定成立；维度展示分按 α 融合，默认双侧对齐时二者自洽 | §2.5.2 |
| 6 | 单次评价落库必须同时写 `formula_version`；跨表聚合不落库 | §2.5.6 |
| 7 | 所有聚合接口必须支持 `?semester=`，默认当前学期 | §2.5.6 |
| 8 | 无评价教师综合分为 `null` 且不参与排序，不得显示 0 | §2.5.5 |
| 9 | 一次课允许多个督导评分，同场次取 `AVG` | §2.5.4 |
| 10 | 正式评价无草稿态，`PUT` 幂等覆盖，写入即生效；草稿另存 `evaluation_drafts`，不进聚合 | §3.1 |
| 11 | 教师只能看自己的评分与评语；音频对教师不可见，仅开放脱敏转写 | §5.1 |
| 12 | 教师评分列表默认按姓名排序，必须展示评价次数 n，标注"不作为考核依据" | §5.3 |
| 13 | **不得改动 Sprint 1 存量表**（`supervision_plans` 保持不动，关联靠 `teaching_sessions.plan_id` 反向指回）；Sprint 2 新增表的列调整一律走新迁移（V4 / V6 为先例） | §3 |
| 14 | 转写必须异步且可重试；智能体故障不得影响督导评分主流程 | §4.4 |
| 15 | 督导评分**五维全部必填**（缺失 40002）；仅智能体侧允许维度为 `NULL` | §2.1 · §4.1 |
| 16 | 样本充足性以 `evaluatedCount`（已评价场次）为准，不以 `sessionCount` | §2.5.5 |
| 17 | 迁移文件命名 `<版本>_<名称>.up.sql` / `.down.sql`；**禁止** Flyway 风格 `V2__xxx` | §3 |
| 18 | 教师评分列表路由为 `GET /teacher-scores`；`GET /teachers` 永远是教师字典 | §4.2 |
| 19 | 新增错误码必须同步 `pkg/errcode` 的 code / 文案 / **HTTP 映射**三处，并补表驱动单测 | §4 |
| 20 | JSON 列（如 `evaluations.evidence`）的 Go 字段必须用指针，禁止以零值 `''` 写入 | §3.3-3 |
| 21 | 智能体评分未接入（T2.6）：不得把 `/agent/*`、`score-trend` 当作可用接口，也不得用 `50002` 描述其真实行为（实为 404） | §4.4 |

### 10.1 遗留待观察项（不阻塞开发，阶段三复盘）

| 项 | 说明 |
|----|------|
| 督导间评分校准 | 当前无法消除 rater bias，阶段三 T3.5 引入示范课基线偏移校准 |
| 智能体评分效力 | α=0.5 是未经校准的等权假设，积累 ≥50 条双侧样本后应重新评估 |
| 样本量阈值 | `minSampleSize=3` 为经验值，首个学期结束后按实际分布调整 |
| 非对齐聚合算法 | 当前采用「综合均值」并默认数据双侧对齐；部分场次仅单侧评价时的通用算法待单独立项（§2.5.2） |
| 智能体评分接入 | T2.6 未完成，阶段二不能判定为交付（§1.1 / §8.2）；`50003` 目前只覆盖转写链路 |

---

## 更新记录

| 日期 | 版本 | 变更 |
|------|------|------|
| 2026-09 | v1.0 → v1.3 | 阶段一联调修订：综合均值口径、五维必填、`evaluatedCount` 样本量、`/teacher-scores` 路由、golang-migrate 命名、错误码 HTTP 映射 |
| 2026-10-07 | v1.4 | ① 并入原按模块分发的《Sprint2-3 任务清单》（该文件随本次合并删除，其按模块视角无额外信息）并为 §7.1/§7.2/§7.3 增加**状态**列；② §3 DDL、§4 接口契约、§6 页面与路由改为指向单一事实源（`backend/database/schema.sql` + `backend/migrations/`、`backend_AGENTS.md` §8、`frontend_AGENTS.md` §6/§7），只保留 SQL/契约之外的口径约定与实现陷阱；③ 修正阶段状态（阶段①已实现 / 阶段②**部分实现**：智能体评分未接入 / 阶段③未实现）与审计表名（`audit_logs` → `recording_playback_logs`）、页面名（「教师管理」→「教师画像」）；④ 修正 §4.4「未实现前统一返回 501/50002」的失实描述为真实行为（无 `/agent/*` 路由 → 404；`50002` 无调用方）；⑤ 补入遗漏路由 `/me/quality`、`/drafts` 与挂链现状 |

*文档版本：v1.4（2026-10-07）*
