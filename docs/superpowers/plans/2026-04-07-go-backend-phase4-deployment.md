# Go Backend Phase 4: Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Dockerfile（ARM64）・k8sマニフェスト・GitHub Actions CI/CD・Cloudflare Tunnelを構成し、Raspberry Pi k3sへ自動デプロイが動作する状態にする。

**Architecture:** GitHub Actions → GHCR（ARM64イメージ）→ Cloudflare Tunnel → k3s kubectl。k8s IngressでTraefikが `/api/*` をGoバックエンドに、`/*` をNext.jsに振り分ける。

**Tech Stack:** Docker buildx, k3s, Traefik, Cloudflare Tunnel, GitHub Actions

**前提条件:** Phase 1〜3完了済み。Raspberry PiにはUbuntu 24.04 LTS + k3sインストール済み。Cloudflareアカウント・ドメイン設定済み。PostgreSQL・RedisはRaspberry Pi上のDocker Compose（k3s外）で運用する。

---

## ファイル構成

| 操作 | ファイル | 内容 |
|------|---------|------|
| 新規作成 | `Dockerfile` | マルチステージビルド（linux/arm64） |
| 新規作成 | `.dockerignore` | Dockerビルド除外ファイル |
| 新規作成 | `k8s/namespace.yaml` | Kubernetesネームスペース定義 |
| 新規作成 | `k8s/fishing-api/deployment.yaml` | Goバックエンドデプロイメント |
| 新規作成 | `k8s/fishing-api/service.yaml` | Goバックエンドサービス |
| 新規作成 | `k8s/fishing-api/ingress.yaml` | `/api/*` → fishing-api ルーティング |
| 新規作成 | `k8s/frontend/deployment.yaml` | Next.jsフロントエンドデプロイメント |
| 新規作成 | `k8s/frontend/service.yaml` | Next.jsフロントエンドサービス |
| 新規作成 | `k8s/frontend/ingress.yaml` | `/*` → frontend ルーティング |
| 新規作成 | `k8s/cloudflared/deployment.yaml` | Cloudflare Tunnelエージェント |
| 新規作成 | `k8s/config/postgres-secret.yaml` | PostgreSQL接続情報シークレット |
| 新規作成 | `k8s/config/redis-secret.yaml` | Redis接続情報シークレット |
| 新規作成 | `k8s/config/fishing-api-secret.yaml` | 環境変数シークレット |
| 新規作成 | `.github/workflows/ci.yml` | Lint・テスト・ビルドCI |
| 新規作成 | `.github/workflows/deploy.yml` | GHCR push → k3sデプロイ |
| 新規作成 | `.github/workflows/sync-schema.yml` | DBリポジトリからschema.sql同期 |
| 新規作成 | `docker-compose.db.yml` | Raspberry Pi上でのDB（PostgreSQL + Redis）構成 |
| 新規作成 | `docs/development.md` | ローカル開発セットアップ手順 |

---

## Task 1: Dockerfile（ARM64マルチステージビルド）

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
.env
.env.*
bin/
```

- [ ] **Step 2: Dockerfile を作成する**

```dockerfile
# Dockerfile
# -----------------------------------------------
# Stage 1: ビルド（linux/arm64対応）
# -----------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

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
# docker buildxがなければセットアップ
docker buildx create --use --name multiarch

# ARM64向けビルドテスト（pushなし）
docker buildx build --platform linux/arm64 -t fishing-api:test --load .
```

Expected: イメージが正常にビルドされる

- [ ] **Step 4: コミット**

```bash
git add Dockerfile .dockerignore
git commit -m "feat: Dockerfile追加（ARM64マルチステージビルド）"
```

---

## Task 2: Kubernetesマニフェスト（namespace + secrets）

**Files:**
- Create: `k8s/namespace.yaml`
- Create: `k8s/config/fishing-api-secret.yaml`
- Create: `k8s/config/postgres-secret.yaml`
- Create: `k8s/config/redis-secret.yaml`

- [ ] **Step 1: namespace.yaml を作成する**

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: fishing
```

