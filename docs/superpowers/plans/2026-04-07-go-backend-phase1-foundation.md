# Go Backend Phase 1: Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Goモジュール初期化・設定管理・ドメイン層・DB接続・Redisキャッシュ・ヘルスチェックエンドポイントを実装し、`go run ./cmd/server` でサーバーが起動できる状態にする。

**Architecture:** クリーンアーキテクチャ。`domain` 層は外側を知らない。`infrastructure` が `domain` のインターフェースを実装し、`main.go` でDIする。

**Tech Stack:** Go 1.24, Gin v1.10, sqlc v2, pgx/v5, go-redis/v9, testify

---

## ファイル構成

| 操作 | ファイル | 内容 |
|------|---------|------|
| 新規作成 | `go.mod` | Goモジュール定義・依存関係 |
| 新規作成 | `config/config.go` | 環境変数読み込み・設定構造体 |
| 新規作成 | `config/config_test.go` | 設定読み込みテスト |
| 新規作成 | `internal/domain/errors.go` | ドメイン共通エラー定義 |
| 新規作成 | `internal/domain/entity/user.go` | Userエンティティ |
| 新規作成 | `internal/domain/entity/location.go` | Location, Port, LocationType |
| 新規作成 | `internal/domain/entity/fishing_score.go` | FishingScore, ScoreRank |
| 新規作成 | `internal/domain/entity/weather.go` | WeatherData |
| 新規作成 | `internal/domain/entity/tide.go` | TideData, TideEvent |
| 新規作成 | `internal/domain/entity/verification_token.go` | VerificationToken |
| 新規作成 | `internal/domain/repository/user_repository.go` | UserRepository interface |
| 新規作成 | `internal/domain/repository/favorite_repository.go` | FavoriteRepository interface |
| 新規作成 | `internal/domain/repository/verification_token_repository.go` | VerificationTokenRepository interface |
| 新規作成 | `internal/domain/repository/location_repository.go` | LocationRepository interface |
| 新規作成 | `db/schema.sql` | PostgreSQLスキーマ（Prismaスキーマから変換） |
| 新規作成 | `db/queries/user.sql` | ユーザークエリ |
| 新規作成 | `db/queries/location.sql` | ロケーションクエリ |
| 新規作成 | `db/queries/favorite.sql` | お気に入りクエリ |
| 新規作成 | `db/queries/verification_token.sql` | トークンクエリ |
| 新規作成 | `sqlc.yaml` | sqlc設定 |
| 新規作成 | `internal/infrastructure/db/db.go` | PostgreSQL接続・プール管理 |
| 新規作成 | `internal/infrastructure/cache/cache.go` | Redisクライアント実装 |
| 新規作成 | `internal/infrastructure/cache/cache_test.go` | Redisクライアントテスト |
| 新規作成 | `internal/interface/handler/health.go` | ヘルスチェックハンドラー |
| 新規作成 | `internal/interface/handler/health_test.go` | ヘルスチェックテスト |
| 新規作成 | `internal/interface/router/router.go` | Ginルーター設定 |
| 新規作成 | `cmd/server/main.go` | エントリーポイント・DI組み立て |
| 新規作成 | `docker-compose.yml` | ローカル開発用PostgreSQL + Redis |
| 新規作成 | `Makefile` | よく使うコマンド集 |
| 新規作成 | `.env.example` | 環境変数テンプレート |

---

## Task 1: Goモジュール初期化と依存関係インストール

**Files:**
- Create: `go.mod`

- [ ] **Step 1: モジュール初期化**

```bash
cd /Users/nosawakazuma/Project/Fishing-api
go mod init github.com/kazumadev619-dev/fishing-api
```

Expected: `go.mod` が生成される

- [ ] **Step 2: 依存関係をインストール**

```bash
go get github.com/gin-gonic/gin@v1.10.0
go get github.com/golang-jwt/jwt/v5@v5.2.1
go get github.com/google/uuid@v1.6.0
go get github.com/jackc/pgx/v5@v5.7.2
go get github.com/redis/go-redis/v9@v9.7.0
go get golang.org/x/crypto@v0.32.0
go get golang.org/x/oauth2@v0.26.0
go get github.com/stretchr/testify@v1.10.0
```

