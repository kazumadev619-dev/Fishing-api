# Go Backend Phase 4: Deployment Implementation Plan（Refined）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **Refined 版について:** 本ファイルは `2026-04-07-go-backend-phase4-deployment.md` を踏襲しつつ、Phase 4 着手前の前提整備（Branch A）で確定した以下を反映した改訂版なのだ。
>
> - **Go バージョンを 1.26 に統一**（Dockerfile / CI / sync-schema すべて旧 1.24 → 1.26）
> - **CI にカバレッジゲート（80% 未満で fail）を追加**
> - **Hurl による本番スモークテスト**を deploy に組み込み
> - **Phase 3 の実配線**（`router.New(handlers, jwtManager)` シグネチャ・`jwtManagerAdapter`）を前提に記述

**Goal:** Dockerfile（ARM64）・k8sマニフェスト・GitHub Actions CI/CD・Cloudflare Tunnelを構成し、Raspberry Pi k3sへ自動デプロイが動作する状態にする。デプロイ後は Hurl スモークテストで `/health` と `/api/weather` の疎通を自動検証する。

**Architecture:** GitHub Actions → GHCR（ARM64イメージ）→ Cloudflare Tunnel → k3s kubectl。k8s IngressでTraefikが `/api/*` をGoバックエンドに、`/*` をNext.jsに振り分ける。

**Tech Stack:** Go 1.26, Docker buildx, k3s, Traefik, Cloudflare Tunnel, GitHub Actions, Hurl

**前提条件:** Phase 1〜3完了済み。Raspberry PiにはUbuntu 24.04 LTS + k3sインストール済み。Cloudflareアカウント・ドメイン設定済み。PostgreSQL は Neon（マネージドクラウド）を使用。Redis は k3s Pod として k3s クラスター内で運用する（ClusterIP Service 経由でアクセス）。

### Phase 3 の出力（このプランが依存する実装）

| 項目 | 実装 | 備考 |
|------|------|------|
| ルーター生成 | `router.New(handlers *router.Handlers, jwtManager *jwtutil.Manager) *gin.Engine` | `internal/interface/router/router.go` |
| ヘルスチェック | `GET /health` → `{"status":"ok"}`（200） | `handler.HealthCheck` |
| 天気エンドポイント | `GET /api/weather?lat=&lon=` | `handlers.Weather.Get` |
| JWT 配線 | `cmd/server/main.go` の `jwtManagerAdapter{m: jwtManager}` を `auth.NewAuthUsecase(...)` に注入。`router.New(handlers, jwtManager)` に **`*jwtutil.Manager` 実体**を渡す（ミドルウェア `middleware.JWTAuth(jwtManager)` が使用） | adapter は usecase 用、router へは実体を渡す二段構成 |

> **重要:** Dockerfile でビルドするバイナリは `./cmd/server`。`main.go` は `config.Load()` → `infradb.NewPool` → `stdlib.OpenDBFromPool` → repositories → usecases → handlers → `router.New(handlers, jwtManager)` → `r.Run(":" + cfg.Server.Port)` の順で組み立てる。環境変数が揃っていないと `config.Load()` で fail するため、k8s Secret の網羅性が要件なのだ。

---

## ファイル構成

| 操作 | ファイル | 内容 |
|------|---------|------|
| 新規作成 | `Dockerfile` | マルチステージビルド（linux/arm64・**golang:1.26-alpine**） |
| 新規作成 | `.dockerignore` | Dockerビルド除外ファイル |
| 新規作成 | `k8s/namespace.yaml` | Kubernetesネームスペース定義 |
| 新規作成 | `k8s/fishing-api/deployment.yaml` | Goバックエンドデプロイメント |
| 新規作成 | `k8s/fishing-api/service.yaml` | Goバックエンドサービス |
| 新規作成 | `k8s/fishing-api/ingress.yaml` | `/api/*` → fishing-api ルーティング |
| 新規作成 | `k8s/frontend/deployment.yaml` | Next.jsフロントエンドデプロイメント |
| 新規作成 | `k8s/frontend/service.yaml` | Next.jsフロントエンドサービス |
| 新規作成 | `k8s/frontend/ingress.yaml` | `/*` → frontend ルーティング |
| 新規作成 | `k8s/cloudflared/deployment.yaml` | Cloudflare Tunnelエージェント |
| 新規作成 | `k8s/redis/deployment.yaml` | Redis Pod デプロイメント |
| 新規作成 | `k8s/redis/service.yaml` | Redis ClusterIP Service |
| 新規作成 | `k8s/config/redis-secret.yaml` | Redis接続情報シークレット |
| 新規作成 | `k8s/config/fishing-api-secret.yaml` | 環境変数シークレット（DATABASE_URL含む） |
| 新規作成 | `.github/workflows/ci.yml` | Lint・テスト（**カバレッジゲート 80%**）・ビルドCI |
| 新規作成 | `.github/workflows/deploy.yml` | GHCR push → k3sデプロイ → **Hurl スモークテスト** |
| 新規作成 | `.github/workflows/sync-schema.yml` | DBリポジトリからschema.sql同期 |
| 新規作成 | `tests/smoke/health.hurl` | `/health` スモークテスト |
| 新規作成 | `tests/smoke/weather.hurl` | `/api/weather` スモークテスト |
| 新規作成 | `docs/development.md` | ローカル開発セットアップ手順 |

