# RAG 知识库与 AI 评价接入方案

> 指导性文档：只说明**策略**与**执行方法**。所有待评审项已裁定，本版为执行依据。
> 改造基线：`evaluations` 表结构已为 agent 侧预留，`pkg/scoring` 双侧融合与三档聚合已实现；缺的是 AI 产出写入通道、教师级报告表、知识库与 LLM 接入。

---

## 1. 决策摘要

| # | 决策项 | 结论 |
|---|--------|------|
| 1 | 范围边界 | 「全文搜索引擎」仅指 ES 类独立检索引擎；**进程内向量检索 + MySQL 存储不构成违规** |
| 2 | 外部模型 | **全部走 API 调用**（百炼）。嵌入与对话共用同一套凭据体系 |
| 3 | 逐维置信度/证据 | **JSON 内嵌** `evaluations.evidence`，不新增维度级表 |
| 4 | 说话人分离 | **关闭**，先跑通业务。`interaction` 维度由**教师提问与发起对话等文本线索**判定（§6.5） |
| 5 | C 类报告 | 仅**教师级**；S1 完成后**异步 + 去抖 + 快照比对**触发生成 |
| 6 | 对话 | **无状态**，前端携带最近 N 轮 `history`，不建对话表 |
| 7 | 知识库 | 由开发者用 **CLI 离线构建**；格式 md/txt/docx/pdf；内容见 §5.1 |
| 8 | S1 输入 | **不传入督导评分**，AI 独立评价，`alignedCount` 才具备一致性含义 |
| 9 | 任务表 | **新增 `ai_tasks`**，S1/S2 可观测、可重试 |

**四条贯穿全局的硬约束**：

1. **AI 全链路与督导主流程解耦**：任何失败只落 `ai_tasks.status='failed'`，不影响转写、评分与页面可用性。
2. **不做同步调用**：S1/S2 一律异步，绝不出现在上传请求的响应路径上。
3. **三样东西必须版本化落库**：`formula_version`（计分口径）、`prompt_version`（提示词）、`embedding_model`（向量模型）。任一变更都会让历史数据不可比。
4. **证据可追溯**：每个非空维度分必须附带能在转写中定位的引用；定位失败的维度降置信度或置 NULL。

---

## 2. 总体架构

```
             ┌─────────────── internal/agent ───────────────┐
  百炼 API ──▶│ llm.go      结构化输出 / SSE 流式             │
  (对话+嵌入) │ embedder.go Embedder 接口（API 实现）        │
             │ retriever.go 进程内余弦 + 集合/课程过滤       │
             │ prompt/     S1/S2/S3 模板 + 版本常量         │
             │ eval.go     S1 单次课评价                    │
             │ report.go   S2 教师级报告                    │
             │ chat.go     S3 对话（SSE）                   │
             └───────────────────┬─────────────────────────┘
                                 │  GORM（沿用现有连接）
                    MySQL（唯一存储，不新增任何基础设施）
                    ├─ ai_tasks              任务与重试
                    ├─ teacher_ai_reports    教师级 AI 报告
                    ├─ knowledge_documents   知识库文档
                    ├─ knowledge_chunks      切片 + 向量
                    └─ evaluations           agent 行（S1 落库）
```

**无向量数据库、无检索引擎服务。** 依据：本平台一学期量级为数百至数千 chunk，1024 维 float32 向量 × 3000 条约 12 MB；进程内余弦单次查询 < 5 ms。数据量超过 10 万 chunk 时再评估专用向量库。

---

## 3. 外部 API 的获取与使用

### 3.1 开通与获取凭据

1. 开通**阿里云百炼（Model Studio）**，进入控制台。
2. **创建 API Key**：控制台 → API-KEY 管理 → 创建。Key 只在创建时完整显示一次，立即保存。
   - 🔴 **API Key 按地域绑定**：北京地域的 Key 不能调用新加坡 endpoint，否则返回 **HTTP 401 `invalid_api_key`**。本项目统一使用**华北 2（北京）**。
3. **获取业务空间 ID（WorkspaceId）**：控制台 → **业务空间详情**。形如 `ws-xxxxxxxx`。
   - 本项目已在使用该 ID（ASR 链路），可直接复用。
4. **确认模型可用**：在控制台「模型列表」中确认所需的**对话模型**与**向量模型**均已开通。
   - 🔴 **模型名必须配置化，不得硬编码**：百炼模型名带版本且会迭代，写死在代码里会在厂商下线旧版本时直接不可用。
5. **确认免费额度与限流**：向量模型各版本通常有免费 Token 额度（有效期约 90 天）；对话模型按输入/输出 Token 分别计费。限流规则见控制台「限流」说明。

### 3.2 环境变量配置

沿用 ASR 链路已验证的范式（`BindEnv` 显式绑定 + `SetDefault` + 启动期 Fail-Fast + 日志只打掩码）。

