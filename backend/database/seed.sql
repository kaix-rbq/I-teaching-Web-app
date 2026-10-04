-- 「爱教学」Sprint 1 + Sprint 2.1 种子数据
-- 事实源：docs/MySQL数据库创建指导.md §5；授课记录与评价见开发计划 §3.4
-- 执行顺序：schema.sql → migrations（V2）→ seed.sql → make seed（make seed 用 bcrypt 覆写 password_hash）
-- 演示账号密码统一为 123456；password_hash 占位串必须由 cmd/seed 覆写，否则登录返回 40101。

USE `aijiaoxue`;

-- 幂等：重复执行先清空（注意外键顺序）
SET FOREIGN_KEY_CHECKS = 0;
TRUNCATE TABLE `evaluation_drafts`;
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

-- 听评课安排（督导：陈静 id=5；一门课程配一位督导）
--   - completed：已听评课程，关联授课记录后可在工作台直接「查看授课记录」；
--   - planned 且 planned_date = CURDATE()：出现在工作台「今日」待评队列，可一键创建并评估；
--   - 本周/本月计划用于演示队列分档（本周/本月不提供一键创建，避免未来日期建记录）。
INSERT INTO `supervision_plans` (`id`,`course_id`,`supervisor_id`,`planned_date`,`status`) VALUES
(1, 1, 5, '2026-09-12',                         'completed'),
(2, 2, 5, CURDATE(),                            'planned'),
(3, 3, 5, '2026-09-08',                         'completed'),
(4, 4, 5, CURDATE(),                            'planned'),
(5, 5, 5, '2026-09-10',                         'completed'),
(6, 6, 5, DATE_ADD(CURDATE(), INTERVAL 2 DAY),  'planned'),
(7, 5, 5, DATE_SUB(CURDATE(), INTERVAL 10 DAY), 'planned');

-- ============================================================
-- Sprint 2.1：授课记录与课堂评价（开发计划 §3.4）
-- 表结构由 migrations/V2 建。本段数据为多教师、多课程、多记录的验收样本：
--   覆盖 4 位教师（李明/张华/刘洋/赵磊）、6 门课程、14 次授课，每门课程配同一位督导（陈静）；
--   每次课同时具备督导与智能体两侧评价，且两侧均覆盖五维（智能体 objective 不再留空，
--   修复雷达图缺角）；单侧总分 = Σ w×dim（w=0.30/0.30/0.20/0.20，frontier 观测项权重 0）。
--   综合分 = 各场次综合分（α=0.5 双源融合后加权）的算术平均（§2.5.2 恒等式）。
--   数值由 pkg/scoring 口径离线计算得出，如算法变更需同步重算本段并更新注释。
-- ============================================================

-- 授课记录：s1/2/3 c1、s4/5/6 c2、s7/8 c3、s9/10 c4、s11/12 c5、s13/14 c6
-- plan_id 仅对「来自听评课计划且已完成」的记录回填，供工作台直达评估页。
INSERT INTO `teaching_sessions`
  (`id`,`course_id`,`class_id`,`teacher_id`,`session_date`,`period`,`topic`,`plan_id`,`status`) VALUES
(1,  1, 0, 2, '2026-09-12', '3-4 节', '项目立项与章程',    1,    'evaluated'),
(2,  1, 0, 2, '2026-09-19', '3-4 节', '需求调研与用户故事', NULL, 'evaluated'),
(3,  1, 0, 2, '2026-09-26', '3-4 节', '迭代计划与估点',     NULL, 'evaluated'),
(4,  2, 0, 3, '2026-09-13', '1-2 节', '进程与线程',         NULL, 'evaluated'),
(5,  2, 0, 3, '2026-09-20', '1-2 节', '死锁与调度',         NULL, 'evaluated'),
(6,  2, 0, 3, '2026-09-27', '1-2 节', '内存管理',           NULL, 'evaluated'),
(7,  3, 0, 4, '2026-09-08', '3-4 节', '关系模型与范式',     3,    'evaluated'),
(8,  3, 0, 4, '2026-09-15', '3-4 节', 'SQL 与查询优化',     NULL, 'evaluated'),
(9,  4, 0, 2, '2026-09-17', '5-6 节', '软件生命周期',       NULL, 'evaluated'),
(10, 4, 0, 2, '2026-09-24', '5-6 节', '需求工程',           NULL, 'evaluated'),
(11, 5, 0, 6, '2026-09-10', '1-2 节', '监督学习概览',       5,    'evaluated'),
(12, 5, 0, 6, '2026-09-17', '1-2 节', '模型评估指标',       NULL, 'evaluated'),
(13, 6, 0, 4, '2026-09-14', '5-6 节', '指令系统',           NULL, 'evaluated'),
(14, 6, 0, 4, '2026-09-21', '5-6 节', '存储层次',           NULL, 'evaluated');

