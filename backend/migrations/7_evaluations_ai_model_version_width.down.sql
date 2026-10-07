-- 回滚 V7：恢复 VARCHAR(32)。
-- 必须先截断再收窄：超长值存在时，严格模式下 MODIFY 同样会报 1406。

UPDATE `evaluations` SET `ai_model_version` = LEFT(`ai_model_version`, 32);

ALTER TABLE `evaluations`
  MODIFY COLUMN `ai_model_version` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '仅 agent';
