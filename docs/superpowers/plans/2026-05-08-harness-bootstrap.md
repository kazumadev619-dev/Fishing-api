# Pre-Phase 4 Bootstrap Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Phase 4（Docker・k8s・CI/CD・Cloudflare Tunnel）に進む前に、Claude Code ハーネスを「ハッカソン・ベストプラクティス2026」基準まで強化し、ドキュメント腐敗・テストカバレッジ不足・古い Phase 4 計画を解消する。B → C → A の3ブランチ・3PRで段階マージ。

**Architecture:** ハーネス（settings.json hooks）→ ドキュメント（CLAUDE.md / ADR / rules）→ Phase 4 着手準備（infra/db tests + refined plan）。各ブランチで `make test` + 手動 verify をパスさせる。

**Tech Stack:** Claude Code Hooks (JSON), MADR template, testcontainers-go (postgres:17), Hurl (smoke test 設計)

**前提条件:** Phase 0〜3 完了済み（PR #2〜#6 マージ済み）。go.mod = 1.26.2。テスト全 PASS（19 ファイル）。

---

## Context

Fishing-api は Phase 4 「Dockerfile・k8s・GitHub Actions・Cloudflare Tunnel デプロイ」に進む段階だが、エージェントに自律で IaC・CI/CD・本番接続を組ませる工程で、現状のハーネスは以下のリスクがある:

