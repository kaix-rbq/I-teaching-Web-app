-- 「爱教学」Sprint 1 + Sprint 2.1 种子数据
-- 事实源：docs/MySQL数据库创建指导.md §5；授课记录与评价见开发计划 §3.4
-- 执行顺序：schema.sql → migrations（V2）→ seed.sql → make seed（make seed 用 bcrypt 覆写 password_hash）
-- 演示账号密码统一为 123456；password_hash 占位串必须由 cmd/seed 覆写，否则登录返回 40101。

USE `aijiaoxue`;

-- 幂等：重复执行先清空（注意外键顺序）
SET FOREIGN_KEY_CHECKS = 0;
TRUNCATE TABLE `evaluations`;
TRUNCATE TABLE `teaching_sessions`;
TRUNCATE TABLE `supervision_plans`;
TRUNCATE TABLE `resources`;
TRUNCATE TABLE `course_classes`;
TRUNCATE TABLE `courses`;
TRUNCATE TABLE `users`;
TRUNCATE TABLE `departments`;
SET FOREIGN_KEY_CHECKS = 1;

-- 教研室
INSERT INTO `departments` (`id`, `name`, `code`) VALUES
(1, '软件工程教研室', 'SE'),
(2, '计算机系统教研室', 'CS'),
(3, '人工智能教研室', 'AI');

-- 用户（password_hash 为占位，见下方"密码哈希"说明）
INSERT INTO `users` (`id`, `username`, `password_hash`, `name`, `role`, `department_id`, `job_no`) VALUES
(1, 'director',   'REPLACE_WITH_BCRYPT_HASH', '王建国', 'director',   1,    'T2001'),
(2, 'teacher',    'REPLACE_WITH_BCRYPT_HASH', '李明',   'teacher',    1,    'T2011'),
(3, 'zhanghua',   'REPLACE_WITH_BCRYPT_HASH', '张华',   'teacher',    1,    'T2012'),
(4, 'liuyang',    'REPLACE_WITH_BCRYPT_HASH', '刘洋',   'teacher',    2,    'T2021'),
(5, 'supervisor', 'REPLACE_WITH_BCRYPT_HASH', '陈静',   'supervisor', NULL, 'T3001'),
(6, 'zhaolei',    'REPLACE_WITH_BCRYPT_HASH', '赵磊',   'teacher',    3,    'T2031');

-- 课程（当前学期 2026-2027-1；c8 为上学期已结课，用于状态标签演示）
INSERT INTO `courses` (`id`, `code`, `name`, `credit`, `hours`, `semester`, `department_id`, `teacher_id`, `description`, `status`) VALUES
(1, 'SE3101', '软件项目管理',     3, 48, '2026-2027-1', 1, 2, '以用户故事地图与敏捷迭代方法为主线，覆盖立项、规划、执行到收尾全流程。', 'open'),
(2, 'SE2104', '操作系统',         4, 64, '2026-2027-1', 1, 3, '进程管理、内存管理、文件系统与 I/O 子系统的原理与实践。', 'open'),
(3, 'CS3302', '数据库原理',       3, 48, '2026-2027-1', 2, 4, '关系模型、SQL、事务与并发控制、索引与查询优化。', 'open'),
(4, 'SE3205', '软件工程导论',     2, 32, '2026-2027-1', 1, 2, '软件生命周期、需求工程与基础设计方法。', 'open'),
(5, 'AI4101', '机器学习',         3, 48, '2026-2027-1', 3, 6, '监督学习、模型评估与经典算法实践。', 'open'),
(6, 'CS2101', '计算机组成原理',   4, 64, '2026-2027-1', 2, 4, '指令系统、CPU 结构、存储层次与总线。', 'open'),
(7, 'SE4102', '软件测试技术',     2, 32, '2026-2027-1', 1, 3, '测试用例设计、自动化测试与质量度量（建设中）。', 'draft'),
(8, 'AI4202', '深度学习',         3, 48, '2025-2026-2', 3, 6, '神经网络基础与主流框架实践。', 'closed');

-- 开课班级
INSERT INTO `course_classes` (`course_id`, `class_name`, `schedule`, `location`, `student_count`) VALUES
(1, '软工2201',  '周一 3-4 节', '逸夫楼301', 40),
(1, '软工2202',  '周一 3-4 节', '逸夫楼302', 46),
(2, '软件2301',  '周二 1-2 节', '综合楼201', 44),
(2, '软件2302',  '周二 1-2 节', '综合楼202', 42),
(2, '软件2303',  '周二 1-2 节', '综合楼203', 42),
(3, '计科2301',  '周三 3-4 节', '知行楼105', 48),
(3, '计科2302',  '周三 3-4 节', '知行楼106', 47),
(4, '软工2401',  '周四 5-6 节', '逸夫楼201', 45),
(4, '软工2402',  '周四 5-6 节', '逸夫楼202', 44),
(5, '智科2401',  '周五 1-2 节', '智慧楼301', 50),
(5, '智科2402',  '周五 1-2 节', '智慧楼302', 48),
(6, '计科2401',  '周一 5-6 节', '知行楼201', 46),
(6, '计科2402',  '周一 5-6 节', '知行楼202', 45),
(8, '智科2301',  '周三 1-2 节', '智慧楼201', 50);

