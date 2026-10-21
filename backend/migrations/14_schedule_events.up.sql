-- V14（2026-10-20 第三轮 · Task B2）：主任事务（周视图事务管理卡）。
-- 仅 owner 本人可读写；时间校验与重叠判断在 service 层。
CREATE TABLE `schedule_events` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `owner_user_id` BIGINT UNSIGNED NOT NULL,
  `event_date`    DATE NOT NULL,
  `start_time`    TIME NOT NULL,
  `end_time`      TIME NOT NULL,
  `event_type`    ENUM('meeting','open_class','lecture','other') NOT NULL DEFAULT 'other',
  `type_text`     VARCHAR(32) NOT NULL DEFAULT '' COMMENT '「其他」的自填名称',
  `remark`        VARCHAR(255) NOT NULL DEFAULT '',
  `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_schedule_owner_date` (`owner_user_id`,`event_date`,`start_time`),
  CONSTRAINT `fk_schedule_owner` FOREIGN KEY (`owner_user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='主任事务';