- [ ] **Step 2: fishing-api-secret.yaml を作成する（値はGitHub Secretsで管理）**

```yaml
# k8s/config/fishing-api-secret.yaml
# NOTE: このファイルはテンプレート。実際の値は kubectl create secret で作成するか、
# GitHub ActionsのSecretsからenvsubstで置換してapplyする。
apiVersion: v1
kind: Secret
metadata:
  name: fishing-api-secret
  namespace: fishing
type: Opaque
stringData:
  JWT_ACCESS_SECRET: "${JWT_ACCESS_SECRET}"
  JWT_REFRESH_SECRET: "${JWT_REFRESH_SECRET}"
  OPENWEATHER_API_KEY: "${OPENWEATHER_API_KEY}"
  GOOGLE_MAPS_API_KEY: "${GOOGLE_MAPS_API_KEY}"
  RESEND_API_KEY: "${RESEND_API_KEY}"
  EMAIL_FROM: "${EMAIL_FROM}"
```

- [ ] **Step 3: postgres-secret.yaml を作成する**

```yaml
# k8s/config/postgres-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: fishing
type: Opaque
stringData:
  DATABASE_URL: "${DATABASE_URL}"
```

- [ ] **Step 4: redis-secret.yaml を作成する**

```yaml
# k8s/config/redis-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: redis-secret
  namespace: fishing
type: Opaque
stringData:
  REDIS_URL: "${REDIS_URL}"
```

- [ ] **Step 5: .gitignore にシークレットの実値ファイルを追加する**

```bash
echo "k8s/config/*.yaml.local" >> .gitignore
```

- [ ] **Step 6: コミット**

```bash
git add k8s/namespace.yaml k8s/config/
git commit -m "feat: k8sネームスペース・シークレットテンプレート追加"
```

---

## Task 3: fishing-apiのk8sマニフェスト

**Files:**
- Create: `k8s/fishing-api/deployment.yaml`
- Create: `k8s/fishing-api/service.yaml`
- Create: `k8s/fishing-api/ingress.yaml`

- [ ] **Step 1: deployment.yaml を作成する**

```yaml
# k8s/fishing-api/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fishing-api
  namespace: fishing
spec:
  replicas: 1
  selector:
    matchLabels:
      app: fishing-api
  template:
    metadata:
      labels:
        app: fishing-api
    spec:
      containers:
        - name: fishing-api
          image: ghcr.io/kazumadev619-dev/fishing-api:latest
          ports:
            - containerPort: 8080
          env:
            - name: PORT
              value: "8080"
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: DATABASE_URL
            - name: REDIS_URL
              valueFrom:
                secretKeyRef:
                  name: redis-secret
                  key: REDIS_URL
            - name: JWT_ACCESS_SECRET
              valueFrom:
                secretKeyRef:
                  name: fishing-api-secret
                  key: JWT_ACCESS_SECRET
            - name: JWT_REFRESH_SECRET
              valueFrom:
                secretKeyRef:
                  name: fishing-api-secret
                  key: JWT_REFRESH_SECRET
            - name: OPENWEATHER_API_KEY
              valueFrom:
                secretKeyRef:
                  name: fishing-api-secret
                  key: OPENWEATHER_API_KEY
            - name: GOOGLE_MAPS_API_KEY
              valueFrom:
                secretKeyRef:
                  name: fishing-api-secret
                  key: GOOGLE_MAPS_API_KEY
            - name: RESEND_API_KEY
              valueFrom:
                secretKeyRef:
                  name: fishing-api-secret
                  key: RESEND_API_KEY
            - name: EMAIL_FROM
              valueFrom:
                secretKeyRef:
                  name: fishing-api-secret
                  key: EMAIL_FROM
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 15
            periodSeconds: 20
          resources:
            requests:
              memory: "64Mi"
              cpu: "50m"
            limits:
              memory: "256Mi"
              cpu: "200m"
```

