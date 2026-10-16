-- V12（2026-10-16 重构 · B11/D6）：最小课表 + 授课记录建课人。
-- 事实源：docs/2026-10-16-教学提优重构开发计划.md §3.2 V12 / §3.4。
-- ① course_timetable：仅用于「创建授课」时的课程/班级下拉选择，不做同步与冲突校验；
-- ② teaching_sessions.created_by：记录建课人（教师自建为主，督导保留代录入口）。
--    过渡期按课程归属判定权限，V12 落地后再收紧为创建者本人（见计划 §4.1 DELETE /sessions/:id）。

CREATE TABLE `course_timetable` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `course_id`  BIGINT UNSIGNED NOT NULL,
  `class_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '开课班级；0=未指定',
  `teacher_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '任课教师；0=未指定（沿用课程主表）',
  `semester`   VARCHAR(16) NOT NULL DEFAULT '',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_timetable` (`course_id`,`class_id`,`teacher_id`,`semester`),
  KEY `idx_timetable_teacher` (`teacher_id`,`semester`),
  CONSTRAINT `fk_timetable_course` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='最小课表（仅下拉用）';

ALTER TABLE `teaching_sessions`
  ADD COLUMN `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '建课人 users.id；0=历史数据' AFTER `teacher_id`;
