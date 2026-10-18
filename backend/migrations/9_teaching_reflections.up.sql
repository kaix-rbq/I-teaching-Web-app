-- V9（2026-10-16 重构 · N5）：教学反思表。
-- 事实源：docs/2026-10-16-教学提优重构开发计划.md §3.2 V9。
-- 语义：教师私有的反思记录（可收藏某条评价 source_eval_id），
-- 督导在关闭改进项时可读关联反思（由 service 层裁剪，见计划 §5 权限矩阵）。

CREATE TABLE `teaching_reflections` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `teacher_id`     BIGINT UNSIGNED NOT NULL COMMENT '反思所属教师',
  `session_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关联授课记录；0=不绑定具体课次',
  `source_eval_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '收藏的评价行 evaluations.id；0=无',
  `content`        TEXT NULL COMMENT '反思内容',
  `plan`           TEXT NULL COMMENT '改进计划',
  `next_action`    TEXT NULL COMMENT '下节行动',
  `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_reflection_teacher` (`teacher_id`,`created_at`),
  KEY `idx_reflection_session` (`session_id`),
  CONSTRAINT `fk_reflection_teacher` FOREIGN KEY (`teacher_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='教学反思';
