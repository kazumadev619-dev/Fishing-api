# 0006. Next を前段（BFF）にして、Route Handler を1本ずつ Go に移す

- Status: Accepted
- Date: 2026-11-27
- Deciders: kazumadev619-dev
- Supersedes: [ADR 0003](0003-bigbang-migration-from-nextjs-api.md)

## Context and Problem Statement

ADR 0003 は Go で全 API を作り直し、k8s Ingress で `/api/*` を一括で Go に切り替える方針だった。前提は「Ingress を切り替えれば、フロントは変えなくてよい」ことだった。

2026-11-27 の差分調査で、この前提は成り立たないと分かった。

- 認証の方式が違う。フロントは Auth.js のセッション Cookie で `/api/*` を呼ぶが、Go は `Authorization: Bearer` しか見ない。お気に入りはすべて 401 になる。
- `/api/auth/*` を Go に回すと、Auth.js 本体（ログイン、Google のコールバック、`useSession`、`signOut`）が 404 になり、ログインできなくなる。
- 応答の形が違う。Go のエンティティには JSON タグが無く、`Temperature` のような大文字始まりのキーで返す。フロントが読む `data.data`、`favorites[].locationId`、`ports` が Go の応答に無い。
- `/api/ports` と `/api/scores` が Go に無い。`DELETE /api/favorites` はパスの形が違う。
- `/dashboard` のサーバーコンポーネントと Server Action（`signup`、`authenticate`）は `/api/*` を通らない。Ingress を切り替えても Next に残り、Next は DB と外部 API のキーを持ち続ける。
- Go の潮汐クライアントは存在しない URL（`/api/get_tide/...`）を呼んでいる。Next が使うのは `get_tide.php`。
- スコアの式が Next と Go で別物である。

移行の目的も改めた。移行は勉強をかねており、Go の特徴を活かすことを目的に含める。活かしたいのは次の特徴である。

- goroutine と errgroup による並行取得
- singleflight による同じ要求の相乗り
- context による期限
- 常駐の goroutine による先読みと定期処理
- 省メモリの単一バイナリ
- sqlc
- OpenTelemetry の Go SDK

この状況で、Next から Go へどう切り替えるかを決める。

## Considered Options

- A. Ingress で一括に切り替え、Go が Auth.js の Cookie を復号して検証する（ADR 0003 の延長）
- B. Next を前段（BFF）にし、Next の Route Handler を1本ずつ Go への中継に置き換える
- C. 先に認証を Go に移し、Auth.js をやめる

## Decision Outcome

**B** を採用する。

### 切り替えの方式

- Ingress は今のまま（`/*` → Next）にする。ブラウザは Go を直接呼ばない。
- Next の Route Handler を1本ずつ「Go を呼ぶ中継」に置き換える。中継にするかどうかは1本ずつ戻せるようにする。
- `/dashboard` のサーバーコンポーネントも最終的に Go を呼ぶ形にする。
- 切り替える前に、同じ入力で Next と Go の応答を比べる試験を通す。

### 認証

- 認証は移行の最後の段で Go に移す。それまでは Auth.js（Next）に残す。
- Next から Go へは、Next が `auth()` で確かめた user.id を内部トークンで渡す。内部トークンは短命で、Next と Go だけが知る鍵で署名する。
- Go の今の登録・ログイン・リフレッシュの実装は、最後の段まで使わない。最後の段で、次を含めて作り直す。
  - メールの正規化
  - リフレッシュトークンのローテーション
  - Google OAuth
  - メール確認の送り直しと、未確認の登録の掃除

### 仕様をどちらに合わせるか

- スコアの式は Next の式を正とする（FishingConditionsApp の #367 と #436 の決定に合わせる）。
- 応答の形（JSON のキー、`data`／`meta` の包み）、エラーの3分類、リトライの対象は Next に合わせる。

### Go に移さないもの

- `/api/client-errors` は Next に残す。Grafana Faro（FishingConditionsApp #284）が入ったら消す。

### 段の順番

1. Renovate の有効化と依存の整理
2. 潮汐クライアントの作り直しと、上流の応答を使った契約テスト。天気とエラー処理を Next に合わせる
3. Phase 4：Pi へのデプロイ、スキーマの正を Go に移す（[ADR 0007](0007-schema-ownership-goose.md)）、内部トークン、`/api/weather` と `/api/conditions/tide` の中継化
4. `/api/scores` と `/dashboard`
5. 常駐の先読みと定期処理
6. DB 系（`/api/ports`、`/api/favorites`、`/api/locations/search`）、レート制限、キャッシュ
7. 計測（OpenTelemetry の Go SDK）
8. Next と Go の応答を比べる試験（各段の切り替え前に使う。比べる仕組みは段3で作ってもよい）
9. 認証を Go に移す（最後）

段ごとの作業は GitHub の親 issue と子 issue で管理する。

## Positive Consequences

- フロントを変えずに、1本ずつ切り替えて戻せる。切り替えの不具合が全 API に広がらない。
- Go は Auth.js の内部の形式（Cookie 名、暗号の鍵の作り方）を知らなくてよい。
- `/dashboard` と Route Handler の両方を「Next から Go を呼ぶ」1つの形にそろえられる。最後には Next から DB と外部 API のキーを外せる。
- 段ごとに、学ぶ Go の特徴をはっきりさせて進められる。

## Negative Consequences

- Next から Go への1回分、応答が遅くなる。
- 移行のあいだ、Next と Go の2つの実装が並ぶ。応答を比べる試験で食い違いを見つける必要がある。
- Next と Go が共有する署名の鍵を、両方の環境に配る必要がある。
- 認証を移すまで、Go の登録・ログイン・リフレッシュのコードは使われずに残る。

## Links

- 置き換えた ADR: [ADR 0003 — Next.js API Routes からのビッグバン移行](0003-bigbang-migration-from-nextjs-api.md)
- スキーマの正: [ADR 0007 — スキーマの正を Go 側に移し、goose で管理する](0007-schema-ownership-goose.md)
- 関連: [ADR 0005 — クリーンアーキテクチャ層構成](0005-clean-architecture-layers.md)
- FishingConditionsApp: #367、#436（スコアの式）、#283（計測）、#284（Grafana Faro）