---

## Task 1: Dockerfile（ARM64マルチステージビルド・Go 1.26）

**Files:**
- Create: `Dockerfile`
- Create: `.dockerignore`

- [ ] **Step 1: .dockerignore を作成する**

```
# .dockerignore
.git
.github
*.md
docs/
k8s/
tests/
.env
.env.*
bin/
cov.out
coverage.out
```

- [ ] **Step 2: Dockerfile を作成する**

```dockerfile
# Dockerfile
# -----------------------------------------------
# Stage 1: ビルド（linux/arm64対応・Go 1.26）
# -----------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /app

# 依存関係をキャッシュするために先にgo.mod/go.sumをコピー
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# ARM64向けバイナリをビルド
ARG TARGETOS=linux
ARG TARGETARCH=arm64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -o bin/server ./cmd/server

# -----------------------------------------------
# Stage 2: 実行イメージ（最小サイズ）
# -----------------------------------------------
FROM --platform=linux/arm64 alpine:3.21

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Tokyo

WORKDIR /app
COPY --from=builder /app/bin/server .

EXPOSE 8080
CMD ["./server"]
```

- [ ] **Step 3: ローカルでDockerビルドが通ることを確認する（ARMエミュレーション）**

```bash
docker buildx create --use --name multiarch
docker buildx build --platform linux/arm64 -t fishing-api:test --load .
```

Expected: `golang:1.26-alpine` をベースにイメージが正常にビルドされる

- [ ] **Step 4: コミット**

```bash
git add Dockerfile .dockerignore
git commit -m "🐳 feat: Dockerfile追加（ARM64マルチステージビルド・Go 1.26）"
```

---

## Task 2: Kubernetesマニフェスト（namespace + secrets）

旧プラン（`2026-04-07-go-backend-phase4-deployment.md` Task 2）をそのまま踏襲する。`fishing-api-secret` には `main.go` の `config.Load()` が要求する全環境変数を含めること：

```
DATABASE_URL            # Neon 接続文字列（?sslmode=require 必須）
JWT_ACCESS_SECRET
JWT_REFRESH_SECRET
OPENWEATHER_API_KEY
GOOGLE_MAPS_API_KEY
RESEND_API_KEY
EMAIL_FROM
APP_BASE_URL            # 認証メールのリンク生成に使用（auth usecase）
```

`redis-secret` には `REDIS_URL` を持たせる。詳細な YAML テンプレートは旧プラン Task 2 を参照するのだ。

---

## Task 3: fishing-apiのk8sマニフェスト

旧プラン Task 3 を踏襲する。`readinessProbe` / `livenessProbe` は `GET /health`（Phase 3 の `handler.HealthCheck`、`{"status":"ok"}` を 200 で返す）を使う。env には Task 2 の全 Secret キーを `secretKeyRef` で注入する（`config.Load()` が欠落時に fail するため網羅必須）。

---

## Task 4: フロントエンド・Cloudflared・Redis k8sマニフェスト

旧プラン Task 4 を踏襲する。加えて Redis を k3s Pod として運用するため `k8s/redis/deployment.yaml` と `k8s/redis/service.yaml`（ClusterIP）を作成し、`redis-secret` の `REDIS_URL` をクラスター内 Service 名（例: `redis://redis.fishing.svc.cluster.local:6379`）で構成するのだ。

---

## Task 5: GitHub Actions CI（Lint・Test・カバレッジゲート・Build）

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: ci.yml を作成する（Go 1.26・カバレッジゲート付き）**

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'
          cache: true

      - name: Install sqlc
        run: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

      - name: Generate sqlc code
        run: sqlc generate

      # infrastructure/db は testcontainers-go を使うため Docker が必要。
      # GitHub Actions の ubuntu-latest には Docker が同梱されているのだ。
      - name: Run tests with coverage
        env:
          JWT_ACCESS_SECRET: test-access-secret-32chars-minimum
          JWT_REFRESH_SECRET: test-refresh-secret-32chars-minimum
        run: go test -coverprofile=coverage.out ./...

      - name: Enforce coverage gate (>= 80%)
        run: |
          COV=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | tr -d %)
          awk -v c="$COV" 'BEGIN{exit !(c+0<80)}' && { echo "coverage $COV% < 80%"; exit 1; } || echo "coverage $COV% OK"

      - name: Build
        run: go build ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'
          cache: true

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
```

> **カバレッジゲートの読み方:** `awk 'BEGIN{exit !(c+0<80)}'` は「カバレッジが 80 未満なら exit 0（=真）」を返す。`&&` 側で `exit 1`（fail）し、80 以上なら `||` 側で `OK` を出力する。`c+0` で文字列を数値強制しているのだ。

- [ ] **Step 2: `.golangci.yml` が存在することを確認する**

```bash
ls .golangci.yml
```

- [ ] **Step 3: コミット**

```bash
git add .github/workflows/ci.yml
git commit -m "👷 ci: CIワークフロー追加（Go 1.26・カバレッジゲート80%）"
```

---

## Task 6: GitHub Actions デプロイ（GHCR → k3s → Hurl スモーク）

**Files:**
- Create: `.github/workflows/deploy.yml`
- Create: `tests/smoke/health.hurl`
- Create: `tests/smoke/weather.hurl`

- [ ] **Step 1: Hurl スモークテストを作成する**

```hurl
# tests/smoke/health.hurl
GET {{base_url}}/health
HTTP 200
[Asserts]
body contains "ok"
```

```hurl
# tests/smoke/weather.hurl
GET {{base_url}}/api/weather?lat=35.0&lon=139.0
HTTP 200
[Asserts]
jsonpath "$.temperature" exists
```

> `base_url` は Hurl の `--variable` で外から注入する（例: `https://fishing.kazuma-lab.com`）。本番 Ingress 経由のエンドツーエンド疎通を確認するのだ。