- [ ] **Step 2: service.yaml を作成する**

```yaml
# k8s/fishing-api/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: fishing-api
  namespace: fishing
spec:
  selector:
    app: fishing-api
  ports:
    - port: 80
      targetPort: 8080
  type: ClusterIP
```

- [ ] **Step 3: ingress.yaml を作成する（Traefik用）**

```yaml
# k8s/fishing-api/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: fishing-api-ingress
  namespace: fishing
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web,websecure
spec:
  rules:
    - host: fishing.kazuma-lab.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: fishing-api
                port:
                  number: 80
```

- [ ] **Step 4: コミット**

```bash
git add k8s/fishing-api/
git commit -m "feat: fishing-api k8sマニフェスト追加（Deployment・Service・Ingress）"
```

---

## Task 4: フロントエンド・Cloudflared k8sマニフェスト

**Files:**
- Create: `k8s/frontend/deployment.yaml`
- Create: `k8s/frontend/service.yaml`
- Create: `k8s/frontend/ingress.yaml`
- Create: `k8s/cloudflared/deployment.yaml`

- [ ] **Step 1: frontend deployment.yaml を作成する**

```yaml
# k8s/frontend/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: fishing
spec:
  replicas: 1
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
        - name: frontend
          image: ghcr.io/kazumadev619-dev/fishing-conditions-app:latest
          ports:
            - containerPort: 3000
          readinessProbe:
            httpGet:
              path: /
              port: 3000
            initialDelaySeconds: 10
            periodSeconds: 10
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
```

- [ ] **Step 2: frontend service.yaml を作成する**

```yaml
# k8s/frontend/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: frontend
  namespace: fishing
spec:
  selector:
    app: frontend
  ports:
    - port: 80
      targetPort: 3000
  type: ClusterIP
```

- [ ] **Step 3: frontend ingress.yaml を作成する**

```yaml
# k8s/frontend/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: frontend-ingress
  namespace: fishing
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web,websecure
spec:
  rules:
    - host: fishing.kazuma-lab.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 80
```

- [ ] **Step 4: cloudflared deployment.yaml を作成する**

```yaml
# k8s/cloudflared/deployment.yaml
# NOTE: CLOUDFLARE_TUNNEL_TOKEN はGitHub Secretsで管理し、
# kubectl create secret で事前に作成しておくこと。
# kubectl create secret generic cloudflare-tunnel-secret \
#   --from-literal=TUNNEL_TOKEN=<your-token> -n fishing
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cloudflared
  namespace: fishing
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cloudflared
  template:
    metadata:
      labels:
        app: cloudflared
    spec:
      containers:
        - name: cloudflared
          image: cloudflare/cloudflared:latest
          args:
            - tunnel
            - --no-autoupdate
            - run
          env:
            - name: TUNNEL_TOKEN
              valueFrom:
                secretKeyRef:
                  name: cloudflare-tunnel-secret
                  key: TUNNEL_TOKEN
          resources:
            requests:
              memory: "32Mi"
              cpu: "20m"
            limits:
              memory: "128Mi"
              cpu: "100m"
```

- [ ] **Step 5: コミット**

```bash
git add k8s/frontend/ k8s/cloudflared/
git commit -m "feat: フロントエンド・cloudflared k8sマニフェスト追加"
```

---

## Task 5: GitHub Actions CI（Lint・Test・Build）

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: ci.yml を作成する**

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
    services:
      postgres:
        image: postgres:17-alpine
        env:
          POSTGRES_DB: fishing_test
          POSTGRES_USER: fishing
          POSTGRES_PASSWORD: fishing_password
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Install sqlc
        run: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

      - name: Generate sqlc code
        run: sqlc generate

      - name: Run tests
        env:
          DATABASE_URL: postgres://fishing:fishing_password@localhost:5432/fishing_test
          REDIS_URL: redis://localhost:6379
          JWT_ACCESS_SECRET: test-access-secret-32chars-minimum
          JWT_REFRESH_SECRET: test-refresh-secret-32chars-minimum
        run: go test ./... -v -count=1

      - name: Build
        run: go build ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
