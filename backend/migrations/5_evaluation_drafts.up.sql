-- V5（督导草稿箱）：评估草稿与正式评价分表存放。
-- 设计要点：草稿是「未生效」的私人工作副本，不能污染 evaluations 聚合口径，
-- 故独立建表；同一督导对同一场次至多一份草稿（uk_draft），提交后删除。
-- 维度分允许 NULL（草稿可只填写部分维度），提交时由 service 层强制五维齐全。

CREATE TABLE `evaluation_drafts` (
  `id`                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id`         BIGINT UNSIGNED NOT NULL COMMENT '授课记录',
  `supervisor_id`      BIGINT UNSIGNED NOT NULL COMMENT '草稿所属督导',
  `objective_score`    TINYINT UNSIGNED NULL,
  `content_score`      TINYINT UNSIGNED NULL,
  `interaction_score`  TINYINT UNSIGNED NULL,
  `organization_score` TINYINT UNSIGNED NULL,
  `frontier_score`     TINYINT UNSIGNED NULL,
  `comment`            TEXT NULL,
  `highlights`         TEXT NULL COMMENT '亮点',
  `improvements`       TEXT NULL COMMENT '待改进',
  `suggestions`        TEXT NULL COMMENT '建议',
  `created_at`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_draft` (`session_id`,`supervisor_id`),
  KEY `idx_draft_supervisor` (`supervisor_id`),
  CONSTRAINT `fk_draft_session` FOREIGN KEY (`session_id`) REFERENCES `teaching_sessions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='评估草稿';
