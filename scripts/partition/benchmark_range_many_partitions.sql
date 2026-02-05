-- RANGE パーティション数のベンチマーク（8 / 100 / 1000 / 8000）
-- 実行時間を測定するためのスクリプト

SET profiling = 1;
SET profiling_history_size = 100;

-- =====================================================
-- 1. PK検索（単一行）- 初回
-- =====================================================
SELECT '=== 1. PK検索（初回） ===' AS benchmark;

SELECT SQL_NO_CACHE * FROM books_range_8 WHERE id = 500000;
SELECT SQL_NO_CACHE * FROM books_range_100 WHERE id = 500000;
SELECT SQL_NO_CACHE * FROM books_range_1000 WHERE id = 500000;
SELECT SQL_NO_CACHE * FROM books_range_8000 WHERE id = 500000;

-- =====================================================
-- 2. PK検索（キャッシュ後）
-- =====================================================
SELECT '=== 2. PK検索（キャッシュ後） ===' AS benchmark;

SELECT * FROM books_range_8 WHERE id = 500000;
SELECT * FROM books_range_100 WHERE id = 500000;
SELECT * FROM books_range_1000 WHERE id = 500000;
SELECT * FROM books_range_8000 WHERE id = 500000;

-- =====================================================
-- 3. ID範囲検索（パーティション内）
-- =====================================================
SELECT '=== 3. ID範囲検索（パーティション内: 100000-110000） ===' AS benchmark;

SELECT SQL_NO_CACHE COUNT(*) FROM books_range_8 WHERE id BETWEEN 100000 AND 110000;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_100 WHERE id BETWEEN 100000 AND 110000;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_1000 WHERE id BETWEEN 100000 AND 110000;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_8000 WHERE id BETWEEN 100000 AND 110000;

-- =====================================================
-- 4. ID範囲検索（パーティション跨ぎ）
-- =====================================================
SELECT '=== 4. ID範囲検索（パーティション跨ぎ: 120000-130000） ===' AS benchmark;

-- 8パーティション: 125000境界をまたぐ
-- 100パーティション: 120000, 130000境界をまたぐ
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_8 WHERE id BETWEEN 120000 AND 130000;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_100 WHERE id BETWEEN 120000 AND 130000;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_1000 WHERE id BETWEEN 120000 AND 130000;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_8000 WHERE id BETWEEN 120000 AND 130000;

-- =====================================================
-- 5. 日付範囲検索（パーティションキー以外）
-- =====================================================
SELECT '=== 5. 日付範囲検索（1ヶ月） ===' AS benchmark;

SELECT SQL_NO_CACHE COUNT(*) FROM books_range_8
WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';

SELECT SQL_NO_CACHE COUNT(*) FROM books_range_100
WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';

SELECT SQL_NO_CACHE COUNT(*) FROM books_range_1000
WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';

SELECT SQL_NO_CACHE COUNT(*) FROM books_range_8000
WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';

-- =====================================================
-- 6. Full COUNT
-- =====================================================
SELECT '=== 6. Full COUNT ===' AS benchmark;

SELECT SQL_NO_CACHE COUNT(*) FROM books_range_8;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_100;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_1000;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_8000;

-- =====================================================
-- 7. author_id 検索（セカンダリインデックス）
-- =====================================================
SELECT '=== 7. author_id検索 ===' AS benchmark;

SELECT SQL_NO_CACHE COUNT(*) FROM books_range_8 WHERE author_id = 5000;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_100 WHERE author_id = 5000;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_1000 WHERE author_id = 5000;
SELECT SQL_NO_CACHE COUNT(*) FROM books_range_8000 WHERE author_id = 5000;

-- =====================================================
-- プロファイル結果
-- =====================================================
SELECT '=== プロファイル結果 ===' AS benchmark;
SHOW PROFILES;

-- =====================================================
-- EXPLAIN 分析
-- =====================================================
SELECT '=== EXPLAIN: PK検索 ===' AS benchmark;
EXPLAIN SELECT * FROM books_range_8 WHERE id = 500000;
EXPLAIN SELECT * FROM books_range_8000 WHERE id = 500000;

SELECT '=== EXPLAIN: ID範囲検索（パーティション内） ===' AS benchmark;
EXPLAIN SELECT COUNT(*) FROM books_range_8 WHERE id BETWEEN 100000 AND 110000;
EXPLAIN SELECT COUNT(*) FROM books_range_8000 WHERE id BETWEEN 100000 AND 110000;

SELECT '=== EXPLAIN: ID範囲検索（パーティション跨ぎ） ===' AS benchmark;
EXPLAIN SELECT COUNT(*) FROM books_range_8 WHERE id BETWEEN 120000 AND 130000;
EXPLAIN SELECT COUNT(*) FROM books_range_8000 WHERE id BETWEEN 120000 AND 130000;

SELECT '=== EXPLAIN: 日付範囲検索 ===' AS benchmark;
EXPLAIN SELECT COUNT(*) FROM books_range_8 WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';
EXPLAIN SELECT COUNT(*) FROM books_range_8000 WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';

SELECT '=== EXPLAIN: Full COUNT ===' AS benchmark;
EXPLAIN SELECT COUNT(*) FROM books_range_8;
EXPLAIN SELECT COUNT(*) FROM books_range_8000;

SELECT '=== EXPLAIN: author_id検索 ===' AS benchmark;
EXPLAIN SELECT COUNT(*) FROM books_range_8 WHERE author_id = 5000;
EXPLAIN SELECT COUNT(*) FROM books_range_8000 WHERE author_id = 5000;

-- =====================================================
-- テーブルサイズ
-- =====================================================
SELECT '=== テーブルサイズ ===' AS benchmark;
SELECT
    TABLE_NAME,
    TABLE_ROWS,
    ROUND(DATA_LENGTH/1024/1024, 2) as data_mb,
    ROUND(INDEX_LENGTH/1024/1024, 2) as index_mb,
    ROUND((DATA_LENGTH + INDEX_LENGTH)/1024/1024, 2) as total_mb
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = DATABASE()
AND TABLE_NAME LIKE 'books_range_%'
ORDER BY TABLE_NAME;
