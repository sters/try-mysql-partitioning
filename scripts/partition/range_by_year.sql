-- RANGE Partition by Year (created_at)
-- 年ごとにパーティション分割

-- books テーブルを年別でパーティション化
-- 注意: パーティションキーはPRIMARY KEYに含める必要がある
-- 外部キー制約は使えないため、author_idの制約は削除される

DROP TABLE IF EXISTS books_range_year;

CREATE TABLE books_range_year (
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

-- 既存データのコピー
INSERT INTO books_range_year (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- book_tags も年別でパーティション化
DROP TABLE IF EXISTS book_tags_range_year;

CREATE TABLE book_tags_range_year (
    book_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (book_id, tag_id, created_at),
    INDEX idx_tag_id (tag_id),
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

INSERT INTO book_tags_range_year (book_id, tag_id, created_at)
SELECT book_id, tag_id, created_at FROM book_tags;

-- パーティション情報確認
SELECT
    TABLE_NAME,
    PARTITION_NAME,
    PARTITION_METHOD,
    PARTITION_EXPRESSION,
    TABLE_ROWS
FROM INFORMATION_SCHEMA.PARTITIONS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME IN ('books_range_year', 'book_tags_range_year')
ORDER BY TABLE_NAME, PARTITION_ORDINAL_POSITION;