```bash
# backend/.env.local（不入库，已由 .gitignore 覆盖）
export AIJIAOXUE_AGENT_API_KEY="sk-xxxxxxxx"                                # 与 ASR 可复用同一 Key
export AIJIAOXUE_AGENT_WORKSPACE_ID="ws-xxxxxxxx"
export AIJIAOXUE_AGENT_LLM_MODEL="<控制台模型列表中的对话模型名>"
export AIJIAOXUE_EMBEDDING_MODEL="text-embedding-v4"
export AIJIAOXUE_EMBEDDING_DIM="1024"
```

> 🔴 **两个必须避开的坑（ASR 链路已实测踩过）**：
> 1. `viper.AutomaticEnv` 只影响 `v.Get`，而 `viper.Unmarshal` 走 `AllSettings→AllKeys`，**未注册的键会被静默丢弃**。新增配置键必须 `SetDefault` 或 `BindEnv`，二者至少其一。
> 2. `SetEnvKeyReplacer` 只把 `.` 换成 `_`，**不拆驼峰**。`agent.apiKey` 自动推导为 `..._APIKEY`（连写）。因此统一用 `BindEnv` 显式绑定 `..._API_KEY`。

**启动期校验**：`agent.enabled=true` 但缺 Key / 缺模型名 → 直接启动失败；`embedding.dim` 与向量模型实际维度不符 → 检索前校验并报错。

### 3.3 文本嵌入 API

**Endpoint（北京，OpenAI 兼容）**

```
POST https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/embeddings
Authorization: Bearer $AIJIAOXUE_AGENT_API_KEY
Content-Type: application/json
```

```jsonc
{
  "model": "text-embedding-v4",
  "input": ["待嵌入文本1", "待嵌入文本2"],   // 批量上限见下表
  "dimensions": 1024,                       // 须与落库的 embedding_dim 一致
  "encoding_format": "float"
}
```

响应取 `data[i].embedding`（`float[]`），按 `index` 对齐输入顺序。

**可选模型（北京地域）**

| 模型 | 维度（默认加粗） | 单行最大 Token | 批量上限 | 单价（每千输入 Token） |
|---|---|---|---|---|
| `text-embedding-v4`（Qwen3-Embedding 系列） | 2048/1536/**1024**/768/512/256/128/64 | 8,192 | 10 | 0.0005 元（Batch 0.00025） |
| `text-embedding-v3` | **1024**/768/512/256/128/64 | 8,192 | 10 | 0.0005 元 |
| `qwen3.7-text-embedding` | 2560/2048/1536/**1024**/768/512/256 | 128,000 | 20 | 0.0005 元 |
| `qwen3.7-text-embedding-flash` | **1024**/768/512/256 | 128,000 | 20 | **0.000125 元** |

**选型结论**：默认 `text-embedding-v4` + `dimensions=1024`。若知识库体量大且追求最低成本，可换 `qwen3.7-text-embedding-flash`。
> 🔴 **一旦选定模型与维度，不得中途更换**：旧向量与新向量不在同一空间，混用等于返回随机结果。更换必须**全量重建**并同步更新 `embedding_model` / `embedding_dim`。

