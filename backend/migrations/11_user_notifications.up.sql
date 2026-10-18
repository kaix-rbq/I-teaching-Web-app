-- V11（2026-10-16 重构 · B12）：站内信。
-- 事实源：docs/2026-10-16-教学提优重构开发计划.md §3.2 V11。
-- 只推 supervisor_comment / ai_issue；payload 承载展示所需信息（标题、课程名、关联 id 等），
-- 进入详情自动已读，红点收敛到「工作台清单 / 督导任务队列」两处。

CREATE TABLE `user_notifications` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id`    BIGINT UNSIGNED NOT NULL,
  `type`       ENUM('supervisor_comment','ai_issue') NOT NULL,
  `ref_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关联对象 id（session / improvement 等）',
  `payload`    JSON NULL COMMENT '展示信息快照',
  `is_read`    TINYINT(1) NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_notification_user` (`user_id`,`is_read`,`created_at`),
  CONSTRAINT `fk_notification_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='站内信';
