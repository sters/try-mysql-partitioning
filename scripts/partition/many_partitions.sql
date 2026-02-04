-- 大量パーティション（8000個）の比較
-- MySQL パーティション上限は 8192

-- =====================================================
-- HASH 8000 パーティション
-- =====================================================
DROP TABLE IF EXISTS books_hash_8000;

CREATE TABLE books_hash_8000 (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY HASH(id) PARTITIONS 8000;

INSERT INTO books_hash_8000 (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- =====================================================
-- HASH 少数パーティション（比較用）
-- =====================================================
DROP TABLE IF EXISTS books_hash_8;

CREATE TABLE books_hash_8 (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY HASH(id) PARTITIONS 8;

INSERT INTO books_hash_8 (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- =====================================================
-- HASH 中間パーティション（比較用）
-- =====================================================
DROP TABLE IF EXISTS books_hash_100;

CREATE TABLE books_hash_100 (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY HASH(id) PARTITIONS 100;

INSERT INTO books_hash_100 (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

DROP TABLE IF EXISTS books_hash_1000;

CREATE TABLE books_hash_1000 (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY HASH(id) PARTITIONS 1000;

INSERT INTO books_hash_1000 (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- =====================================================
-- KEY パーティション 8000
-- =====================================================
DROP TABLE IF EXISTS books_key_8000;

CREATE TABLE books_key_8000 (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY KEY() PARTITIONS 8000;

INSERT INTO books_key_8000 (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- =====================================================
-- テーブル情報確認
-- =====================================================
SELECT
    TABLE_NAME,
    TABLE_ROWS,
    ROUND(DATA_LENGTH/1024/1024, 2) as data_mb,
    ROUND(INDEX_LENGTH/1024/1024, 2) as index_mb
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE()
AND TABLE_NAME LIKE 'books_hash_%' OR TABLE_NAME LIKE 'books_key_%'
ORDER BY TABLE_NAME;
