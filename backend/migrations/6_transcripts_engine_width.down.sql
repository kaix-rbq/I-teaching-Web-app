-- 回滚 V6：恢复 VARCHAR(32)。
-- 回滚前必须先把超长值截断，否则 MODIFY 会因严格模式报 1406。
UPDATE `transcripts` SET `engine` = LEFT(`engine`, 32), `engine_version` = LEFT(`engine_version`, 32);

ALTER TABLE `transcripts`
  MODIFY COLUMN `engine` VARCHAR(32) NOT NULL DEFAULT '',
  MODIFY COLUMN `engine_version` VARCHAR(32) NOT NULL DEFAULT '';
