-- HASH Partition by author_id (Foreign Key)
-- 外部キー参照カラムでのパーティション分割

DROP TABLE IF EXISTS books_hash_author;

CREATE TABLE books_hash_author (
    id BIGINT AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, author_id),
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY HASH(author_id) PARTITIONS 8;

-- 既存データのコピー
INSERT INTO books_hash_author (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- book_tags も book_id でパーティション（JOINの最適化用）
DROP TABLE IF EXISTS book_tags_hash_bookid;

CREATE TABLE book_tags_hash_bookid (
    book_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (book_id, tag_id),
    INDEX idx_tag_id (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY HASH(book_id) PARTITIONS 8;

INSERT INTO book_tags_hash_bookid (book_id, tag_id, created_at)
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
  AND TABLE_NAME IN ('books_hash_author', 'book_tags_hash_bookid')
ORDER BY TABLE_NAME, PARTITION_ORDINAL_POSITION;
