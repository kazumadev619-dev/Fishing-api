# Clean Architecture

## 依存方向（厳守）

```
interface/ → usecase/ → domain/ ← infrastructure/
```

`domain/` が最内層。外側のレイヤーが内側に依存し、内側は外側を知らない。

## 禁止 import パターン

| FROM | TO | 理由 |
|------|----|------|
| `domain/` | `usecase/` `infrastructure/` `interface/` | 最内層は外を知らない |
| `usecase/` | `infrastructure/` （直接） | インターフェース経由のみ許可 |
| `usecase/` | `interface/` | ハンドラー層に依存しない |
| `interface/handler/` | `infrastructure/` | usecase 経由のみ |
| `interface/handler/` | `db/generated/` | sqlc 生成コードに直接触らない |

```go
// WRONG: usecase が infrastructure を直接 import
// internal/usecase/auth/usecase.go
import "fishing-api/internal/infrastructure/db"  // NG

// CORRECT: usecase は domain のインターフェースのみ依存
// internal/usecase/auth/usecase.go
import "fishing-api/internal/domain/repository"  // OK
```

## レイヤー別責務

| ディレクトリ | 責務 | 持っていいもの |
|------------|------|-------------|
| `internal/domain/` | エンティティ・リポジトリ IF 定義 | struct, interface, 定数, errors |
| `internal/usecase/` | ビジネスロジック | domain の型・IF のみ |
| `internal/infrastructure/` | DB・Redis・外部 API 実装 | domain IF の実装 |
| `internal/interface/` | Gin ハンドラー・ミドルウェア・ルーター | usecase の型・IF のみ |
| `cmd/server/` | エントリポイント・DI 組み立て | 全レイヤーの import 許可 |

## DI（依存性注入）パターン

インターフェースの組み立ては `cmd/server/main.go` のみが行う。

```go
// cmd/server/main.go — ここだけが全レイヤーに依存してよい
func main() {
    db := infrastructure.NewPostgresDB(cfg.DatabaseURL)
    cache := infrastructure.NewRedisCache(cfg.RedisAddr)

    userRepo := dbimpl.NewUserRepository(db)         // infrastructure
    authUC := auth.NewAuthUsecase(userRepo, cache)   // usecase
    authHandler := handler.NewAuthHandler(authUC)    // interface

    r := router.New(authHandler)
    r.Run(":8080")
}
```

## ドメインエンティティのルール

- エンティティ（`entity/`）はビジネスの概念のみを持つ
- DB のカラム名や JSON タグをエンティティに直接書かない
- `FishingScore` のような計算結果は DB 永続化しない（値オブジェクト扱い）

```go
// WRONG: DB 依存がエンティティに混入
type User struct {
    ID    uuid.UUID `db:"id" json:"id"`  // NG: インフラの関心事
}

// CORRECT: エンティティはピュアな Go struct
type User struct {
    ID    uuid.UUID
    Email string
}
```

## 関連ルール

- [testing.md](./testing.md) — レイヤー別テスト戦略
- [go-coding-style.md](./go-coding-style.md) — インターフェース設計のスタイル
