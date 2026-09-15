# 「爱教学」Sprint 2 / Sprint 3 开发计划 —— 课堂评价与教学提优

> **文档定位**：本文件是 Sprint 2「看课堂」与 Sprint 3「帮教师」的功能与设计事实源，承接 `docs/backend_AGENTS.md`、`docs/frontend_AGENTS.md`、`docs/MySQL数据库创建指导.md` 的 Sprint 1 基线。
> **维护人**：成员四（胡凯翔，后端架构）· 评审：成员二（刘子杰，前端）· DRI 复核：成员一（徐仕杰）
> **版本**：v1.2
>
> **使用方式**：本文件只描述**要开发什么、按什么规范做**。§10 汇总了必须遵守的硬性约定，变更任一条需先更新本文档再改代码（契约先行）。
> 任务分发用的简短清单见 [`Sprint2-3-任务清单.md`](./Sprint2-3-任务清单.md)。

---

## 1. 范围与目标

### 1.1 三阶段推进（正式排期）

| 阶段 | 主题 | 核心内容 | 是否依赖智能体 |
|------|------|---------|--------------|
| **阶段一**（Sprint 2.1） | 评价闭环 | 授课记录 · 督导评分 · 结构化评语 · 三级聚合 · 主任教师管理 · 教师提优页（只读督导分） | ❌ 不依赖 |
| **阶段二**（Sprint 2.2） | 智能体接入 | 课堂录音 · 异步转写 · 智能体评分 · 综合分（双侧融合）· 督导页智能体参考面板 | ✅ |
| **阶段三**（Sprint 3） | 帮教师 | 智能体提优建议 · 对话（SSE）· 趋势视图 · 申诉复核 · 质量报告导出 | ✅ |

> **核心架构：评分源可插拔**
> `supervisor` 与 `agent` 只是 `evaluations.evaluator_type` 的两个取值，向同一张表写同样的结构；聚合层只认该表，不关心分数从哪来。
> 收益：① 智能体延期不阻塞阶段一交付；② 智能体效果不佳时可降权甚至停用；③ 未来加「同行评议」「学生评教」只需新增一个枚举值，不改表、不改聚合。

### 1.2 用户故事

| 编号 | 阶段 | 角色 | 故事 | 优先级 | 落点 |
|------|------|------|------|--------|------|
| S6.1 | 一 | 督导 | 为一次课建立授课记录（课程/班级/日期/节次/主题），以便评价有落点 | M | `POST /sessions` |
| S6.2 | 一 | 任一 | 在课程详情页查看该课程的历史授课记录列表 | M | `GET /courses/:id/sessions` |
| S6.3 | 一 | 督导 | 在当堂课评估页按 5 个维度打分并留下结构化评语 | M | `PUT /sessions/:id/supervisor-evaluation` |
| S6.4 | 一 | 主任 | 在教师管理页查看本室教师综合评分列表 | M | `GET /teachers` |
| S6.5 | 一 | 主任 | 点击教师查看其评分数据面板（综合分 + 分维度 + 分课程） | M | `GET /teachers/:id/evaluation-summary` |
| S6.6 | 一 | 教师 | 在教学提优页查看本人各维度评分与督导评语 | M | `GET /courses/:id/evaluation-summary` |
| S6.7 | 一 | 督导 | 工作台与「听评课管理」页分离，完整安排列表可分页筛选 | S | `GET /supervision/plans` |
| S7.1 | 二 | 督导 | 上传课堂录音并查看转写文本 | M | `POST /sessions/:id/recording` · `GET /sessions/:id/transcript` |
| S7.2 | 二 | 督导 | 查看智能体对该堂课的分维度评分作为参考 | M | 评估页聚合接口 |
| S7.3 | 二 | 任一 | 综合分 = 督导评分与智能体评分的加权融合 | M | 聚合服务 |
| S7.4 | 二 | 教师 | 在教学提优页看到综合分随智能体接入而变化 | S | 同 S6.6 |
| S8.1 | 三 | 教师 | 查看智能体针对课堂记录给出的提优建议 | M | 评估聚合 |
| S8.2 | 三 | 教师 | 与智能体就课堂改进对话 | C | `POST /agent/chat`（SSE） |
| S8.3 | 三 | 教师 | 查看本人各维度分数的历史趋势 | S | 新增趋势接口 |
| S8.4 | 三 | 教师 | 对评分提出申诉，督导复核 | S | 新增申诉表 |

> `[M]` Must；`[S]` Should；`[C]` Could。

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
 （单次评价）              （教学提优页）       （教师管理面板）
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

#### 2.5.2 聚合口径：全部在「维度层」完成

