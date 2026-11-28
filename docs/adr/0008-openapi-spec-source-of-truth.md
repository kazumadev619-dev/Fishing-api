# 0008. OpenAPI の仕様を正にして Go と Next の型を生成する

- Status: Accepted
- Date: 2026-11-28
- Deciders: kazumadev619-dev

## Context and Problem Statement

[ADR 0006](0006-incremental-migration-via-nextjs-bff.md) に従い、Next（FishingConditionsApp）の Route Handler を 1 本ずつ Go（Fishing-api）への中継に置き換えている。

段 2（#26、PR #38）では、Go の `/api/weather` と `/api/conditions/tide` の応答を Next と同じ形に揃えた。しかし、この時点では応答の形が Go 側の手書き構造体（`weatherResponse`、`tideResponse` 等）と Next 側の型（`FormattedWeatherData`、`FormattedTideData` 等）の 2 か所に二重定義されており、片方だけを変更しても食い違いを機械的に検知できない。

段 3（#27）で Next から Go へ中継を始める前に、Go と Next が同じ API 契約を参照できる単一の正（Source of Truth）と型生成の仕組みを決める必要がある。

## Considered Options

- **OpenAPI 仕様（`api/openapi.yaml`）を Fishing-api に置き、Go と Next の両方で型を生成する**
  - OpenAPI 3.0.3 + `oapi-codegen` v2（strict server）+ Next は `openapi-typescript`
- OpenAPI 3.1.0 を採用する
  - `oapi-codegen` v2 は OpenAPI 3.1 に未対応のため採用できない
- 手書き構造体と TypeScript 型をそのまま維持し、テストだけで整合性を保つ
  - 仕様と型の乖離をコンパイル時に防げず、段が進むごとに保守コストが増える
- 契約専用の別リポジトリを作る
  - 3 リポジトリ運用に 4 つ目のリポジトリが増え、変更の同期コストが高い

## Decision Outcome

**`api/openapi.yaml` を Fishing-api に置き、HTTP API 契約の正（Source of Truth）とする。Go と Next の型はここから生成する。**

### 仕様の置き場所と版

- **置き場所**: `Fishing-api` の `api/openapi.yaml`。API を提供する側が契約の正を持つ。
- **OpenAPI の版**: **3.0.3**。`oapi-codegen` v2 が 3.0 系のみ対応しており 3.1 に未対応であるため。Next 側の `openapi-typescript` は 3.0 と 3.1 の両方に対応している。
- **適用範囲**: 段ごとに段階的に広げる。最初は段 2 で Next と形を揃えた `GET /api/weather` と `GET /api/conditions/tide` だけを書く。残りのエンドポイントは、それぞれを移行する段で仕様に追加する。

### Go 側の生成（oapi-codegen strict server + ErrorHandler フォールバック）

- `github.com/oapi-codegen/oapi-codegen/v2` を `go.mod` の `tool` ディレクティブで固定し、`make openapi-gen` で `internal/interface/openapi/openapi.gen.go` を生成・コミットする（設定は `api/oapi-codegen.yaml`）。
- 生成対象は `models`、`gin-server`、`strict-server` とする。strict server を使うことで、各ステータスコード（200・400・500・502・503・504）やヘッダー（`Cache-Control`）を型で縛れる。
- **ErrorHandler フォールバックを置く理由**: 生成された `ServerInterfaceWrapper` はハンドラ呼び出し前にクエリパラメータを型へバインドする。必須クエリ欠落や数値パース失敗時は既定の `ErrorHandler` が `{"msg": "..."}` を返してしまい、以下 2 点が段 2 の契約（Next 互換）と食い違う。
  1. 400 のエラー文言（`lat and lon are required`、`invalid lat: ...`、`Missing required parameter: prefectureCode` 等）
  2. `/api/weather` で OpenWeatherMap API キー未設定時は、クエリ検証より先に 503（`API_NOT_CONFIGURED`）を返す順序
- そのため `GinServerOptions.ErrorHandler` にルート（`c.FullPath()`）ごとのフォールバック関数を渡し、段 2 と同じ順序・検証ロジック・文言で応答する。
- 生成コードと仕様の食い違いは CI（`make openapi-gen && git diff --exit-code`）で検知し、実応答と仕様の整合は `kin-openapi`（`openapi3filter.ValidateResponse`）を使ったテストで検証する。

### Next への渡し方

- Next（FishingConditionsApp）は `Fishing-api` の特定コミットに固定して `api/openapi.yaml` を取得し、`openapi-typescript` で TypeScript の型を生成する（FishingConditionsApp 側の実装は別 issue）。

## Positive Consequences

- Go と Next が同じ `api/openapi.yaml` から型を生成するため、フィールド名・型・必須/任意のずれをコンパイル時に検知できる。
- Go ハンドラから手書き応答構造体が消え、ステータスコード・ヘッダー・エラー形式が strict server の型で強制される。
- CI で仕様と生成物の乖離を防ぎ、`kin-openapi` のテストでランタイム応答（JSON シリアライズ結果・ヘッダー）も仕様準拠を保証できる。
- `go.mod` の `tool` により `oapi-codegen` の版が固定され、ローカルと CI で再現性があり Renovate で更新できる。

## Negative Consequences

- `oapi-codegen` のラッパーはクエリバインド失敗時に既定形式で弾くため、Next 互換のエラー文言と評価順を保つための `ErrorHandler` フォールバックの維持が必要になる。
- 生成コードがリポジトリに含まれるため、仕様変更のたびに `make openapi-gen` の実行と生成物のコミットが必要になる（CI で漏れを検知する）。
- OpenAPI 3.0.3 制約により 3.1 固有の機能は使えない。

## Links

- 関連: [ADR 0006 — Next を BFF にして1本ずつ Go に移す](0006-incremental-migration-via-nextjs-bff.md)、[ADR 0007 — スキーマの正を Go 側に移し、goose で管理する](0007-schema-ownership-goose.md)
- OpenAPI 運用ルール: [`.claude/rules/openapi.md`](../../.claude/rules/openapi.md)
- 仕様ファイル: [`api/openapi.yaml`](../../api/openapi.yaml)
- oapi-codegen: https://github.com/oapi-codegen/oapi-codegen
- kin-openapi: https://github.com/getkin/kin-openapi
