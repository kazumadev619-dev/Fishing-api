# Go Backend Phase 0: リポジトリ整備 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Goバックエンド実装を始める前にリポジトリの骨格を整備する（README・.gitignore・エディタ設定・lint設定・GitHub テンプレート）。

**Architecture:** コード実装ゼロのリポジトリ整備フェーズ。設定ファイルとドキュメントのみを追加する。Phase 1〜4 の実装時に開発者が迷わないよう、規約・コマンド・アーキテクチャ概要をすべてここに集約する。

**Tech Stack:** gofumpt, golangci-lint, Lefthook, markdownlint, cspell, editorconfig, GitHub Templates

---

## ファイル構成

| 操作 | ファイル | 内容 |
|------|---------|------|
| 更新 | `README.md` | プロジェクト概要・アーキテクチャ・コマンド一覧 |
| 新規作成 | `.gitignore` | Go標準 + .env + bin/ + k8s secrets |
| 新規作成 | `.editorconfig` | tab indent・LF改行（Go標準） |
| 新規作成 | `.markdownlint.json` | Markdown lint 設定 |
| 新規作成 | `cspell.json` | Go/k8s 固有ワード許可リスト |
| 新規作成 | `.golangci.yml` | golangci-lint 設定（fast subset + full） |
| 新規作成 | `lefthook.yml` | pre-commit フック（golangci-lint --fix） |
| 更新 | `.claude/settings.json` | PostToolUse（gofumpt + 高速lint）+ PreToolUse（設定保護） |
| 新規作成 | `.github/PULL_REQUEST_TEMPLATE.md` | PRテンプレート |
| 新規作成 | `.github/ISSUE_TEMPLATE/bug_report.md` | バグ報告テンプレート |
| 新規作成 | `.github/ISSUE_TEMPLATE/feature_request.md` | 機能要望テンプレート |
| 新規作成 | `CONTRIBUTING.md` | 開発フロー・コミット規約・Phase構成 |

---

## Task 1: .gitignore 作成

**Files:**
- Create: `.gitignore`

- [ ] **Step 1: .gitignore を作成する**

```
# .gitignore

# -----------------------------------------------
# Go
# -----------------------------------------------
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
vendor/

# -----------------------------------------------
# 環境変数・シークレット
# -----------------------------------------------
.env
.env.*
!.env.example

# -----------------------------------------------
# k8s シークレット実値ファイル（テンプレートはGit管理対象）
# -----------------------------------------------
k8s/config/*.yaml.local

# -----------------------------------------------
# Raspberry Pi DB 設定（本番シークレット）
# -----------------------------------------------
.env.db

# -----------------------------------------------
# ビルド成果物
# -----------------------------------------------
dist/
build/

# -----------------------------------------------
# IDE / エディタ
# -----------------------------------------------
.idea/
.vscode/
*.swp
*.swo
.DS_Store

# -----------------------------------------------
# テスト・カバレッジ
# -----------------------------------------------
coverage.out
coverage.html

# -----------------------------------------------
# kubeconfig ローカルコピー
# -----------------------------------------------
kubeconfig.yaml
kubeconfig*.yaml
```

- [ ] **Step 2: .gitignore が正しく機能することを確認する**

```bash
cd /Users/nosawakazuma/Project/Fishing-api
git status
```

Expected: `.env` があれば untracked に出ないこと（すでに .gitignore 対象であることを確認）

- [ ] **Step 3: コミット**

```bash
git add .gitignore
git commit -m "chore: .gitignore 追加"
```

---

## Task 2: .editorconfig 作成

**Files:**
- Create: `.editorconfig`

- [ ] **Step 1: .editorconfig を作成する**

```ini
# .editorconfig
root = true

[*]
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
charset = utf-8

# Go はタブインデント（gofmt 標準）
[*.go]
indent_style = tab
indent_size = 4

# YAML・JSON はスペース2つ
[*.{yaml,yml,json}]
indent_style = space
indent_size = 2

# Markdown はトレイリングスペース保持（改行に使う場合がある）
[*.md]
trim_trailing_whitespace = false
indent_style = space
indent_size = 2

# Dockerfile はスペース4つ
[Dockerfile]
indent_style = space
indent_size = 4

# Makefile はタブ必須
[Makefile]
indent_style = tab
```

- [ ] **Step 2: コミット**

```bash
git add .editorconfig
git commit -m "chore: .editorconfig 追加（Go tab indent・LF改行）"
```