```
① 单次评价的维度分（0-100 百分制）
   dim_i = (v_i − 1) / 4 × 100                       v_i ∈ {1..5}

② 单次总分（仅用于列表排序与快速展示，非聚合来源）
   total = Σ w_i × dim_i                             某侧缺维度时，权重在其余维度上重新归一化

③ 场次级双侧融合（逐维度）
   dim_i^session = α × dim_i^sup + (1−α) × dim_i^agent
   某一侧该维度为 NULL 时，该维度直接取另一侧的值

④ 课程级 / 教师级
   dim_i^level = α × mean(dim_i^sup over sessions) + (1−α) × mean(dim_i^agent over sessions)
   composite   = Σ w_i × dim_i^level                  w 取 §2.1 生效权重，frontier 为 0
```

> 🔴 **为什么融合必须放在维度层（易错点）**：加权是线性运算，**"先融合再加权"与"先加权再融合"等价**，因此
>
> ```
> composite(教师级) ≡ mean over sessions( composite(场次级) )
> ```
>
> 即**教师面板的总分恒等于其各次课总分的平均值**。用 §3.4 种子数据验证：三次课综合分 58.75 / 70.00 / 82.50，教师级为 **70.42**。
> 若改成在"总分"层先融合、再对智能体缺失的维度做归一化，该恒等式会被破坏——页面会出现"总分 70.42、各次课平均却是 68.10"这类无法解释的差异（数值随数据而变，此处仅示意）。

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

**样本量规则**：

- `sessionCount = 0` → 综合分 `null`，不参与排序
- `sessionCount < minSampleSize(默认3)` → 置 `sampleSufficient: false`，前端标注"样本不足（n=x）"

#### 2.5.6 口径版本与学期切片

- **口径版本**：维度权重可配 → 改权重会让历史分数同时变化且无法解释。因此**单次评价的 `total_score` 落库时必须同时写 `formula_version`**；改口径只对新评价生效，历史分按旧口径保留；聚合 SQL 按版本隔离，版本混杂时返回提示。
  > 这是「汇总不落库」原则的**必要例外**——该原则针对跨表聚合，单行内确定性函数不在此列。**跨表跨行的教师级/课程级聚合仍然查询时计算，不落库。**
- **时间衰减**：本轮 **不做**。它与「教学提优」诉求冲突——教师改进后应看到分数上升，用**趋势折线**体现进步比用衰减更直观、更好解释。
- **学期切片**：所有聚合接口必须支持 `?semester=`，默认当前学期。跨学期平均会抹平改进。

---

## 3. 数据模型与建表 DDL

> **迁移方式**：按 MySQL 文档 §9，从 Sprint 2 起引入 `golang-migrate`。新增表全部**只新增、不改存量表**（`supervision_plans` 完全不动，关联通过 `teaching_sessions.plan_id` 反向指回）。

### 3.1 `migrations/V2__teaching_sessions_and_evaluations.sql`（阶段一）

```sql
-- ① 授课记录：一切评价的落点
-- 定义："某年某月某日第几节，某班级，某教师上的那一次课"
CREATE TABLE `teaching_sessions` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `course_id`    BIGINT UNSIGNED NOT NULL,
  `class_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '开课班级；0=未指定',
  `teacher_id`   BIGINT UNSIGNED NOT NULL COMMENT '冗余自 courses，避免每次联表',
  `session_date` DATE            NOT NULL,
  `period`       VARCHAR(32)     NOT NULL DEFAULT '' COMMENT '节次，如 3-4 节',
  `topic`        VARCHAR(128)    NOT NULL DEFAULT '' COMMENT '本次课主题',
  `plan_id`      BIGINT UNSIGNED NULL COMMENT '来源听评课计划（可空）',
  `status`       ENUM('scheduled','recorded','evaluated') NOT NULL DEFAULT 'scheduled',
  `created_at`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_session` (`course_id`,`class_id`,`session_date`,`period`),
  KEY `idx_session_course` (`course_id`),
  KEY `idx_session_teacher_date` (`teacher_id`,`session_date`),
  CONSTRAINT `fk_session_course` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`),
  CONSTRAINT `fk_session_plan`   FOREIGN KEY (`plan_id`)   REFERENCES `supervision_plans` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='授课记录';