来源：[通用文本向量同步接口 API 详情](https://help.aliyun.com/zh/model-studio/text-embedding-synchronous-api)

### 3.4 评价与对话 LLM API

**Endpoint（北京，OpenAI 兼容）**

```
POST https://{WorkspaceId}.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/chat/completions
Authorization: Bearer $AIJIAOXUE_AGENT_API_KEY
Content-Type: application/json
```

```jsonc
// S1 / S2：非流式，要求结构化 JSON
{
  "model": "<对话模型名>",
  "messages": [
    { "role": "system", "content": "<S1 提示词 + 检索到的评分量表片段>" },
    { "role": "user",   "content": "<课程元信息 + 脱敏转写全文>" }
  ],
  "temperature": 0.2,                        // 评分场景要求稳定，取低值
  "response_format": { "type": "json_object" },
  "max_tokens": 2000
}

// S3：流式
{
  "model": "<对话模型名>",
  "messages": [ ... ],
  "stream": true,
  "stream_options": { "include_usage": true }
}
```

**调用约定**：

- `messages[0]` 可为 `system`，其余 `user`/`assistant` 须交替，**最后一条必须是 `user`**。
- 评分场景 `temperature` 取 0.2 以下；对话场景可放宽至 0.7。
- 流式响应为增量 `choices[0].delta.content`，以 `finish_reason="stop"` 结束；启用 `include_usage` 后末帧带 `usage`。
- **`response_format` 若模型不支持**：退化为「提示词强约束只输出 JSON」+ 服务端容错解析（剥离 ```json 代码块、截取首个 `{` 到末个 `}`、解析失败则记 `ai_tasks.failed` 并允许重试）。

来源：[OpenAI Chat 接口兼容](https://help.aliyun.com/zh/model-studio/compatibility-of-openai-with-dashscope)

### 3.5 连通性自检

落库前先跑通最小调用，避免把凭据问题误判为代码问题：

```bash
# 1) 嵌入自检：期望返回 data[0].embedding，长度 = dimensions
curl -s -X POST "https://$AIJIAOXUE_AGENT_WORKSPACE_ID.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/embeddings" \
  -H "Authorization: Bearer $AIJIAOXUE_AGENT_API_KEY" -H 'Content-Type: application/json' \
  -d "{\"model\":\"$AIJIAOXUE_EMBEDDING_MODEL\",\"input\":\"测试\",\"dimensions\":$AIJIAOXUE_EMBEDDING_DIM}" \
  | head -c 200

# 2) 对话自检：期望返回 choices[0].message.content
curl -s -X POST "https://$AIJIAOXUE_AGENT_WORKSPACE_ID.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/chat/completions" \
  -H "Authorization: Bearer $AIJIAOXUE_AGENT_API_KEY" -H 'Content-Type: application/json' \
  -d "{\"model\":\"$AIJIAOXUE_AGENT_LLM_MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"只回复两个字：正常\"}]}"
```

**常见错误定位**：`401 invalid_api_key` → Key 与 endpoint 地域不匹配（不是 Key 失效）；`404` → 模型名或 endpoint 路径错误；`403` → 模型未在控制台开通。

---

## 4. 数据模型

> **迁移编号**：V7 = 列宽修正（阶段一已实施）；V8 = AI 评价与任务；V9 = 知识库。
> 阶段一只需要列宽修正，故 AI 表与知识库表顺延一位。

### 4.1 V8 迁移：AI 评价与任务

> `ai_model_version` 列宽修正已单独落在 **V7**（`7_evaluations_ai_model_version_width`），见 §9 阶段一 1.3。

```sql
-- ① AI 任务表：可观测、可重试、幂等
CREATE TABLE `ai_tasks` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `kind`          ENUM('session_eval','teacher_report','embed_document') NOT NULL,
  `ref_type`      VARCHAR(16) NOT NULL COMMENT 'session / teacher / document',
  `ref_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `status`        ENUM('pending','running','done','failed') NOT NULL DEFAULT 'pending',
  `attempts`      TINYINT UNSIGNED NOT NULL DEFAULT 0,
  `error_message` VARCHAR(255) NOT NULL DEFAULT '',
  `payload`       JSON NULL COMMENT '任务输入快照，便于排查与重放',
  `started_at`    DATETIME NULL,
  `finished_at`   DATETIME NULL,
  `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  -- 幂等：同一目标同一类任务至多一条；「重新生成」即把 status 复位为 pending
  UNIQUE KEY `uk_ai_task` (`kind`,`ref_type`,`ref_id`),
  KEY `idx_ai_task_status` (`status`,`kind`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 任务';

-- ② 教师级 AI 报告
CREATE TABLE `teacher_ai_reports` (
  `id`                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `teacher_id`         BIGINT UNSIGNED NOT NULL,
  `semester`           VARCHAR(16) NOT NULL,
  -- 🔴 必须 NOT NULL DEFAULT 0：MySQL 唯一索引对 NULL 不去重，
  --    若 course_id 可空，uk_ai_report 形同虚设（本项目已记录的坑）
  `course_id`          BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `formula_version`    VARCHAR(16) NOT NULL DEFAULT 'v1',
  `ai_model_version`   VARCHAR(64) NOT NULL DEFAULT '',
  `prompt_version`     VARCHAR(16) NOT NULL DEFAULT 'v1',
  `summary`            TEXT NULL COMMENT '教师综合 AI 评价',
  `directions`         JSON NULL COMMENT '可提优方向 string[]（对齐前端 improveAnalysis.directions）',
  `suggestions`        JSON NULL COMMENT '提优建议 string[]（对齐前端 improveAnalysis.suggestions）',
  `highlights`         TEXT NULL,
  `trend_comment`      TEXT NULL COMMENT '历次趋势解读',
  `evidence`           JSON NULL COMMENT '引用的知识库片段 + 关键课次',
  `source_session_ids` JSON NULL COMMENT '生成时的样本快照，用于判断是否需要重算',
  `sample`             JSON NULL COMMENT 'evaluatedCount / sessionCount / flags 快照',
  `confidence`         DECIMAL(3,2) NULL,
  `status`             ENUM('pending','running','done','failed') NOT NULL DEFAULT 'pending',
  `error_message`      VARCHAR(255) NOT NULL DEFAULT '',
  `generated_at`       DATETIME NULL,
  `created_at`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_report` (`teacher_id`,`semester`,`course_id`),
  CONSTRAINT `fk_ai_report_teacher` FOREIGN KEY (`teacher_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='教师级 AI 综合报告';
```

### 4.2 V9 迁移：知识库

```sql
CREATE TABLE `knowledge_documents` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `doc_key`         VARCHAR(128) NOT NULL COMMENT '稳定标识，同一文档重复入库即覆盖',
  `title`           VARCHAR(255) NOT NULL DEFAULT '',
  `collection`      VARCHAR(32) NOT NULL COMMENT 'kb-rubric / kb-standard / kb-textbook / kb-strategy / kb-case',
  `course_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '0 = 通用，不绑定具体课程',
  `subject`         VARCHAR(64) NOT NULL DEFAULT '' COMMENT '学科，用于教材类集合过滤',
  `source_format`   ENUM('md','txt','docx','pdf') NOT NULL,
  `source_path`     VARCHAR(255) NOT NULL DEFAULT '',
  `checksum`        CHAR(64) NOT NULL COMMENT '内容 sha256，用于跳过未变更文档',
  `chunk_count`     INT UNSIGNED NOT NULL DEFAULT 0,
  `embedding_model` VARCHAR(64) NOT NULL DEFAULT '',
  `embedding_dim`   SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  `status`          ENUM('pending','done','failed') NOT NULL DEFAULT 'pending',
  `error_message`   VARCHAR(255) NOT NULL DEFAULT '',
  `ingested_at`     DATETIME NULL,
  `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_kdoc_key` (`doc_key`),
  KEY `idx_kdoc_collection` (`collection`,`course_id`,`subject`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='知识库文档';

CREATE TABLE `knowledge_chunks` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `document_id`     BIGINT UNSIGNED NOT NULL,
  `collection`      VARCHAR(32) NOT NULL,
  `course_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `subject`         VARCHAR(64) NOT NULL DEFAULT '',
  `seq`             INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '文档内顺序，便于引用定位',
  `heading`         VARCHAR(255) NOT NULL DEFAULT '' COMMENT '所属标题路径，切片时保留',
  `content`         TEXT NOT NULL,
  `token_count`     INT UNSIGNED NOT NULL DEFAULT 0,
  `embedding`       LONGBLOB NULL COMMENT 'float32 小端紧密排列；JSON 可读但体积约 3 倍',
  `embedding_model` VARCHAR(64) NOT NULL DEFAULT '',
  `embedding_dim`   SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_kc_scope` (`collection`,`course_id`,`subject`),
  KEY `idx_kc_model` (`embedding_model`,`embedding_dim`),
  CONSTRAINT `fk_kc_doc` FOREIGN KEY (`document_id`) REFERENCES `knowledge_documents` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='知识库切片与向量';
```

> **向量编码**：`[]float32` 按小端序紧密写入 `LONGBLOB`（1024 维 = 4096 字节），读取时按 `embedding_dim` 还原。检索前必须同时按 `embedding_model` 与 `embedding_dim` 过滤。

### 4.3 `evidence` JSON 规范（JSON 内嵌，不建维度级表）

写入 `evaluations.evidence`（agent 行）：

```jsonc
{
  "schemaVersion": 1,
  "citedChunks": ["kb-rubric#3", "kb-textbook#12"],   // 本次引用的知识库片段，可追溯
  "dimensions": {
    "objective":    { "confidence": 0.55, "quotes": [{"start": 123.4, "end": 145.0, "quote": "…"}] },
    "content":      { "confidence": 0.60, "quotes": [ … ] },
    "interaction":  { "confidence": 0.75, "quotes": [ … ] },
    "organization": { "confidence": 0.70, "quotes": [ … ] },
    "frontier":     { "confidence": 0.00, "quotes": [] }
  },
  "notObservable": ["frontier"],
  "promptVersion": "v1"
}
```

- `start` / `end` 单位为**秒**，与 `transcripts.segments` 对齐；
- `ai_confidence` 列存**整条**置信度（各维度置信度的加权或最小值）；
- `notObservable` 显式声明无法评价的维度，是 DB 写 `NULL` 的依据。

### 4.4 既有代码必须同步修正

| # | 位置 | 问题 | 修正 |
|---|---|---|---|
| 1 | `repository/recording.go` `UpsertAgentEvaluation` | `DoUpdates` **漏 `comment`/`highlights`/`improvements`/`suggestions`**，重复触发时评语被静默丢弃 | 四列补入 `DoUpdates` |
| 2 | `dto/evaluation.go` `EvaluationDTO` + `service/mapper.go` | 有 DB 列与 model 字段，但 DTO 未导出，前端拿不到证据 | 增加 `evidence` 字段并映射 |
| 3 | 前端 flags 字典 | 多出后端不产生的 `formula_mixed`；缺 `sup_only`/`ai_only`，页面显示英文原始 key | 与 `pkg/scoring` 对齐 |
| 4 | `pkg/scoring` `composite` | 某维度**双侧皆缺**时不归一化，分数被系统性压低（无测试覆盖） | 补归一化决策 + 单测 |

> 另：`errcode` 的 `50003` 文案写死为「转写服务不可用」。AI 链路复用该码时需参数化文案。
>
> **实施状态**：以上 4 项已在「阶段一」修复（见 §9 阶段一）。其中第 4 项为**口径变更**，
> 已按「契约先行」先更新开发计划 §2.5.2 再改代码，且对满覆盖数据零影响
> （种子对账 `TestSessionCompositeSeed` / `TestAggregateSeed` 数值不变）。

---

## 5. 知识库构建

### 5.1 集合划分与内容映射

| 集合 | 收录内容 | 主要消费者 |
|---|---|---|
| `kb-rubric` | **听评课标准**、**输出模板与提示词**、五维定义与评分锚点 | S1（评分口径）、S2、S3 |
| `kb-standard` | **国家课程标准** | S1（对照 `objective`/`content`）、S3 |
| `kb-textbook` | **学科权威教材**（按 `subject` 过滤） | S1（内容准确性）、S3 |
| `kb-strategy` | **通用教学/管理策略** | S2、S3 |
| `kb-case` | **优秀案例**（含优秀课堂实例、优秀评语范例） | S2、S3 |

> **S1 不检索 `kb-case` / `kb-strategy`**：评分需要的是"标准"，不是"范例"，混入会让打分向案例靠拢而失去一致性。

### 5.2 文档摄取流程（CLI，不做 Web 上传）

知识库由开发者构建，因此**不新增前端上传界面与权限面**，改用命令行工具 `cmd/kb`：

```bash
cd backend
go run ./cmd/kb ingest --dir ./kb-src --collection kb-rubric    # 摄取目录
go run ./cmd/kb list   --collection kb-textbook                # 查看已入库文档与切片数
go run ./cmd/kb rebuild --collection kb-textbook               # 换嵌入模型后全量重建
```

**摄取流水线**：

```
扫描目录 → 按扩展名解析 → 清洗（去页眉页脚/空行/重复标题）
        → 切片 → 批量嵌入（按模型批量上限分批）
        → 写 knowledge_documents + knowledge_chunks
```

**格式处理策略**：

| 格式 | 处理方式 |
|---|---|
| `.md` / `.txt` | 直接读取（`os.ReadFile`），md 额外解析标题层级用于切片 |
| `.docx` | 解压 zip 读 `word/document.xml`，用标准库 `encoding/xml` 抽段落文本（**无需第三方依赖**） |
| `.pdf` | **推荐入库前离线转为 md/txt**。纯 Go 抽取版式复杂 PDF 的文本质量不可靠，且**扫描版无文本层**；若必须直接入库，引入纯 Go 库并接受质量损耗 |

> 目录约定：`backend/kb-src/<collection>/…`，目录名即集合名，减少人工参数出错。

### 5.3 切片规范

| 参数 | 取值 | 说明 |
|---|---|---|
| 切片长度 | 约 **500 汉字**（≈700 Token） | 过短丢上下文，过长稀释检索精度 |
| 重叠 | **10%–15%** | 防止关键句被切断 |
| 切分边界 | **标题优先** → 段落 → 句号 | md/docx 保留标题路径写入 `heading` |
| 转写文本 | 按**连续发言窗口**聚合成 500 字块 | 保留 `session_id` 与起止时间，便于引用回原文 |

**必做**：切片内容在入库前**过 `Scrubber` 脱敏**，与转写落库同一标准——否则知识库会成为绕过脱敏的后门。

### 5.4 检索策略

```
查询构造 → 过滤（collection / course_id / subject / embedding_model+dim）
        → 进程内余弦相似度 top-k → 组装进提示词（带片段编号）
```

| 参数 | 取值 |
|---|---|
| `topK` | 5–8（S1 取 8；S3 取 5，避免上下文膨胀） |
| 相似度阈值 | 低于 0.35 的片段丢弃（宁缺勿滥，避免无关内容干扰评分） |
| 分数归一化 | 向量已 L2 归一化时，点积即余弦 |
| 缓存 | 按 `(collection, course_id, subject, embedding_model)` 缓存切片向量于内存，文档变更时失效 |

**混合检索**：元数据过滤（集合/课程/学科）承担主要精度职责，向量负责语义排序。**不引入 ES**（决策 1）。

### 5.5 增量与重建

- **增量**：摄取前比对 `checksum`（内容 sha256），未变更则跳过，避免重复计费。
- **重建**：变更 `embedding_model` 或 `embedding_dim` 后，必须 `rebuild` 全量重算；新旧向量不得混存使用（`idx_kc_model` 用于隔离与排查）。

---

## 6. AI 调用方案

### 6.1 场景矩阵

| 场景 | 触发 | 输入上下文 | 检索集合 | 落库 | 流式 |
|---|---|---|---|---|---|
| **S1 单次课评价** | 转写 `done` | ①脱敏转写全文（含句级时间戳）②课程元信息（课程名/课次/教师/日期/节次/主题）③**不含督导评分** | `kb-rubric` + `kb-standard` + `kb-textbook`(本学科) | `evaluations`(agent 行) | 否 |
| **S2 教师级报告** | S1 `done`（去抖） | ①本学期**各场次结构化评价摘要**（不喂全文转写）②维度趋势序列 ③样本量与 flags ④督导评语汇总 ⑤定向检索的弱项原文片段 | `kb-rubric` + `kb-strategy` + `kb-case` | `teacher_ai_reports` | 否 |
| **S3 提优对话** | 教师提问 | ①最近 N 轮 `history` ②该课程聚合分与维度明细 ③该课程各场次 AI 建议摘要 | `kb-rubric` + `kb-textbook` + `kb-strategy` + `kb-case` | 无（无状态） | **是** |

> **S1 与 S2 合并的收益点**：S1 一次调用同时产出「五维分 + 置信度 + 证据 + 四段评语」，保证**分数与评语依据自洽**（分开调用会出现"评语说互动不错、互动却给 2 分"）。
> **S2 绝不喂全文转写**：N 次课转写达数十万 Token，既超上下文又昂贵且噪声大。S2 的输入是 S1 的结构化产出 + **定向检索**的少量原文片段。

### 6.2 S1 输出契约

```jsonc
{
  "dimensions": {
    "objective":    {"score": 4, "confidence": 0.55, "quotes": [{"start": 123.4, "end": 145.0, "quote": "…"}]},
    "content":      {"score": 3, "confidence": 0.60, "quotes": [ … ]},
    "interaction":  {"score": 2, "confidence": 0.75, "quotes": [ … ]},
    "organization": {"score": 4, "confidence": 0.70, "quotes": [ … ]},
    "frontier":     {"score": null, "confidence": 0.0, "quotes": []}
  },
  "confidence": 0.68,
  "comment": "…", "highlights": "…", "improvements": "…", "suggestions": "…",
  "notObservable": ["frontier"],
  "citedChunks": ["kb-rubric#3", "kb-textbook#12"]
}
```

**服务端强制校验（不通过则该维降置信度或置 NULL）**：

1. `score` ∈ {1,2,3,4,5} 或 `null`；**无法观测必须写 `null`，禁止强行给分**。
2. 每个非 `null` 维度**至少 1 条 `quote`**，且 `quote` 必须能在脱敏转写中定位到子串。
3. 遵守两条硬约束：`organization` **不得以"学生安静/音量低"作为正向证据**；`frontier` **不得因基础课性质扣分**。
4. 输出必须为**纯 JSON**。

**映射到 DB**：五维 → `objective_score…frontier_score`；`confidence` → `ai_confidence`；`dimensions` + `notObservable` + `citedChunks` → `evidence`（§4.3）；四段评语 → 同名列；`formula_version` **必须与 `evaluation.formulaVersion` 一致**（否则被聚合 JOIN 静默排除，表现为 `sup_only`）；`prompt_version` 写入 `evidence`。

### 6.3 S2 输出契约

```jsonc
{
  "summary": "本学期共 5 次课…整体呈上升趋势，互动维度仍为短板…",
  "directions": ["学生互动维度相对偏弱：…", "内容深度可再加强：…"],
  "suggestions": ["提问后保持 3-5 秒沉默…", "每个核心知识点配 1 个近两年的行业案例…"],
  "highlights": "…",
  "trendComment": "综合分由 58.75 提升至 78.75，主要贡献来自 organization…",
  "confidence": 0.70,
  "citedChunks": ["kb-strategy#7"]
}
```

- `directions` / `suggestions` **直接对齐前端 `improveAnalysis.directions/suggestions`**；
- `summary` 对应新增的「教师综合 AI 评价」展示位；
- **样本不足时必须在文本中显式声明**（`evaluatedCount < minSampleSize`），不得给出确定性结论。

### 6.4 S3 提优对话（SSE）

```
POST /agent/chat
Body: { "courseId": 12, "question": "如何提升课堂互动？",
        "history": [{"role":"user","content":"…"},{"role":"agent","content":"…"}] }

响应 text/event-stream：
  data: {"delta":"针对「学生互动」："}\n\n
  data: {"delta":"你本次课的有效提问比例…"}\n\n
  data: {"done":true,"citations":[{"chunkId":"kb-rubric#3","title":"互动维度锚点","quote":"…"}]}\n\n
  data: [DONE]\n\n
错误：
  data: {"error":{"code":50003,"message":"智能体暂时不可用"}}\n\n
```

- **无状态**：`history` 由前端携带并截断至最近 N 轮（建议 ≤6 轮），服务端不落库。
- 前端需补 `AbortController`（当前 mock 仅 `clearTimeout`，无取消、无断流重连）。
- 引用来自本次检索命中，需回传 `citations` 以便前端展示出处。

### 6.5 `interaction` 维度的文本线索（说话人分离关闭时的替代口径）

不依赖音量/静音占比（既因未开说话人分离，也因标准明确禁止以"安静"为正向证据）。提示词要求 AI 从以下**语言线索**取证，并在 `quotes` 中给出原文：

| 线索 | 高分表现 | 低分表现 |
|---|---|---|
| **教师设问** | 开放性/追问式问题占比高 | 以"是不是""对不对"的封闭确认题为主 |
| **主动发起对话** | 点名提问、邀请学生表达、组织小组并巡视参与 | 全程单向讲授，无主动发起 |
| **问题后处理** | 给出思考时间、倾听后追问一层 | 自问自答、立即给答案 |
| **学生话语痕迹** | 转写中出现学生的回答、提问、讨论内容 | 只有教师独白 |
| **反馈质量** | 针对回答内容做具体评价 | 仅"很好""对"式笼统回应 |

> 这些线索可从文本判定，无需声学分离。**置信度应反映证据强度**：仅凭教师独白推断互动弱时给中等置信度；缺学生话语证据时写 `notObservable`。

---

## 7. 接口契约（需先写入 `backend_AGENTS.md` §8 再改代码）

| 方法 路径 | 权限 | 说明 |
|---|---|---|
| `GET /sessions/:id/evaluation` | 登录（裁剪） | **扩展现有接口**：`agentScore` 增加 `evidence` 结构化字段（仅加字段，向后兼容） |
| `GET /teachers/:id/ai-report?semester=` | teacher(仅本人)/director(本室)/supervisor | 读 C 类报告；无报告返回 `null` + `status` 供轮询 |
| `POST /teachers/:id/ai-report/regenerate` | supervisor/director | 手动重生成（异步，返回 `pending`） |
| `GET /courses/:id/agent-suggestions?semester=&page=&pageSize=` | 登录（裁剪） | 供 `AgentSuggestionList`，按课次倒序 |
| `POST /agent/chat` | 登录（裁剪） | SSE 流式对话 |

**`AgentSuggestion` 契约**（前端 `types/agent.ts` 已定型，后端直接对齐）：

```
id, courseId, sessionId, sessionDate, period, topic,
summary, highlights, improvements,
evidence: string,     // 字符串形态的转写片段，与 evaluations.evidence 的结构化 JSON 不同
modelVersion, confidence: number, createdAt
```

> ⚠️ `evidence` 在两处类型不同是**有意设计**：评估页需要结构化引用（可定位、可校验），建议卡片只需要人类可读片段。后端在 `agent-suggestions` 接口里把 JSON 渲染为 `转写片段：「…」（12:03–12:45）`，避免前端改类型。

**新增配置组**：

```yaml
agent:
  enabled: false
  apiKey: ""            # 仅环境变量 AIJIAOXUE_AGENT_API_KEY
  workspaceId: ""
  llmModel: ""          # 必填，模型名配置化
  timeout: "3m"         # 单次评价的超时
  maxConcurrency: 2     # 并发闸门
  reportDebounce: "10m" # S2 去抖窗口
embedding:
  model: "text-embedding-v4"
  dim: 1024
  batchSize: 10         # 对齐模型批量上限
  topK: 8
  minScore: 0.35
```

---

## 8. 触发与编排

```
督导上传录音 → 转写 done
   └─ ai_tasks(kind=session_eval, ref=session:id) 置 pending
        └─ S1 执行 → 写 evaluations(agent 行) → done
             └─ ai_tasks(kind=teacher_report, ref=teacher:id) 置 pending
                  └─ 去抖窗口到期 + 快照比对（source_session_ids 有变化）
                       └─ S2 执行 → 写 teacher_ai_reports → done
```

**规则**：

1. **全部异步**：S1/S2 只在后台 worker 执行，绝不出现在 HTTP 响应路径上。
2. **去抖**：同一教师 `reportDebounce` 窗口内多次上传只触发一次 S2。
3. **快照比对**：S2 执行前比对 `source_session_ids` 与当前已评价场次集合，无变化直接跳过。
4. **幂等**：`ai_tasks.uk_ai_task` 保证同一目标同一类任务至多一条；「重新生成」= 复位 `status='pending'`。
5. **失败可重试**：`attempts` 达上限后置 `failed`，由督导/主任在页面点「重新生成」复位。
6. **重启恢复**：进程启动时把遗留的 `running` 任务复位为 `pending` 重新入队（沿用转写链路的 `RequeueStuck` 范式）。

---

## 9. 执行计划

### 阶段 0：契约冻结（无代码）

| # | 任务 | 产出 |
|---|---|---|
| 0.1 | 把 §7 接口与 DTO 写入 `backend_AGENTS.md` §8 新增章节 | 契约事实源 |
| 0.2 | 固化 §4.3 `evidence` schema、§6.2/§6.3 输出契约、§6.4 SSE 帧格式 | 同上 |
| 0.3 | 登记 `prompt_version` / `formula_version` 策略 | 同上 |

### 阶段 1：补齐既有缺口（低风险，可独立验收）

| # | 任务 | 验收 |
|---|---|---|
| 1.1 | 修 `UpsertAgentEvaluation` 的 `DoUpdates` 补四个评语列 | 单测：重复 upsert 后评语仍在 |
| 1.2 | `EvaluationDTO` 增加 `evidence` 并在 mapper 映射 | 前端可渲染证据 |
| 1.3 | V7 迁移：`ai_model_version` → VARCHAR(64) | 迁移可重复执行 |
| 1.4 | 补 flags 常量 + 修前端 flags 字典 | 页面不出现英文原始 key |
| 1.5 | `composite` 双侧皆缺维度的归一化决策 + 单测 | 单测覆盖双侧全缺 |

### 阶段 2：AI 基础设施与评价链路

| # | 任务 | 验收 |
|---|---|---|
| 2.1 | `config` 新增 `agent.*` / `embedding.*`（`BindEnv` + `SetDefault` + Fail-Fast + 掩码） | 缺 Key 启动失败，日志只打掩码 |
| 2.2 | `internal/agent/llm`：结构化 JSON 输出、超时、重试、并发闸门 | 上游失败包装为 `ErrAgentUnavailable` |
| 2.3 | `internal/agent/prompt`：S1/S2/S3 模板 + 版本常量 | 单测：模板渲染 + JSON 容错解析 |
| 2.4 | V8 迁移：`ai_tasks` + `teacher_ai_reports` | 唯一键幂等覆盖验证 |
| 2.5 | S1 服务：转写 done → 异步评价 → 写 agent 行 | 端到端：真实音频产出五维分 + 评语 + 证据 |
| 2.6 | S2 服务：教师级报告（去抖 + 快照比对 + 幂等） | 同教师多次上传只生成一次 |
| 2.7 | 接口：`ai-report` 读写、`agent-suggestions` | 契约与 `src/types` 逐字对齐 |

### 阶段 3：知识库

| # | 任务 | 验收 |
|---|---|---|
| 3.1 | V9 迁移：`knowledge_documents` + `knowledge_chunks` | 向量与 `embedding_model/dim` 一并落库 |
| 3.2 | `internal/agent/embedder`：Embedder 接口 + 百炼实现 + 失败降级 | 单测：失败时降级为无知识库仍返回 |
| 3.3 | `internal/agent/retriever`：缓存、余弦、元数据过滤、阈值 | 单测：top-k 命中 + 模型过滤 + 阈值丢弃 |
| 3.4 | `cmd/kb`：ingest / list / rebuild（md+txt+docx 解析，PDF 预处理） | 摄取一个真实文档并检索命中 |
| 3.5 | `kb-rubric` 冷启动：导入听评课标准、五维锚点、输出模板与提示词 | 检索"互动维度锚点"能命中 |
| 3.6 | S1/S2/S3 接入检索与 `citedChunks` 回传 | 证据里可见知识库引用 |

### 阶段 4：对话与前端

| # | 任务 | 验收 |
|---|---|---|
| 4.1 | `POST /agent/chat` SSE（帧格式 + 错误 + 取消） | `curl -N` 可见分帧 |
| 4.2 | 前端 `api/agent.ts` 换真实请求（**函数签名不变**） | 页面无需改动即切真数据 |
| 4.3 | 前端 `AgentChat` 接 SSE（AbortController + onError） | 取消/失败有可见反馈 |
| 4.4 | 前端「教师综合 AI 评价」新增 `summary` 展示位 | 我的质量档案显示 AI 总结 |
| 4.5 | 删除 `src/mocks/teacherImprove.ts` | 全站无 mock |

---

## 10. 运维与降级

| 情形 | 行为 |
|---|---|
| 嵌入 API 不可用 | 降级为**无知识库**评价（仅转写 + 内置量表），`evidence.citedChunks` 为空；页面标注"未引用知识库" |
| 检索无命中（全部低于阈值） | 同上降级，不报错 |
| LLM API 失败 / 超时 | `ai_tasks.status='failed'` + `error_message`；**不影响督导评分与页面可用性**；可重试 |
| LLM 返回非法 JSON | 容错解析一次；仍失败则置 `failed` 并记录原始响应片段（截断至 255 字符） |
| 模型名下线 | 配置项切换，无需改代码；**历史数据的 `ai_model_version` 保留原值**，不做追溯改写 |
| 换嵌入模型 | `rebuild` 全量重建知识库；旧向量按 `embedding_model` 隔离，未重建前不参与检索 |
| 成本控制 | 知识库增量摄取靠 `checksum` 跳过未变更文档；S2 靠去抖与快照比对避免重复调用；LLM 与嵌入分别设并发上限 |

**可观测性**：每次 AI 调用记录 `kind / ref / model / prompt_version / 输入输出 Token / 耗时 / 结果状态`，落 `ai_tasks` 与结构化日志；`ai_model_version`、`prompt_version`、`embedding_model` 三者随产物落库，保证历史可解释。
