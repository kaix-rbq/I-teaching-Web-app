-- V7：放宽 evaluations.ai_model_version 列宽（32 → 64）。
--
-- 背景（与 transcripts.engine_version 同型的坑，V6 已踩过一次）：
-- 该列直接存 LLM 模型名，而厂商模型名不受本项目控制，常带日期或版本后缀
-- （如 qwen3-max-2026-01-xx），很容易超过 32 字符。MySQL 8 严格模式下超长触发
--   ERROR 1406 (22001): Data too long for column 'ai_model_version'
-- 并让**整条 UPDATE 回滚**——表现为 LLM 调用已成功并计费，但五维分与四段评语
-- 全部没落库，而外层只看到一条写库失败日志。
--
-- 因此必须在库侧放宽，不能只靠应用层截断：截断模型名会让
-- 「换模型后分数不可比」的版本追溯失效（ai_model_version 的唯一用途就是追溯）。

ALTER TABLE `evaluations`
  MODIFY COLUMN `ai_model_version` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '仅 agent：生成该行的模型名';
