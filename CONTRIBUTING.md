# Contributing Guide

## 開発フロー

このプロジェクトは以下の Phase 構成で実装を進めているのだ。

| Phase | 内容 | ドキュメント |
|-------|------|-------------|
| Phase 0 | リポジトリ整備（本フェーズ） | `docs/superpowers/plans/2026-04-07-go-backend-phase0-repo-setup.md` |
| Phase 1 | 基盤構築（DB・Redis・ヘルスチェック） | `docs/superpowers/plans/2026-04-07-go-backend-phase1-foundation.md` |
| Phase 2 | 認証（JWT + Google OAuth） | `docs/superpowers/plans/2026-04-07-go-backend-phase2-auth.md` |
| Phase 3 | コアAPI（天気・潮汐・スコア・お気に入り） | `docs/superpowers/plans/2026-04-07-go-backend-phase3-core-apis.md` |
| Phase 4 | デプロイ（k3s + Cloudflare Tunnel） | `docs/superpowers/plans/2026-04-07-go-backend-phase4-deployment.md` |

## ブランチ戦略

```
main          本番ブランチ（直接pushは禁止）
feature/*     機能開発ブランチ
fix/*         バグ修正ブランチ
chore/*       設定・ドキュメント変更
```

## コミットメッセージ規約

```
<type>: <description>

<optional body>
```

**type 一覧:**

| type | 用途 |
|------|------|
| `feat` | 新機能追加 |
| `fix` | バグ修正 |
| `refactor` | リファクタリング（動作変更なし） |
| `docs` | ドキュメント変更 |
| `test` | テスト追加・修正 |
| `chore` | 設定・ビルド・依存関係変更 |
| `perf` | パフォーマンス改善 |
| `ci` | CI/CD 設定変更 |

**例:**
```
feat: 天気APIエンドポイント追加（GET /api/weather）

OpenWeatherMap APIを呼び出し、Redisに30分キャッシュする実装。
```

## 開発環境セットアップ

### 前提条件

- Go 1.24+
- Docker + Docker Compose
- make
- sqlc: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- golangci-lint: `brew install golangci-lint`
- gofumpt: `go install mvdan.cc/gofumpt@latest`
- lefthook: `brew install lefthook`

### 手順

```bash
# 1. リポジトリクローン
git clone https://github.com/kazumadev619-dev/fishing-api.git
cd fishing-api

# 2. 依存関係インストール
go mod download

# 3. 環境変数設定
cp .env.example .env
# .env を編集して各 API キーを設定

# 4. DB・Redis 起動（ローカル開発用）
make docker-up

# 5. sqlc コード生成
make sqlc-gen

# 6. サーバー起動
make run

# 動作確認
curl http://localhost:8080/health
```

## よく使うコマンド

```bash
make run        # 開発サーバー起動
make test       # 全テスト実行
make lint       # Lint チェック（golangci-lint）
make sqlc-gen   # sqlc コード再生成
make docker-up  # DB・Redis 起動
make docker-down # DB・Redis 停止
make build      # バイナリビルド（./bin/server）
```

## アーキテクチャ概要

クリーンアーキテクチャを採用しているのだ。依存関係は内側（domain）に向かう。

```
cmd/server/          エントリポイント（main.go）
config/              設定読み込み（環境変数）
domain/              ドメイン層（エンティティ・リポジトリIF・ドメインエラー）
usecase/             ユースケース層（ビジネスロジック）
infrastructure/      インフラ層（DB・外部API・キャッシュの実装）
  db/                sqlc 生成コード + リポジトリ実装
  external/          外部 API クライアント（天気・潮汐・Maps）
  email/             メール送信（Resend）
interface/           インターフェース層（HTTP ハンドラー・ミドルウェア）
  handler/           Gin ハンドラー
  middleware/        JWT 認証ミドルウェア
  router/            ルーティング定義
pkg/                 共通パッケージ（JWT・バリデーター）
db/                  スキーマ・sqlc クエリ・生成コード
k8s/                 Kubernetes マニフェスト（Traefik Ingress）
```

## PR のルール

1. `main` への直接 push 禁止
2. PR には `PULL_REQUEST_TEMPLATE.md` に従って記載する
3. `make test` と `make lint` が通っていること
4. セキュリティ関連の変更は security-reviewer エージェントでレビューすること
5. CRITICAL・HIGH 指摘がある場合はマージ不可

## 質問・相談

Issue を立てるか、PR のコメントで相談してくださいなのだ。