- [ ] **Step 2: deploy.yml を作成する（rollout 後に Hurl スモーク）**

ビルド＆プッシュ・kubeconfig 構成・Secret apply・rollout は旧プラン Task 6 を踏襲する。`deploy` ジョブの `kubectl rollout status` 成功後に、以下の Hurl ステップを追加するのだ：

```yaml
      - name: Deploy to k3s
        env:
          KUBECONFIG: kubeconfig.yaml
        run: |
          kubectl apply -f k8s/fishing-api/
          kubectl set image deployment/fishing-api \
            fishing-api=ghcr.io/${{ github.repository_owner }}/fishing-api:sha-${{ github.sha }} \
            -n fishing
          kubectl rollout status deployment/fishing-api -n fishing --timeout=300s

      - name: Install Hurl
        run: |
          curl -L https://github.com/Orange-OpenSource/hurl/releases/latest/download/hurl_amd64.deb -o hurl.deb
          sudo dpkg -i hurl.deb

      - name: Smoke test (Hurl)
        run: |
          hurl --variable base_url=https://fishing.kazuma-lab.com \
               --test tests/smoke/*.hurl
```

> **失敗時の挙動:** `hurl --test` は 1 件でもアサーション失敗すると非ゼロ終了し、デプロイジョブを fail させる。これによりイメージ更新が成功しても疎通が壊れていればワークフローが赤くなり、ロールバック判断ができるのだ。

- [ ] **Step 3: コミット**

```bash
git add .github/workflows/deploy.yml tests/smoke/
git commit -m "🚀 ci: デプロイワークフロー追加（GHCR → k3s → Hurlスモーク）"
```

---

## Task 7: DB schema.sql 自動同期ワークフロー

旧プラン Task 7 を踏襲する。ただし `actions/setup-go` の `go-version` は **`'1.26'`** に変更するのだ。

```yaml
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'
```

---

## Task 8: 開発ドキュメント作成

旧プラン Task 8 を踏襲する。`docs/development.md` の前提条件は Go 1.26+。`make test` のカバレッジ確認手順に「ローカルでも CI と同じく `go tool cover -func=coverage.out | tail -1` で 80% を確認する」旨を追記するのだ。infrastructure/db のテストは Docker（testcontainers）が必要な点も明記する。

---

## Task 9 / 10: Raspberry Pi Redis 構成・k3s 初回デプロイ

旧プラン Task 9・Task 10 を踏襲する。Redis は k3s Pod 運用（Task 4 で作成）に統一したため、旧 Task 9 の Docker Compose 手順は「k3s 外運用を選ぶ場合のフォールバック」として残す。本番疎通確認（旧 Task 10 Step 7）は Hurl スモークテスト（Task 6）に置き換わり、CI/CD で自動化されるのだ。

---

## 完了条件チェックリスト

- [ ] `Dockerfile` のビルドステージが `golang:1.26-alpine` を使用している
- [ ] `docker buildx build --platform linux/arm64` でビルドが通る
- [ ] CI の `setup-go` が `go-version: '1.26'`（ci.yml / sync-schema.yml すべて）
- [ ] CI に `coverage.out` を用いたカバレッジゲートがあり、80% 未満で fail する
- [ ] GitHub Actions CIが全テストPASS（カバレッジ 80%+ を含む）
- [ ] mainへのpushでGHCRにARM64イメージがpushされる
- [ ] k3s上で全Podが `Running` 状態
- [ ] `tests/smoke/health.hurl` と `tests/smoke/weather.hurl` が存在する
- [ ] deploy.yml の `kubectl rollout status` 成功後に `hurl --test tests/smoke/*.hurl` が実行される
- [ ] Hurl スモークが `https://fishing.kazuma-lab.com/health`（body に `ok`）と `/api/weather`（`$.temperature` exists）を検証する
- [ ] DBスキーマ変更時にsync-schema.ymlが自動でPRを作成する
