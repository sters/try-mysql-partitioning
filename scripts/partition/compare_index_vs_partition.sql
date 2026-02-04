-- パーティション vs インデックス 比較用テーブル作成
-- 4パターンで比較:
-- 1. インデックスなし・パーティションなし (books_bare)
-- 2. インデックスあり・パーティションなし (books - 元テーブル)
-- 3. インデックスなし・パーティションあり (books_part_no_idx)
-- 4. インデックスあり・パーティションあり (books_part_with_idx)

-- =====================================================
-- 1. インデックスなし・パーティションなし
-- =====================================================
DROP TABLE IF EXISTS books_bare;

CREATE TABLE books_bare (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    -- インデックスなし
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO books_bare (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- =====================================================
-- 2. インデックスあり・パーティションなし (元テーブル: books)
-- =====================================================
-- 既存のbooksテーブルを使用
-- INDEX: idx_author_id, idx_created_at

-- =====================================================
-- 3. インデックスなし・パーティションあり (RANGE by year)
-- =====================================================
DROP TABLE IF EXISTS books_part_no_idx;

CREATE TABLE books_part_no_idx (
    id BIGINT AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, created_at)
    -- セカンダリインデックスなし
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY RANGE (YEAR(created_at)) (
    PARTITION p2020 VALUES LESS THAN (2021),
    PARTITION p2021 VALUES LESS THAN (2022),
    PARTITION p2022 VALUES LESS THAN (2023),
    PARTITION p2023 VALUES LESS THAN (2024),
    PARTITION p2024 VALUES LESS THAN (2025),
    PARTITION p2025 VALUES LESS THAN (2026),
    PARTITION pmax VALUES LESS THAN MAXVALUE
);

INSERT INTO books_part_no_idx (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- =====================================================
-- 4. インデックスあり・パーティションあり (RANGE by year)
-- =====================================================
DROP TABLE IF EXISTS books_part_with_idx;

CREATE TABLE books_part_with_idx (
    id BIGINT AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, created_at),
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY RANGE (YEAR(created_at)) (
    PARTITION p2020 VALUES LESS THAN (2021),
    PARTITION p2021 VALUES LESS THAN (2022),
    PARTITION p2022 VALUES LESS THAN (2023),
    PARTITION p2023 VALUES LESS THAN (2024),
    PARTITION p2024 VALUES LESS THAN (2025),
    PARTITION p2025 VALUES LESS THAN (2026),
    PARTITION pmax VALUES LESS THAN MAXVALUE
);

INSERT INTO books_part_with_idx (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- =====================================================
-- author_id での RANGE パーティション比較用
-- =====================================================

-- 5. インデックスなし・パーティションあり (RANGE by author_id)
DROP TABLE IF EXISTS books_part_author_no_idx;

CREATE TABLE books_part_author_no_idx (
    id BIGINT AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, author_id)
    -- セカンダリインデックスなし
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY RANGE (author_id) (
    PARTITION p0 VALUES LESS THAN (10000),
    PARTITION p1 VALUES LESS THAN (20000),
    PARTITION p2 VALUES LESS THAN (30000),
    PARTITION p3 VALUES LESS THAN (40000),
    PARTITION p4 VALUES LESS THAN (50000),
    PARTITION p5 VALUES LESS THAN (60000),
    PARTITION p6 VALUES LESS THAN (70000),
    PARTITION p7 VALUES LESS THAN (80000),
    PARTITION p8 VALUES LESS THAN (90000),
    PARTITION p9 VALUES LESS THAN (100000),
    PARTITION pmax VALUES LESS THAN MAXVALUE
);

INSERT INTO books_part_author_no_idx (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- 6. インデックスあり・パーティションあり (RANGE by author_id)
DROP TABLE IF EXISTS books_part_author_with_idx;

CREATE TABLE books_part_author_with_idx (
    id BIGINT AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, author_id),
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY RANGE (author_id) (
    PARTITION p0 VALUES LESS THAN (10000),
    PARTITION p1 VALUES LESS THAN (20000),
    PARTITION p2 VALUES LESS THAN (30000),
    PARTITION p3 VALUES LESS THAN (40000),
    PARTITION p4 VALUES LESS THAN (50000),
    PARTITION p5 VALUES LESS THAN (60000),
    PARTITION p6 VALUES LESS THAN (70000),
    PARTITION p7 VALUES LESS THAN (80000),
    PARTITION p8 VALUES LESS THAN (90000),
    PARTITION p9 VALUES LESS THAN (100000),
    PARTITION pmax VALUES LESS THAN MAXVALUE
);

INSERT INTO books_part_author_with_idx (id, title, author_id, created_at)
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
AND TABLE_NAME IN ('books', 'books_bare', 'books_part_no_idx', 'books_part_with_idx',
                   'books_part_author_no_idx', 'books_part_author_with_idx')
ORDER BY TABLE_NAME;
