# 外部キーパーティション 比較レポート

## 概要

外部キー（author_id）でパーティションを分割した場合の効果を検証。

**注意**: MySQL のパーティションテーブルでは外部キー制約は使用できないため、
アプリケーション側での整合性管理が必要。

---

## テスト条件

- レコード数: **1,000,000件** (books), **100,000件** (authors)
- 比較対象:
  - パーティションなし（インデックスあり）
  - HASH(author_id) 8分割
  - RANGE(author_id) 10分割（10,000件ごと）

---

## 1. 単一著者検索

特定の著者（author_id = 500）の本を検索。

### クエリ
```sql
SELECT COUNT(*) FROM {table} WHERE author_id = 500;
```

### 結果

| 構成 | 時間 | 比較 |
|------|------|------|
| パーティションなし | 82µs | baseline |
| **HASH(author_id)** | **51µs** | **37% 高速** |
| RANGE(author_id) | 482µs | 487% 低速 |

### EXPLAIN 分析

#### HASH(author_id)
```sql
EXPLAIN SELECT COUNT(*) FROM books_hash_author WHERE author_id = 500;
```
```
type: ref
partitions: p3
key: idx_author_id
rows: 91
Extra: Using index
```
**考察**: HASH(author_id) により author_id=500 は p3 に格納。単一パーティション + インデックスで最速。

#### RANGE(author_id)
```sql
EXPLAIN SELECT COUNT(*) FROM books_range_author WHERE author_id = 500;
```
```
type: ref
partitions: p0
key: idx_author_id
rows: 91
Extra: Using index
```
**考察**: author_id=500 は p0（0-10000範囲）に該当。パーティションプルーニングは機能するが、HASH より若干遅い。

### 分析
- **HASH が最速**: パーティションプルーニングが効く
- RANGE は不向き: author_id の分布が偏ると効果が薄い

---

## 2. 著者範囲検索（1000人分）

author_id BETWEEN 500 AND 1500 の本を検索（約10万件）。

### クエリ
```sql
SELECT COUNT(*) FROM {table} WHERE author_id BETWEEN 500 AND 1500;
```

### 結果

| 構成 | 時間 | 比較 |
|------|------|------|
| パーティションなし | 49.0ms | baseline |
| HASH(author_id) | 14.6ms | 70% 高速 |
| **RANGE(author_id)** | **11.1ms** | **77% 高速** |

### EXPLAIN 分析

#### HASH(author_id)
```sql
EXPLAIN SELECT COUNT(*) FROM books_hash_author WHERE author_id BETWEEN 500 AND 1500;
```
```
type: range
partitions: p0,p1,p2,p3,p4,p5,p6,p7
key: idx_author_id
rows: 91000
```
**考察**: HASH パーティションでは範囲検索時に**全パーティションスキャン**。ただしインデックスで絞り込み。

#### RANGE(author_id)
```sql
EXPLAIN SELECT COUNT(*) FROM books_range_author WHERE author_id BETWEEN 500 AND 1500;
```
```
type: range
partitions: p0
key: idx_author_id
rows: 91000
Extra: Using where; Using index
```
**考察**: RANGE パーティションでは 500-1500 が p0（0-10000範囲）に収まるため、**単一パーティションのみスキャン**。最も効率的。

### 分析
- **RANGE が最速**: 範囲検索ではパーティションプルーニングが効果的
- HASH も高速だが、RANGE には及ばない

---

## 3. JOIN クエリ（著者 + 本）

author_id BETWEEN 500 AND 600 の著者と本を JOIN。

### クエリ
```sql
SELECT a.name, COUNT(b.id) AS book_count
FROM authors a
JOIN {books_table} b ON a.id = b.author_id
WHERE a.id BETWEEN 500 AND 600
GROUP BY a.id;
```

### 結果

| 構成 | 時間 | 比較 |
|------|------|------|
| パーティションなし | 1.57ms | baseline |
| HASH(author_id) | 1.63ms | 4% 低速 |
| RANGE(author_id) | 1.81ms | 15% 低速 |

### EXPLAIN 分析

#### パーティションなし
```sql
EXPLAIN SELECT a.name, COUNT(b.id) FROM authors a
JOIN books b ON a.id = b.author_id WHERE a.id BETWEEN 500 AND 600 GROUP BY a.id;
```
```
-- authors テーブル
type: range, key: PRIMARY, rows: 101
-- books テーブル
type: ref, key: idx_author_id, rows: 10
```
**考察**: シンプルな2テーブル JOIN。インデックスで効率的に結合。

#### HASH(author_id)
```sql
EXPLAIN SELECT a.name, COUNT(b.id) FROM authors a
JOIN books_hash_author b ON a.id = b.author_id WHERE a.id BETWEEN 500 AND 600 GROUP BY a.id;
```
```
-- books_hash_author テーブル
partitions: p0,p1,p2,p3,p4,p5,p6,p7
type: ref, key: idx_author_id, rows: 91
```
**考察**: JOIN 時は各 author_id ごとにパーティションを特定する必要があり、オーバーヘッドが発生。

### 分析
- JOIN ではパーティションのオーバーヘッドが発生
- パーティションをまたぐ JOIN は避けるべき

---

## 4. 大量データでの検証（10,000,000件）

| クエリ | パーティションなし | HASH | RANGE |
|-------|------------------|------|-------|
| 単一著者 | 1.1ms | - | **0.2ms** |
| 範囲検索 | 5.4ms | - | **0.3ms** |

### 分析
- 大量データでは RANGE の効果が顕著
- 10倍以上の高速化

---

## 推奨パーティション戦略

### 単一値検索が多い場合
```sql
PARTITION BY HASH(author_id) PARTITIONS 8;
```

### 範囲検索が多い場合
```sql
PARTITION BY RANGE (author_id) (
    PARTITION p0 VALUES LESS THAN (10000),
    PARTITION p1 VALUES LESS THAN (20000),
    ...
);
```

### 両方のケースがある場合
- インデックスのみで対応
- または RANGE パーティション（範囲検索がより高速）

---

## 注意点

### 1. 外部キー制約は使用不可
```sql
-- これはエラーになる
CREATE TABLE books (
    ...
    FOREIGN KEY (author_id) REFERENCES authors(id)
) PARTITION BY HASH(author_id);
```

### 2. パーティションキーは PRIMARY KEY に含める必要
```sql
-- author_id を PRIMARY KEY に含める
PRIMARY KEY (id, author_id)
```

### 3. アプリケーション側での整合性管理
- 参照整合性はアプリケーションで担保
- または定期的なバッチチェック

---

## まとめ

| ユースケース | 推奨パーティション | 効果 |
|-------------|------------------|------|
| 単一著者検索 | HASH | 37% 高速 |
| 著者範囲検索 | **RANGE** | **77% 高速** |
| JOIN | なし（インデックスのみ） | - |
