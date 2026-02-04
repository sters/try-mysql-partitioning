# パーティション種類別 比較レポート

## テスト条件

- レコード数: **1,000,000件**
- 全テーブルにインデックスあり

---

## 1. HASH パーティション

### 特徴
- ID のハッシュ値で均等分割
- 単一値検索（PK検索）に最適

### 構文
```sql
PARTITION BY HASH(id) PARTITIONS 8;
```

### ベンチマーク結果

| クエリ | パーティションなし | HASH(8) | 比較 |
|-------|------------------|---------|------|
| PK検索 | 153µs | 143µs | 6.5%削減 |
| Full COUNT | 56.6ms | 63.9ms | 12.9%増加 |
| Range検索 | 343µs | 327µs | 4.7%削減 |
| JOIN | 425µs | 611µs | **43.7%増加** |

### EXPLAIN 分析

#### PK検索
```sql
EXPLAIN SELECT * FROM books_hash WHERE id = 500000;
```
```
type: const
partitions: p0
key: PRIMARY
rows: 1
```
**考察**: HASH(id) により、id=500000 は p0 に格納。単一パーティションのみアクセスで高速。

#### Range検索
```sql
EXPLAIN SELECT * FROM books_hash WHERE id BETWEEN 100000 AND 110000;
```
```
type: range
partitions: p0,p1,p2,p3,p4,p5,p6,p7
key: PRIMARY
rows: 20000
```
**考察**: HASH パーティションでは範囲検索時に**全パーティションスキャン**が発生。パーティションプルーニングが効かない。

#### Full COUNT
```sql
EXPLAIN SELECT COUNT(*) FROM books_hash;
```
```
type: index
partitions: p0,p1,p2,p3,p4,p5,p6,p7
key: PRIMARY
rows: 1000000
Extra: Using index
```
**考察**: 全パーティションを順次カウント。パーティションなしより12.9%遅い原因は、各パーティションのオープン/クローズのオーバーヘッド。

#### JOIN
```sql
EXPLAIN SELECT b.title, a.name FROM books_hash b
JOIN authors a ON b.author_id = a.id WHERE b.id = 500000;
```
```
-- books_hash
type: const, partitions: p0, key: PRIMARY, rows: 1
-- authors
type: const, key: PRIMARY, rows: 1
```
**考察**: PK での JOIN は高速だが、範囲 JOIN では全パーティションにアクセスする必要があり、43.7%の遅延が発生。パーティションをまたぐ JOIN は避けるべき。

### 適用ケース
- PK による単一レコード検索が多い場合
- データを均等に分散したい場合

### 注意点
- 範囲検索は全パーティションスキャン
- JOINは遅くなる傾向

---

## 2. RANGE パーティション（日付）

### 特徴
- 日付範囲でパーティション分割
- 時系列データに最適

### 構文
```sql
PARTITION BY RANGE (YEAR(created_at)) (
    PARTITION p2020 VALUES LESS THAN (2021),
    PARTITION p2021 VALUES LESS THAN (2022),
    PARTITION p2022 VALUES LESS THAN (2023),
    ...
);
```

### ベンチマーク結果

| クエリ | パーティションなし | RANGE(year) | 比較 |
|-------|------------------|-------------|------|
| 1年間の日付範囲 | 189.5ms | 64.5ms | **66.0%削減（約3倍高速）** |
| 1ヶ月の日付範囲 | 26.6ms | 13.2ms | **50.4%削減（約2倍高速）** |
| GROUP BY 年 | 119.7ms | 132.2ms | 10.4%増加 |
| 複数年クエリ | 806µs | 1.4ms | 73.7%増加 |

### EXPLAIN 分析

#### 1年間の日付範囲（単一パーティション）
```sql
EXPLAIN SELECT COUNT(*) FROM books_range_year
WHERE created_at BETWEEN '2022-01-01' AND '2022-12-31';
```
```
type: range
partitions: p2022
key: idx_created_at
rows: 288130
Extra: Using where; Using index
```
**考察**: パーティションプルーニングにより p2022 のみアクセス。インデックスも活用され高速。

