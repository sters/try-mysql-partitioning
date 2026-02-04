# try-mysql-partitioning

MySQL パーティション機能をローカル環境で実験するプロジェクト。

## 起動

```bash
# MySQL と App を起動
docker compose up -d

# ログ確認
docker compose logs -f
```

## データ投入

```bash
# 100万件のデータを投入（初回）
go run ./cmd/seed -truncate

# オプション
go run ./cmd/seed \
  -authors 10000 \
  -books 1000000 \
  -tags 1000 \
  -book-tags 5000000 \
  -author-tags 50000 \
  -truncate
```

## ベンチマーク

```bash
# 基本実行
go run ./cmd/benchmark

# EXPLAIN付き
go run ./cmd/benchmark -explain -iterations 20
```

## パーティション比較

パーティションの有無でパフォーマンスを比較するスクリプト。

```bash
# HASH パーティションで比較（パーティションテーブルを自動作成）
go run ./cmd/compare -type hash -setup

# RANGE (年) パーティションで比較
go run ./cmd/compare -type range_year -setup

# RANGE (ID) パーティションで比較
go run ./cmd/compare -type range_id -setup

# LIST パーティションで比較
go run ./cmd/compare -type list -setup

# KEY パーティションで比較
go run ./cmd/compare -type key -setup

# 全パーティション種類で比較
go run ./cmd/compare -type all -setup -iterations 20

# パーティションテーブルが既にある場合は -setup 不要
go run ./cmd/compare -type hash -iterations 20
```

## パーティション適用

```bash
# MySQL に接続
docker compose exec mysql mysql -uapp -papp bookdb

# パーティション適用（例: RANGE by year）
source /path/to/scripts/partition/range_by_year.sql

# クリーンアップ
source /path/to/scripts/partition/cleanup.sql
```

### パーティション種類

| ファイル | 方式 | 説明 |
|---------|-----|------|
| `range_by_year.sql` | RANGE | created_at の年で分割 |
| `range_by_id.sql` | RANGE | ID 範囲で分割（10万件ごと） |
| `hash.sql` | HASH | ID のハッシュ値で均等分割 |
| `list.sql` | LIST | ステータス値で分割 |
| `key.sql` | KEY | 複合キーで分割 |

## API

```bash
# 著者
curl http://localhost:8080/authors
curl -X POST http://localhost:8080/authors -d '{"name":"Author Name"}'

# 本
curl http://localhost:8080/books
curl -X POST http://localhost:8080/books -d '{"title":"Book Title","author_id":1}'

# タグ
curl http://localhost:8080/tags
curl -X POST http://localhost:8080/tags -d '{"name":"Tag Name"}'

# 本にタグ付け
curl -X POST http://localhost:8080/books/1/tags -d '{"tag_id":1}'
```

## 停止

```bash
docker compose down

# データも削除
docker compose down -v
```
