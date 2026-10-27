-- V16（2026-10-27）：知识库文档与切片（RAG 方案 §4.2）。
-- 只新增表，不改既有业务表；embedding 以小端 float32 紧凑写入 LONGBLOB。
CREATE TABLE `knowledge_documents` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `doc_key`         VARCHAR(128) NOT NULL COMMENT '稳定标识，同一文档重复入库即覆盖',
  `title`           VARCHAR(255) NOT NULL DEFAULT '',
  `collection`      VARCHAR(32) NOT NULL COMMENT 'kb-rubric / kb-standard / kb-textbook / kb-strategy / kb-case',
  `course_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '0 = 通用，不绑定具体课程',
  `subject`         VARCHAR(64) NOT NULL DEFAULT '' COMMENT '学科，用于教材类集合过滤',
  `source_format`   ENUM('md','txt','docx','pdf') NOT NULL,
  `source_path`     VARCHAR(255) NOT NULL DEFAULT '',
  `checksum`        CHAR(64) NOT NULL COMMENT '内容 sha256，用于跳过未变更文档',
  `chunk_count`     INT UNSIGNED NOT NULL DEFAULT 0,
  `embedding_model` VARCHAR(64) NOT NULL DEFAULT '',
  `embedding_dim`   SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  `status`          ENUM('pending','done','failed') NOT NULL DEFAULT 'pending',
  `error_message`   VARCHAR(255) NOT NULL DEFAULT '',
  `ingested_at`     DATETIME NULL,
  `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_kdoc_key` (`doc_key`),
  KEY `idx_kdoc_collection` (`collection`,`course_id`,`subject`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='知识库文档';

CREATE TABLE `knowledge_chunks` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `document_id`     BIGINT UNSIGNED NOT NULL,
  `collection`      VARCHAR(32) NOT NULL,
  `course_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `subject`         VARCHAR(64) NOT NULL DEFAULT '',
  `seq`             INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '文档内顺序，便于引用定位',
  `heading`         VARCHAR(255) NOT NULL DEFAULT '' COMMENT '所属标题路径，切片时保留',
  `content`         TEXT NOT NULL,
  `token_count`     INT UNSIGNED NOT NULL DEFAULT 0,
  `embedding`       LONGBLOB NULL COMMENT 'float32 小端紧密排列；JSON 可读但体积约 3 倍',
  `embedding_model` VARCHAR(64) NOT NULL DEFAULT '',
  `embedding_dim`   SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_kc_scope` (`collection`,`course_id`,`subject`),
  KEY `idx_kc_model` (`embedding_model`,`embedding_dim`),
  CONSTRAINT `fk_kc_doc` FOREIGN KEY (`document_id`) REFERENCES `knowledge_documents` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='知识库切片与向量';