-- 督导评价（陈静 id=5）：五维齐全；total_score 为单侧加权总分
INSERT INTO `evaluations`
  (`session_id`,`evaluator_type`,`evaluator_id`,`objective_score`,`content_score`,
   `interaction_score`,`organization_score`,`frontier_score`,`total_score`,
   `comment`,`highlights`,`improvements`,`suggestions`) VALUES
(1,'supervisor',5, 4,3,2,3,2, 52.50, '开篇结构完整，但互动偏少。',
   '课程框架清晰','提问后等待时间不足','增加案例讨论环节'),
(2,'supervisor',5, 4,4,3,4,3, 70.00, '用户故事讲解透彻，小组讨论有效。',
   '小组讨论组织得当','个别小组偏离主题','为每组设定明确产出物'),
(3,'supervisor',5, 5,4,4,4,3, 82.50, '估点练习设计巧妙，学生参与度高。',
   '练习设计贴近实战','时间略紧','预留 5 分钟总结'),
(4,'supervisor',5, 5,4,3,4,2, 77.50, '进程模型讲解清晰，课堂节奏平稳。',
   '概念对比清晰','形成性提问偏少','每讲插入一道课堂小测'),
(5,'supervisor',5, 4,4,4,4,3, 75.00, '死锁案例生动，学生讨论投入。',
   '案例贴近操作系统实践','讨论收敛稍慢','为讨论设置明确时限'),
(6,'supervisor',5, 5,5,4,5,4, 95.00, '内存管理重点突出，练习反馈及时。',
   '重难点处理到位','节奏偏快','给基础薄弱学生留缓冲'),
(7,'supervisor',5, 3,3,2,3,2, 45.00, '范式概念铺垫充分，但互动不足。',
   '概念铺垫完整','课堂互动薄弱','增加小组判断练习'),
(8,'supervisor',5, 4,4,3,4,3, 70.00, 'SQL 优化讲解到位，示例贴近实战。',
   '示例贴近实战','学生动手时间不足','安排随堂上机环节'),
(9,'supervisor',5, 4,3,3,4,2, 62.50, '生命周期串讲完整，学生参与一般。',
   '脉络清晰','案例略旧','引入近两年项目案例'),
(10,'supervisor',5, 5,4,4,4,3, 82.50, '需求访谈演练扎实，产出物明确。',
   '演练设计扎实','组间互评缺失','增加组间互评环节'),
(11,'supervisor',5, 4,3,2,3,3, 52.50, '监督学习框架清晰，但等待时间偏短。',
   '知识框架清晰','提问等待时间不足','延长等待至 3-5 秒'),
(12,'supervisor',5, 5,5,4,4,4, 90.00, '模型评估指标讲解透彻，案例丰富。',
   '指标讲解透彻','讲授时长偏多','压缩讲授增加练习'),
(13,'supervisor',5, 4,3,3,3,2, 57.50, '指令系统讲解准确，节奏略快。',
   '讲解准确','关键推导略快','放慢流水线推导'),
(14,'supervisor',5, 5,4,4,4,4, 82.50, '存储层次结构清晰，举例恰当。',
   '层次结构清晰','互动偏少','增加缓存命中问答');

-- 智能体评价：五维全覆盖（含 objective），用于修复雷达图缺角并验收双源融合
INSERT INTO `evaluations`
  (`session_id`,`evaluator_type`,`evaluator_id`,`ai_model_version`,
   `objective_score`,`content_score`,`interaction_score`,`organization_score`,`frontier_score`,
   `total_score`,`ai_confidence`) VALUES
(1,'agent',0,'qwen-audio-v1', 4,4,3,3,3, 65.00, 0.70),
(2,'agent',0,'qwen-audio-v1', 4,4,3,4,3, 70.00, 0.70),
(3,'agent',0,'qwen-audio-v1', 4,4,4,4,4, 75.00, 0.70),
(4,'agent',0,'qwen-audio-v1', 4,4,3,4,3, 70.00, 0.70),
(5,'agent',0,'qwen-audio-v1', 4,5,4,4,4, 82.50, 0.70),
(6,'agent',0,'qwen-audio-v1', 4,4,4,4,4, 75.00, 0.70),
(7,'agent',0,'qwen-audio-v1', 3,3,2,3,3, 45.00, 0.70),
(8,'agent',0,'qwen-audio-v1', 4,4,3,4,3, 70.00, 0.70),
(9,'agent',0,'qwen-audio-v1', 4,4,3,4,3, 70.00, 0.70),
(10,'agent',0,'qwen-audio-v1', 4,5,4,4,4, 82.50, 0.70),
(11,'agent',0,'qwen-audio-v1', 4,4,3,3,4, 65.00, 0.70),
(12,'agent',0,'qwen-audio-v1', 4,5,4,5,4, 87.50, 0.70),
(13,'agent',0,'qwen-audio-v1', 4,4,3,3,3, 65.00, 0.70),
(14,'agent',0,'qwen-audio-v1', 4,4,4,4,4, 75.00, 0.70);