---

## Task 3: cspell.json 作成（診断エラー解消）

**Files:**
- Create: `cspell.json`

このタスクで `docs/` 内の cSpell 診断エラー（sqlc, usecase, cloudflared 等）を解消するのだ。

- [ ] **Step 1: cspell.json を作成する**

```json
{
  "$schema": "https://raw.githubusercontent.com/streetsidesoftware/cspell/main/packages/cspell-types/cspell.schema.json",
  "version": "0.2",
  "language": "en,ja",
  "ignorePaths": [
    "vendor/**",
    "bin/**",
    "*.sum",
    "node_modules/**"
  ],
  "words": [
    "sqlc",
    "sqlcgen",
    "usecase",
    "usecases",
    "cloudflared",
    "kazuma",
    "kazumadev",
    "buildx",
    "httptest",
    "pgx",
    "txdb",
    "bcrypt",
    "godotenv",
    "golangci",
    "traefik",
    "ghcr",
    "kubeconfig",
    "envsubst",
    "resend",
    "openweather",
    "openweathermap",
    "ingress",
    "namespace",
    "clusterip",
    "kubectl",
    "pullpolicy",
    "livenessProbe",
    "readinessProbe",
    "configmap",
    "statefulset",
    "pgsql",
    "postgres",
    "postgresql",
    "multiarch",
    "BUILDPLATFORM",
    "TARGETOS",
    "TARGETARCH",
    "stripprefix",
    "entrypoints",
    "websecure",
    "ldflags",
    "ginctx",
    "middlewares",
    "pathtype",
    "secretkeyref",
    "fieldref",
    "valueFrom",
    "stringdata"
  ],
  "overrides": [
    {
      "filename": "**/*.go",
      "words": [
        "struct",
        "func",
        "chan",
        "goroutine",
        "goroutines",
        "pkgerrors"
      ]
    }
  ]
}
```

- [ ] **Step 2: 診断エラーが減ったことを確認する**

VSCode の PROBLEMS タブ、または以下のコマンドで確認：

```bash
# cspell CLI がインストール済みの場合
npx cspell "docs/**/*.md" --no-progress 2>&1 | grep -c "Unknown word" || true
```

Expected: `docs/superpowers/specs/2026-04-07-go-backend-design.md` のエラーが 0 になる

- [ ] **Step 3: コミット**

```bash
git add cspell.json
git commit -m "chore: cspell.json 追加（Go/k8s固有ワード許可リスト）"
```

---

## Task 4: .markdownlint.json 作成

**Files:**
- Create: `.markdownlint.json`

- [ ] **Step 1: .markdownlint.json を作成する**

```json
{
  "default": true,
  "MD013": false,
  "MD033": false,
  "MD041": false,
  "MD024": {
    "siblings_only": true
  },
  "MD007": {
    "indent": 2
  }
}
```

各ルールの意味：
- `MD013: false` — 1行の長さ制限を無効化（日本語混在のため）
- `MD033: false` — インラインHTML許可（バッジ等で使用する場合）
- `MD041: false` — ファイル先頭のH1強制を無効化
- `MD024: siblings_only` — 同一見出し名を兄弟レベルのみ禁止（異なるセクションでの重複は許可）
- `MD007: indent: 2` — リストのインデントを2スペースに統一

- [ ] **Step 2: コミット**

```bash
git add .markdownlint.json
git commit -m "chore: .markdownlint.json 追加"
```

---

## Task 5: .golangci.yml 作成（fast + full 2段構成）

**Files:**
- Create: `.golangci.yml`

golangci-lint を2段構成で使うのだ。PostToolUse フックでは `--fast` オプションで高速サブセットのみ実行し、pre-commit・CI ではフル設定で全解析する。

- [ ] **Step 1: .golangci.yml を作成する**