```

- [ ] **Step 2: .golangci.yml が存在することを確認する**

```bash
ls .golangci.yml
```

Expected: `.golangci.yml` が存在する（Phase 0 Task 5 で作成済み）

存在しない場合は Phase 0 Task 5 を先に実施すること。CI では `.golangci.yml` の full 設定（gofumpt, gocritic, bodyclose 等を含む）が使用される。

- [ ] **Step 3: コミット**

```bash
git add .github/workflows/ci.yml .golangci.yml
git commit -m "feat: GitHub Actions CIワークフロー追加"
```

---

## Task 6: GitHub Actions デプロイ（GHCR → k3s）

**Files:**
- Create: `.github/workflows/deploy.yml`

- [ ] **Step 1: GitHub Secretsに以下を登録する（GitHubリポジトリ設定から）**

```
# 登録が必要なSecrets:
CLOUDFLARE_TUNNEL_TOKEN    # Cloudflare Access用トークン（k3s APIアクセス用）
K3S_API_URL                # k3s APIサーバーURL（Cloudflare Access経由）
K3S_KUBECONFIG             # kubeconfigの内容（base64エンコード）
DATABASE_URL               # 本番DB接続文字列
REDIS_URL                  # 本番Redis URL
JWT_ACCESS_SECRET          # JWT秘密鍵
JWT_REFRESH_SECRET         # JWTリフレッシュ秘密鍵
OPENWEATHER_API_KEY        # OpenWeatherMap APIキー
GOOGLE_MAPS_API_KEY        # Google Maps APIキー
RESEND_API_KEY             # Resend APIキー
EMAIL_FROM                 # 送信元メールアドレス
```

- [ ] **Step 2: deploy.yml を作成する**

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository_owner }}/fishing-api

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    outputs:
      image-tag: ${{ steps.meta.outputs.tags }}

    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=sha,prefix=sha-
            type=raw,value=latest,enable=${{ github.ref == 'refs/heads/main' }}

      - name: Build and push (ARM64)
        uses: docker/build-push-action@v6
        with:
          context: .
          platforms: linux/arm64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  deploy:
    needs: build-and-push
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Install cloudflared
        run: |
          curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb -o cloudflared.deb
          sudo dpkg -i cloudflared.deb

      - name: Install kubectl
        uses: azure/setup-kubectl@v4

      - name: Configure kubeconfig via Cloudflare Tunnel
        run: |
          echo "${{ secrets.K3S_KUBECONFIG }}" | base64 -d > kubeconfig.yaml
          # Cloudflare Access経由でk3s APIサーバーに接続
          cloudflared access kubeconfig --hostname ${{ secrets.K3S_API_URL }} >> kubeconfig.yaml
        env:
          CLOUDFLARE_TOKEN: ${{ secrets.CLOUDFLARE_TUNNEL_TOKEN }}

      - name: Apply Secrets
        env:
          KUBECONFIG: kubeconfig.yaml
          DATABASE_URL: ${{ secrets.DATABASE_URL }}
          REDIS_URL: ${{ secrets.REDIS_URL }}
          JWT_ACCESS_SECRET: ${{ secrets.JWT_ACCESS_SECRET }}
          JWT_REFRESH_SECRET: ${{ secrets.JWT_REFRESH_SECRET }}
          OPENWEATHER_API_KEY: ${{ secrets.OPENWEATHER_API_KEY }}
          GOOGLE_MAPS_API_KEY: ${{ secrets.GOOGLE_MAPS_API_KEY }}
          RESEND_API_KEY: ${{ secrets.RESEND_API_KEY }}
          EMAIL_FROM: ${{ secrets.EMAIL_FROM }}
        run: |
          kubectl apply -f k8s/namespace.yaml
          # envsubstでシークレットテンプレートを置換してapply
          envsubst < k8s/config/fishing-api-secret.yaml | kubectl apply -f -
          envsubst < k8s/config/postgres-secret.yaml | kubectl apply -f -
          envsubst < k8s/config/redis-secret.yaml | kubectl apply -f -

      - name: Deploy to k3s
        env:
          KUBECONFIG: kubeconfig.yaml
        run: |
          kubectl apply -f k8s/fishing-api/
          # 新しいイメージに更新してローリングデプロイ
          kubectl set image deployment/fishing-api \
            fishing-api=ghcr.io/${{ github.repository_owner }}/fishing-api:sha-${{ github.sha }} \
            -n fishing
          # デプロイ完了を待機（タイムアウト5分）
          kubectl rollout status deployment/fishing-api -n fishing --timeout=300s

      - name: Verify deployment
        env:
          KUBECONFIG: kubeconfig.yaml
        run: |
          kubectl get pods -n fishing
          kubectl get ingress -n fishing
```