- PostToolUse hook がリント結果を**エージェントに戻していない**（`additionalContext` JSON 未使用）
- Stop hook が無く、ビルド/テスト未通過のまま完了宣言される可能性
- `.env` 編集や `rm -rf /` `git push --force main` `--no-verify` を構造的にブロックしていない
- CLAUDE.md に技術スタック解説が混入し腐敗リスク（[harness-engineering 記事](https://nyosegawa.com/posts/harness-engineering-best-practices-2026/) 推奨と乖離）
- 設計判断が CLAUDE.md に narrative で埋没（ADR 化されていない）
- `.claude/rules/` に重複（agents.md / code-review.md / development-workflow.md で同一概念が3箇所）
- `infra/db` カバレッジ 35.7%（Favorite/VerificationToken/UpdateEmailVerified が未テスト）→ Phase 4 の CI カバレッジゲートを通せない
- Phase 4 計画（2026-04-07）が **Go 1.24** 表記（実体は 1.26.2）で実装すれば古い Go でビルドされる
- E2E 検証が `curl /health` のみで Phase 4 完了判定が浅い

これらをハッカソン記事に沿って **B → C → A** の3ブランチで先に潰し、Phase 4 を「ハーネスがエラーを叫んでくれる」状態で着手するのが本 plan の目的。

---

## ユーザー確定事項

| 項目 | 確定値 |
|------|--------|
| ブランチ戦略 | **3 ブランチ・3 PR**（B → C → A 順次マージ） |
| ADR スコープ | **5件すべて**起こす（CLAUDE.md からは除去） |
| M1 スコープ | Favorite + VerificationToken + UpdateEmailVerified = **8関数すべて** |
| 対象範囲 | **Low 以外**（B1, B2, H1〜H5, M1〜M5）。L1〜L3 は対象外 |
| トーン | ずんだもん口調維持（`Project/CLAUDE.md` 由来） |

---

## 全体構造

```
feature/harness-tighten          B-pre / B-1 / B-2 / B-3            (M4, H1, H2, H5)
   ↓ merge
feature/docs-tidy                C-1 / C-2 / C-3 / C-4              (H3, H4, M2, M5)
   ↓ merge
feature/pre-phase4-prep          A-1 / A-2                          (M1, B1, B2, M3)
   ↓ merge → ここから Phase 4 着手
```

---

## Branch B: `feature/harness-tighten`

### B-pre — `.serena/project.yml` ドリフト解消 (M4)

- [ ] **Step 1:** ワークツリー残差分の性質を確認

```bash
git diff --stat .serena/project.yml
git diff .serena/project.yml | grep -E "^[+-][^+#]" | head -20
```

- [ ] **Step 2:** コメント/空行のみなら revert、機能変更を含むなら別 commit で取り込む

```bash
# コメント/空行のみの場合
git checkout -- .serena/project.yml
git status --porcelain .serena/project.yml   # 空であること
```

### B-1 (H1) — PostToolUse Hook を `additionalContext` JSON 形式に変更

- [ ] **Step 1:** `.claude/settings.json` の PostToolUse を以下に差し替え（`--fix` 後に残る issue だけを Claude に注入。常に exit 0 で advisory）

```json
"PostToolUse": [
  {
    "matcher": "Write|Edit",
    "hooks": [
      {
        "type": "command",
        "command": "bash -c 'INPUT=$(cat); FILE=$(echo \"$INPUT\" | jq -r \".tool_input.file_path // .file_path // .path // empty\" 2>/dev/null); if [ -z \"$FILE\" ] || ! echo \"$FILE\" | grep -q \"\\.go$\"; then exit 0; fi; gofumpt -w \"$FILE\" >/dev/null 2>&1 || true; golangci-lint run --fast --fix \"$FILE\" >/dev/null 2>&1 || true; REMAINING=$(golangci-lint run --fast --out-format=line-number \"$FILE\" 2>&1 || true); if [ -n \"$REMAINING\" ] && echo \"$REMAINING\" | grep -qE \":[0-9]+:[0-9]+:\"; then jq -nc --arg ctx \"golangci-lint reports unresolved issues in $FILE after --fix:\\n$REMAINING\" '\\''{hookSpecificOutput:{hookEventName:\"PostToolUse\",additionalContext:$ctx}}'\\''; fi; exit 0'"
      }
    ]
  }
]
```

- [ ] **Step 2:** Verify

```bash
jq . .claude/settings.json >/dev/null && echo "JSON OK"
# clean .go ファイル Edit → 出力なし
# 未使用 import を含む .go Edit → additionalContext が次ターンに見える
```

### B-2 (H2) — Stop Hook 追加（build + test、失敗時のみ注入）

- [ ] **Step 1:** `.claude/settings.json` の `hooks` 直下に Stop セクション追加

```json
"Stop": [
  {
    "matcher": "",
    "hooks": [
      {
        "type": "command",
        "command": "bash -c 'cd /Users/nosawakazuma/Project/Fishing-api && OUT=$( { go build ./... && go test ./... -short -count=1; } 2>&1 | head -c 4000); if [ ${PIPESTATUS[0]} -ne 0 ]; then jq -nc --arg ctx \"Stop hook: build/test failed (truncated to 4KB):\\n$OUT\" '\\''{hookSpecificOutput:{hookEventName:\"Stop\",additionalContext:$ctx}}'\\''; fi; exit 0'"
      }
    ]
  }
]
```

- [ ] **Step 2:** Verify — `.go` に意図的な構文エラー → ターン終了で Stop 出力 → revert

**Risk:** 持続的失敗時は毎ターン context 注入でループ化リスク。`head -c 4000` 出力切り詰め + `-short` で testcontainers 除外で緩和。再発するなら sentinel 機構を後付け。

### B-3 (H5) — PreToolUse Safety Gate 拡張

- [ ] **Step 1:** 既存リンター設定保護 (settings.json:17-27) に**シークレット保護**と**Bash危険コマンド保護**を追加

```json
{
  "matcher": "Write|Edit|MultiEdit",
  "hooks": [{
    "type": "command",
    "command": "bash -c 'INPUT=$(cat); FILE=$(echo \"$INPUT\" | jq -r \".tool_input.file_path // .file_path // .path // empty\"); if echo \"$FILE\" | grep -qE \"(^|/)\\.env($|\\.)|\\.key$|kubeconfig$|_rsa$\"; then echo \"BLOCKED: secret/credential file edits forbidden ($FILE)\" >&2; exit 2; fi; exit 0'"
  }]
},
{
  "matcher": "Bash",
  "hooks": [{
    "type": "command",
    "command": "bash -c 'CMD=$(cat | jq -r \".tool_input.command // empty\"); if echo \"$CMD\" | grep -qE \"rm[[:space:]]+-rf?[[:space:]]+(/|~|\\*)\"; then echo \"BLOCKED: destructive rm -rf root\" >&2; exit 2; fi; if echo \"$CMD\" | grep -qE \"git[[:space:]]+push.*--force.*(main|master)|git[[:space:]]+push.*(main|master).*--force\"; then echo \"BLOCKED: force-push to main/master\" >&2; exit 2; fi; if echo \"$CMD\" | grep -qE \"--no-verify\\b\"; then echo \"BLOCKED: --no-verify forbidden\" >&2; exit 2; fi; exit 0'"
  }]
}
```

- [ ] **Step 2:** Verify

```bash
echo '{"tool_input":{"command":"git push --force origin main"}}' | bash -c "$(jq -r '.hooks.PreToolUse[1].hooks[0].command' .claude/settings.json)"   # exit 2
echo '{"tool_input":{"command":"git push origin main"}}' | bash -c "$(jq -r '.hooks.PreToolUse[1].hooks[0].command' .claude/settings.json)"           # exit 0
echo '{"tool_input":{"command":"rm -rf ./bin"}}' | bash -c "$(jq -r '.hooks.PreToolUse[1].hooks[0].command' .claude/settings.json)"                   # exit 0（誤爆チェック）
```

### Branch B PR

- **Title:** `chore(harness): tighten Claude Code hooks before Phase 4`
- **Body skeleton:**
  - Summary（lint feedback loop / Stop verification / safety gate）
  - Why（Phase 4 自律デプロイに備えハーネス強化）
  - Test plan（上記 verify コマンド）

---

## Branch C: `feature/docs-tidy`

### C-1 (H3) — CLAUDE.md スリム化（47行 → ~25行）

- [ ] **Step 1:** `Fishing-api/CLAUDE.md` を改訂
  - 残す: H1、1行プロジェクト説明、ルーティング指示（`docs/adr/`, `docs/superpowers/specs/`, `docs/superpowers/plans/`, `.claude/rules/`）、禁止事項リスト（schema.sql 直接編集禁止 / db/generated/ 編集禁止 / .env コミット禁止）、README + ADR への誘導
  - 削除: 「技術スタック」表（lines 5–16）、「アーキテクチャ」セクション（lines 18–28）

- [ ] **Step 2:** 削除内容を `Fishing-api/README.md` に移植（技術スタック表 + アーキテクチャ図）

### C-2 (H4) — ADR セット作成（5件 + index）

- [ ] **Step 1:** `docs/adr/` ディレクトリ作成

- [ ] **Step 2:** MADR テンプレート（各 ADR 共通形式）

```markdown
# ADR NNNN: <Title>

- Status: Accepted
- Date: 2026-05-08
- Deciders: kazumadev619-dev

## Context and Problem Statement
<background>

## Considered Options
- Option A: <name>
- Option B: <name>
- Option C: <name>

## Decision Outcome
Chosen option: "<Option A>", because <rationale>.

### Positive Consequences
- ...

### Negative Consequences / Trade-offs
- ...

## Links
- docs/superpowers/specs/2026-04-07-go-backend-design.md
- (related ADRs / code paths)
```

- [ ] **Step 3:** 5 ADR 作成（根拠ソースに沿って執筆）

| ADR | ファイル | Source |
|-----|---------|--------|
| 0001 | `docs/adr/0001-database-provider-neon.md` | CLAUDE.md L32（Supabase Free 7日停止） |
| 0002 | `docs/adr/0002-redis-on-k3s-pod.md` | CLAUDE.md L33（ステートレスキャッシュ） |
| 0003 | `docs/adr/0003-bigbang-migration-from-nextjs-api.md` | CLAUDE.md L34–35 + design spec L8–11 |
| 0004 | `docs/adr/0004-db-access-sqlc-pgx.md` | CLAUDE.md L12 + design spec |
| 0005 | `docs/adr/0005-clean-architecture-layers.md` | CLAUDE.md L18–28 + `clean-architecture.md` |

- [ ] **Step 4:** `docs/adr/README.md` を索引として作成（5 ADR の1行サマリ + テンプレートへのリンク）

### C-3 (M2) — ルール3つを `workflow.md` に統合

- [ ] **Step 1:** `.claude/rules/workflow.md` 新規作成（~80行）
  - Section 1: Feature Pipeline（development-workflow.md から）
  - Section 2: Code Review Triggers + Checklist（code-review.md コア部分）
  - Section 3: Agent Dispatch（agents.md + code-review.md エージェント表を統合）
  - Section 4: Coverage & TDD（testing.md と重複しないようリンクのみ）

- [ ] **Step 2:** 旧3ファイルを削除

```bash
git rm .claude/rules/agents.md
git rm .claude/rules/code-review.md
git rm .claude/rules/development-workflow.md
```

- [ ] **Step 3:** 参照漏れチェック

```bash
grep -rn "rules/agents.md\|rules/code-review.md\|rules/development-workflow.md" CLAUDE.md .claude/ docs/
# → 空 or 本 plan 自身のみ
```

### C-4 (M5) — `Project/CLAUDE.md` に共通禁止事項追加

- [ ] **Step 1:** `/Users/nosawakazuma/Project/CLAUDE.md`（現状3行）に追記

```markdown
- シークレットを含むファイル (.env, *.key, kubeconfig, *_rsa) は git add 禁止。誤って add した場合は即 git restore --staged。
- 破壊的 git 操作 (push --force / reset --hard / branch -D) は事前に「実行してよい？」と確認すること。
```

### Branch C verify

```bash
wc -l CLAUDE.md                                     # ~25
ls docs/adr/ | wc -l                                # 6
test ! -e .claude/rules/agents.md
test ! -e .claude/rules/code-review.md
test ! -e .claude/rules/development-workflow.md
test -e .claude/rules/workflow.md
go build ./... && go test ./... -short              # ハーネス回帰なし
```

### Branch C PR

- **Title:** `docs: slim CLAUDE.md, add 5 ADRs, consolidate rules`
- **Body skeleton:** Summary → Why（context 削減 + 設計判断の固定化）→ Test plan

---

## Branch A: `feature/pre-phase4-prep`

### A-1 (M1) — testcontainers 統合テスト 8本追加（infra/db 35.7% → 80%+）

- [ ] **Step 1:** `setupTestDB` を `internal/infrastructure/db/helpers_test.go` に切り出し
  - 既存: `user_repository_test.go:21-56`
  - schema パスは `../../../db/schema.sql` のまま
  - 位置情報シードは `db.ExecContext` で直接 INSERT（locations は repo 経由で作成しないため）

- [ ] **Step 2:** `internal/infrastructure/db/favorite_repository_test.go` 新規作成
  - `TestFavoriteRepository_Add_FindByUserID` — user + location 挿入 → Add → FindByUserID で 1 件
  - `TestFavoriteRepository_Exists_TrueFalse` — Add 前後で boolean 変化
  - `TestFavoriteRepository_Delete_Success` — Add → Delete → Exists=false
  - `TestFavoriteRepository_Delete_NotFound` — 未存在 Delete で `errors.Is(err, domain.ErrNotFound)`

- [ ] **Step 3:** `internal/infrastructure/db/verification_token_repository_test.go` 新規作成
  - `TestVerificationTokenRepository_Create_FindByToken`
  - `TestVerificationTokenRepository_FindByToken_NotFound` → `domain.ErrNotFound`
  - `TestVerificationTokenRepository_DeleteByEmail` → 0件でもエラー無し（idempotent 確認）

- [ ] **Step 4:** `internal/infrastructure/db/user_repository_test.go` に `TestUserRepository_UpdateEmailVerified` 追加
  - email_verified=false で Create → UpdateEmailVerified → Refetch → true

**既存パターン参照:**
- testcontainers セットアップ: `user_repository_test.go:21-56`
- Delete 戻り値: `favorite_repository.go:46-62`
- ErrNotFound: `internal/domain/errors.go:6`

- [ ] **Step 5:** Verify

```bash
go test ./internal/infrastructure/db/... -count=1 -v
go test ./internal/infrastructure/db/... -coverprofile=cov.out
go tool cover -func=cov.out | tail -1               # ≥80%
```

### A-2 (B1+B2+M3) — Phase 4 refined plan 作成

- [ ] **Step 1:** `docs/superpowers/plans/2026-05-08-go-backend-phase4-deployment-refined.md` を作成

ベース: `2026-04-07-go-backend-phase4-deployment.md` (1224 行) を踏襲しつつ以下を差し替え

- [ ] **Step 2:** Go バージョン更新
  - Dockerfile: `golang:1.24-alpine` → `golang:1.26-alpine`
  - CI: `go-version: '1.24'` → `'1.26'`

- [ ] **Step 3:** `.github/workflows/ci.yml` の test ジョブ末尾にカバレッジゲート追加

```bash
go test -coverprofile=coverage.out ./...
COV=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | tr -d %)
awk -v c="$COV" 'BEGIN{exit !(c+0<80)}' && { echo "coverage $COV% < 80%"; exit 1; } || echo "coverage $COV% OK"
```

- [ ] **Step 4:** Hurl smoke test セクション新設
  - `tests/smoke/health.hurl` — `GET {{base_url}}/health` → 200, body contains `"ok"`
  - `tests/smoke/weather.hurl` — `GET {{base_url}}/api/weather?lat=35.0&lon=139.0` → 200, `jsonpath "$.temperature" exists`
  - `deploy.yml` の rollout status 後に `hurl --variable base_url=https://fishing.kazuma-lab.com --test tests/smoke/*.hurl`

- [ ] **Step 5:** Phase 3 出力参照を反映
  - `internal/interface/router/router.go` の `New(handlers, jwtManager)` シグネチャ
  - `cmd/server/main.go` の `jwtManagerAdapter`

- [ ] **Step 6:** 完了条件チェックリストを Hurl smoke + coverage gate を含む形に更新

### Branch A verify

```bash
go test ./... -count=1
go test ./internal/infrastructure/db/... -coverprofile=cov.out && go tool cover -func=cov.out | tail -1   # ≥80%
test -f docs/superpowers/plans/2026-05-08-go-backend-phase4-deployment-refined.md
grep -c "golang:1.26-alpine" docs/superpowers/plans/2026-05-08-go-backend-phase4-deployment-refined.md   # ≥1
grep -c "coverage.out" docs/superpowers/plans/2026-05-08-go-backend-phase4-deployment-refined.md         # ≥1
grep -c "hurl" docs/superpowers/plans/2026-05-08-go-backend-phase4-deployment-refined.md                 # ≥1
```

### Branch A PR

- **Title:** `test+docs: infra/db coverage to 80%+ and refined Phase 4 plan`
- **Body skeleton:**
  - Summary（8 testcontainers tests / Phase 4 refined w/ Go 1.26 + coverage gate + Hurl smoke）
  - Before/after coverage（35.7% → ≥80%）
  - Test plan

---

## Critical Files

```
.claude/settings.json                                              # B-1, B-2, B-3
.serena/project.yml                                                # B-pre (revert or commit)
CLAUDE.md                                                          # C-1
README.md                                                          # C-1（受け入れ先）
docs/adr/README.md                                                 # C-2 新規
docs/adr/0001-database-provider-neon.md                            # C-2 新規
docs/adr/0002-redis-on-k3s-pod.md                                  # C-2 新規
docs/adr/0003-bigbang-migration-from-nextjs-api.md                 # C-2 新規
docs/adr/0004-db-access-sqlc-pgx.md                                # C-2 新規
docs/adr/0005-clean-architecture-layers.md                         # C-2 新規
.claude/rules/workflow.md                                          # C-3 新規（統合先）
.claude/rules/agents.md                                            # C-3 削除
.claude/rules/code-review.md                                       # C-3 削除
.claude/rules/development-workflow.md                              # C-3 削除
/Users/nosawakazuma/Project/CLAUDE.md                              # C-4 追記
internal/infrastructure/db/helpers_test.go                         # A-1 新規（setupTestDB 抽出）
internal/infrastructure/db/user_repository_test.go                 # A-1（UpdateEmailVerified 追加 + helpers 移行）
internal/infrastructure/db/favorite_repository_test.go             # A-1 新規
internal/infrastructure/db/verification_token_repository_test.go   # A-1 新規
docs/superpowers/plans/2026-05-08-go-backend-phase4-deployment-refined.md  # A-2 新規
```

## 既存資産の再利用

- `setupTestDB` (user_repository_test.go:21-56) → A-1 で helpers_test.go に切り出し
- 既存 PreToolUse pattern (settings.json:17-27) → B-3 で同形式を踏襲
- 既存 Phase 4 plan (`docs/superpowers/plans/2026-04-07-go-backend-phase4-deployment.md`) → A-2 のベース
- design spec (`docs/superpowers/specs/2026-04-07-go-backend-design.md`) → C-2 ADR の根拠ソース
- Phase 3 refined plan の `router.New(handlers, jwtManager)` 記述 → A-2 の entrypoint 配線参照

## Risk Callouts

- **Stop hook ループ**: 持続的失敗時はターン毎に additionalContext 注入。`head -c 4000` で出力切り詰め + `-short` で testcontainers 除外。発生したら sentinel 機構を後付け。
- **Bash 正規表現の誤爆**: `rm -rf` は `/`, `~`, `*` ルートのみマッチ。`rm -rf ./bin` のような相対パスは通過する設計（要 verify）。
- **JSON quote 地獄**: settings.json 編集後は必ず `jq . .claude/settings.json` で構文チェックし、stdin パイプで動作確認。
- **ADR 内容ドリフト**: ADR の決定が実コードと乖離しないよう、各 ADR の Links に対応コード/spec パスを明記。
- **testcontainers schema ドリフト**: helper は `db/schema.sql` を raw apply。Phase 3 で migration を別途追加していないか `git log -- db/schema.sql` で確認。既存 `TestUserRepository_*` が PASS することを smoke として先に確認。
- **ブランチ順序**: B → C → A で固定。理由 (1) A の testcontainers 実行が B-2 Stop hook を実条件で検証する、(2) C の CLAUDE.md スリム化は C-3 で削除されるルール参照を含まないようにする必要がある。

## Verification（end-to-end）

```bash
# Branch B 完了後
jq . .claude/settings.json >/dev/null
echo '{"tool_input":{"command":"git push --force origin main"}}' | bash -c "$(jq -r '.hooks.PreToolUse[1].hooks[0].command' .claude/settings.json)"  # exit 2

# Branch C 完了後
wc -l CLAUDE.md                                          # ~25
ls docs/adr/ | wc -l                                     # 6
test ! -e .claude/rules/code-review.md

# Branch A 完了後
go test ./... -count=1
go test ./internal/infrastructure/db/... -coverprofile=cov.out
go tool cover -func=cov.out | tail -1                    # ≥80%
test -f docs/superpowers/plans/2026-05-08-go-backend-phase4-deployment-refined.md

# 全体（A マージ後・Phase 4 着手前の最終チェック）
go build ./...
go test ./... -count=1
make lint
git log --oneline -10                                    # B/C/A の3 PR が main にある
```

## Next Steps（Phase 4 着手）

このプランの A までマージ完了後、`docs/superpowers/plans/2026-05-08-go-backend-phase4-deployment-refined.md` を `superpowers:executing-plans` で実行する。

## 対象外（明示的に Low）

- **L1**: schema.sql 自動同期（Phase 4 Task 7 で扱う、本 plan 範囲外）
- **L2**: gocritic/wrapcheck を PostToolUse に流す（CI で検出する現状を維持）
- **L3**: superpowers 系スキルの追加導入
