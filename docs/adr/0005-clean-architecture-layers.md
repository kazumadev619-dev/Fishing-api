# 0005. クリーンアーキテクチャ層構成を採用する

- Status: Accepted
- Date: 2026-05-08
- Deciders: kazumadev619-dev

## Context and Problem Statement

Go バックエンドの内部構造を決める。外部依存（Neon・Redis・天気/潮汐/Maps API・Resend）が多く、これらは差し替えやモック化が必要なのだ。ビジネスロジック（認証・スコア算出）を外部技術の詳細から守り、テスト容易性（usecase をモックで高速にテスト、infra を testcontainers で検証）を確保したい。

## Considered Options

- レイヤードアーキテクチャ（単純な 3 層、依存方向の制約なし）
- パッケージ・バイ・フィーチャー（機能ごとにフラットに分割）
- クリーンアーキテクチャ（依存は内側のみ：`interface → usecase → domain ← infrastructure`）

## Decision Outcome

**クリーンアーキテクチャ**を採用する。依存方向は内側のみとし、`domain/` を最内層に置く。

```
interface/ → usecase/ → domain/ ← infrastructure/
```

- `internal/domain/` エンティティ・リポジトリ IF（最内層・外を知らない）
- `internal/usecase/` ビジネスロジック（domain の型・IF のみ依存）
- `internal/infrastructure/` DB・Redis・外部 API 実装（domain IF の実装）
- `internal/interface/` Gin ハンドラー・ミドルウェア・ルーター（usecase のみ依存）
- `cmd/server/main.go` DI 組み立て（全レイヤーの import を許可する唯一の場所）

主要な禁止 import: `usecase → infrastructure`（IF 経由のみ）、`interface/handler → infrastructure`、`interface/handler → db/generated`。

## Positive Consequences

- ビジネスロジックが外部技術の詳細から独立し、Neon→他 DB や Redis→他キャッシュへの差し替えが容易
- usecase は domain IF のモックで高速にユニットテストでき、infra は testcontainers で実 DB 検証できる
- 依存方向が一方向に固定され、循環依存や層の漏れを防げる

## Negative Consequences

- レイヤー間のインターフェース定義・DTO 変換でボイラープレートが増える
- 小さな機能追加でも複数層を横断する手間がある
- DI を `cmd/server/main.go` に集約する規律が必要（散らばると利点が崩れる）

## Links

- 設計仕様: [`docs/superpowers/specs/2026-04-07-go-backend-design.md`](../superpowers/specs/2026-04-07-go-backend-design.md)（「アーキテクチャ：クリーンアーキテクチャ」節）
- クリーンアーキテクチャルール: [`.claude/rules/clean-architecture.md`](../../.claude/rules/clean-architecture.md)
- 実装: `internal/domain/`, `internal/usecase/`, `internal/infrastructure/`, `internal/interface/`, `cmd/server/main.go`
- 関連: [ADR 0003 — ビッグバン移行](0003-bigbang-migration-from-nextjs-api.md), [ADR 0004 — DB アクセス sqlc + pgx](0004-db-access-sqlc-pgx.md)
