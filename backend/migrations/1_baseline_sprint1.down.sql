-- V1 回滚：按外键依赖逆序删除 Sprint 1 全部存量表。
DROP TABLE IF EXISTS `supervision_plans`;
DROP TABLE IF EXISTS `resources`;
DROP TABLE IF EXISTS `course_classes`;
DROP TABLE IF EXISTS `courses`;
DROP TABLE IF EXISTS `users`;
DROP TABLE IF EXISTS `departments`;
