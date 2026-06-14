# Workflow

開発パイプライン・コードレビュー・エージェント振り分けを 1 ファイルに統合したルールなのだ。
（旧 `agents.md` / `code-review.md` / `development-workflow.md` を統合）

## Section 1: Feature Pipeline

Git 操作の前に行う開発プロセス。Git 操作自体は [git-workflow.md](./git-workflow.md) を参照。

0. **Research & Reuse**（新規実装の前に必須）
   - **GitHub code search first:** `gh search repos` / `gh search code` で既存実装・テンプレート・パターンを探す。
   - **Library docs second:** Context7 やベンダー公式ドキュメントで API 挙動・バージョン差を確認する。
   - **Exa は最後の手段:** 上記 2 つで足りないときのみ広域 Web リサーチに使う。
   - **パッケージレジストリを確認:** npm / PyPI / crates.io 等を調べ、自前実装より実績あるライブラリを優先する。
   - 80% 以上を解決する OSS があれば、新規実装よりフォーク・移植・ラップを優先する。

1. **Plan First**
   - **planner** エージェントで実装計画を作る。
   - コーディング前に計画ドキュメント（PRD / architecture / system_design / task_list）を生成する。
   - 依存とリスクを洗い出し、フェーズに分割する。

2. **TDD Approach**
   - **tdd-guide** エージェントを使う。
   - テストを先に書く（RED）→ 通す実装（GREEN）→ リファクタ（IMPROVE）。
   - 詳細・カバレッジ要件は Section 4 を参照。

3. **Code Review**
   - 書いた直後に **code-reviewer** エージェントでレビュー（Section 2）。
   - CRITICAL / HIGH を対応し、可能なら MEDIUM も直す。

4. **Commit & Push**
   - 詳細なコミットメッセージ。Conventional Commits 形式。
   - 形式・PR 手順は [git-workflow.md](./git-workflow.md) を参照。

5. **Pre-Review Checks**
   - CI/CD が全てパス・コンフリクト解消・ターゲットブランチに追従済みを確認してからレビュー依頼する。

## Section 2: Code Review Triggers + Checklist

**レビュー必須トリガー:**

- コードを書いた / 変更した後
- 共有ブランチへのコミット前
- 認証・認可・ユーザーデータなどセキュリティ関連の変更時
- アーキテクチャ変更時
- PR マージ前

**レビュー前提:** CI/CD パス済み・コンフリクト解消済み・ターゲットブランチに追従済み。

**チェックリスト（完了前に確認）:**

- [ ] 読みやすく命名が適切
- [ ] 関数は焦点が絞られている（< 50 行）
- [ ] ファイルは凝集的（< 800 行）
- [ ] 深いネストがない（> 4 階層を避ける）
- [ ] エラーは明示的に処理
- [ ] ハードコードされた秘密情報・認証情報がない
- [ ] デバッグ出力（`console.log` 等）が残っていない
- [ ] 新機能にテストがある（カバレッジ 80% 以上、Section 4）

**重大度レベル:**

| Level | 意味 | アクション |
|-------|------|-----------|
| CRITICAL | セキュリティ脆弱性・データ損失リスク | **BLOCK**（マージ前に必須修正） |
| HIGH | バグ・重大な品質問題 | **WARN**（マージ前に修正すべき） |
| MEDIUM | 保守性の懸念 | **INFO**（可能なら修正） |
| LOW | スタイル・軽微な提案 | **NOTE**（任意） |

承認基準: CRITICAL/HIGH なし → Approve、HIGH のみ → Warning、CRITICAL あり → Block。

**セキュリティレビュー必須トリガー**（→ Section 3 の security-reviewer）: 認証/認可・ユーザー入力処理・DB クエリ・ファイル操作・外部 API 呼び出し・暗号/ハッシュ処理。詳細は [security.md](./security.md) を参照。

## Section 3: Agent Dispatch

エージェントは `~/.claude/agents/` に配置。プロンプト不要で即時起動するケースもある。

| エージェント | 用途 | 起動タイミング |
|------------|------|--------------|
| planner | 実装計画 | 複雑な機能・リファクタ（即時） |
| architect | システム設計 | アーキテクチャ判断（即時） |
| tdd-guide | テスト駆動開発 | 新機能・バグ修正（即時） |
| code-reviewer | 一般的な品質・パターン・ベストプラクティス | コード記述/変更の直後（即時） |
| security-reviewer | セキュリティ脆弱性・OWASP Top 10 | コミット前・セキュリティ関連変更時 |
| go-reviewer | Go 固有のレビュー | Go コード記述後 |
| build-error-resolver | ビルドエラー修正 | ビルド失敗時 |
| e2e-runner | E2E テスト | 重要ユーザーフロー |
| refactor-cleaner | デッドコード除去 | コード保守 |
| doc-updater | ドキュメント更新 | ドキュメント更新時 |

**並列実行:** 独立した操作は必ず並列で Task 実行する（例: 認証モジュールのセキュリティ分析 / キャッシュの性能レビュー / ユーティリティの型チェックを同時起動）。

**多視点分析:** 複雑な問題は役割分担したサブエージェント（事実確認 / シニアエンジニア / セキュリティ専門 / 一貫性 / 冗長性）で検討する。

## Section 4: Coverage & TDD

TDD の進め方とカバレッジ要件（全体 80%+ 必須・usecase 90%+ 目標）、テーブルドリブンテスト、testify/mock・testcontainers-go・httptest+Gin の使い方は重複定義を避け、[testing.md](./testing.md) に一本化する。本セクションはリンクのみ。
