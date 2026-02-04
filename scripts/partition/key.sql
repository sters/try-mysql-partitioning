-- KEY Partition
-- MySQLが内部的にハッシュ関数を使用して分割
-- 複合キーでのパーティショニングが可能

-- book_tags を複合キー (book_id, tag_id) でパーティション
DROP TABLE IF EXISTS book_tags_key;

CREATE TABLE book_tags_key (
    book_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (book_id, tag_id),
    INDEX idx_tag_id (tag_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY KEY(book_id, tag_id) PARTITIONS 16;

INSERT INTO book_tags_key (book_id, tag_id, created_at)
SELECT book_id, tag_id, created_at FROM book_tags;

-- author_tags も複合キーでパーティション
DROP TABLE IF EXISTS author_tags_key;

CREATE TABLE author_tags_key (
    author_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (author_id, tag_id),
    INDEX idx_tag_id (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY KEY(author_id, tag_id) PARTITIONS 8;

INSERT INTO author_tags_key (author_id, tag_id, created_at)
SELECT author_id, tag_id, created_at FROM author_tags;

-- books を PRIMARY KEY (id) で KEY パーティション
DROP TABLE IF EXISTS books_key;

CREATE TABLE books_key (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY KEY() PARTITIONS 8;  -- KEY() uses PRIMARY KEY by default

INSERT INTO books_key (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- パーティション情報確認
SELECT
    TABLE_NAME,
    PARTITION_NAME,
    PARTITION_METHOD,
    PARTITION_EXPRESSION,
    TABLE_ROWS
FROM INFORMATION_SCHEMA.PARTITIONS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME IN ('book_tags_key', 'author_tags_key', 'books_key')
ORDER BY TABLE_NAME, PARTITION_ORDINAL_POSITION;