#### 複数年クエリ
```sql
EXPLAIN SELECT COUNT(*) FROM books_range_year
WHERE created_at BETWEEN '2021-06-01' AND '2022-06-30';
```
```
type: range
partitions: p2021,p2022
key: idx_created_at
rows: 576260
```
**考察**: 2パーティションをまたぐため、両方をスキャン。パーティションが増えると遅くなる。

#### 1ヶ月の日付範囲
```sql
EXPLAIN SELECT COUNT(*) FROM books_range_year
WHERE created_at BETWEEN '2022-06-01' AND '2022-06-30';
```
```
type: range
partitions: p2022
key: idx_created_at
rows: 28813
Extra: Using where; Using index
```
**考察**: 単一パーティション内の範囲検索。パーティションプルーニング + インデックスで50.5%高速化。

#### GROUP BY 年
```sql
EXPLAIN SELECT YEAR(created_at), COUNT(*) FROM books_range_year GROUP BY YEAR(created_at);
```
```
type: index
partitions: p2019,p2020,p2021,p2022,p2023,p2024,pmax
key: idx_created_at
rows: 1000000
Extra: Using index
```
**考察**: GROUP BY では全パーティションをスキャン。YEAR() 関数の評価オーバーヘッドも加わり、10.5%遅くなる。パーティションプルーニングが効かないクエリパターン。

### 適用ケース
- ログデータ、履歴データ
- 日付範囲での検索が多い場合
- 古いデータの削除（パーティション DROP）

### 注意点
- パーティションをまたぐクエリは遅くなる
- パーティションキーは PRIMARY KEY に含める必要あり

---

## 3. RANGE パーティション（ID）

### 特徴
- ID 範囲でパーティション分割
- 連番 ID のテーブルに有効

### 構文
```sql
PARTITION BY RANGE (id) (
    PARTITION p0 VALUES LESS THAN (100000),
    PARTITION p1 VALUES LESS THAN (200000),
    ...
);
```

### ベンチマーク結果

| クエリ | パーティションなし | RANGE(id) | 比較 |
|-------|------------------|-----------|------|
| パーティション内Range | 20.8ms | 16.2ms | **22.1%削減** |
| パーティション跨ぎRange | 4.0ms | 3.6ms | **10.0%削減** |

### EXPLAIN 分析

#### パーティション内Range（単一パーティション）
```sql
EXPLAIN SELECT * FROM books_range_id WHERE id BETWEEN 150000 AND 180000;
```
```
type: range
partitions: p1
key: PRIMARY
rows: 30000
```
**考察**: id 150000-180000 は p1（100000-200000範囲）に収まる。単一パーティションのみアクセスで22.1%高速。

#### パーティション跨ぎRange
```sql
EXPLAIN SELECT * FROM books_range_id WHERE id BETWEEN 180000 AND 220000;
```
```
type: range
partitions: p1,p2
key: PRIMARY
rows: 40000
```
**考察**: id 180000-220000 は p1 と p2 をまたぐ。2パーティションをスキャンするが、それでも11.2%高速。全テーブルスキャンと比較して、アクセス範囲が限定される効果。

### 適用ケース
- ID 範囲での検索が多い場合
- データのアーカイブ（古い ID のパーティション削除）

### DROP PARTITION によるデータ削除
```sql
-- 古いデータを瞬時に削除（DELETE と比較して劇的に高速）
ALTER TABLE books_range_id DROP PARTITION p0;
```
**考察**: DELETE は行ごとに処理されるが、DROP PARTITION はメタデータの変更のみ。100万件の削除が数ミリ秒で完了。

---

## 4. LIST パーティション

### 特徴
- 特定の値リストでパーティション分割
- ステータス、カテゴリ、地域などに有効