```yaml
# .golangci.yml
version: "2"

linters:
  # デフォルト有効linterはそのままにする
  default: standard

  enable:
    # --- fast subset（PostToolUse・pre-commitで使用） ---
    - errcheck        # エラー無視の検出
    - govet           # go vet 相当
    - staticcheck     # 高度な静的解析
    - unused          # 未使用コード
    - gosimple        # 簡略化できるコードの検出
    - ineffassign     # 無効な代入の検出
    # --- full（CI のみで実行） ---
    - gofumpt         # gofumpt フォーマットチェック
    - goimports       # import 順序チェック
    - gocritic        # 追加静的解析
    - noctx           # context なし HTTP リクエスト検出
    - bodyclose       # response body の close 漏れ検出
    - nilerr          # nil エラー返却の検出
    - exhaustive      # switch 文の網羅性チェック
    - wrapcheck       # エラーラップ漏れ検出（外部パッケージ呼び出し時）

linters-settings:
  gofumpt:
    module-path: github.com/kazumadev619-dev/fishing-api
  exhaustive:
    default-signifies-exhaustive: true

issues:
  exclude-rules:
    # テストコードは errcheck・wrapcheck を緩和
    - path: _test\.go
      linters:
        - errcheck
        - wrapcheck
    # 生成コードは除外
    - path: db/generated/
      linters:
        - all
```

- [ ] **Step 2: fast モードで動作確認する（Go モジュール初期化後に実行）**

```bash
# NOTE: go.mod が存在しない場合はスキップ。Phase 1 完了後に実行する。
# go.mod がある場合:
golangci-lint run --fast ./... 2>&1 | head -20 || true
```

Expected: エラーまたは "no go files" メッセージ（設定が構文エラーでないことを確認）

- [ ] **Step 3: コミット**

```bash
git add .golangci.yml
git commit -m "chore: .golangci.yml 追加（fast subset + full 2段構成）"
```

---

## Task 6: Lefthook 設定（pre-commit: golangci-lint --fix）

**Files:**
- Create: `lefthook.yml`

- [ ] **Step 1: lefthook.yml を作成する**

```yaml
# lefthook.yml
pre-commit:
  parallel: false
  commands:
    gofumpt:
      glob: "*.go"
      run: gofumpt -w {staged_files}
      stage_fixed: true

    golangci-lint:
      glob: "*.go"
      run: golangci-lint run --fix --fast {staged_files}
      stage_fixed: true
```

- [ ] **Step 2: Lefthook をインストールする（ローカル開発環境）**

```bash
# macOS
brew install lefthook

# インストール確認
lefthook --version
```

Expected: `lefthook version X.X.X`

- [ ] **Step 3: Git フックをインストールする**

```bash
cd /Users/nosawakazuma/Project/Fishing-api
lefthook install
```

Expected: `.git/hooks/pre-commit` が作成される

- [ ] **Step 4: コミット**

```bash
git add lefthook.yml
git commit -m "chore: lefthook.yml 追加（pre-commit: gofumpt + golangci-lint --fix）"
```

---

## Task 7: .claude/settings.json 更新（PostToolUse + PreToolUse フック）

**Files:**
- Modify: `.claude/settings.json`

3層リント構成の PostToolUse フック（gofumpt + fast golangci-lint）と、リンター設定保護の PreToolUse フックを追加するのだ。

**保護対象ファイル:** `.golangci.yml` / `cspell.json` / `.markdownlint.json`

エージェントがリンターエラーを回避するためにリンター設定を書き換えることを防ぐのだ。

- [ ] **Step 1: .claude/settings.json を更新する**

```json
{
  "env": {
    "Max_THINKING_TOKENS": "10000"
  },
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "bash -c 'FILE=$(cat | jq -r \".file_path // .path // empty\" 2>/dev/null); if [ -n \"$FILE\" ] && echo \"$FILE\" | grep -q \"\\.go$\"; then gofumpt -w \"$FILE\" 2>/dev/null || true; golangci-lint run --fast --fix \"$FILE\" 2>/dev/null || true; fi'"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Write|Edit|MultiEdit",
        "hooks": [
          {
            "type": "command",
            "command": "bash -c 'FILE=$(cat | jq -r \".file_path // .path // empty\" 2>/dev/null); if echo \"$FILE\" | grep -qE \"(\\.golangci\\.ya?ml|cspell\\.json|\\.markdownlint\\.json)$\"; then echo \"BLOCKED: リンター設定ファイルは直接編集禁止。コードを修正してリンターエラーを解消してください。\"; exit 2; fi'"
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 2: 設定ファイルを保存して動作確認する（任意の .go ファイルを編集してフックが動くか確認）**

```bash
# .go ファイルが存在する場合（Phase 1 以降）に確認
# echo "package main" > /tmp/test_hook.go
# Claude Code 上で /tmp/test_hook.go を編集し、
# gofumpt と golangci-lint が自動実行されることを確認する
```

- [ ] **Step 3: PreToolUse 保護フックの動作を確認する（メモ）**

`.golangci.yml` を Claude Code から直接編集しようとすると以下のメッセージでブロックされることを確認する：

```
BLOCKED: リンター設定ファイルは直接編集禁止。コードを修正してリンターエラーを解消してください。
```

- [ ] **Step 4: コミット**

```bash
git add .claude/settings.json
git commit -m "chore: Claude Code フック追加（PostToolUse gofumpt/lint + PreToolUse 設定保護）"
```

---

## Task 8: GitHub テンプレート作成

**Files:**
- Create: `.github/PULL_REQUEST_TEMPLATE.md`
- Create: `.github/ISSUE_TEMPLATE/bug_report.md`
- Create: `.github/ISSUE_TEMPLATE/feature_request.md`

- [ ] **Step 1: .github ディレクトリ構造を確認する**

```bash
ls -la /Users/nosawakazuma/Project/Fishing-api/.github/ 2>/dev/null || echo "なし"
```

- [ ] **Step 2: PULL_REQUEST_TEMPLATE.md を作成する**

```markdown
## 概要

