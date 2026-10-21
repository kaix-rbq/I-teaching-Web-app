-- V13 down：还原最小课表（B11）结构。
-- 周课表可能让同一班级在一周内多次排课；旧唯一键不允许重复，回滚前保留每组最早一条。
DELETE t1 FROM `course_timetable` AS t1
JOIN `course_timetable` AS t2
  ON t1.course_id = t2.course_id
 AND t1.class_id = t2.class_id
 AND t1.teacher_id = t2.teacher_id
 AND t1.semester = t2.semester
 AND t1.id > t2.id;

ALTER TABLE `course_timetable` DROP INDEX `idx_timetable_week`;
ALTER TABLE `course_timetable`
  DROP INDEX `uk_timetable`,
  ADD UNIQUE KEY `uk_timetable` (`course_id`,`class_id`,`teacher_id`,`semester`);
ALTER TABLE `course_timetable`
  DROP COLUMN `weekday`,
  DROP COLUMN `period_start`,
  DROP COLUMN `period_end`,
  DROP COLUMN `classroom`,
  DROP COLUMN `remark`;
