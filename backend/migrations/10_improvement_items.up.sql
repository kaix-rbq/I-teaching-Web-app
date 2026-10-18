-- V10（2026-10-16 重构 · N4）：待改进清单（简化版）。
-- 事实源：docs/2026-10-16-教学提优重构开发计划.md §3.2 V10 + §4.4。
-- 状态机：open → done_self（教师自评完成） → closed（督导确认关闭）；
-- 不支持打回，打回即退回 open 并留言（done_note 复用为留言）。
-- title 为 §4.4 「N4 简化版：标题+来源+完成说明+三态」所要求的标题列。

CREATE TABLE `improvement_items` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `teacher_id`  BIGINT UNSIGNED NOT NULL,
  `session_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '来源授课记录；0=不绑定',
  `title`       VARCHAR(255) NOT NULL DEFAULT '' COMMENT '改进项标题',
  `source`      VARCHAR(32)  NOT NULL DEFAULT '' COMMENT '来源：ai / supervisor / self',
  `status`      ENUM('open','done_self','closed') NOT NULL DEFAULT 'open',
  `done_note`   TEXT NULL COMMENT '完成说明（教师自评）/ 打回留言',
  `closed_by`   BIGINT UNSIGNED NULL COMMENT '确认关闭的督导 user_id',
  `closed_at`   DATETIME NULL,
  `created_at`  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_improvement_teacher` (`teacher_id`,`status`),
  KEY `idx_improvement_session` (`session_id`),
  CONSTRAINT `fk_improvement_teacher` FOREIGN KEY (`teacher_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='待改进清单';