<!-- このPRで何を変更したか、1〜3行で説明してください -->

## 変更内容

- 

## テスト方法

- [ ] `make test` が全PASS
- [ ] `make lint` がエラーなし
- [ ] ローカルで動作確認済み（`curl http://localhost:8080/health`）

## 関連 Issue

Closes #

## チェックリスト

- [ ] コードレビューの準備ができている
- [ ] ドキュメント更新が必要な場合は更新した
- [ ] `.env.example` に変更が必要な場合は更新した
- [ ] DB スキーマ変更がある場合は `db/schema.sql` を更新した
```

- [ ] **Step 3: bug_report.md を作成する**

```markdown
---
name: バグ報告
about: 不具合を報告してください
title: 'fix: '
labels: bug
assignees: ''
---

## バグの概要

<!-- 何が起きているか簡潔に説明してください -->

## 再現手順

1. 
2. 
3. 

## 期待される動作

<!-- 本来どうあるべきか -->

## 実際の動作

<!-- 実際に何が起きているか -->

## 環境

- OS:
- Go バージョン: `go version`
- ブランチ:

## ログ・エラー出力

```
（ここに貼り付け）
```
```

- [ ] **Step 4: feature_request.md を作成する**

```markdown
---
name: 機能要望
about: 新しい機能を提案してください
title: 'feat: '
labels: enhancement
assignees: ''
---

## 概要

<!-- 何を実現したいか -->

## 動機・背景

<!-- なぜこの機能が必要か -->

## 提案する実装方針

<!-- どう実現するか（任意） -->

## 代替案

<!-- 検討した他のアプローチ（任意） -->
```

- [ ] **Step 5: コミット**

```bash
git add .github/
git commit -m "chore: GitHub PR・Issueテンプレート追加"
```

---

## Task 9: CONTRIBUTING.md 作成

**Files:**
- Create: `CONTRIBUTING.md`

- [ ] **Step 1: CONTRIBUTING.md を作成する**

```markdown
# Contributing Guide

## 開発フロー

このプロジェクトは以下の Phase 構成で実装を進めているのだ。

| Phase | 内容 | ドキュメント |
|-------|------|-------------|
| Phase 0 | リポジトリ整備（本フェーズ） | `docs/superpowers/plans/2026-04-07-go-backend-phase0-repo-setup.md` |
| Phase 1 | 基盤構築（DB・Redis・ヘルスチェック） | `docs/superpowers/plans/2026-04-07-go-backend-phase1-foundation.md` |
| Phase 2 | 認証（JWT + Google OAuth） | `docs/superpowers/plans/2026-04-07-go-backend-phase2-auth.md` |
| Phase 3 | コアAPI（天気・潮汐・スコア・お気に入り） | `docs/superpowers/plans/2026-04-07-go-backend-phase3-core-apis.md` |
| Phase 4 | デプロイ（k3s + Cloudflare Tunnel） | `docs/superpowers/plans/2026-04-07-go-backend-phase4-deployment.md` |

## ブランチ戦略

```
main          本番ブランチ（直接pushは禁止）
feature/*     機能開発ブランチ
fix/*         バグ修正ブランチ
chore/*       設定・ドキュメント変更
```

## コミットメッセージ規約

```
<type>: <description>

<optional body>
```

**type 一覧:**