- [ ] **Step 3: コミット**

```bash
git add .github/workflows/deploy.yml
git commit -m "feat: GitHub Actions デプロイワークフロー追加（GHCR → k3s）"
```

---

## Task 7: DB schema.sql 自動同期ワークフロー

**Files:**
- Create: `.github/workflows/sync-schema.yml`

- [ ] **Step 1: sync-schema.yml を作成する**

```yaml
# .github/workflows/sync-schema.yml
name: Sync DB Schema

on:
  # DBリポジトリからrepository_dispatchで起動される
  repository_dispatch:
    types: [db-schema-updated]

  # 手動実行も可能
  workflow_dispatch:
    inputs:
      schema_content:
        description: "新しいschema.sqlの内容（base64）"
        required: false

jobs:
  sync-schema:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write

    steps:
      - uses: actions/checkout@v4
        with:
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install sqlc
        run: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

      - name: Update schema.sql from dispatch payload
        if: github.event_name == 'repository_dispatch'
        run: |
          echo "${{ github.event.client_payload.schema_base64 }}" | base64 -d > db/schema.sql

      - name: Regenerate sqlc code
        run: sqlc generate

      - name: Check for changes
        id: check-changes
        run: |
          git diff --exit-code db/ || echo "changes=true" >> $GITHUB_OUTPUT

      - name: Create Pull Request
        if: steps.check-changes.outputs.changes == 'true'
        uses: peter-evans/create-pull-request@v7
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          commit-message: "chore: DBスキーマ・sqlc生成コード自動更新"
          title: "chore: DBスキーマ自動同期 - ${{ github.run_number }}"
          body: |
            ## DBスキーマ自動同期

            DBリポジトリのスキーマ変更を検知し、以下を自動更新しました：
            - `db/schema.sql`
            - `db/generated/` (sqlc生成コード)

            **レビューポイント:**
            - [ ] スキーマ変更内容を確認
            - [ ] 既存クエリへの影響を確認
            - [ ] 必要に応じて `db/queries/` を更新
          branch: "chore/sync-schema-${{ github.run_number }}"
          base: main
```

- [ ] **Step 2: DBリポジトリ側のdispatch設定（メモ）**

DBリポジトリ（例: `Fishing-database`）のGitHub Actionsに以下を追加する：

```yaml
# Fishing-databaseリポジトリの .github/workflows/notify-api.yml（参考）
- name: Notify fishing-api of schema change
  run: |
    SCHEMA_BASE64=$(base64 -w0 prisma/migrations/combined_schema.sql)
    curl -X POST \
      -H "Authorization: token ${{ secrets.FISHING_API_DISPATCH_TOKEN }}" \
      -H "Accept: application/vnd.github.v3+json" \
      https://api.github.com/repos/kazumadev619-dev/fishing-api/dispatches \
      -d "{\"event_type\":\"db-schema-updated\",\"client_payload\":{\"schema_base64\":\"${SCHEMA_BASE64}\"}}"
```