Expected: `go.sum` が生成され、`go.mod` に依存関係が追加される

- [ ] **Step 3: ビルド確認**

```bash
go build ./...
```

Expected: エラーなし（まだファイルがないので何も起きない）

---

## Task 2: 設定管理（config/config.go）

**Files:**
- Create: `config/config.go`
- Create: `config/config_test.go`

- [ ] **Step 1: テストを書く**

```go
// config/config_test.go
package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Success(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/fishing")
	os.Setenv("JWT_ACCESS_SECRET", "access-secret-32chars-minimum!!")
	os.Setenv("JWT_REFRESH_SECRET", "refresh-secret-32chars-minimum!")
	t.Cleanup(func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("JWT_ACCESS_SECRET")
		os.Unsetenv("JWT_REFRESH_SECRET")
	})

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "postgres://localhost/fishing", cfg.Database.URL)
	assert.Equal(t, "redis://localhost:6379", cfg.Redis.URL)
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Setenv("JWT_ACCESS_SECRET", "access-secret")
	os.Setenv("JWT_REFRESH_SECRET", "refresh-secret")
	t.Cleanup(func() {
		os.Unsetenv("JWT_ACCESS_SECRET")
		os.Unsetenv("JWT_REFRESH_SECRET")
	})

	_, err := Load()
	assert.ErrorContains(t, err, "DATABASE_URL")
}

func TestLoad_CustomPort(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/fishing")
	os.Setenv("JWT_ACCESS_SECRET", "access-secret")
	os.Setenv("JWT_REFRESH_SECRET", "refresh-secret")
	os.Setenv("PORT", "9090")
	t.Cleanup(func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("JWT_ACCESS_SECRET")
		os.Unsetenv("JWT_REFRESH_SECRET")
		os.Unsetenv("PORT")
	})

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Server.Port)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./config/... -v
```

Expected: FAIL（`config` パッケージが存在しない）

- [ ] **Step 3: 実装**

```go
// config/config.go
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	External ExternalConfig
	Email    EmailConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
}

type ExternalConfig struct {
	OpenWeatherAPIKey string
	GoogleMapsAPIKey  string
}

type EmailConfig struct {
	ResendAPIKey string
	FromAddress  string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		Database: DatabaseConfig{
			URL: os.Getenv("DATABASE_URL"),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		JWT: JWTConfig{
			AccessSecret:  os.Getenv("JWT_ACCESS_SECRET"),
			RefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		},
		External: ExternalConfig{
			OpenWeatherAPIKey: os.Getenv("OPENWEATHER_API_KEY"),
			GoogleMapsAPIKey:  os.Getenv("GOOGLE_MAPS_API_KEY"),
		},
		Email: EmailConfig{
			ResendAPIKey: os.Getenv("RESEND_API_KEY"),
			FromAddress:  getEnv("EMAIL_FROM", "noreply@fishing-app.com"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":       c.Database.URL,
		"JWT_ACCESS_SECRET":  c.JWT.AccessSecret,
		"JWT_REFRESH_SECRET": c.JWT.RefreshSecret,
	}
	for key, val := range required {
		if val == "" {
			return fmt.Errorf("required environment variable not set: %s", key)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./config/... -v
```

Expected: PASS（3テスト全部通る）

- [ ] **Step 5: コミット**

```bash
git add config/
git commit -m "feat: 設定管理モジュール追加"
```

---

## Task 3: ドメインエラー定義

**Files:**
- Create: `internal/domain/errors.go`

- [ ] **Step 1: 実装**

```go
// internal/domain/errors.go
package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrInvalidToken  = errors.New("invalid or expired token")
)
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./internal/domain/...
```

Expected: エラーなし

---

## Task 4: ドメインエンティティ定義

**Files:**
- Create: `internal/domain/entity/user.go`
- Create: `internal/domain/entity/location.go`
- Create: `internal/domain/entity/fishing_score.go`
- Create: `internal/domain/entity/weather.go`
- Create: `internal/domain/entity/tide.go`
- Create: `internal/domain/entity/verification_token.go`

