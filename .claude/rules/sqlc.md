# sqlc / DB 管理ルール

## 生成コード保護

`db/generated/` は **sqlc が自動生成するため編集禁止**。

```
db/
├── schema.sql        # 手動編集禁止（CI 自動同期）
├── queries/          # ← ここだけ手動編集する
│   ├── user.sql
│   ├── favorite.sql
│   └── verification_token.sql
└── generated/        # ← 編集禁止（make sqlc-gen で再生成）
    ├── db.go
    ├── models.go
    └── user.sql.go
```

変更手順:
1. `db/queries/*.sql` にクエリを追加・編集
2. `make sqlc-gen` で再生成
3. `go build ./...` でコンパイルエラーがないか確認

## スキーマ管理

`db/schema.sql` は **手動編集禁止**。

- `Fishing-database` リポジトリで管理
- GitHub Actions (`sync-schema.yml`) が自動同期 → PR 自動作成
- スキーマ変更は `Fishing-database` リポジトリ側で行う

## sqlc クエリ記述ルール

```sql
-- WRONG: コメントなし
SELECT * FROM users WHERE email = $1;

-- CORRECT: sqlc コメント必須
-- name: GetUserByEmail :one
SELECT id, email, password_hash, name, avatar_url, is_sso, email_verified_at, created_at, updated_at
FROM users
WHERE email = $1;
```

| コメント形式 | 戻り値 | 使いどころ |
|------------|--------|-----------|
| `:one` | 1 行（なければ error） | 主キー・ユニークキー検索 |
| `:many` | スライス | 一覧取得 |
| `:exec` | なし | INSERT/UPDATE/DELETE |
| `:execresult` | sql.Result | 影響行数が必要な場合 |

## N+1 防止

```sql
-- WRONG: N+1 になるクエリ（ループ内で個別取得）
-- name: GetLocationByID :one
SELECT * FROM locations WHERE id = $1;

-- CORRECT: JOIN で一括取得
-- name: GetFavoriteLocations :many
SELECT l.id, l.name, l.latitude, l.longitude, l.region
FROM favorites f
JOIN locations l ON f.location_id = l.id
WHERE f.user_id = $1
ORDER BY f.created_at DESC;
```

## ローカル開発フロー

```bash
# 1. クエリ追加・編集
vim db/queries/user.sql

# 2. sqlc で Go コード再生成
make sqlc-gen

# 3. コンパイル確認
go build ./...

# 4. テスト実行
make test
```

## 関連ルール

- [security.md](./security.md) — SQL インジェクション防止（文字列連結禁止）
- [clean-architecture.md](./clean-architecture.md) — `db/generated/` は `infrastructure/db/` からのみ import
