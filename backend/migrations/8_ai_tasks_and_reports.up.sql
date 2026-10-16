-- V8（2026-10-16 重构 · D3）：AI 任务表 + 教师级 AI 报告表。
-- 事实源：docs/RAG知识库与AI评价接入方案.md §4.1（DDL 一字不改），
-- 与 docs/2026-10-16-教学提优重构开发计划.md §3.2 V8 对齐（知识库表顺延 V13+，本轮不建）。
-- 设计要点：
--   ① ai_tasks 以 uk_ai_task 保证「同一目标同一类任务至多一条」，重试即复位 status='pending'；
--   ② teacher_ai_reports.course_id 必须 NOT NULL DEFAULT 0 —— MySQL 唯一索引对 NULL 不去重，
--      可空会让 uk_ai_report 形同虚设（本项目已记录的坑）。

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
  UNIQUE KEY `uk_ai_task` (`kind`,`ref_type`,`ref_id`),
  KEY `idx_ai_task_status` (`status`,`kind`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 任务';

CREATE TABLE `teacher_ai_reports` (
  `id`                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `teacher_id`         BIGINT UNSIGNED NOT NULL,
  `semester`           VARCHAR(16) NOT NULL,
  `course_id`          BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `formula_version`    VARCHAR(16) NOT NULL DEFAULT 'v1',
  `ai_model_version`   VARCHAR(64) NOT NULL DEFAULT '',
  `prompt_version`     VARCHAR(16) NOT NULL DEFAULT 'v1',
  `summary`            TEXT NULL COMMENT '教师综合 AI 评价',
  `directions`         JSON NULL COMMENT '可提优方向 string[]',
  `suggestions`        JSON NULL COMMENT '提优建议 string[]',
  `highlights`         TEXT NULL,
  `trend_comment`      TEXT NULL COMMENT '历次趋势解读',
  `evidence`           JSON NULL COMMENT '引用的知识库片段 + 关键课次',
  `source_session_ids` JSON NULL COMMENT '生成时的样本快照',
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
