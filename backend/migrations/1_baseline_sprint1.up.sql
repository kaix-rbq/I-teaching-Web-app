-- V1 基线：Sprint 1 全部存量表（内容与 database/schema.sql 一致，事实源：docs/MySQL数据库创建指导.md）。
-- 说明：沿用 IF NOT EXISTS，使「已按 schema.sql 手工建过表的库」执行本迁移时不报错，只补记版本号。
-- 建库与开发账号不在迁移范围内（golang-migrate 不建库），见 scripts/init_db.sh 第 1 步。

-- 4.1 departments — 教研室
CREATE TABLE IF NOT EXISTS `departments` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name`       VARCHAR(64)  NOT NULL COMMENT '教研室名称',
  `code`       VARCHAR(16)  NOT NULL COMMENT '编码，如 SE/CS/AI（课程编码前缀来源）',
  `created_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_dept_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='教研室';

-- 4.2 users — 用户（三角色）
CREATE TABLE IF NOT EXISTS `users` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username`      VARCHAR(32) NOT NULL COMMENT '登录名（演示账号：director/teacher/supervisor）',
  `password_hash` VARCHAR(72) NOT NULL COMMENT 'bcrypt 哈希，由 cmd/seed 写入，禁止明文',
  `name`          VARCHAR(32) NOT NULL COMMENT '姓名',
  `role`          ENUM('director','teacher','supervisor') NOT NULL COMMENT '角色',
  `department_id` BIGINT UNSIGNED NULL COMMENT '所属教研室（督导可为 NULL）',
  `job_no`        VARCHAR(16) NOT NULL COMMENT '工号',
  `status`        TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '1 启用 / 0 停用',
  `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  KEY `idx_user_dept` (`department_id`),
  CONSTRAINT `fk_user_dept` FOREIGN KEY (`department_id`) REFERENCES `departments` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户';

-- 4.3 courses — 课程主表
CREATE TABLE IF NOT EXISTS `courses` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code`          VARCHAR(12)  NOT NULL COMMENT '课程编码，如 SE3101',
  `name`          VARCHAR(128) NOT NULL COMMENT '课程名称',
  `credit`        TINYINT UNSIGNED NOT NULL COMMENT '学分 1-6',
  `hours`         SMALLINT UNSIGNED NOT NULL COMMENT '学时 16-128',
  `semester`      VARCHAR(16)  NOT NULL COMMENT '学期，如 2026-2027-1',
  `department_id` BIGINT UNSIGNED NOT NULL COMMENT '开课教研室',
  `teacher_id`    BIGINT UNSIGNED NOT NULL COMMENT '授课教师',
  `description`   VARCHAR(500) NOT NULL DEFAULT '' COMMENT '课程简介',
  `status`        ENUM('open','draft','closed') NOT NULL DEFAULT 'draft' COMMENT '开课状态',
  `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_course_code_semester` (`code`, `semester`),
  KEY `idx_course_dept` (`department_id`),
  KEY `idx_course_teacher` (`teacher_id`),
  KEY `idx_course_semester_status` (`semester`, `status`),
  CONSTRAINT `fk_course_dept` FOREIGN KEY (`department_id`) REFERENCES `departments` (`id`),
  CONSTRAINT `fk_course_teacher` FOREIGN KEY (`teacher_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='课程';

-- 4.4 course_classes — 开课班级
CREATE TABLE IF NOT EXISTS `course_classes` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `course_id`     BIGINT UNSIGNED NOT NULL,
  `class_name`    VARCHAR(32)  NOT NULL COMMENT '班级名，如 软工2201',
  `schedule`      VARCHAR(64)  NOT NULL COMMENT '上课时间，如 周一 3-4 节',
  `location`      VARCHAR(64)  NOT NULL COMMENT '上课地点',
  `student_count` SMALLINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '学生数',
  `created_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_class_course` (`course_id`),
  CONSTRAINT `fk_class_course` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='开课班级';

-- 4.5 resources — 课程资源
CREATE TABLE IF NOT EXISTS `resources` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `course_id`   BIGINT UNSIGNED NOT NULL,
  `name`        VARCHAR(128) NOT NULL COMMENT '展示文件名',
  `type`        ENUM('pdf','doc','ppt','video','zip','other') NOT NULL COMMENT '按扩展名映射',
  `size`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '字节数',
  `file_path`   VARCHAR(255) NOT NULL COMMENT '相对存储路径 uploads/{courseId}/{uuid}{ext}',
  `uploader_id` BIGINT UNSIGNED NOT NULL COMMENT '上传人',
  `uploaded_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `created_at`  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_res_course` (`course_id`),
  KEY `idx_res_uploader` (`uploader_id`),
  CONSTRAINT `fk_res_course` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`),
  CONSTRAINT `fk_res_uploader` FOREIGN KEY (`uploader_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='课程资源';

-- 4.6 supervision_plans — 听评课安排
CREATE TABLE IF NOT EXISTS `supervision_plans` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `course_id`      BIGINT UNSIGNED NOT NULL COMMENT '被听评课程',
  `supervisor_id`  BIGINT UNSIGNED NOT NULL COMMENT '督导',
  `planned_date`   DATE NOT NULL COMMENT '计划听课日期',
  `status`         ENUM('planned','completed') NOT NULL DEFAULT 'planned',
  `created_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_plan_status` (`status`),
  KEY `idx_plan_course` (`course_id`),
  KEY `idx_plan_supervisor` (`supervisor_id`),
  KEY `idx_plan_date` (`planned_date`),
  CONSTRAINT `fk_plan_course` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`),
  CONSTRAINT `fk_plan_supervisor` FOREIGN KEY (`supervisor_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='听评课安排';
