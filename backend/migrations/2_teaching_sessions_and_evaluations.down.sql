-- V2 回滚：先删评价（外键依赖授课记录），再删授课记录。
DROP TABLE IF EXISTS `evaluations`;
DROP TABLE IF EXISTS `teaching_sessions`;