-- ② 评价：督导与智能体同表，靠 evaluator_type 区分（可插拔 scorer 的落地）
CREATE TABLE `evaluations` (
  `id`                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id`        BIGINT UNSIGNED NOT NULL,
  `evaluator_type`    ENUM('supervisor','agent') NOT NULL,
  `evaluator_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '督导 user_id；agent 固定 0',
  `ai_model_version`  VARCHAR(32) NOT NULL DEFAULT '' COMMENT '仅 agent',
  `formula_version`   VARCHAR(16) NOT NULL DEFAULT 'v1' COMMENT '计分口径版本',
  -- 维度分 1-5；NULL = 该评分源无法评价此维度
  `objective_score`    TINYINT UNSIGNED NULL,
  `content_score`      TINYINT UNSIGNED NULL,
  `interaction_score`  TINYINT UNSIGNED NULL,
  `organization_score` TINYINT UNSIGNED NULL,
  `frontier_score`     TINYINT UNSIGNED NULL,
  `total_score`       DECIMAL(5,2) NULL COMMENT '按 formula_version 算出的单次总分；仅用于排序与展示，非聚合来源',
  `ai_confidence`     DECIMAL(3,2) NULL COMMENT '仅 agent',
  `evidence`          JSON NULL COMMENT '仅 agent：维度→转写片段引用',
  -- 结构化评语
  `comment`           TEXT NULL,
  `highlights`        TEXT NULL COMMENT '亮点',
  `improvements`      TEXT NULL COMMENT '待改进',
  `suggestions`       TEXT NULL COMMENT '建议 / 智能体提优建议',
  `created_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_eval` (`session_id`,`evaluator_type`,`evaluator_id`),
  KEY `idx_eval_session` (`session_id`),
  CONSTRAINT `fk_eval_session` FOREIGN KEY (`session_id`) REFERENCES `teaching_sessions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='课堂评价';
```

> **无草稿态**：`evaluations` 不设 `status` / `submitted_at`——写入即生效，`created_at` 即提交时间。督导评分通过 `PUT /sessions/:id/supervisor-evaluation` 幂等覆盖，不产生中间态。

### 3.2 `migrations/V3__recordings_and_transcripts.sql`（阶段二）

```sql
-- ③ 课堂录音
CREATE TABLE `recordings` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id`    BIGINT UNSIGNED NOT NULL,
  `file_path`     VARCHAR(255) NOT NULL,
  `original_name` VARCHAR(128) NOT NULL,
  `format`        VARCHAR(16)  NOT NULL COMMENT 'mp3/wav/m4a',
  `size`          BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `duration_sec`  INT UNSIGNED NOT NULL DEFAULT 0,
  `uploaded_by`   BIGINT UNSIGNED NOT NULL,
  `uploaded_at`   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rec_session` (`session_id`) COMMENT '一次课一条主录音',
  CONSTRAINT `fk_rec_session` FOREIGN KEY (`session_id`) REFERENCES `teaching_sessions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='课堂录音';

-- ④ 转写文本（异步任务产物）
CREATE TABLE `transcripts` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id`     BIGINT UNSIGNED NOT NULL,
  `recording_id`   BIGINT UNSIGNED NOT NULL,
  `content`        MEDIUMTEXT NOT NULL COMMENT '纯文本全文（脱敏后）',
  `segments`       JSON NULL COMMENT '[{start,end,speaker,text}]',
  `engine`         VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'qwen-audio / whisper',
  `engine_version` VARCHAR(32) NOT NULL DEFAULT '',
  `status`         ENUM('pending','running','done','failed') NOT NULL DEFAULT 'pending',
  `error_message`  VARCHAR(255) NOT NULL DEFAULT '',
  `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tr_session` (`session_id`),
  CONSTRAINT `fk_tr_session` FOREIGN KEY (`session_id`) REFERENCES `teaching_sessions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='课堂转写';
```

### 3.3 三个必须写进迁移说明的实现陷阱

1. **`class_id` 必须 `NOT NULL DEFAULT 0`。** MySQL 唯一索引**对 NULL 不去重**——若 `class_id` 可空，`uk_session` 形同虚设，同一节课能建出无数条记录。此坑只会在线上暴露。
2. **转写必须异步。** 45 分钟音频的 ASR 需数分钟，同步 HTTP 必然超时。`transcripts.status` 即为此设计，前端轮询；`failed` 状态必须可重试。
3. **为什么用列式而非 EAV**（`evaluation_scores(evaluation_id, dimension_key, score)`）：维度已冻结 → 列式可直接 `AVG(content_score)`、类型安全、索引友好；EAV 每次聚合都要 PIVOT，SQL 复杂且极易出错。代价是新增维度需 `ALTER TABLE`，在"维度冻结"前提下可接受。

### 3.4 演示种子数据（阶段一联调用）

```sql
-- 为 c1（软件项目管理，教师李明 id=2）补 3 次授课记录与评价
INSERT INTO `teaching_sessions`
  (`course_id`,`class_id`,`teacher_id`,`session_date`,`period`,`topic`,`status`) VALUES
(1, 0, 2, '2026-09-12', '3-4 节', '项目立项与章程',     'evaluated'),
(1, 0, 2, '2026-09-19', '3-4 节', '需求调研与用户故事', 'evaluated'),
(1, 0, 2, '2026-09-26', '3-4 节', '迭代计划与估点',     'evaluated');

-- 督导（陈静 id=5）评价：3 次，逐次提升以演示"趋势"
INSERT INTO `evaluations`
  (`session_id`,`evaluator_type`,`evaluator_id`,`objective_score`,`content_score`,
   `interaction_score`,`organization_score`,`frontier_score`,`total_score`,
   `comment`,`highlights`,`improvements`) VALUES
(1,'supervisor',5, 4,3,2,3,2, 52.50, '开篇结构完整，但互动偏少。',
   '课程框架清晰','提问后等待时间不足','增加案例讨论环节'),
(2,'supervisor',5, 4,4,3,4,3, 70.00, '用户故事讲解透彻，小组讨论有效。',
   '小组讨论组织得当','个别小组偏离主题','为每组设定明确产出物'),
(3,'supervisor',5, 5,4,4,4,3, 82.50, '估点练习设计巧妙，学生参与度高。',
   '练习设计贴近实战','时间略紧','预留 5 分钟总结');

-- 智能体评价（阶段二启用；objective 恒为 NULL，用于验收"维度可空"的融合逻辑）
INSERT INTO `evaluations`
  (`session_id`,`evaluator_type`,`evaluator_id`,`ai_model_version`,
   `objective_score`,`content_score`,`interaction_score`,`organization_score`,`frontier_score`,
   `total_score`,`ai_confidence`) VALUES
(1,'agent',0,'qwen-audio-v1', NULL,4,3,3,3, 60.71, 0.72),
(2,'agent',0,'qwen-audio-v1', NULL,4,3,4,3, 67.86, 0.68),
(3,'agent',0,'qwen-audio-v1', NULL,4,4,4,4, 75.00, 0.75);
```

> **对账基准（可直接用于联调验收）**：
>
> | 指标 | 第 1 次课 | 第 2 次课 | 第 3 次课 | 教师级汇总 |
> |------|----------|----------|----------|-----------|
> | 督导单次总分 | 52.50 | 70.00 | 82.50 | **68.33** |
> | 智能体单次总分 | 60.71 | 67.86 | 75.00 | **67.86** |
> | **综合分** | **58.75** | **70.00** | **82.50** | **70.42** |
>
> - 三次课综合分逐次上升，趋势折线应呈现**上升**，用于验收 S8.3；
> - 教师级综合分 70.42 **恰好等于三次课综合分的算术平均**（§2.5.2 的恒等式），若实现后两者不等即为聚合层次写错；
> - 智能体侧 `objective_score` 恒为 `NULL`，其维度融合自动退化为"只取督导侧"，用于验收该分支；
> - `total_score` 仅为展示与排序用，**聚合以维度分为准**；
> - 第 2 次课督导与智能体在三个共同维度上完全一致，故综合分 70.00 与督导分相同；而智能体单次总分 67.86 是其在**可评的三个维度上重新归一化**的结果，因不含 `objective`（督导独占维度）而略低——**这是预期行为，不是计算错误**。

---

## 4. 接口契约

前缀 `/api/v1`，沿用 Sprint 1 统一响应信封与错误码。新增错误码：

| code | HTTP | 含义 | 触发示例 |
|------|------|------|---------|
| 40002 | 400 | 业务规则校验失败 | 授课记录日期晚于今天、评分维度缺失 |
| 40902 | 409 | 重复提交 | 同一督导对同一场次重复评分（可用 `PUT` 幂等覆盖） |
| 50002 | 501 | 功能未实现 | 智能体接口在阶段一返回 |
| 50003 | 503 | 依赖服务不可用 | ASR 引擎未配置或转写任务失败 |

### 4.1 授课记录

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| `GET /courses/:id/sessions?semester=&page=&pageSize=` | 登录（数据裁剪） | 课程历史授课记录列表，含每场的双侧评分摘要 |
| `POST /sessions` | supervisor | 创建授课记录 |
| `GET /sessions/:id` | 登录（数据裁剪） | 单场次基本信息 |
| `GET /sessions/:id/evaluation` | 登录（数据裁剪） | **当堂课评估页聚合接口**：场次信息 + 音频 + 转写 + 督导评分 + 智能体评分 + 评语 |
| `PUT /sessions/:id/supervisor-evaluation` | supervisor | 提交/覆盖督导评分与评语（幂等） |

### 4.2 评价聚合（主任页与教师页的唯一事实源）

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| `GET /teachers?departmentId=&semester=&page=&pageSize=` | director（本室）/ supervisor（全校） | 教师评分列表：综合分、双侧分、分维度、样本量 |
| `GET /teachers/:id/evaluation-summary?semester=` | director（本室）/ teacher（仅自己）/ supervisor | 教师级评分面板 |
| `GET /courses/:id/evaluation-summary?semester=` | 登录（数据裁剪） | **课程级**评分（教学提优页用），与教师级共用聚合函数 |
| `GET /teachers/:id/score-trend?semester=&dimension=` | director（本室）/ teacher（仅自己）/ supervisor | 阶段三：趋势序列 |

**`GET /teachers/:id/evaluation-summary` 响应**：

```json
{
  "teacherId": 2, "teacherName": "李明", "semester": "2026-2027-1",
  "compositeScore": 70.42,
  "supervisorScore": 68.33,
  "agentScore": 67.86,
  "weights": { "supervisor": 0.5, "agent": 0.5 },
  "dimensions": [
    { "key": "objective",    "name": "教学目标与内容准确性", "score": 83.33, "supervisorScore": 83.33, "agentScore": null,  "weight": 0.30 },
    { "key": "content",      "name": "内容质量与深度",       "score": 70.83, "supervisorScore": 66.67, "agentScore": 75.00, "weight": 0.30 },
    { "key": "interaction",  "name": "学生互动与参与",       "score": 54.17, "supervisorScore": 50.00, "agentScore": 58.33, "weight": 0.20 },
    { "key": "organization", "name": "课堂组织与节奏",       "score": 66.67, "supervisorScore": 66.67, "agentScore": 66.67, "weight": 0.20 },
    { "key": "frontier",     "name": "前沿与交叉学科",       "score": 50.00, "supervisorScore": 41.67, "agentScore": 58.33, "weight": 0.00, "isObservation": true }
  ],
  "sample": {
    "sessionCount": 3, "supervisorCount": 3, "agentCount": 3,
    "alignedCount": 3, "sampleSufficient": true
  },
  "flags": [],
  "formulaVersion": "v1"
}
```

> 该响应与 §3.4 种子数据一一对应，可直接作为联调断言：`compositeScore` 应等于
> `0.30×83.33 + 0.30×70.83 + 0.20×54.17 + 0.20×66.67 = 70.42`，也等于三次课综合分 `58.75/70.00/82.50` 的平均。
> `frontier` 返回 `weight: 0` 与 `isObservation: true`，前端据此把它渲染为"亮点标记"而非计分维度。

> `flags` 是**必须让使用者知道的状态**，由后端显式传给前端，不让前端猜：`no_data` / `sup_only` / `ai_only` / `disjoint` / `sample_insufficient` / `agent_not_calibrated` / `formula_mixed`。

### 4.3 录音与转写（阶段二）

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| `POST /sessions/:id/recording` | supervisor | multipart 上传音频（扩展名与大小白名单校验） |
| `GET /recordings/:id/stream` | supervisor（教师不可见，见 §5.1） | 音频流式播放，支持 Range |
| `GET /sessions/:id/transcript` | 登录（数据裁剪） | 转写文本；未完成时返回 `status` 供轮询 |
| `POST /sessions/:id/transcript/retry` | supervisor | 转写失败后重试 |

### 4.4 智能体（阶段二起，未实现前统一返回 `501 / 50002`）

| 方法 路径 | 权限 | 说明 |
|-----------|------|------|
| `POST /agent/transcribe` | 内部 | 触发转写任务（异步） |
| `POST /agent/evaluate` | 内部 | 触发智能体评分（写 `evaluations` 的 agent 行） |
| `POST /agent/chat` | 登录 | 阶段三：SSE 流式对话 |

> 智能体接口**必须与主链路解耦**：调用失败只记录日志并保留 `transcripts.status='failed'`，**不得影响督导评分与主流程**。

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
> **音频对教师不可见**：音频中的学生人声与教师人声不可分割，教师端只开放**已脱敏的转写文本**（§5.2-5）。

### 5.2 隐私与合规（采集音频前必须落地）

课堂录音含**学生声音**，属个人信息。一旦开始采集即产生合规义务：

1. **告知**：录音前需课堂口头告知或课程公告
2. **访问控制 + 审计**：谁在何时听了谁的课，必须留痕（阶段二引入 `audit_logs`）
3. **保留期限**：如 1 学期后归档或删除，不得无限期保存
4. **转写脱敏**：转写文本中的学生姓名做 NER 替换（"学生A"），避免评语出现可识别个人的内容
5. **最小可见**：教师可见**自己的转写文本**用于改进；音频默认仅督导（音频里的人声不只是教师的）

### 5.3 评分伦理

- 教师管理列表**默认按姓名/工号排序**，按分排序作为显式操作
- 每行必须展示**评价次数 n**，`n < 3` 标注样本不足
- UI 显著标注「**仅用于教学支持，不作为考核依据**」
- 督导间评分校准（rater bias）列为已知局限（§10.1），阶段三引入

---

## 6. 页面与路由设计

### 6.1 路由变更表

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
| `/supervision` | supervision | 督导总览 | **改造为「听评课管理」**完整分页列表 | supervisor | 改造（不删路由） |
| `/profile` | profile | — | 不变 | 登录 | — |

> `/supervision` **不删除**：工作台只展示听评课安排前 8 条，删页会导致第 9 条以后无法访问；改造后承载完整分页与状态/日期筛选。

### 6.2 「教学提优」与课程详情的关系

**采用并列路由**：保留 `/courses/:id`；教师从**「我的课程」点击时跳 `/courses/:id/improve`**。

| 方案 | 做法 | 评价 |
|------|------|------|
| A 真替换 | `/courses/:id` 对 teacher 渲染提优页 | 路由语义随角色漂移；教师把链接发给督导，对方看到完全不同的页面 |
| B 加 Tab | 详情页内加「教学提优」Tab | 改动最小，但教师每次要多点一次 |
| **C 并列路由（采用）** | 教师从「我的课程」跳 `/courses/:id/improve` | 同时满足"点课程直接进提优"与"路径语义清晰"，教师仍可访问原详情 |

教学提优页保留课程基本信息（学分/学时/学期/班级/学生人次）与资源上传接口。

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
> 工作台**移除**「本室教师开课情况」表格与「近期开课」列表（与课程管理重复），仅保留统计卡与两个入口。

**② 督导**
```
/dashboard  统计卡 + 覆盖率环 + 听评课安排（前 8 条）＋「查看全部」
   ├─▶ /supervision  听评课管理（完整分页 + 状态/日期筛选）
   └─▶ /courses/:id  课程详情
         └─ 历史授课记录列表
               └─▶ /sessions/:id/evaluation  当堂课质量评估页
                     ├─ 音频播放器（阶段二）
                     ├─ 转写文本（阶段二，含说话人分离）
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
         └─ 智能体提优建议（按时间倒序）
```

### 6.4 前端新增目录

```
src/
├── api/
│   ├── session.ts          # 授课记录、单场次评估聚合
│   ├── teacher.ts          # 教师评分列表与面板
│   └── agent.ts            # 阶段三：对话
├── components/
│   ├── evaluation/         # 新增：评分域组件
│   │   ├── ScoreRadar.vue          # 五维雷达/条形
│   │   ├── ScoreTrendChart.vue     # 阶段三：趋势
│   │   ├── EvaluationForm.vue      # 督导评分表单（含锚点 tooltip）
│   │   ├── EvaluationCompare.vue   # 督导 vs 智能体 对比
│   │   ├── CommentPanel.vue        # 结构化评语展示
│   │   └── TranscriptViewer.vue    # 阶段二：转写文本
│   ├── session/
│   │   ├── SessionTable.vue        # 历史授课记录列表
│   │   └── AudioPlayer.vue         # 阶段二
│   └── teacher/
│       ├── TeacherScoreTable.vue
│       └── TeacherScorePanel.vue
└── views/
    ├── TeacherListView.vue
    ├── TeacherDetailView.vue
    ├── SessionEvaluationView.vue
    └── CourseImproveView.vue
```

---

## 7. 任务列表

> 分工沿用 4+n 模式：**成员一**（徐仕杰，DRI/产品）、**成员二**（刘子杰，前端）、**成员三**（后端）、**成员四**（胡凯翔，后端架构）。
> 依赖列中的编号表示必须先完成的任务。分发用的简短清单见 [`Sprint2-3-任务清单.md`](./Sprint2-3-任务清单.md)。

### 7.1 阶段一（Sprint 2.1）—— 评价闭环，不依赖智能体

| 编号 | 任务 | 负责 | 依赖 | 验收 |
|------|------|------|------|------|
| T1.1 | 引入 `golang-migrate`，编写 `V2` 迁移脚本（§3.1） | 成员四 | — | 空库执行迁移可建成 2 张表 |
| T1.2 | 扩展 `model`：`TeachingSession`、`Evaluation` | 成员四 | T1.1 | `go build` 通过，字段与 DDL 一一对应 |
| T1.3 | `pkg/scoring`：维度权重、单次总分、`Aggregate()` 纯函数 | 成员四 | — | 表驱动单测覆盖 §2.4 验算例与 §2.5.5 缺失矩阵全部 5 行 |
| T1.4 | repository：`session.go`、`evaluation.go`（含 §2.5.4 聚合 SQL） | 成员三 | T1.2 | `go test` 通过；聚合结果与手算一致 |
| T1.5 | service：`session.go`（授课记录 CRUD + 归属校验） | 成员三 | T1.4 | 越权用例返回 40302 |
| T1.6 | service：`evaluation.go`（提交评分、幂等覆盖、`formula_version` 写入） | 成员三 | T1.5 | 重复提交覆盖而非报错 |
| T1.7 | service：`teacherscore.go`（教师级/课程级聚合，单一事实源） | 成员四 | T1.3 T1.4 | 主任视角与教师视角数字**完全一致** |
| T1.8 | handler + router：§4.1 / §4.2 全部接口 | 成员三 | T1.5–T1.7 | `/healthz` 与 Sprint 1 接口无回归 |
| T1.9 | 前端 `types/` + `api/`：session / teacher 接口层 | 成员二 | T1.8 | `vue-tsc` 零错误 |
| T1.10 | 前端：课程详情页新增「历史授课记录」列表 | 成员二 | T1.9 | 三态完整（加载/空/失败） |
| T1.11 | 前端：当堂课质量评估页（评分表单 + 锚点 tooltip + 结构化评语） | 成员二 | T1.9 | 5 维 1-5 分必填校验；提交后列表即时反映 |
| T1.12 | 前端：教师管理页 `TeacherListView` | 成员二 | T1.9 | 展示评价次数 n；n<3 有样本不足标记；默认按姓名排序 |
| T1.13 | 前端：教师评分面板 `TeacherDetailView` | 成员二 | T1.12 | 综合分 + 5 维 + 按课程明细 + 历次时间线 |
| T1.14 | 前端：教学提优页 `CourseImproveView`（只读督导分 + 评语） | 成员二 | T1.9 | 保留基本信息与资源上传 |
| T1.15 | 前端：`/supervision` 改造为听评课管理；主任工作台改为入口卡 | 成员二 | — | 完整分页列表可用；工作台不再重复课程表格 |
| T1.16 | 种子数据扩展（§3.4）+ `scripts/verify.sh` 新增断言 | 成员三 | T1.8 | 新增 ≥25 条断言全绿 |
| T1.17 | 文档同步：三份 AGENTS/MySQL 文档的范围与契约章节 | 成员四 | — | 与本文档无矛盾 |

### 7.2 阶段二（Sprint 2.2）—— 智能体接入

| 编号 | 任务 | 负责 | 依赖 | 验收 |
|------|------|------|------|------|
| T2.1 | `V3` 迁移脚本（§3.2） | 成员四 | T1.1 | 迁移可重复执行 |
| T2.2 | 音频上传（白名单 + 大小限制 + 时长解析） | 成员三 | T2.1 | 非法扩展名返回 40001 |
| T2.3 | 异步转写任务框架（worker + 状态机 + 重试） | 成员四 | T2.2 | 任务失败不影响主流程；`failed` 可重试 |
| T2.4 | ASR 适配层（Qwen-Audio 接入，接口先行、实现可后补） | 成员四 | T2.3 | 未配置引擎时返回 50003 而非崩溃 |
| T2.5 | 转写脱敏（学生姓名 NER 替换） | 成员四 | T2.3 | 单测覆盖姓名替换 |
| T2.6 | 智能体评分服务（写 `evaluations` 的 agent 行） | 成员四 | T2.4 | 无法观测的维度写 `NULL` |
| T2.7 | 前端：音频播放器 + 转写查看器（含轮询） | 成员二 | T2.3 | 转写中/失败/完成三态 |
| T2.8 | 前端：评估页「智能体参考」对比面板 | 成员二 | T2.6 | 标注"AI 参考"；低置信度维度有提示 |
| T2.9 | 综合分生效：聚合 SQL 纳入 agent 侧 + `flags` 全量返回 | 成员四 | T2.6 | §2.5.5 缺失矩阵全部有单测 |
| T2.10 | 审计日志（谁听了谁的课） | 成员三 | T2.2 | 播放接口写入审计 |

### 7.3 阶段三（Sprint 3）—— 帮教师

| 编号 | 任务 | 负责 | 依赖 | 验收 |
|------|------|------|------|------|
| T3.1 | 智能体提优建议生成与展示 | 成员四/二 | T2.6 | 教师端可见，按时间倒序 |
| T3.2 | 趋势接口 + 趋势折线组件 | 成员三/二 | T1.7 | §3.4 种子数据应呈上升趋势 |
| T3.3 | 智能体对话（SSE 流式） | 成员四/二 | T2.6 | 断流可重连；失败不阻塞页面 |
| T3.4 | 教师申诉 / 督导复核流程 | 成员三/二 | T1.6 | 申诉记录留痕，状态可追溯 |
| T3.5 | 督导间评分校准（示范课基线偏移） | 成员四 | T2.9 | 校准前后分数可对比 |
| T3.6 | 质量报告导出 | 成员三 | T1.7 | 导出内容与页面数字一致 |

---

## 8. 验收标准（DoD）

### 8.1 阶段一

| 项 | 标准 |
|----|------|
| 数据链路 | 督导建授课记录 → 打分 → 教师端 10 秒内看到分数与评语 |
| **一致性** | 同一教师，主任端与教师端综合分、各维度分**逐位相同**（由单一聚合函数保证） |
| 数据范围 | 教师访问 `/teachers/3/evaluation-summary` 返回 40302；主任访问他室教师同样 40302 |
| 算法 | §2.4 验算例返回 **82.50**；§2.5.5 五种缺失组合全部有断言 |
| 恒等式 | 教师级综合分 ≡ 其各次课综合分的算术平均（§2.5.2）；用 §3.4 种子数据对账应为 **70.42** |
| 空数据 | 无评价教师综合分为 `null`、列表置底，**不得显示 0** |
| 构建 | `go build` / `go vet` / `gofmt` 零告警；`vue-tsc` 零错误；`npm run build` 通过 |

### 8.2 阶段二

| 项 | 标准 |
|----|------|
| 转写异步 | 上传 45 分钟音频不阻塞接口；轮询可见 `pending→running→done` |
| 降级 | ASR 引擎不可用时，督导评分与聚合**完全不受影响**；接口返回 50003 |
| 维度可空 | 智能体 `objective_score` 为 `NULL` 时，该维度自动退化为"只取督导侧"，不出现 NaN |
| 综合分 | 双侧齐备时按 α 融合；`disjoint` 场景返回显式提示 |
| 隐私 | 转写文本中学生姓名已脱敏；音频播放写入审计日志 |

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

---

## 10. 关键设计约定速查

下表汇总开发中必须遵守的硬性约定。**变更任一条必须先更新本文档再改代码**（契约先行）。

| # | 约定 | 出处 |
|---|------|------|
| 1 | 授课记录由督导在听课后创建；无教务对接，不自动生成 | S6.1 · T1.5 |
| 2 | 5 个维度冻结；`frontier` 为观测项，权重 0，不计入加权总分 | §2.1 · §2.2 |
| 3 | 维度权重非零项合计必须为 1.00，服务启动时校验 | §2.2 |
| 4 | 综合分 α 默认 0.5（督导 : 智能体），可配置 | §2.2 |
| 5 | 聚合一律在**维度层**完成；教师级综合分 ≡ 各次课综合分的算术平均 | §2.5.2 |
| 6 | 单次评价落库必须同时写 `formula_version`；跨表聚合不落库 | §2.5.6 |
| 7 | 所有聚合接口必须支持 `?semester=`，默认当前学期 | §2.5.6 |
| 8 | 无评价教师综合分为 `null` 且不参与排序，不得显示 0 | §2.5.5 |
| 9 | 一次课允许多个督导评分，同场次取 `AVG` | §2.5.4 |
| 10 | 评分无草稿态，`PUT` 幂等覆盖，写入即生效 | §3.1 |
| 11 | 教师只能看自己的评分与评语；音频对教师不可见，仅开放脱敏转写 | §5.1 |
| 12 | 教师评分列表默认按姓名排序，必须展示评价次数 n，标注"不作为考核依据" | §5.3 |
| 13 | 新增表只新增、不改存量表；`supervision_plans` 保持不动 | §3 |
| 14 | 转写必须异步且可重试；智能体故障不得影响督导评分主流程 | §4.4 |

### 10.1 遗留待观察项（不阻塞开发，阶段三复盘）

| 项 | 说明 |
|----|------|
| 督导间评分校准 | 当前无法消除 rater bias，阶段三 T3.5 引入示范课基线偏移校准 |
| 智能体评分效力 | α=0.5 是未经校准的等权假设，积累 ≥50 条双侧样本后应重新评估 |
| 样本量阈值 | `minSampleSize=3` 为经验值，首个学期结束后按实际分布调整 |

---

*文档版本：v1.2 · 维护人：成员四（胡凯翔）· DRI 复核：成员一（徐仕杰）*
