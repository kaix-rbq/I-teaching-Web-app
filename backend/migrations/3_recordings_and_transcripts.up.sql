CREATE TABLE `recordings` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id` BIGINT UNSIGNED NOT NULL,
  `file_path` VARCHAR(255) NOT NULL,
  `original_name` VARCHAR(128) NOT NULL,
  `format` VARCHAR(16) NOT NULL,
  `size` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `duration_sec` INT UNSIGNED NOT NULL DEFAULT 0,
  `uploaded_by` BIGINT UNSIGNED NOT NULL,
  `uploaded_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rec_session` (`session_id`),
  KEY `idx_rec_uploaded_by` (`uploaded_by`),
  CONSTRAINT `fk_rec_session` FOREIGN KEY (`session_id`) REFERENCES `teaching_sessions` (`id`),
  CONSTRAINT `fk_rec_uploaded_by` FOREIGN KEY (`uploaded_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='课堂录音';

CREATE TABLE `transcripts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id` BIGINT UNSIGNED NOT NULL,
  `recording_id` BIGINT UNSIGNED NOT NULL,
  `content` MEDIUMTEXT NOT NULL,
  `segments` JSON NULL,
  `engine` VARCHAR(32) NOT NULL DEFAULT '',
  `engine_version` VARCHAR(32) NOT NULL DEFAULT '',
  `status` ENUM('pending','running','done','failed') NOT NULL DEFAULT 'pending',
  `error_message` VARCHAR(255) NOT NULL DEFAULT '',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tr_session` (`session_id`),
  KEY `idx_tr_recording` (`recording_id`),
  CONSTRAINT `fk_tr_session` FOREIGN KEY (`session_id`) REFERENCES `teaching_sessions` (`id`),
  CONSTRAINT `fk_tr_recording` FOREIGN KEY (`recording_id`) REFERENCES `recordings` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='课堂转写';

CREATE TABLE `recording_playback_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `recording_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `started_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_playback_recording` (`recording_id`),
  CONSTRAINT `fk_playback_recording` FOREIGN KEY (`recording_id`) REFERENCES `recordings` (`id`),
  CONSTRAINT `fk_playback_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='录音播放审计日志';