- [ ] **Step 1: user.go を作成する**

```go
// internal/domain/entity/user.go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID
	Email           string
	PasswordHash    *string
	Name            *string
	AvatarURL       *string
	IsSSO           bool
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
```

- [ ] **Step 2: location.go を作成する**

```go
// internal/domain/entity/location.go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type LocationType string

const (
	LocationTypeShore    LocationType = "SHORE"
	LocationTypeSurf     LocationType = "SURF"
	LocationTypePort     LocationType = "PORT"
	LocationTypeRiver    LocationType = "RIVER"
	LocationTypeLake     LocationType = "LAKE"
	LocationTypeOffshore LocationType = "OFFSHORE"
	LocationTypeOther    LocationType = "OTHER"
)

type Location struct {
	ID           uuid.UUID
	Name         string
	Latitude     float64
	Longitude    float64
	Region       *string
	Prefecture   *string
	LocationType LocationType
	PortID       *uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Port struct {
	ID             uuid.UUID
	Name           string
	PrefectureCode string
	PrefectureName *string
	PortCode       string
	Latitude       *float64
	Longitude      *float64
	CreatedAt      time.Time
}
```

- [ ] **Step 3: fishing_score.go を作成する**

```go
// internal/domain/entity/fishing_score.go
package entity

type ScoreRank string

const (
	ScoreRankExcellent ScoreRank = "excellent"
	ScoreRankGood      ScoreRank = "good"
	ScoreRankFair      ScoreRank = "fair"
	ScoreRankPoor      ScoreRank = "poor"
	ScoreRankBad       ScoreRank = "bad"
)

type FishingScore struct {
	Total        int
	Rank         ScoreRank
	TideScore    int
	WeatherScore int
	TimeScore    int
	Explanation  string
}

func GetScoreRank(score int) ScoreRank {
	switch {
	case score >= 80:
		return ScoreRankExcellent
	case score >= 60:
		return ScoreRankGood
	case score >= 40:
		return ScoreRankFair
	case score >= 20:
		return ScoreRankPoor
	default:
		return ScoreRankBad
	}
}
```

- [ ] **Step 4: weather.go を作成する**

```go
// internal/domain/entity/weather.go
package entity

import "time"

type WeatherData struct {
	Temperature float64
	FeelsLike   float64
	WindSpeed   float64
	WindDeg     int
	Pressure    float64
	Humidity    int
	Description string
	DateTime    time.Time
}
```

- [ ] **Step 5: tide.go を作成する**

```go
// internal/domain/entity/tide.go
package entity

import "time"

type TideEvent struct {
	Time   time.Time
	Height float64
}

type TideData struct {
	PortCode  string
	Date      string
	HighTides []TideEvent
	LowTides  []TideEvent
	TideType  string
}
```

- [ ] **Step 6: verification_token.go を作成する**

```go
// internal/domain/entity/verification_token.go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type VerificationToken struct {
	ID        uuid.UUID
	Email     string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}
```

- [ ] **Step 7: ビルド確認**

```bash
go build ./internal/domain/...
```

Expected: エラーなし

- [ ] **Step 8: コミット**

```bash
git add internal/domain/
git commit -m "feat: ドメインエンティティ・エラー定義追加"
```

---

## Task 5: リポジトリインターフェース定義

**Files:**
- Create: `internal/domain/repository/user_repository.go`
- Create: `internal/domain/repository/verification_token_repository.go`
- Create: `internal/domain/repository/favorite_repository.go`
- Create: `internal/domain/repository/location_repository.go`

- [ ] **Step 1: user_repository.go を作成する**

```go
// internal/domain/repository/user_repository.go
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) (*entity.User, error)
	UpdateEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) (*entity.User, error)
}
```

- [ ] **Step 2: verification_token_repository.go を作成する**

```go
// internal/domain/repository/verification_token_repository.go
package repository

import (
	"context"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type VerificationTokenRepository interface {
	Create(ctx context.Context, token *entity.VerificationToken) (*entity.VerificationToken, error)
	FindByToken(ctx context.Context, token string) (*entity.VerificationToken, error)
	DeleteByEmail(ctx context.Context, email string) error
}
```

