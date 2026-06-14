# 0001. DB プロバイダに Neon を採用する

- Status: Accepted
- Date: 2026-05-08
- Deciders: kazumadev619-dev

## Context and Problem Statement

FishingConditionsApp のバックエンドは Raspberry Pi 5 上の k3s で運用する。永続データ（ユーザー・お気に入り・釣り場マスタ）を格納する PostgreSQL をどこに置くべきか。Raspberry Pi の SD カードはランダム書き込みに弱く、PostgreSQL データが乗った状態で停電や HW 障害が起きると全データ消失のリスクがある。個人運用かつ無料枠で、非アクティブによる停止が起きないマネージド DB が望ましい。

## Considered Options

- Raspberry Pi Docker（自前 PostgreSQL）
- Supabase Free
- Neon Free
- Supabase Pro（$25/月）

## Decision Outcome

**Neon Free** を採用する。サーバーレス PostgreSQL（マネージドクラウド）で、無料枠ながら非アクティブ停止がなく、7 日分の自動バックアップが付く。標準 PostgreSQL なので `pgx/v5 + sqlc` の接続文字列を変えるだけで移行でき、Go コードの変更は不要（接続文字列に `sslmode=require` を追加するのみ）。

| 選択肢 | コスト | 非アクティブ停止 | 管理コスト | 採用 |
|--------|-------|----------------|-----------|------|
| Raspberry Pi Docker（自前） | ゼロ | なし | 高（バックアップ・パッチ自己管理） | ✗ |
| Supabase Free | ゼロ | 7日停止あり | 低 | ✗ |
| **Neon Free** | **ゼロ** | **なし** | **低** | **✓** |
| Supabase Pro | $25/月 | なし | 低 | ✗ |

## Positive Consequences

- 停電・SD カード障害によるデータ消失リスクを排除（クラウドで冗長化）
- 無料枠で非アクティブ停止がない（Supabase Free の 7 日停止問題を回避）
- 7 日分の自動バックアップを標準提供
- 標準 PostgreSQL のため移行コストがほぼゼロ

## Negative Consequences

- 外部クラウドへの依存が生じる（ネットワーク経由のレイテンシ）
- DB 接続文字列に `sslmode=require` が必須（設定漏れで接続失敗）
- Neon には Redis 相当のサービスがないため、キャッシュは別途自前運用が必要（[ADR 0002](0002-redis-on-k3s-pod.md)）

## Links

- 設計仕様: [`docs/superpowers/specs/2026-04-07-go-backend-design.md`](../superpowers/specs/2026-04-07-go-backend-design.md)（「DB 選定理由（Neon 採用）」節）
- セキュリティルール: [`.claude/rules/security.md`](../../.claude/rules/security.md)（DB 接続 / `sslmode=require`）
- 関連: [ADR 0002 — Redis on k3s Pod](0002-redis-on-k3s-pod.md), [ADR 0004 — DB アクセス sqlc + pgx](0004-db-access-sqlc-pgx.md)
