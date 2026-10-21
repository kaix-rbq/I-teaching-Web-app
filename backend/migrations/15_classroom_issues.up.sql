-- V15（2026-10-20 第三轮 · Task B3）：课堂问题登记。
-- 教师登记 → 督导/主任处理（handled）→ 教师确认（解决硬删除 / 未解决回退 open）。
CREATE TABLE `classroom_issues` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `teacher_id`  BIGINT UNSIGNED NOT NULL,
  `issue_type`  ENUM('equipment','absence','progress','material','other') NOT NULL DEFAULT 'other',
  `content`     VARCHAR(2000) NOT NULL,
  `status`      ENUM('open','handled') NOT NULL DEFAULT 'open',
  `handle_note` VARCHAR(1000) NOT NULL DEFAULT '',
  `handler_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `handled_at`  DATETIME NULL,
  `created_at`  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_issue_teacher_status` (`teacher_id`,`status`),
  KEY `idx_issue_status_created` (`status`,`created_at`),
  CONSTRAINT `fk_issue_teacher` FOREIGN KEY (`teacher_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='课堂问题登记';