- [ ] **Step 3: favorite_repository.go を作成する**

```go
// internal/domain/repository/favorite_repository.go
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type FavoriteRepository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error)
	Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error
	Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error
	Exists(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) (bool, error)
}
```

- [ ] **Step 4: location_repository.go を作成する**

```go
// internal/domain/repository/location_repository.go
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type LocationRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Location, error)
}
```

- [ ] **Step 5: ビルド確認**

```bash
go build ./internal/domain/...
```

Expected: エラーなし

- [ ] **Step 6: コミット**

```bash
git add internal/domain/repository/
git commit -m "feat: リポジトリインターフェース定義追加"
```

---

## Task 6: DBスキーマとsqlcクエリ定義

**Files:**
- Create: `db/schema.sql`
- Create: `db/queries/user.sql`
- Create: `db/queries/location.sql`
- Create: `db/queries/favorite.sql`
- Create: `db/queries/verification_token.sql`
- Create: `sqlc.yaml`

- [ ] **Step 1: db/schema.sql を作成する**

```sql
-- db/schema.sql
-- NOTE: 本番環境ではDBリポジトリからCI/CDで自動同期される。
-- このファイルはローカル開発・sqlcコード生成用。

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE location_type AS ENUM ('SHORE', 'SURF', 'PORT', 'RIVER', 'LAKE', 'OFFSHORE', 'OTHER');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    name VARCHAR(100),
    avatar_url VARCHAR(500),
    is_sso_user BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE identities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    identity_data JSONB,
    last_sign_in_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_id)
);

CREATE TABLE verification_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    prefecture_code VARCHAR(10) NOT NULL,
    prefecture_name VARCHAR(50),
    port_code VARCHAR(10) NOT NULL,
    latitude FLOAT,
    longitude FLOAT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(prefecture_code, port_code)
);

CREATE TABLE locations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    latitude FLOAT NOT NULL,
    longitude FLOAT NOT NULL,
    region VARCHAR(100),
    prefecture VARCHAR(50),
    location_type location_type NOT NULL DEFAULT 'OTHER',
    port_id UUID REFERENCES ports(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_favorites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, location_id)
);

CREATE TABLE user_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    default_location_id UUID REFERENCES locations(id),
    notification_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    theme VARCHAR(20) NOT NULL DEFAULT 'system',
    unit_system VARCHAR(20) NOT NULL DEFAULT 'metric',
    preferences JSONB,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- [ ] **Step 2: db/queries/user.sql を作成する**

```sql
-- db/queries/user.sql

-- name: FindUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: FindUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, name, is_sso_user)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateUserEmailVerified :one
UPDATE users
SET email_verified_at = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
```

- [ ] **Step 3: db/queries/verification_token.sql を作成する**

```sql
-- db/queries/verification_token.sql

