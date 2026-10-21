-- V13（2026-10-20 第三轮 · Task B1）：课表扩展到周网格。
-- weekday 1=周一 … 7=周日；0=历史数据未排课（前端不渲染进网格）。
-- period_start/period_end 1-11 节；0=未指定。classroom/remark 空串=未填写。
-- 唯一键加入 weekday/period_start，允许同一课程班级在一周内多次排课；
-- 历史行 weekday=0/period_start=0 时仍保持原有「同课同班同教师同学期唯一」语义。

ALTER TABLE `course_timetable`
  ADD COLUMN `weekday`      TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '1=周一…7=周日；0=未排课' AFTER `semester`,
  ADD COLUMN `period_start` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '起始节次 1-11；0=未指定' AFTER `weekday`,
  ADD COLUMN `period_end`   TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '结束节次 1-11；0=未指定' AFTER `period_start`,
  ADD COLUMN `classroom`    VARCHAR(64) NOT NULL DEFAULT '' COMMENT '上课教室' AFTER `period_end`,
  ADD COLUMN `remark`       VARCHAR(255) NOT NULL DEFAULT '' COMMENT '教师备注' AFTER `classroom`;

ALTER TABLE `course_timetable`
  DROP INDEX `uk_timetable`,
  ADD UNIQUE KEY `uk_timetable` (`course_id`,`class_id`,`teacher_id`,`semester`,`weekday`,`period_start`);

ALTER TABLE `course_timetable`
  ADD KEY `idx_timetable_week` (`teacher_id`,`semester`,`weekday`,`period_start`);
