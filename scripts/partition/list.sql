-- LIST Partition
-- 特定の値リストでパーティション分割
-- ユースケース: ステータス、カテゴリ、地域などで分割

-- books にステータスカラムを追加したバージョン
DROP TABLE IF EXISTS books_list;

CREATE TABLE books_list (
    id BIGINT AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    author_id BIGINT NOT NULL,
    status TINYINT NOT NULL DEFAULT 1 COMMENT '0:draft, 1:published, 2:archived, 3:deleted',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, status),
    INDEX idx_author_id (author_id),
    INDEX idx_created_at (created_at),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY LIST(status) (
    PARTITION p_draft VALUES IN (0),
    PARTITION p_published VALUES IN (1),
    PARTITION p_archived VALUES IN (2),
    PARTITION p_deleted VALUES IN (3)
);

-- 既存データをランダムなステータスでコピー
INSERT INTO books_list (id, title, author_id, status, created_at)
SELECT
    id,
    title,
    author_id,
    -- 80% published, 10% archived, 5% draft, 5% deleted
    CASE
        WHEN RAND() < 0.80 THEN 1
        WHEN RAND() < 0.90 THEN 2
        WHEN RAND() < 0.95 THEN 0
        ELSE 3
    END as status,
    created_at
FROM books;

-- author_tags を author_id の範囲でリスト分割（地域的な分割を模倣）
-- 例: author_id を 1000 で割った余りでグループ化
DROP TABLE IF EXISTS author_tags_list;

CREATE TABLE author_tags_list (
    author_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    region TINYINT NOT NULL DEFAULT 0 COMMENT '0-9 region codes',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (author_id, tag_id, region),
    INDEX idx_tag_id (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY LIST(region) (
    PARTITION p_region_0 VALUES IN (0),
    PARTITION p_region_1 VALUES IN (1),
    PARTITION p_region_2 VALUES IN (2),
    PARTITION p_region_3 VALUES IN (3),
    PARTITION p_region_4 VALUES IN (4),
    PARTITION p_region_5 VALUES IN (5),
    PARTITION p_region_6 VALUES IN (6),
    PARTITION p_region_7 VALUES IN (7),
    PARTITION p_region_8 VALUES IN (8),
    PARTITION p_region_9 VALUES IN (9)
);

INSERT INTO author_tags_list (author_id, tag_id, region, created_at)
SELECT
    author_id,
    tag_id,
    (author_id % 10) as region,
    created_at
FROM author_tags;

-- パーティション情報確認
SELECT
    TABLE_NAME,
    PARTITION_NAME,
    PARTITION_METHOD,
    PARTITION_EXPRESSION,
    TABLE_ROWS
FROM INFORMATION_SCHEMA.PARTITIONS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME IN ('books_list', 'author_tags_list')
ORDER BY TABLE_NAME, PARTITION_ORDINAL_POSITION;
