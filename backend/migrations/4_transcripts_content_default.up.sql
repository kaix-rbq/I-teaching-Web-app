-- 转写结果在 ASR 完成前允许为空：为 content 补默认值，
-- 避免任何绕过 GORM 的写入在 STRICT_TRANS_TABLES 下触发 1364。
ALTER TABLE `transcripts`
  MODIFY COLUMN `content` MEDIUMTEXT NOT NULL DEFAULT ('');
