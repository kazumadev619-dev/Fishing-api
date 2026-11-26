# 0007. スキーマの正を Go 側に移し、goose で管理する

- Status: Accepted
- Date: 2026-11-27
- Deciders: kazumadev619-dev
- Amends: [ADR 0004](0004-db-access-sqlc-pgx.md)（`db/schema.sql` の同期元の記述）

## Context and Problem Statement

ADR 0004 と `.claude/rules/sqlc.md` には「`db/schema.sql` は `Fishing-database` リポジトリから CI で自動同期する」とある。しかし `Fishing-database` リポジトリは存在せず、同期の workflow（`sync-schema.yml`）も無い。

本番の Neon のテーブルを作っているのは、FishingConditionsApp（Next）の `prisma/migrations` である。Go の `schema.sql` は本番と差がある。たとえば `locations` の座標の一意制約（#151）や、`locations.prefecture` と `ports.prefecture_code` の索引が無い。

[ADR 0006](0006-incremental-migration-via-nextjs-bff.md) で、バックエンドを1本ずつ Go に移すと決めた。最後には Next から DB を外すので、スキーマの正をどこに置き、何で migration を流すかを決める。

ADR 0004（sqlc + pgx）は「Go からどう DB を読み書きするか」の決定で、この点は変えない。スキーマの正と migration のツールは別の関心事なので、別の ADR にする。

## Considered Options

- スキーマの正を Next の `prisma/migrations` に置いたまま、Go の `schema.sql` をそこから作る
- スキーマの正を Go 側に移し、goose（SQL ファイル）で管理する
- スキーマの正を Go 側に移し、golang-migrate や Atlas で管理する

## Decision Outcome

**スキーマの正を Go 側に移し、goose で管理する。**

- migration は SQL ファイルで書き、`embed` でバイナリに埋め込む。
- 移す時期は Go を Pi に出す段（Phase 4）とする。
- 最初の migration は本番の今の形（Prisma の最新の migration を当てたあとの形）を起点にする。本番には適用済みとして記録し、テーブルを作り直さない。
- 移したあとは、Prisma の migration を止める。Next の `schema.prisma` は `prisma db pull` で本番に追従させる。
- k8s の db-migrate Job は、Go 側のイメージで流す。
- sqlc のスキーマは goose の migration のフォルダを読む形に変える予定（sqlc は goose の書式を読める）。形は Phase 4 の issue で決める。

### Phase 4 までの扱い

- `db/schema.sql` は、本番（Prisma の migration）に合わせるときだけ手で直す。`Fishing-database` からの同期は無い。

## Positive Consequences

- 最後に Next から DB を外したとき、スキーマの持ち主が残らない問題を避けられる。
- migration を Go のバイナリ1つに入れられるので、Pi に配るものが増えない。
- goose は素の SQL で書くので、sqlc のクエリと同じ言葉でスキーマを読める。

## Negative Consequences

- Phase 4 の切り替えの日に、Prisma 側の migration を確実に止める必要がある。両方から変更すると本番の形がずれる。
- Next の開発者はスキーマを変えたいときに Fishing-api に PR を出す必要がある。
- 最初の migration を本番の形と一致させる確認が要る。

## Links

- 関連: [ADR 0004 — DB アクセスに sqlc + pgx/v5](0004-db-access-sqlc-pgx.md)、[ADR 0006 — Next を BFF にして1本ずつ Go に移す](0006-incremental-migration-via-nextjs-bff.md)
- sqlc / DB 管理ルール: [`.claude/rules/sqlc.md`](../../.claude/rules/sqlc.md)
- goose: https://github.com/pressly/goose
