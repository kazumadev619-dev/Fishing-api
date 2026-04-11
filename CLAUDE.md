# Fishing-api

Go バックエンド for [FishingConditionsApp](https://github.com/kazumadev619-dev/FishingConditionsApp)

## 技術スタック

| 項目 | 内容 |
|------|------|
| 言語 | Go 1.26 |
| フレームワーク | Gin |
| DB | **Neon**（マネージドサーバーレス PostgreSQL） |
| DB アクセス | sqlc + pgx/v5（接続文字列に `sslmode=require` 必須） |
| キャッシュ | Redis（k3s Pod + ClusterIP Service） |
| 認証 | JWT（15分） + Refresh Token（7日）+ Google OAuth |
| デプロイ | k3s on Raspberry Pi 5 → Cloudflare Tunnel |
| CI/CD | GitHub Actions → GHCR（linux/arm64） → kubectl rolling deploy |

## アーキテクチャ

クリーンアーキテクチャ。依存は内側のみ：`interface → usecase → domain ← infrastructure`

- `cmd/server/` エントリポイント・DI組み立て
- `internal/domain/` エンティティ・リポジトリIF（最内層）
- `internal/usecase/` ビジネスロジック
- `internal/infrastructure/` DB・Redis・外部API実装
- `internal/interface/` Gin ハンドラー・ミドルウェア・ルーター
- `db/queries/` sqlc 用 SQL クエリ（手動管理）
- `db/generated/` sqlc 自動生成コード（編集不可）

## 主要設計決定

- **DB は Neon** を採用（Supabase Free は7日不使用で停止するため却下）
- **Redis は k3s Pod** で運用（キャッシュはステートレスなので Pod 再起動でデータが消えても DB から再取得するだけ）
- **フロント→バックエンド通信**：ブラウザは Cloudflare 経由、Next.js SSR は ClusterIP 直通（`fishing-api-service:8080`）
- **ビッグバン移行**：Next.js API Routes を全廃し Go で再実装。k8s Ingress で `/api/*` を切り替え
- `db/schema.sql` は `Fishing-database` リポジトリから GitHub Actions で自動同期（手動編集しない）

## テスト方針

- usecase: `testify/mock` でモックリポジトリ
- infrastructure/db: `testcontainers-go` で実 PostgreSQL（Neon への依存なし）
- interface/handler: `httptest + Gin` で統合テスト

## 詳細ドキュメント

- 設計仕様: `docs/superpowers/specs/2026-04-07-go-backend-design.md`
- 実装計画: `docs/superpowers/plans/`
