---
paths:
  - "**/*.go"
---
# Go Coding Style

## エラーハンドリング

エラーは必ず戻り値で返す。`panic` は `main`・`init` の初期化失敗時のみ許可。

```go
// WRONG: panic でエラーを握りつぶす
func getUser(id string) *User {
    u, err := db.Find(id)
    if err != nil {
        panic(err)
    }
    return u
}

// CORRECT: error を戻り値で返す
func getUser(ctx context.Context, id string) (*User, error) {
    u, err := db.Find(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("getting user %s: %w", id, err)
    }
    return u, nil
}
```

- エラーラップは `fmt.Errorf("doing X: %w", err)` で文脈を付ける
- sentinel error は `var ErrNotFound = errors.New("not found")` で定義
- エラーチェックを省略しない（`_` への代入禁止）

## 命名規則

| 対象 | ルール | 例 |
|------|--------|-----|
| パッケージ | 小文字・単数形・略さない | `user`, `weather`, `tide` |
| インターフェース | 動詞+er 形 | `UserStorer`, `WeatherFetcher` |
| レシーバ名 | 型名の頭文字 1〜2 文字 | `u *User`, `uc *authUsecase` |
| エラー変数 | `Err` プレフィックス | `ErrNotFound`, `ErrUnauthorized` |
| bool 返却関数 | `Is`/`Has`/`Can` プレフィックス | `IsExpired()`, `HasPermission()` |

```go
// WRONG: 不明瞭な名前
func (x *AuthUsecaseImpl) do(s string) (interface{}, error) { ... }

// CORRECT: 明確な名前
func (uc *authUsecase) Login(ctx context.Context, email string) (*TokenPair, error) { ... }
```

## インターフェース設計

- インターフェースは**使う側**のパッケージで定義する（`internal/domain/repository/`）
- 小さいインターフェースを優先（1〜3 メソッドが理想）
- 実装を先に書き、インターフェースは後から抽出する

```go
// WRONG: 実装パッケージでインターフェースを定義
// internal/infrastructure/db/user_repository.go
type UserRepository interface { ... }  // NG: 使う側で定義すべき

// CORRECT: domain パッケージで定義（依存の向きが正しい）
// internal/domain/repository/user.go
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (*entity.User, error)
    Create(ctx context.Context, user *entity.User) error
}
```

## context.Context

```go
// WRONG: context を struct に保存
type Service struct {
    ctx context.Context  // NG
}

// CORRECT: 関数の第1引数として受け渡す
func (s *weatherUsecase) GetCurrent(ctx context.Context, lat, lon float64) (*entity.WeatherData, error) {
    return s.client.Fetch(ctx, lat, lon)
}
```

- DB・外部 API・Redis へのアクセスには必ず `ctx` を渡す
- `context.Background()` は `main`・テスト・ゴルーチン起点のみ使用

## ゼロ値・初期化

```go
// WRONG: 不要な初期化
results := make([]string, 0)
var count int = 0

// CORRECT: ゼロ値を活用
var results []string
var count int
```

- `make([]T, 0)` ではなく `var s []T` で宣言（append するだけなら容量指定不要）
- struct フィールドのゼロ値が有効な初期状態になるよう設計する

## Early Return

深いネストを避け、早期 return でガード節を書く。

```go
// WRONG: 深いネスト
func validate(u *User) error {
    if u != nil {
        if u.Email != "" {
            if isValidEmail(u.Email) {
                return nil
            } else {
                return ErrInvalidEmail
            }
        } else {
            return ErrEmptyEmail
        }
    }
    return ErrNilUser
}

// CORRECT: Early return
func validate(u *User) error {
    if u == nil {
        return ErrNilUser
    }
    if u.Email == "" {
        return ErrEmptyEmail
    }
    if !isValidEmail(u.Email) {
        return ErrInvalidEmail
    }
    return nil
}
```

## ログ・デバッグ

- `fmt.Println` / `log.Println` を本番コードに残さない
- ロガーは `slog`（Go 1.21+）または DI 注入されたロガーを使用
- デバッグ用 `fmt.Printf` はコミット前に必ず削除
