# OpenAPI 管理ルール

## 生成コード保護

`internal/interface/openapi/` は **oapi-codegen が自動生成するため編集禁止**。

```
api/
├── openapi.yaml           # ← ここだけ手動編集する（HTTP API 契約の正・ADR 0008）
└── oapi-codegen.yaml      # oapi-codegen 生成設定

internal/interface/openapi/
└── openapi.gen.go         # ← 編集禁止（make openapi-gen で再生成）
```

変更手順:
1. `api/openapi.yaml` を編集する
2. `make openapi-gen` で再生成する
3. 必要なハンドラ（`internal/interface/handler/`）と仕様照合テスト（`openapi_test.go`）を更新する
4. `go build ./...` と `make test` でコンパイル・テストエラーがないか確認する

## 仕様管理

API 契約の正は [ADR 0008](../../docs/adr/0008-openapi-spec-source-of-truth.md) に従う。

- OpenAPI の版は **3.0.3** を使う（`oapi-codegen` v2 は 3.1 未対応）
- 対象エンドポイントは移行の段ごとに広げる（最初は `/api/weather` と `/api/conditions/tide`）
- `internal/interface/openapi/` は `.golangci.yml` の lint 対象外および `make cover` のカバレッジ計測対象外とする
- CI で `make openapi-gen && git diff --exit-code` を実行し、仕様と生成物の食い違いを検知する

## ローカル開発フロー

```bash
# 1. OpenAPI 仕様を編集
vim api/openapi.yaml

# 2. oapi-codegen で Go コード再生成
make openapi-gen

# 3. コンパイル確認
go build ./...

# 4. テスト実行（仕様照合テストを含む）
make test
```

## 関連ルール

- [clean-architecture.md](./clean-architecture.md) — `internal/interface/openapi/` は `interface/` 層のみから import する（`domain/`・`usecase/`・`infrastructure/` からは import しない）
- [sqlc.md](./sqlc.md) — `db/generated/` と同じ生成コードの扱い
