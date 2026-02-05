-- RANGE パーティション数の比較（8 / 100 / 1000 / 8000）
-- HASH との比較のため同じパーティション数で検証

-- =====================================================
-- RANGE 8 パーティション（ID範囲）
-- =====================================================
DROP TABLE IF EXISTS books_range_8;

CREATE TABLE books_range_8 (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY RANGE (id) (
    PARTITION p0 VALUES LESS THAN (125000),
    PARTITION p1 VALUES LESS THAN (250000),
    PARTITION p2 VALUES LESS THAN (375000),
    PARTITION p3 VALUES LESS THAN (500000),
    PARTITION p4 VALUES LESS THAN (625000),
    PARTITION p5 VALUES LESS THAN (750000),
    PARTITION p6 VALUES LESS THAN (875000),
    PARTITION p7 VALUES LESS THAN MAXVALUE
);

INSERT INTO books_range_8 (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- =====================================================
-- RANGE 100 パーティション（ID範囲）
-- =====================================================
DROP TABLE IF EXISTS books_range_100;

DROP PROCEDURE IF EXISTS create_range_100_partitions;

DELIMITER //
CREATE PROCEDURE create_range_100_partitions()
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE partition_sql TEXT DEFAULT '';
    DECLARE step INT DEFAULT 10000; -- 1M / 100 = 10000

    SET partition_sql = 'CREATE TABLE books_range_100 (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        author_id BIGINT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        INDEX idx_author_id (author_id),
        INDEX idx_created_at (created_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
    PARTITION BY RANGE (id) (';

    WHILE i < 99 DO
        SET partition_sql = CONCAT(partition_sql,
            'PARTITION p', i, ' VALUES LESS THAN (', (i + 1) * step, '),');
        SET i = i + 1;
    END WHILE;

    SET partition_sql = CONCAT(partition_sql,
        'PARTITION p99 VALUES LESS THAN MAXVALUE)');

    SET @sql = partition_sql;
    PREPARE stmt FROM @sql;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
END //
DELIMITER ;

CALL create_range_100_partitions();
DROP PROCEDURE create_range_100_partitions;

INSERT INTO books_range_100 (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- =====================================================
-- RANGE 1000 パーティション（ID範囲）
-- =====================================================
DROP TABLE IF EXISTS books_range_1000;

DROP PROCEDURE IF EXISTS create_range_1000_partitions;

DELIMITER //
CREATE PROCEDURE create_range_1000_partitions()
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE partition_sql TEXT DEFAULT '';
    DECLARE step INT DEFAULT 1000; -- 1M / 1000 = 1000

    SET partition_sql = 'CREATE TABLE books_range_1000 (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        author_id BIGINT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        INDEX idx_author_id (author_id),
        INDEX idx_created_at (created_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
    PARTITION BY RANGE (id) (';

    WHILE i < 999 DO
        SET partition_sql = CONCAT(partition_sql,
            'PARTITION p', i, ' VALUES LESS THAN (', (i + 1) * step, '),');
        SET i = i + 1;
    END WHILE;

    SET partition_sql = CONCAT(partition_sql,
        'PARTITION p999 VALUES LESS THAN MAXVALUE)');

    SET @sql = partition_sql;
    PREPARE stmt FROM @sql;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
END //
DELIMITER ;

CALL create_range_1000_partitions();
DROP PROCEDURE create_range_1000_partitions;

INSERT INTO books_range_1000 (id, title, author_id, created_at)
SELECT id, title, author_id, created_at FROM books;

-- =====================================================
-- RANGE 8000 パーティション（ID範囲）
-- =====================================================
DROP TABLE IF EXISTS books_range_8000;

DROP PROCEDURE IF EXISTS create_range_8000_partitions;

DELIMITER //
CREATE PROCEDURE create_range_8000_partitions()
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE partition_sql LONGTEXT DEFAULT '';
    DECLARE step INT DEFAULT 125; -- 1M / 8000 = 125

    SET partition_sql = 'CREATE TABLE books_range_8000 (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        author_id BIGINT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        INDEX idx_author_id (author_id),
        INDEX idx_created_at (created_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
    PARTITION BY RANGE (id) (';

    WHILE i < 7999 DO
        SET partition_sql = CONCAT(partition_sql,
            'PARTITION p', i, ' VALUES LESS THAN (', (i + 1) * step, '),');
        SET i = i + 1;
    END WHILE;

    SET partition_sql = CONCAT(partition_sql,
        'PARTITION p7999 VALUES LESS THAN MAXVALUE)');

    SET @sql = partition_sql;
    PREPARE stmt FROM @sql;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
END //
DELIMITER ;

CALL create_range_8000_partitions();
DROP PROCEDURE create_range_8000_partitions;

INSERT INTO books_range_8000 (id, title, author_id, created_at)
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
AND TABLE_NAME LIKE 'books_range_%'
ORDER BY TABLE_NAME;

-- =====================================================
-- ベンチマーククエリ
-- =====================================================

-- 1. PK検索（単一行）
SELECT '=== PK検索 ===' AS benchmark;

SELECT 'books_range_8' AS tbl, id, title FROM books_range_8 WHERE id = 500000;
SELECT 'books_range_100' AS tbl, id, title FROM books_range_100 WHERE id = 500000;
SELECT 'books_range_1000' AS tbl, id, title FROM books_range_1000 WHERE id = 500000;
SELECT 'books_range_8000' AS tbl, id, title FROM books_range_8000 WHERE id = 500000;

-- 2. ID範囲検索
SELECT '=== ID範囲検索（パーティション内） ===' AS benchmark;

SELECT 'books_range_8' AS tbl, COUNT(*) FROM books_range_8 WHERE id BETWEEN 100000 AND 110000;
SELECT 'books_range_100' AS tbl, COUNT(*) FROM books_range_100 WHERE id BETWEEN 100000 AND 110000;
SELECT 'books_range_1000' AS tbl, COUNT(*) FROM books_range_1000 WHERE id BETWEEN 100000 AND 110000;
SELECT 'books_range_8000' AS tbl, COUNT(*) FROM books_range_8000 WHERE id BETWEEN 100000 AND 110000;

-- 3. 日付範囲検索（全パーティションスキャン）
SELECT '=== 日付範囲検索 ===' AS benchmark;

SELECT 'books_range_8' AS tbl, COUNT(*) FROM books_range_8
WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';

SELECT 'books_range_100' AS tbl, COUNT(*) FROM books_range_100
WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';

SELECT 'books_range_1000' AS tbl, COUNT(*) FROM books_range_1000
WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';

SELECT 'books_range_8000' AS tbl, COUNT(*) FROM books_range_8000
WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';

-- 4. Full COUNT
SELECT '=== Full COUNT ===' AS benchmark;

SELECT 'books_range_8' AS tbl, COUNT(*) FROM books_range_8;
SELECT 'books_range_100' AS tbl, COUNT(*) FROM books_range_100;
SELECT 'books_range_1000' AS tbl, COUNT(*) FROM books_range_1000;
SELECT 'books_range_8000' AS tbl, COUNT(*) FROM books_range_8000;

-- 5. EXPLAIN 確認
SELECT '=== EXPLAIN ===' AS benchmark;

EXPLAIN SELECT * FROM books_range_8 WHERE id = 500000;
EXPLAIN SELECT * FROM books_range_8000 WHERE id = 500000;

EXPLAIN SELECT COUNT(*) FROM books_range_8 WHERE id BETWEEN 100000 AND 110000;
EXPLAIN SELECT COUNT(*) FROM books_range_8000 WHERE id BETWEEN 100000 AND 110000;

EXPLAIN SELECT COUNT(*) FROM books_range_8 WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';
EXPLAIN SELECT COUNT(*) FROM books_range_8000 WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';