- [ ] **Step 3: コミット**

```bash
git add .github/workflows/sync-schema.yml
git commit -m "feat: DBスキーマ自動同期ワークフロー追加"
```

---

## Task 8: 開発ドキュメント作成

**Files:**
- Create: `docs/development.md`

- [ ] **Step 1: development.md を作成する**

```markdown
# 開発セットアップガイド

## 前提条件

- Go 1.24+
- Docker + Docker Compose
- sqlc: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- golangci-lint: `brew install golangci-lint`

## ローカル開発の開始

### 1. リポジトリのクローン

```bash
git clone https://github.com/kazumadev619-dev/fishing-api.git
cd fishing-api
```

### 2. 依存関係インストール

```bash
go mod download
```

### 3. 環境変数設定

```bash
cp .env.example .env
# .envを編集して各APIキーを設定する
```

### 4. DBとRedisをDockerで起動

```bash
make docker-up
# または: docker-compose up -d
```

### 5. sqlcコード生成

```bash
make sqlc-gen
```

### 6. サーバー起動

```bash
make run
# 起動確認: curl http://localhost:8080/health
```

## よく使うコマンド

| コマンド | 説明 |
|---------|------|
| `make run` | 開発サーバー起動 |
| `make test` | 全テスト実行 |
| `make lint` | Lintチェック |
| `make sqlc-gen` | sqlcコード再生成 |
| `make docker-up` | DB・Redis起動 |
| `make docker-down` | DB・Redis停止 |
| `make build` | バイナリビルド |

## テスト実行

```bash
# 全テスト
make test

# 特定パッケージのみ
go test ./internal/usecase/... -v

# DBが必要な統合テスト（docker-up後に実行）
go test ./internal/infrastructure/db/... -v
```

## デプロイアーキテクチャ

```
[GitHub] push to main
    ↓
[GitHub Actions] CI（test + lint）
    ↓
[GitHub Actions] Docker buildx（linux/arm64）
    ↓
[GHCR] ghcr.io/kazumadev619-dev/fishing-api:sha-xxxxx
    ↓
[Cloudflare Tunnel] k3s API Serverに接続
    ↓
[kubectl] Raspberry Pi k3sにローリングデプロイ
```

## Raspberry Pi k3sへの手動デプロイ（緊急時）

```bash
# kubeconfigを設定
export KUBECONFIG=~/.kube/config-k3s

# Podの状態確認
kubectl get pods -n fishing

# ログ確認
kubectl logs -f deployment/fishing-api -n fishing

# 手動でイメージ更新
kubectl set image deployment/fishing-api \
  fishing-api=ghcr.io/kazumadev619-dev/fishing-api:latest \
  -n fishing

# ロールバック（1つ前のバージョンに戻す）
kubectl rollout undo deployment/fishing-api -n fishing
```
```

- [ ] **Step 2: コミット**

```bash
git add docs/development.md
git commit -m "docs: 開発セットアップガイド追加"
```

---

## Task 9: Raspberry Pi上のDB構成（Docker Compose、k3s外）

PostgreSQL・Redisはk3sのPod管理外で動かすのだ。Podはエフェメラルなため、DBをk8s StatefulSetで管理するのはベストプラクティスに反する。

**Files:**
- Create: `docker-compose.db.yml`

- [ ] **Step 1: Raspberry Pi上にDockerをインストールする**

```bash
# Raspberry Pi上で実行（Ubuntu 24.04）
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# 再ログイン後に確認
docker --version
```

Expected: `Docker version 27.x.x` 等が表示される

- [ ] **Step 2: docker-compose.db.yml を作成する**

このファイルはGitリポジトリで管理し、Raspberry Piにデプロイして使う。

