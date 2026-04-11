# Security

## シークレット管理（最重要）

シークレット（API キー・JWT 秘密鍵・DB 接続文字列）は**環境変数のみ**から取得する。

```go
// WRONG: ハードコード
const jwtSecret = "my-super-secret-key"
dsn := "postgres://user:password@host/db"

// CORRECT: 環境変数から取得
jwtSecret := os.Getenv("JWT_SECRET")
dsn := os.Getenv("DATABASE_URL")  // sslmode=require を含む
```

- コードレビューでハードコードが見つかった場合は **CRITICAL ブロック**
- `.env` ファイルは `.gitignore` に含める（コミット禁止）
- テストコードにも本物の API キー・DB 接続情報を書かない

## DB 接続（Neon 必須設定）

```go
// WRONG: sslmode=require なし
dsn := "postgres://user:pass@neon-host/db"

// CORRECT: sslmode=require 必須
dsn := os.Getenv("DATABASE_URL")
// DATABASE_URL には必ず ?sslmode=require を含める
// 例: postgres://user:pass@host/db?sslmode=require
```

## SQL インジェクション防止

SQL は **sqlc 管理クエリのみ**使用する。文字列連結・`fmt.Sprintf` による SQL 構築は禁止。

```go
// WRONG: 文字列連結で SQL を構築
query := "SELECT * FROM users WHERE email = '" + email + "'"
db.Query(query)

// CORRECT: sqlc 生成クエリを使用
user, err := q.GetUserByEmail(ctx, email)
```

- 動的 WHERE 句が必要な場合も sqlc の Named Parameters を使う
- Raw SQL が必要な場合は `db/queries/` に追加して `make sqlc-gen` で再生成

## JWT

| 項目 | 設定値 |
|------|--------|
| アクセストークン有効期限 | 15 分 |
| リフレッシュトークン有効期限 | 7 日 |
| 秘密鍵取得元 | 環境変数 `JWT_SECRET` |
| アルゴリズム | HS256 |

```go
// リフレッシュトークンはローテーション（使用済みは無効化）
// アクセストークンはステートレス（サーバー側で保存しない）
```

## Gin リクエストバリデーション

すべてのリクエスト構造体に `binding:"required"` タグを付ける。

```go
// WRONG: バリデーションなし
type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

// CORRECT: binding タグでバリデーション
type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // ...
}
```

## CORS

フロントエンドのオリジン（`fishing.kazuma-lab.com`）のみ許可する。ワイルドカード `*` は禁止。

```go
// WRONG: 全オリジン許可
config.AllowAllOrigins = true

// CORRECT: 明示的に許可オリジンを指定
config.AllowOrigins = []string{os.Getenv("FRONTEND_ORIGIN")}
```

## セキュリティレビュートリガー

以下を変更する場合は **security-reviewer エージェント**を使用する：

- 認証・認可ロジック（JWT 検証・ミドルウェア）
- ユーザー入力を処理するハンドラー
- DB クエリ（新規追加・変更）
- 外部 API クライアント（リクエスト・レスポンス処理）
- 暗号・ハッシュ処理（パスワードハッシュ等）

## 関連ルール

- [sqlc.md](./sqlc.md) — SQL インジェクション防止の詳細
- [code-review.md](./code-review.md) — セキュリティレビューのチェックリスト