| type | 用途 |
|------|------|
| `feat` | 新機能追加 |
| `fix` | バグ修正 |
| `refactor` | リファクタリング（動作変更なし） |
| `docs` | ドキュメント変更 |
| `test` | テスト追加・修正 |
| `chore` | 設定・ビルド・依存関係変更 |
| `perf` | パフォーマンス改善 |
| `ci` | CI/CD 設定変更 |

**例:**
```
feat: 天気APIエンドポイント追加（GET /api/weather）

OpenWeatherMap APIを呼び出し、Redisに30分キャッシュする実装。
```

## 開発環境セットアップ

### 前提条件

- Go 1.26+
- Docker + Docker Compose
- make
- sqlc: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- golangci-lint: `brew install golangci-lint`
- gofumpt: `go install mvdan.cc/gofumpt@latest`
- lefthook: `brew install lefthook`

### 手順

```bash
# 1. リポジトリクローン
git clone https://github.com/kazumadev619-dev/fishing-api.git
cd fishing-api

# 2. 依存関係インストール
go mod download

# 3. 環境変数設定
cp .env.example .env
# .env を編集して各 API キーを設定

# 4. Redis 起動（DB は Neon のため docker-compose 不要）
make redis-up

# 5. sqlc コード生成
make sqlc-gen

# 6. サーバー起動
make run

# 動作確認
curl http://localhost:8080/health
```

## よく使うコマンド

```bash
make run        # 開発サーバー起動
make test       # 全テスト実行
make lint       # Lint チェック（golangci-lint）
make sqlc-gen   # sqlc コード再生成
make redis-up   # Redis 起動（DB は Neon を使用）
make redis-down # Redis 停止
make build      # バイナリビルド（./bin/server）
```

## アーキテクチャ概要

クリーンアーキテクチャを採用しているのだ。依存関係は内側（domain）に向かう。

```
cmd/server/          エントリポイント（main.go）
config/              設定読み込み（環境変数）
domain/              ドメイン層（エンティティ・リポジトリIF・ドメインエラー）
usecase/             ユースケース層（ビジネスロジック）
infrastructure/      インフラ層（DB・外部API・キャッシュの実装）
  db/                sqlc 生成コード + リポジトリ実装
  external/          外部 API クライアント（天気・潮汐・Maps）
  email/             メール送信（Resend）
interface/           インターフェース層（HTTP ハンドラー・ミドルウェア）
  handler/           Gin ハンドラー
  middleware/        JWT 認証ミドルウェア
  router/            ルーティング定義
pkg/                 共通パッケージ（JWT・バリデーター）
db/                  スキーマ・sqlc クエリ・生成コード
k8s/                 Kubernetes マニフェスト（Traefik Ingress）
```

## PR のルール

1. `main` への直接 push 禁止
2. PR には `PULL_REQUEST_TEMPLATE.md` に従って記載する
3. `make test` と `make lint` が通っていること
4. セキュリティ関連の変更は security-reviewer エージェントでレビューすること
5. CRITICAL・HIGH 指摘がある場合はマージ不可

## 質問・相談

Issue を立てるか、PR のコメントで相談してくださいなのだ。
```

- [ ] **Step 2: コミット**

```bash
git add CONTRIBUTING.md
git commit -m "docs: CONTRIBUTING.md 追加（開発フロー・コミット規約・アーキテクチャ概要）"
```

---

## Task 10: README.md 更新

**Files:**
- Modify: `README.md`

- [ ] **Step 1: README.md を書き直す**

```markdown
# Fishing-api