```yaml
# docker-compose.db.yml
# Raspberry Pi上でk3s外として動かすDB専用Compose
# 起動: docker compose -f docker-compose.db.yml up -d

services:
  postgres:
    image: postgres:17-alpine
    container_name: fishing-postgres
    restart: always
    environment:
      POSTGRES_DB: fishing
      POSTGRES_USER: fishing
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U fishing"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: fishing-redis
    restart: always
    command: redis-server --requirepass ${REDIS_PASSWORD}
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
  redis_data:
```

- [ ] **Step 3: Raspberry Piに .env.db を作成する（Gitには入れない）**

```bash
# Raspberry Pi上で実行
cat > /opt/fishing-db/.env.db << 'EOF'
POSTGRES_PASSWORD=<強力なパスワードを設定>
REDIS_PASSWORD=<強力なパスワードを設定>
EOF
chmod 600 /opt/fishing-db/.env.db
```

- [ ] **Step 4: docker-compose.db.yml をRaspberry Piにコピーしてコンテナを起動する**

```bash
# ローカルマシンからコピー（sshでPiにコピー）
scp docker-compose.db.yml pi@<pi-ip>:/opt/fishing-db/

# Raspberry Pi上で起動
cd /opt/fishing-db
docker compose -f docker-compose.db.yml --env-file .env.db up -d

# 確認
docker compose -f docker-compose.db.yml ps
```

Expected: `fishing-postgres` と `fishing-redis` が `healthy` 状態

- [ ] **Step 5: systemdでDocker Composeを自動起動設定する（Pi再起動時にも自動復旧）**

```bash
# Raspberry Pi上で実行
sudo tee /etc/systemd/system/fishing-db.service << 'EOF'
[Unit]
Description=Fishing App Database Services
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/fishing-db
ExecStart=/usr/bin/docker compose -f docker-compose.db.yml --env-file .env.db up -d
ExecStop=/usr/bin/docker compose -f docker-compose.db.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable fishing-db
sudo systemctl start fishing-db

# 確認
sudo systemctl status fishing-db
```

Expected: `Active: active (exited)` が表示される

- [ ] **Step 6: Raspberry PiのプライベートIPを確認してk3s Podからの接続URLを設定する**

```bash
# Raspberry Pi上で実行
ip addr show | grep "inet " | grep -v 127.0.0.1
```

確認したIPアドレス（例: `192.168.1.100`）を使って、GitHub Secretsに登録するURLを構成する：

```
DATABASE_URL = postgres://fishing:<POSTGRES_PASSWORD>@192.168.1.100:5432/fishing
REDIS_URL    = redis://:<REDIS_PASSWORD>@192.168.1.100:6379
```

**注意:** k3s Podはホストネットワーク経由でこのIPにアクセスできる。`localhost` は不可（Pod内から見ると自分自身を指すため）。

- [ ] **Step 7: DBスキーマを適用する（初回のみ）**

```bash
# Phase 1で作成した db/schema.sql をRaspberry Piにコピーして適用する
scp db/schema.sql pi@<pi-ip>:/tmp/schema.sql

# Raspberry Pi上で実行
docker exec -i fishing-postgres psql -U fishing -d fishing < /tmp/schema.sql
```

Expected: スキーマが正常に適用される

- [ ] **Step 8: コミット**

```bash
git add docker-compose.db.yml
git commit -m "feat: Raspberry Pi用DB専用Docker Compose追加（k3s外運用）"
```

---

## Task 10: k3sへの初回デプロイ

ここからはRaspberry Pi上での作業なのだ。

- [ ] **Step 1: Raspberry Pi上でk3sをインストールする**

```bash
# Raspberry Pi上で実行
curl -sfL https://get.k3s.io | sh -
# インストール確認
kubectl get nodes
```

Expected: ノードが `Ready` 状態

- [ ] **Step 2: kubeconfigをローカルマシンにコピーする**