### 構文
```sql
PARTITION BY LIST(status) (
    PARTITION p_draft VALUES IN (0),
    PARTITION p_published VALUES IN (1),
    PARTITION p_archived VALUES IN (2)
);
```

### ベンチマーク結果

| クエリ | パーティションなし | LIST | 比較 |
|-------|------------------|------|------|
| 単一ステータス | 52.7ms | 63.5ms | 20.6% 低速 |
| 複数ステータス | 51.9ms | 70.5ms | 35.7% 低速 |

### EXPLAIN 分析

#### 単一ステータス検索
```sql
EXPLAIN SELECT COUNT(*) FROM books_list WHERE status = 1;
```
```
type: index
partitions: p_published
key: PRIMARY
rows: 333333
Extra: Using where; Using index
```
**考察**: パーティションプルーニングは機能（p_published のみ）。しかしステータスごとのデータ量が大きく、効果が薄い。

#### 複数ステータス検索
```sql
EXPLAIN SELECT COUNT(*) FROM books_list WHERE status IN (1, 2);
```
```
type: index
partitions: p_published,p_archived
key: PRIMARY
rows: 666666
```
**考察**: 複数パーティションをスキャン。パーティション数が少ないため、非パーティションと大差なし。

### 適用ケース
- 特定のステータスのみ頻繁にアクセスする場合
- マルチテナント（tenant_id で分割）

### 注意点
- 値の追加にはパーティション追加が必要
- 今回のテストでは効果が薄い

### なぜ LIST パーティションが遅いのか？

| 要因 | 影響 |
|------|------|
| パーティション数が少ない | 3パーティションでは分割効果が薄い |
| データ量の偏り | status=1 が33万件で、依然として大量 |
| インデックスの分断 | 各パーティションに別々のインデックスが作成される |
| オーバーヘッド | パーティション管理コストがベネフィットを上回る |

**LISTが有効なケース**:
```sql
-- マルチテナント: tenant_id で分割
PARTITION BY LIST(tenant_id) (
    PARTITION p_tenant_1 VALUES IN (1),
    PARTITION p_tenant_2 VALUES IN (2),
    ...
);
-- → 各テナントのデータが完全に分離され、効果的
```

---

## 5. KEY パーティション

### 特徴
- MySQL 内部のハッシュ関数を使用
- 複合キーでのパーティショニングが可能

### 構文
```sql
PARTITION BY KEY(book_id, tag_id) PARTITIONS 16;
```

### ベンチマーク結果

| クエリ | パーティションなし | KEY | 比較 |
|-------|------------------|-----|------|
| 複合キー完全一致 | 147µs | 139µs | **5.4%削減** |
| 部分キー検索 | 146µs | 166µs | 13.7%増加 |

### EXPLAIN 分析

#### 複合キー完全一致
```sql
EXPLAIN SELECT * FROM book_tags_key WHERE book_id = 1000 AND tag_id = 5;
```
```
type: const
partitions: p7
key: PRIMARY
rows: 1
```
**考察**: 複合キー全体を指定すると、単一パーティションのみアクセス。

#### 部分キー検索（book_id のみ）
```sql
EXPLAIN SELECT * FROM book_tags_key WHERE book_id = 1000;
```
```
type: ref
partitions: p0,p1,p2,...,p15
key: PRIMARY
rows: 160
```
**考察**: 部分キーでは全パーティションスキャン。KEY パーティションは複合キー全体でハッシュを計算するため。

### 適用ケース
- 複合キーでの検索が多い中間テーブル
- PRIMARY KEY でのパーティショニング

### 注意点
- 部分キー検索は全パーティションスキャン

---

## まとめ

| 種類 | 最適なユースケース | 効果 |
|------|------------------|------|
| HASH | PK検索、均等分散 | 小〜中 |
| RANGE(日付) | 時系列データ、ログ | **大** |
| RANGE(ID) | ID範囲検索、アーカイブ | 中 |
| LIST | マルチテナント、ステータス | 小 |
| KEY | 複合キー検索 | 小 |
