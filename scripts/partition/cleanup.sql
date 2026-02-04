-- Cleanup: パーティションテーブルを削除

DROP TABLE IF EXISTS books_range_year;
DROP TABLE IF EXISTS book_tags_range_year;

DROP TABLE IF EXISTS books_range_id;
DROP TABLE IF EXISTS book_tags_range_id;

DROP TABLE IF EXISTS books_hash;
DROP TABLE IF EXISTS authors_hash;
DROP TABLE IF EXISTS book_tags_hash;

DROP TABLE IF EXISTS books_list;
DROP TABLE IF EXISTS author_tags_list;

DROP TABLE IF EXISTS book_tags_key;
DROP TABLE IF EXISTS author_tags_key;
DROP TABLE IF EXISTS books_key;

SELECT 'Cleanup completed' as status;
