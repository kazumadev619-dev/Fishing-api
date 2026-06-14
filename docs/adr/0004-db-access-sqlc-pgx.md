# 0004. DB アクセスに sqlc + pgx/v5 を採用する

- Status: Accepted
- Date: 2026-05-08
- Deciders: kazumadev619-dev

## Context and Problem Statement

Go から PostgreSQL（Neon、[ADR 0001](0001-database-provider-neon.md)）にアクセスする手段を決める。型安全性・SQL インジェクション耐性・既存 Prisma スキーマとの 1:1 対応のしやすさ・Neon への接続容易性が要件なのだ。

## Considered Options

- ORM（GORM など）
- 生の `database/sql` + 手書きスキャン
- sqlc（SQL からタイプセーフな Go コードを生成）+ pgx/v5（ドライバ）

## Decision Outcome

**sqlc + pgx/v5** を採用する。`db/queries/*.sql` に SQL を手書きし、`make sqlc-gen` でタイプセーフな Go コード（`db/generated/`）を生成する。ドライバは pgx/v5 を使い、Neon への接続文字列には `sslmode=require` を必須とする。

ルール:
- SQL は sqlc 管理クエリのみ使用し、文字列連結・`fmt.Sprintf` による SQL 構築は禁止（SQL インジェクション防止）
- `db/queries/` のみ手動編集し、`db/generated/` は編集禁止（再生成で上書き）
- `db/schema.sql` は `Fishing-database` リポジトリから CI 自動同期（手動編集禁止）
- N+1 を避け、JOIN で一括取得する

## Positive Consequences

- SQL がコンパイル時に型チェックされ、タイポやカラム不一致を早期検出できる
- パラメータ化クエリが生成されるため SQL インジェクションに強い
- 生 SQL を書くので JOIN による N+1 回避など最適化を直接コントロールできる
- pgx/v5 は Neon との接続が容易（`sslmode=require` を付けるのみ）

## Negative Consequences

- クエリ変更のたびに `make sqlc-gen` の再生成ステップが必要
- ORM のような自動マイグレーション・動的クエリ構築はできない（動的 WHERE も Named Parameters で対応）
- 生成コードがリポジトリに含まれる（編集禁止の運用ルールが必要）

## Links

- 設計仕様: [`docs/superpowers/specs/2026-04-07-go-backend-design.md`](../superpowers/specs/2026-04-07-go-backend-design.md)（「インフラストラクチャ層 / sqlc 設定」節）
- sqlc / DB 管理ルール: [`.claude/rules/sqlc.md`](../../.claude/rules/sqlc.md)
- セキュリティルール: [`.claude/rules/security.md`](../../.claude/rules/security.md)（SQL インジェクション防止）
- 設定: `sqlc.yaml`、クエリ: `db/queries/`、生成コード: `db/generated/`
- 関連: [ADR 0001 — DB プロバイダ Neon](0001-database-provider-neon.md)
