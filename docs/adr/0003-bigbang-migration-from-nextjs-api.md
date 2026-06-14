# 0003. Next.js API Routes からのビッグバン移行

- Status: Accepted
- Date: 2026-05-08
- Deciders: kazumadev619-dev

## Context and Problem Statement

FishingConditionsApp の API は当初 Next.js API Routes で実装されていた。これを独立した Go サービスへ移行する。フロントエンド（Next.js）はそのままに、バックエンドをどのような戦略で切り替えるか。段階的（ストラングラーパターン）に少しずつ移すか、一括で切り替えるか。

## Considered Options

- ストラングラー移行（エンドポイントごとに段階的に Go へ移す）
- ビッグバン移行（Go で全 API を再実装し、k8s Ingress で `/api/*` を一括切り替え）

## Decision Outcome

**ビッグバン移行**を採用する。Next.js API Routes を全廃し、Go バックエンドですべての API を再実装する。k8s Ingress（Traefik）のルールで `/api/*` を Go バックエンドへ転送するため、フロントエンドのコード変更は不要。

移行手順:
1. Go バックエンドをローカルで完成させ、全 API を実装
2. k3s 上に Go バックエンドをデプロイ（Next.js と並走）
3. k8s Ingress のルールを Go バックエンドへ切り替え（`/api/*` → fishing-api）
4. Next.js の API Routes を削除

## Positive Consequences

- フロントエンドのコード変更が不要（Ingress のルーティング切替のみ）
- 2 実装を長期間並走させる複雑さ・整合性管理を回避できる
- エラーレスポンス形式を既存と同一に保てば互換性を維持できる

## Negative Consequences

- 切り替え時点で全 API が完成している必要があり、リリースが大きくなる
- 切り替え時の不具合が全 API に波及するリスク（Ingress ルール巻き戻しで緩和）
- 個人開発規模だから許容できる戦略であり、大規模チームには不向き

## Links

- 設計仕様: [`docs/superpowers/specs/2026-04-07-go-backend-design.md`](../superpowers/specs/2026-04-07-go-backend-design.md)（「移行戦略」節）
- Ingress 定義: `k8s/fishing-api/ingress.yaml`（`/api/*` → fishing-api）
- 関連: [ADR 0005 — クリーンアーキテクチャ層構成](0005-clean-architecture-layers.md)
