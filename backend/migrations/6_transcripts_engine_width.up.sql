-- V6：放宽转写引擎标识列宽。
--
-- 背景（端到端联调实测发现）：transcripts.engine_version 原为 VARCHAR(32)，
-- 而百炼真实模型名 `qwen-audio-3.1-asr-flash-filetrans` 有 34 个字符，
-- 写入时触发 MySQL 1406 (22001) Data too long，整条 UPDATE 回滚——
-- 表现为：音频识别成功、文本已拿到，但 content/segments 全部没落库，
-- 且 status 停留在 running，前端无限轮询。
--
-- 因此 engine / engine_version 一并放宽到 VARCHAR(64)：engine_version 直接存模型名，
-- 厂商模型名长度不受控（如带日期后缀的版本号），32 太紧。
-- 注：MySQL 8 严格模式下超长即报错而非截断，故必须在库侧放宽，不能只靠应用层截断。

ALTER TABLE `transcripts`
  MODIFY COLUMN `engine` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '逻辑引擎名，如 dashscope-qwen-audio-asr',
  MODIFY COLUMN `engine_version` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '具体模型名，如 qwen-audio-3.1-asr-flash-filetrans';