-- name: CreateVerificationToken :one
INSERT INTO verification_tokens (id, email, token, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: FindVerificationToken :one
SELECT * FROM verification_tokens WHERE token = $1 LIMIT 1;

-- name: DeleteVerificationTokensByEmail :exec
DELETE FROM verification_tokens WHERE email = $1;

-- name: DeleteExpiredVerificationTokens :exec
DELETE FROM verification_tokens WHERE expires_at < NOW();
```

- [ ] **Step 4: db/queries/location.sql を作成する**

```sql
-- db/queries/location.sql

-- name: FindLocationByID :one
SELECT * FROM locations WHERE id = $1 LIMIT 1;
```

- [ ] **Step 5: db/queries/favorite.sql を作成する**

```sql
-- db/queries/favorite.sql

-- name: FindFavoritesByUserID :many
SELECT l.*
FROM user_favorites uf
JOIN locations l ON uf.location_id = l.id
WHERE uf.user_id = $1
ORDER BY uf.created_at DESC;

-- name: AddFavorite :exec
INSERT INTO user_favorites (id, user_id, location_id)
VALUES ($1, $2, $3);

-- name: DeleteFavorite :exec
DELETE FROM user_favorites
WHERE user_id = $1 AND location_id = $2;

-- name: FavoriteExists :one
SELECT EXISTS(
    SELECT 1 FROM user_favorites
    WHERE user_id = $1 AND location_id = $2
) AS "exists";
```

- [ ] **Step 6: sqlc.yaml を作成する**

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/schema.sql"
    gen:
      go:
        package: "sqlcgen"
        out: "db/generated/"
        emit_interface: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
```

- [ ] **Step 7: sqlcをインストールしてコード生成する**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc generate
```

Expected: `db/generated/` 配下にGoファイルが生成される（`db.go`, `models.go`, `user.sql.go` 等）

- [ ] **Step 8: ビルド確認**

```bash
go build ./db/...
```

Expected: エラーなし

- [ ] **Step 9: コミット**

```bash
git add db/ sqlc.yaml
git commit -m "feat: DBスキーマ・sqlcクエリ・生成コード追加"
```

---

## Task 7: PostgreSQL接続

**Files:**
- Create: `internal/infrastructure/db/db.go`
- Create: `docker-compose.yml`

- [ ] **Step 1: docker-compose.yml を作成する（ローカル開発用）**

```yaml
# docker-compose.yml
version: '3.8'
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_DB: fishing_dev
      POSTGRES_USER: fishing
      POSTGRES_PASSWORD: fishing_password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./db/schema.sql:/docker-entrypoint-initdb.d/schema.sql

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

- [ ] **Step 2: ローカルDBを起動する**

```bash
docker-compose up -d postgres redis
```

Expected: PostgreSQLとRedisが起動する

- [ ] **Step 3: db.go を作成する**

```go
// internal/infrastructure/db/db.go
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
```

- [ ] **Step 4: ビルド確認**

```bash
go build ./internal/infrastructure/db/...
```

Expected: エラーなし

- [ ] **Step 5: コミット**

```bash
git add internal/infrastructure/db/ docker-compose.yml
git commit -m "feat: PostgreSQL接続・ローカル開発環境追加"
```

---

## Task 8: Redisキャッシュクライアント

**Files:**
- Create: `internal/infrastructure/cache/cache.go`
- Create: `internal/infrastructure/cache/cache_test.go`

- [ ] **Step 1: テストを書く**

```go
// internal/infrastructure/cache/cache_test.go
package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: このテストはRedisが localhost:6379 で起動していることが前提。
// docker-compose up -d redis で起動すること。
func TestCacheClient_SetAndGet(t *testing.T) {
	client, err := NewCacheClient("redis://localhost:6379")
	require.NoError(t, err)

	ctx := context.Background()
	key := "test:cache:set-get"
	value := []byte(`{"test": "value"}`)

	err = client.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	result, err := client.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)

	t.Cleanup(func() { client.Delete(ctx, key) })
}

func TestCacheClient_Get_Miss(t *testing.T) {
	client, err := NewCacheClient("redis://localhost:6379")
	require.NoError(t, err)

	ctx := context.Background()
	result, err := client.Get(ctx, "test:cache:nonexistent-key")

	assert.NoError(t, err)
	assert.Nil(t, result) // キャッシュミスはnilを返す
}

func TestCacheClient_Delete(t *testing.T) {
	client, err := NewCacheClient("redis://localhost:6379")
	require.NoError(t, err)

	ctx := context.Background()
	key := "test:cache:delete"

	err = client.Set(ctx, key, []byte("value"), 1*time.Minute)
	require.NoError(t, err)

	err = client.Delete(ctx, key)
	require.NoError(t, err)

	result, err := client.Get(ctx, key)
	assert.NoError(t, err)
	assert.Nil(t, result)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/infrastructure/cache/... -v
```

Expected: FAIL（`cache` パッケージが存在しない）

- [ ] **Step 3: 実装**

```go
// internal/infrastructure/cache/cache.go
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheClient struct {
	client *redis.Client
}

func NewCacheClient(redisURL string) (*CacheClient, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &CacheClient{client: client}, nil
}

// Get はキャッシュからデータを取得する。キャッシュミスの場合は (nil, nil) を返す。
func (c *CacheClient) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache get error: %w", err)
	}
	return val, nil
}

func (c *CacheClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache set error: %w", err)
	}
	return nil
}

func (c *CacheClient) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache delete error: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/infrastructure/cache/... -v
```

Expected: PASS（3テスト全部通る）

- [ ] **Step 5: コミット**

```bash
git add internal/infrastructure/cache/
git commit -m "feat: Redisキャッシュクライアント実装"
```

---

## Task 9: ヘルスチェック + Ginルーター

**Files:**
- Create: `internal/interface/handler/health.go`
- Create: `internal/interface/handler/health_test.go`
- Create: `internal/interface/router/router.go`

- [ ] **Step 1: テストを書く**

```go
// internal/interface/handler/health_test.go
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", HealthCheck)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/interface/handler/... -v
```

Expected: FAIL

- [ ] **Step 3: health.go を実装する**

```go
// internal/interface/handler/health.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/interface/handler/... -v
```

Expected: PASS

- [ ] **Step 5: router.go を実装する**

```go
// internal/interface/router/router.go
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
)

func New() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/health", handler.HealthCheck)

	return r
}
```

- [ ] **Step 6: ビルド確認**

```bash
go build ./internal/interface/...
```

Expected: エラーなし

- [ ] **Step 7: コミット**

```bash
git add internal/interface/
git commit -m "feat: ヘルスチェックハンドラー・Ginルーター追加"
```

---

## Task 10: main.go + Makefile + .env.example

**Files:**
- Create: `cmd/server/main.go`
- Create: `Makefile`
- Create: `.env.example`

- [ ] **Step 1: main.go を作成する**

```go
// cmd/server/main.go
package main

import (
	"context"
	"log"

	"github.com/kazumadev619-dev/fishing-api/config"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/cache"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/db"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	cacheClient, err := cache.NewCacheClient(cfg.Redis.URL)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	_ = cacheClient // Phase 2以降で使用

	r := router.New()

	log.Printf("server starting on :%s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 2: .env.example を作成する**

```bash
# .env.example
PORT=8080
DATABASE_URL=postgres://fishing:fishing_password@localhost:5432/fishing_dev
REDIS_URL=redis://localhost:6379

# JWT（本番では十分に長いランダム文字列を使うこと）
JWT_ACCESS_SECRET=your-access-secret-here-minimum-32-characters
JWT_REFRESH_SECRET=your-refresh-secret-here-minimum-32-characters

# 外部API
OPENWEATHER_API_KEY=your-openweather-api-key
GOOGLE_MAPS_API_KEY=your-google-maps-api-key

# Email
RESEND_API_KEY=your-resend-api-key
EMAIL_FROM=noreply@your-domain.com
```

- [ ] **Step 3: .env を作成する（.env.exampleをコピー・値を設定する）**

```bash
cp .env.example .env
# .envを編集して実際の値を設定する
```

- [ ] **Step 4: Makefile を作成する**

```makefile
# Makefile
.PHONY: run build test lint sqlc-gen docker-up docker-down

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -v

test-short:
	go test ./... -short -v

lint:
	golangci-lint run

sqlc-gen:
	sqlc generate

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate:
	psql $$DATABASE_URL -f db/schema.sql
```

- [ ] **Step 5: .gitignore に .env を追加する**

```bash
echo ".env" >> .gitignore
echo "bin/" >> .gitignore
```

- [ ] **Step 6: サーバーを起動してヘルスチェックを確認する**

```bash
# DBとRedisが起動していることを確認
docker-compose up -d

# .envから環境変数を読み込んでサーバー起動
export $(cat .env | xargs) && make run
```

別ターミナルで確認：

```bash
curl http://localhost:8080/health
```

Expected:
```json
{"status":"ok"}
```

- [ ] **Step 7: コミット**

```bash
git add cmd/ Makefile .env.example .gitignore
git commit -m "feat: main.go・Makefile・.env.example追加、Phase 1完了"
```

---

## 完了条件チェックリスト

- [ ] `go build ./...` がエラーなし
- [ ] `go test ./...` が全テストPASS（DB・Redis接続テストを除く）
- [ ] `make run` でサーバーが起動する
- [ ] `curl http://localhost:8080/health` が `{"status":"ok"}` を返す
- [ ] `docker-compose up -d` でPostgreSQLとRedisが起動する