[FishingConditionsApp](https://github.com/kazumadev619-dev/FishingConditionsApp) のGoバックエンド。
釣り条件（天気・潮汐・スコア）を提供するREST API。

## 概要

| 項目 | 内容 |
|------|------|
| 言語 | Go 1.26 |
| フレームワーク | Gin v1.10 |
| DB | PostgreSQL（Neon マネージドクラウド） |
| キャッシュ | Redis 7（Docker on Raspberry Pi） |
| デプロイ先 | Raspberry Pi 5 + k3s + Cloudflare Tunnel |
| ドメイン | `fishing.kazuma-lab.com` |

## API エンドポイント

| メソッド | パス | 説明 | 認証 |
|---------|------|------|------|
| GET | `/health` | ヘルスチェック | 不要 |
| POST | `/api/auth/register` | ユーザー登録 | 不要 |
| POST | `/api/auth/login` | ログイン | 不要 |
| POST | `/api/auth/refresh` | トークンリフレッシュ | 不要 |
| GET | `/api/auth/verify-email` | メール認証 | 不要 |
| GET | `/api/weather` | 天気情報取得 | 不要 |
| GET | `/api/tide` | 潮汐情報取得 | 不要 |
| GET | `/api/location/search` | 地点検索 | 不要 |
| GET | `/api/score` | 釣り条件スコア取得 | 不要 |
| GET | `/api/favorites` | お気に入り一覧 | JWT必要 |
| POST | `/api/favorites` | お気に入り追加 | JWT必要 |
| DELETE | `/api/favorites/:id` | お気に入り削除 | JWT必要 |

## アーキテクチャ

クリーンアーキテクチャ（依存は内側のみ）：

```
domain → usecase → infrastructure / interface
```

```
cmd/server/        エントリポイント
config/            環境変数設定
domain/            エンティティ・リポジトリIF
usecase/           ビジネスロジック
infrastructure/    DB・外部API実装
interface/         HTTPハンドラー・ミドルウェア
pkg/               JWT・バリデーター
db/                sqlc スキーマ・クエリ
k8s/               Kubernetes マニフェスト
```

## デプロイ構成

```
[GitHub push to main]
      ↓
[GitHub Actions] CI（test + lint）
      ↓
[Docker buildx] linux/arm64 イメージビルド
      ↓
[GHCR] ghcr.io/kazumadev619-dev/fishing-api:sha-xxxxx
      ↓
[Cloudflare Tunnel] k3s API アクセス
      ↓
[kubectl] Raspberry Pi k3s ローリングデプロイ
```

- **PostgreSQL**: Neon（マネージドクラウド）
- **Redis**: Raspberry Pi 上で Docker Compose（k3s 外）
- **アプリ**: k3s Pod
- **ルーティング**: Traefik Ingress（`/api/*` → Go, `/*` → Next.js）

## ローカル開発

```bash
# 前提: Go 1.26+, Docker, sqlc, golangci-lint

# 環境変数設定（DATABASE_URL は Neon の接続文字列を設定）
cp .env.example .env

# Redis 起動（DB は Neon のため不要）
make redis-up

# sqlc コード生成
make sqlc-gen

# サーバー起動
make run

# 動作確認
curl http://localhost:8080/health
```

## コマンド一覧

```bash
make run         # 開発サーバー起動（:8080）
make test        # 全テスト実行
make lint        # Lint チェック
make sqlc-gen    # sqlc コード再生成
make redis-up    # Redis 起動（DB は Neon のため不要）
make redis-down  # Redis 停止
make build       # バイナリビルド（./bin/server）
```

## ドキュメント

- [設計ドキュメント](docs/superpowers/specs/2026-04-07-go-backend-design.md)
- [開発コントリビューションガイド](CONTRIBUTING.md)
- [Phase 0: リポジトリ整備](docs/superpowers/plans/2026-04-07-go-backend-phase0-repo-setup.md)
- [Phase 1: 基盤構築](docs/superpowers/plans/2026-04-07-go-backend-phase1-foundation.md)
- [Phase 2: 認証](docs/superpowers/plans/2026-04-07-go-backend-phase2-auth.md)
- [Phase 3: コアAPI](docs/superpowers/plans/2026-04-07-go-backend-phase3-core-apis.md)
- [Phase 4: デプロイ](docs/superpowers/plans/2026-04-07-go-backend-phase4-deployment.md)
```

- [ ] **Step 2: コミット**

```bash
git add README.md
git commit -m "docs: README.md 更新（アーキテクチャ・API一覧・開発手順）"
```

---

## 完了条件チェックリスト

- [ ] `git status` で `.env` が untracked に出ない
- [ ] `npx cspell "docs/**/*.md"` でエラーが 0
- [ ] VSCode の PROBLEMS タブで cSpell 診断エラーが解消している
- [ ] `golangci-lint run --fast ./...` が設定エラーなく実行できる（Phase 1 以降）
- [ ] `lefthook install` が成功し `.git/hooks/pre-commit` が存在する
- [ ] `.claude/settings.json` に PostToolUse（gofumpt + fast lint）フックが設定されている
- [ ] `.golangci.yml` を Claude Code から直接編集しようとするとブロックされる
- [ ] `README.md` にアーキテクチャ・API一覧・コマンド一覧が記載されている
- [ ] `.github/` に PR テンプレートと Issue テンプレートが存在する
- [ ] `CONTRIBUTING.md` にコミット規約と開発フローが記載されている