```bash
# Raspberry Pi上で実行
sudo cat /etc/rancher/k3s/k3s.yaml
# 表示されたyamlをローカルの ~/.kube/config-k3s にコピーし、
# serverのIPアドレスをRaspberry PiのIPに変更する
```

- [ ] **Step 3: Cloudflare DashboardでTunnelを作成する**

```
1. Cloudflare Dashboard → Zero Trust → Networks → Tunnels
2. 「Create a tunnel」→ cloudflaredを選択
3. トークンをコピー（GitHub SecretのCLOUDFLARE_TUNNEL_TOKENに設定）
4. Public Hostname を設定:
   - fishing.kazuma-lab.com → http://traefik.fishing.svc.cluster.local:80
   - k3s-api.kazuma-lab.com → https://localhost:6443 （CI/CD用）
```

- [ ] **Step 4: Raspberry Pi上でcloudflaredをk3sにデプロイする**

```bash
# kubectl create secret でトークンを登録
kubectl create namespace fishing
kubectl create secret generic cloudflare-tunnel-secret \
  --from-literal=TUNNEL_TOKEN=<your-tunnel-token> \
  -n fishing

# cloudflaredをデプロイ
kubectl apply -f k8s/cloudflared/deployment.yaml

# 確認
kubectl get pods -n fishing
```

Expected: `cloudflared` Podが `Running` 状態

- [ ] **Step 5: GitHub Secretsに全シークレットを登録する**

```
GitHubリポジトリ → Settings → Secrets and variables → Actions で以下を登録：

CLOUDFLARE_TUNNEL_TOKEN  = Cloudflare AccessトークンCLOUDFLARE TOKEN
K3S_API_URL              = k3s-api.kazuma-lab.com
K3S_KUBECONFIG           = base64エンコードしたkubeconfig
DATABASE_URL             = 本番DBのURL
REDIS_URL                = 本番RedisのURL
JWT_ACCESS_SECRET        = 32文字以上のランダム文字列
JWT_REFRESH_SECRET       = 32文字以上のランダム文字列
OPENWEATHER_API_KEY      = OpenWeatherMap APIキー
GOOGLE_MAPS_API_KEY      = Google Maps APIキー
RESEND_API_KEY           = Resend APIキー
EMAIL_FROM               = noreply@kazuma-lab.com
```

- [ ] **Step 6: mainにpushしてデプロイをトリガーする**

```bash
git push origin main
```

GitHub Actions → Deploy ワークフローを確認する。

Expected: 
1. CIが全テストPASS
2. ARM64イメージがGHCRにpushされる
3. k3sにデプロイされる
4. `kubectl get pods -n fishing` で全PodがRunning

- [ ] **Step 7: 本番動作確認**

```bash
curl https://fishing.kazuma-lab.com/health
```

Expected:
```json
{"status":"ok"}
```

```bash
curl "https://fishing.kazuma-lab.com/api/weather?lat=35.6895&lon=139.6917&type=current"
```

Expected: 天気データのJSONレスポンス

- [ ] **Step 8: 最終コミット**

```bash
git add .
git commit -m "feat: Phase 4完了 - k3s・Cloudflare Tunnel・CI/CDデプロイ構成"
git push origin main
```

---

## 完了条件チェックリスト

- [ ] `docker buildx build --platform linux/arm64` でビルドが通る
- [ ] GitHub Actions CIが全テストPASS
- [ ] mainへのpushでGHCRにARM64イメージがpushされる
- [ ] Raspberry Pi上で `fishing-postgres` と `fishing-redis` が `healthy` 状態
- [ ] `sudo systemctl status fishing-db` が `active (exited)` 状態（Pi再起動後も自動復旧）
- [ ] k3s上で全Podが `Running` 状態
- [ ] `https://fishing.kazuma-lab.com/health` が `{"status":"ok"}` を返す
- [ ] `https://fishing.kazuma-lab.com/api/weather` が天気データを返す
- [ ] DBスキーマ変更時にsync-schema.ymlが自動でPRを作成する
