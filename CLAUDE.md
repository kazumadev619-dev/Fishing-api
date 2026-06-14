# Fishing-api

Go バックエンド for [FishingConditionsApp](https://github.com/kazumadev619-dev/FishingConditionsApp)。釣り条件（天気・潮汐・スコア）を提供する REST API なのだ。

## ドキュメントの歩き方

| 知りたいこと | 参照先 |
|------------|--------|
| 技術スタック・アーキテクチャ・起動方法 | [`README.md`](./README.md) |
| 設計判断の経緯（なぜこの構成か） | [`docs/adr/`](./docs/adr/) |
| 設計仕様 | [`docs/superpowers/specs/`](./docs/superpowers/specs/) |
| 実装計画 | [`docs/superpowers/plans/`](./docs/superpowers/plans/) |
| 開発ルール（アーキ・テスト・セキュリティ等） | [`.claude/rules/`](./.claude/rules/) |

## 禁止事項

- `db/schema.sql` を直接編集しない（`Fishing-database` リポジトリから CI 自動同期）
- `db/generated/` を編集しない（`make sqlc-gen` で再生成）
- `.env` をコミットしない（シークレットは環境変数のみ）
