# Architecture Decision Records (ADR)

Fishing-api の主要な設計判断を記録した索引なのだ。各 ADR は [MADR](https://adr.github.io/madr/) テンプレートに沿って「なぜその構成を選んだか」を残す。

| No. | タイトル | 要約 |
|-----|---------|------|
| [0001](0001-database-provider-neon.md) | DB プロバイダに Neon を採用 | Supabase Free は 7 日不使用で停止するため却下。無料・非アクティブ停止なし・自動バックアップ付きの Neon を採用 |
| [0002](0002-redis-on-k3s-pod.md) | Redis を k3s Pod で運用 | キャッシュはステートレスな揮発データ。Pod 再起動で消えても DB から再取得するだけなので PVC 永続化は不要 |
| [0003](0003-bigbang-migration-from-nextjs-api.md) | Next.js API からのビッグバン移行（Superseded by 0006） | Next.js API Routes を全廃し Go で全 API を再実装。k8s Ingress で `/api/*` を一括切替（フロント変更不要） |
| [0004](0004-db-access-sqlc-pgx.md) | DB アクセスに sqlc + pgx/v5 | タイプセーフな SQL 生成で SQL インジェクション耐性を確保。Neon 接続は `sslmode=require` 必須 |
| [0005](0005-clean-architecture-layers.md) | クリーンアーキテクチャ層構成 | 依存は内側のみ `interface → usecase → domain ← infrastructure`。差し替え容易性とテスト容易性を確保 |
| [0006](0006-incremental-migration-via-nextjs-bff.md) | Next を BFF にして1本ずつ Go に移す | Ingress の一括切替をやめ、Next の Route Handler を1本ずつ Go への中継に置き換える。認証は最後の段まで Auth.js に残し、Next から Go へは署名した短命の内部トークンで user.id を渡す |
| [0007](0007-schema-ownership-goose.md) | スキーマの正を Go 側に移し、goose で管理する | Phase 4 でスキーマの正を Prisma から Go に移す。goose の SQL をバイナリに埋め込み、db-migrate Job は Go のイメージで流す |
| [0008](0008-openapi-spec-source-of-truth.md) | OpenAPI の仕様を正にして Go と Next の型を生成する | `api/openapi.yaml`（OpenAPI 3.0.3）を契約の正にし、Go は `oapi-codegen`（strict server）、Next はコミット固定で取得して `openapi-typescript` で型を生成する |

## ADR の書き方

- ファイル名: `NNNN-kebab-case-title.md`（連番 4 桁）
- セクション: Status / Date / Deciders / Context and Problem Statement / Considered Options / Decision Outcome / Positive Consequences / Negative Consequences / Links
- Status は `Proposed` → `Accepted` → （必要なら）`Deprecated` / `Superseded by NNNN`
