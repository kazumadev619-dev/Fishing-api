# 0002. Redis を k3s Pod で運用する

- Status: Accepted
- Date: 2026-05-08
- Deciders: kazumadev619-dev

## Context and Problem Statement

天気・潮汐・場所検索など外部 API の結果をキャッシュするために Redis が必要なのだ。永続データは Neon（マネージドクラウド）に置く（[ADR 0001](0001-database-provider-neon.md)）が、Neon には Redis 相当のマネージドサービスがない。キャッシュ層をどこに・どう運用すべきか。

## Considered Options

- マネージド Redis（Upstash / Redis Cloud などの外部クラウド）
- k3s Pod として Redis を運用（ClusterIP Service 経由）
- k3s Pod + PVC で永続化した Redis

## Decision Outcome

**Redis を k3s Pod として k3s クラスター内で運用する**（ClusterIP Service `redis-service` 経由でアクセス）。Pod なので `kubectl` でライフサイクル管理でき、クラスター内 DNS 名で接続できる。キャッシュはステートレスな揮発データなので **PVC による永続化は不要**。

## Positive Consequences

- 追加コストなしでクラスター内に閉じて運用できる
- `kubectl` でライフサイクル管理でき、DNS 名（`redis-service`）で接続できる
- Pod 再起動でキャッシュが消えても DB（Neon）から再取得するだけで復旧する
- 永続化（PVC）が不要なので運用がシンプル

## Negative Consequences

- Pod 再起動直後はキャッシュミスが増え、一時的に外部 API 呼び出しが増加する
- マネージド Redis のような自動冗長化・バックアップはない（キャッシュ用途なので許容）
- Raspberry Pi のリソースを消費する

## Links

- 設計仕様: [`docs/superpowers/specs/2026-04-07-go-backend-design.md`](../superpowers/specs/2026-04-07-go-backend-design.md)（「確定事項」/ キャッシュ戦略・デプロイメント構成）
- キャッシュ実装: `internal/infrastructure/cache/`
- 関連: [ADR 0001 — DB プロバイダ Neon](0001-database-provider-neon.md)
