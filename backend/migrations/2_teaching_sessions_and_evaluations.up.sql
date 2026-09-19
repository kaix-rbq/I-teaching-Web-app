-- V2（Sprint 2.1 阶段一）：授课记录与课堂评价两张表。
-- 事实源：docs/Sprint2-3-教学评价与提优-开发计划.md §3.1（DDL 一字不改）。
-- 设计要点（§3.3）：class_id 必须 NOT NULL DEFAULT 0 —— MySQL 唯一索引对 NULL 不去重；
-- 维度用列式而非 EAV：维度已冻结，列式可直接 AVG、类型安全、索引友好。

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
  `improvements`     TEXT NULL COMMENT '待改进',
  `suggestions`      TEXT NULL COMMENT '建议 / 智能体提优建议',
  `created_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_eval` (`session_id`,`evaluator_type`,`evaluator_id`),
  KEY `idx_eval_session` (`session_id`),
  CONSTRAINT `fk_eval_session` FOREIGN KEY (`session_id`) REFERENCES `teaching_sessions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='课堂评价';
