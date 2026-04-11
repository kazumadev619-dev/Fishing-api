# Testing

## テスト戦略（レイヤー別）

| レイヤー | 手法 | ツール | 理由 |
|---------|------|--------|------|
| `usecase/` | モックリポジトリ | testify/mock | 外部依存なしで高速テスト |
| `infrastructure/db/` | 実 PostgreSQL コンテナ | testcontainers-go | Neon 非依存・実 DB で検証 |
| `infrastructure/external/` | HTTP モックサーバー | net/http/httptest | 外部 API 呼び出しをインターセプト |
| `interface/handler/` | ルーター統合テスト | httptest + Gin | ミドルウェア込みの動作確認 |

## カバレッジ要件

- **全体**: 80%+ 必須（CI で計測・未達はビルド失敗）
- **usecase 層**: 90%+ を目標（ビジネスロジックの核心）
- `make test` でカバレッジレポート生成

## テーブルドリブンテスト

複数ケースは必ずテーブル形式で書く。

```go
func TestAuthUsecase_Login(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        password string
        wantErr  error
    }{
        {name: "正常ログイン", email: "user@example.com", password: "pass123", wantErr: nil},
        {name: "メール不一致", email: "wrong@example.com", password: "pass123", wantErr: usecase.ErrInvalidCredentials},
        {name: "パスワード不一致", email: "user@example.com", password: "wrong", wantErr: usecase.ErrInvalidCredentials},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

## testify/mock 使用ルール

```go
// go:generate でモック生成（mockery 使用）
//go:generate mockery --name=UserRepository --output=./mocks

// テストファイルでのみモックを使用
func TestLogin(t *testing.T) {
    mockRepo := new(mocks.UserRepository)
    mockRepo.On("FindByEmail", ctx, "user@example.com").Return(&entity.User{...}, nil)

    uc := usecase.NewAuthUsecase(mockRepo)
    _, err := uc.Login(ctx, "user@example.com", "pass123")
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

- 本番コードに `mocks/` パッケージを import しない
- `AssertExpectations(t)` を必ず呼ぶ（未呼び出しのモックを検出）

## testcontainers-go 使用ルール

```go
func TestUserRepository(t *testing.T) {
    ctx := context.Background()

    // PostgreSQL コンテナ起動
    container, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:17"),
        postgres.WithDatabase("testdb"),
    )
    require.NoError(t, err)
    defer container.Terminate(ctx)  // 必須：テスト後にクリーンアップ

    connStr, _ := container.ConnectionString(ctx, "sslmode=disable")
    // ...テスト実行
}
```

- `defer container.Terminate(ctx)` を忘れない（リソースリーク防止）
- CI 環境では Docker デーモンが必要（GitHub Actions は対応済み）
- Neon（本番 DB）に直接接続するテストは書かない

## httptest + Gin 統合テスト

```go
func TestWeatherHandler_GetCurrent(t *testing.T) {
    mockUsecase := new(mocks.WeatherUsecase)
    mockUsecase.On("GetCurrent", mock.Anything, 35.68, 139.69).
        Return(&entity.WeatherData{Temperature: 20.0}, nil)

    router := gin.New()
    handler := handler.NewWeatherHandler(mockUsecase)
    router.GET("/api/weather", handler.GetCurrent)

    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/weather?lat=35.68&lon=139.69", nil)
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
}
```

## 関連ルール

- [clean-architecture.md](./clean-architecture.md) — レイヤー間の依存ルール
- [security.md](./security.md) — テストデータに本物の機密情報を使わない
