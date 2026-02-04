-- HASH Partition
-- IDのハッシュ値で均等分割

DROP TABLE IF EXISTS books_hash;

CREATE TABLE books_hash (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY HASH(id) PARTITIONS 8;

-- 既存データのコピー
INSERT INTO books_hash (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- authors も HASH パーティション
DROP TABLE IF EXISTS authors_hash;

CREATE TABLE authors_hash (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY HASH(id) PARTITIONS 4;

INSERT INTO authors_hash (id, name, created_at)
SELECT id, name, created_at FROM authors;

-- book_tags を book_id で HASH パーティション
DROP TABLE IF EXISTS book_tags_hash;

CREATE TABLE book_tags_hash (
    book_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (book_id, tag_id),
    INDEX idx_tag_id (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY HASH(book_id) PARTITIONS 8;

INSERT INTO book_tags_hash (book_id, tag_id, created_at)
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
  AND TABLE_NAME IN ('books_hash', 'authors_hash', 'book_tags_hash')
ORDER BY TABLE_NAME, PARTITION_ORDINAL_POSITION;