-- 课程资源（file_path 为示例相对路径，实际由上传接口生成）
INSERT INTO `resources` (`course_id`, `name`, `type`, `size`, `file_path`, `uploader_id`, `uploaded_at`) VALUES
(1, '第01讲-项目立项与章程.pdf', 'pdf',  2516582,  'uploads/1/3f2a-demo-01.pdf', 2, '2026-09-02 09:30:00'),
(1, '课程大纲-2026版.doc',       'doc',  483328,   'uploads/1/3f2a-demo-02.doc', 2, '2026-09-01 14:00:00'),
(1, '实验1-需求调研模板.zip',    'zip',  1048576,  'uploads/1/3f2a-demo-03.zip', 2, '2026-09-08 10:15:00'),
(2, '第01讲-操作系统概述.ppt',   'ppt',  15728640, 'uploads/2/5b1c-demo-01.ppt', 3, '2026-09-03 08:45:00'),
(3, '第01讲-数据库绪论.pdf',     'pdf',  3145728,  'uploads/3/7d4e-demo-01.pdf', 4, '2026-09-04 16:20:00'),
(5, '第01讲-机器学习概述.pdf',   'pdf',  4194304,  'uploads/5/9c6f-demo-01.pdf', 6, '2026-09-05 11:00:00');

-- 听评课安排（督导：陈静；覆盖 c1/c3/c5 三门已完成的听评）
INSERT INTO `supervision_plans` (`course_id`, `supervisor_id`, `planned_date`, `status`) VALUES
(1, 5, '2026-09-12', 'completed'),
(2, 5, '2026-09-18', 'planned'),
(3, 5, '2026-09-08', 'completed'),
(5, 5, '2026-09-10', 'completed'),
(6, 5, '2026-09-22', 'planned');

-- ============================================================
-- Sprint 2.1：授课记录与课堂评价（开发计划 §3.4）
-- 表结构由 migrations/V2 建；此处数据与 §3.4 对账基准一一对应：
--   第 1/2/3 次课督导总分 52.50 / 70.00 / 82.50，教师级 68.33；
--   智能体总分 60.71 / 67.86 / 75.00（objective 恒 NULL，验证维度可空融合）；
--   综合分 58.75 / 70.00 / 82.50，教师级 70.42（= 三次综合分的算术平均）。
-- ============================================================

-- 为 c1（软件项目管理，教师李明 id=2）补 3 次授课记录与评价
INSERT INTO `teaching_sessions`
  (`course_id`,`class_id`,`teacher_id`,`session_date`,`period`,`topic`,`status`) VALUES
(1, 0, 2, '2026-09-12', '3-4 节', '项目立项与章程',     'evaluated'),
(1, 0, 2, '2026-09-19', '3-4 节', '需求调研与用户故事', 'evaluated'),
(1, 0, 2, '2026-09-26', '3-4 节', '迭代计划与估点',     'evaluated');

-- 督导（陈静 id=5）评价：3 次，逐次提升以演示"趋势"
INSERT INTO `evaluations`
  (`session_id`,`evaluator_type`,`evaluator_id`,`objective_score`,`content_score`,
   `interaction_score`,`organization_score`,`frontier_score`,`total_score`,
   `comment`,`highlights`,`improvements`) VALUES
(1,'supervisor',5, 4,3,2,3,2, 52.50, '开篇结构完整，但互动偏少。',
   '课程框架清晰','提问后等待时间不足','增加案例讨论环节'),
(2,'supervisor',5, 4,4,3,4,3, 70.00, '用户故事讲解透彻，小组讨论有效。',
   '小组讨论组织得当','个别小组偏离主题','为每组设定明确产出物'),
(3,'supervisor',5, 5,4,4,4,3, 82.50, '估点练习设计巧妙，学生参与度高。',
   '练习设计贴近实战','时间略紧','预留 5 分钟总结');

-- 智能体评价（阶段二启用；objective 恒为 NULL，用于验收"维度可空"的融合逻辑）
INSERT INTO `evaluations`
  (`session_id`,`evaluator_type`,`evaluator_id`,`ai_model_version`,
   `objective_score`,`content_score`,`interaction_score`,`organization_score`,`frontier_score`,
   `total_score`,`ai_confidence`) VALUES
(1,'agent',0,'qwen-audio-v1', NULL,4,3,3,3, 60.71, 0.72),
(2,'agent',0,'qwen-audio-v1', NULL,4,3,4,3, 67.86, 0.68),
(3,'agent',0,'qwen-audio-v1', NULL,4,4,4,4, 75.00, 0.75);
