# Go Backend Phase 2: Authentication Implementation Plan

**Goal:** JWT発行・検証、ユーザー登録・メール確認・ログイン・トークンリフレッシュ・JWTミドルウェアを実装し、認証APIが動作する状態にする。

**Architecture:** `pkg/jwt` でJWT生成/検証。`infrastructure/db` が `domain/repository` インターフェースを実装。`usecase/auth` がビジネスロジックを担当。`interface/handler` がHTTPを処理。DIは `cmd/server/main.go` のみ。

**Tech Stack:** golang-jwt/jwt v5, golang.org/x/crypto (既存indirect), resend-go/v2

**参照プラン:** `docs/superpowers/plans/2026-04-07-go-backend-phase2-auth.md`

---

## 重要な修正点（プランからの差分）

1. **sqlc型変換**: `sqlcgen.User` は `sql.NullString`/`sql.NullTime` を使用。`entity.User` は `*string`/`*time.Time`。`toUserEntity` で変換が必要。
2. **未インストールパッケージ**: `golang-jwt/jwt/v5` と `resend-go/v2` を `go get` で追加。
3. **main.go は slog 使用中**: プランの `log.Fatalf` ではなく `slog.Error` + `os.Exit(1)` を維持。
4. **sqlcgenインポートパス**: `github.com/kazumadev619-dev/fishing-api/db/generated`（パッケージ名: `sqlcgen`）。
5. **CreateUserParams**: `PasswordHash` と `Name` は `sql.NullString` 型。
6. **UpdateUserEmailVerifiedParams**: `EmailVerifiedAt` は `sql.NullTime` 型。

---

## ファイル構成

| 操作 | ファイル |
|------|---------|
| 新規作成 | `pkg/jwt/jwt.go` |
| 新規作成 | `pkg/jwt/jwt_test.go` |
| 新規作成 | `pkg/validator/validator.go` |
| 新規作成 | `pkg/validator/validator_test.go` |
| 新規作成 | `internal/infrastructure/db/user_repository.go` |
| 新規作成 | `internal/infrastructure/db/verification_token_repository.go` |
| 新規作成 | `internal/infrastructure/email/email.go` |
| 新規作成 | `internal/usecase/auth/auth.go` |
| 新規作成 | `internal/usecase/auth/auth_test.go` |
| 新規作成 | `internal/interface/handler/auth_handler.go` |
| 新規作成 | `internal/interface/handler/auth_handler_test.go` |
| 新規作成 | `internal/interface/middleware/auth.go` |
| 新規作成 | `internal/interface/middleware/auth_test.go` |
| 変更 | `internal/interface/router/router.go` |
| 変更 | `cmd/server/main.go` |

---

## Task 1: 依存パッケージ追加

- [ ] **Step 1: jwt と resend を go get**

```bash
go get github.com/golang-jwt/jwt/v5@latest
go get github.com/resend/resend-go/v2@latest
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./...
```

Expected: エラーなし

- [ ] **Step 3: コミット**

```bash
git add go.mod go.sum
git commit -m "chore: golang-jwt/jwt v5 と resend-go/v2 を追加"
```

---

## Task 2: JWT生成・検証パッケージ

**Files:**
- Create: `pkg/jwt/jwt.go`
- Create: `pkg/jwt/jwt_test.go`

- [ ] **Step 1: テストを書く**

プラン `Task 1: Step 1` のコードをそのまま使用（変更不要）。

```go
// pkg/jwt/jwt_test.go
package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	manager := NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")
	userID := uuid.New()

	token, err := manager.GenerateAccessToken(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), claims.ExpiresAt.Time, 5*time.Second)
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	manager := NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")
	userID := uuid.New()

	token, err := manager.GenerateRefreshToken(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	manager := NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")
	_, err := manager.ValidateAccessToken("invalid.token.here")
	assert.Error(t, err)
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	manager1 := NewManager("secret-one-32chars-minimum!!!!", "refresh-secret")
	manager2 := NewManager("secret-two-32chars-minimum!!!!", "refresh-secret")
	userID := uuid.New()

	token, err := manager1.GenerateAccessToken(userID)
	require.NoError(t, err)

	_, err = manager2.ValidateAccessToken(token)
	assert.Error(t, err)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./pkg/jwt/... -v
```

Expected: FAIL（パッケージが存在しない）

- [ ] **Step 3: 実装**

プランのコードをそのまま使用（変更不要）。

```go
// pkg/jwt/jwt.go
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

type Manager struct {
	accessSecret  string
	refreshSecret string
}

func NewManager(accessSecret, refreshSecret string) *Manager {
	return &Manager{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
	}
}

func (m *Manager) GenerateAccessToken(userID uuid.UUID) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.accessSecret))
}

func (m *Manager) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.refreshSecret))
}

func (m *Manager) ValidateAccessToken(tokenStr string) (*Claims, error) {
	return m.validateToken(tokenStr, m.accessSecret)
}

func (m *Manager) ValidateRefreshToken(tokenStr string) (*Claims, error) {
	return m.validateToken(tokenStr, m.refreshSecret)
}

func (m *Manager) validateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./pkg/jwt/... -v
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add pkg/jwt/
git commit -m "feat: JWT生成・検証パッケージ追加"
```

---

## Task 3: バリデーションパッケージ

**Files:**
- Create: `pkg/validator/validator.go`
- Create: `pkg/validator/validator_test.go`

- [ ] **Step 1: テストを書く**

プラン `Task 2: Step 1` のコードをそのまま使用。

```go
// pkg/validator/validator_test.go
package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidEmail(t *testing.T) {
	assert.True(t, IsValidEmail("user@example.com"))
	assert.True(t, IsValidEmail("user+tag@example.co.jp"))
	assert.False(t, IsValidEmail("invalid"))
	assert.False(t, IsValidEmail("@example.com"))
	assert.False(t, IsValidEmail(""))
}

func TestIsValidUUID(t *testing.T) {
	assert.True(t, IsValidUUID("550e8400-e29b-41d4-a716-446655440000"))
	assert.False(t, IsValidUUID("not-a-uuid"))
	assert.False(t, IsValidUUID(""))
}

func TestRoundCoordinate(t *testing.T) {
	assert.Equal(t, 35.6895, RoundCoordinate(35.68954321, 4))
	assert.Equal(t, 139.6917, RoundCoordinate(139.69174321, 4))
}

func TestParseAndValidateCoordinates(t *testing.T) {
	lat, lon, err := ParseAndValidateCoordinates("35.6895", "139.6917")
	assert.NoError(t, err)
	assert.Equal(t, 35.6895, lat)
	assert.Equal(t, 139.6917, lon)

	_, _, err = ParseAndValidateCoordinates("", "139.6917")
	assert.Error(t, err)

	_, _, err = ParseAndValidateCoordinates("91.0", "0")
	assert.Error(t, err)

	_, _, err = ParseAndValidateCoordinates("0", "181.0")
	assert.Error(t, err)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./pkg/validator/... -v
```

Expected: FAIL

- [ ] **Step 3: 実装**

プランのコードをそのまま使用。

```go
// pkg/validator/validator.go
package validator

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	uuidRegex  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func IsValidUUID(id string) bool {
	return uuidRegex.MatchString(id)
}

func RoundCoordinate(value float64, precision int) float64 {
	p := math.Pow(10, float64(precision))
	return math.Round(value*p) / p
}

func ParseAndValidateCoordinates(latStr, lonStr string) (float64, float64, error) {
	if latStr == "" || lonStr == "" {
		return 0, 0, fmt.Errorf("lat and lon are required")
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lat: %w", err)
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lon: %w", err)
	}

	if lat < -90 || lat > 90 {
		return 0, 0, fmt.Errorf("lat must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return 0, 0, fmt.Errorf("lon must be between -180 and 180")
	}

	return lat, lon, nil
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./pkg/validator/... -v
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add pkg/validator/
git commit -m "feat: バリデーションパッケージ追加"
```

---

## Task 4: UserRepository sqlc実装

**Files:**
- Create: `internal/infrastructure/db/user_repository.go`

**注意:** sqlcgenの型は `sql.NullString`/`sql.NullTime`。`toUserEntity` で変換する。

- [ ] **Step 1: 実装**

```go
// internal/infrastructure/db/user_repository.go
package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	sqlcgen "github.com/kazumadev619-dev/fishing-api/db/generated"
)

type userRepository struct {
	queries *sqlcgen.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *userRepository {
	return &userRepository{queries: sqlcgen.New(pool)}
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	row, err := r.queries.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toUserEntity(row), nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	row, err := r.queries.FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toUserEntity(row), nil
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) (*entity.User, error) {
	var passwordHash sql.NullString
	if user.PasswordHash != nil {
		passwordHash = sql.NullString{String: *user.PasswordHash, Valid: true}
	}
	var name sql.NullString
	if user.Name != nil {
		name = sql.NullString{String: *user.Name, Valid: true}
	}

	row, err := r.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: passwordHash,
		Name:         name,
		IsSsoUser:    user.IsSSO,
	})
	if err != nil {
		return nil, err
	}
	return toUserEntity(row), nil
}

func (r *userRepository) UpdateEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) (*entity.User, error) {
	row, err := r.queries.UpdateUserEmailVerified(ctx, sqlcgen.UpdateUserEmailVerifiedParams{
		ID:              id,
		EmailVerifiedAt: sql.NullTime{Time: verifiedAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return toUserEntity(row), nil
}

func toUserEntity(row sqlcgen.User) *entity.User {
	u := &entity.User{
		ID:        row.ID,
		Email:     row.Email,
		IsSSO:     row.IsSsoUser,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if row.PasswordHash.Valid {
		u.PasswordHash = &row.PasswordHash.String
	}
	if row.Name.Valid {
		u.Name = &row.Name.String
	}
	if row.AvatarUrl.Valid {
		u.AvatarURL = &row.AvatarUrl.String
	}
	if row.EmailVerifiedAt.Valid {
		u.EmailVerifiedAt = &row.EmailVerifiedAt.Time
	}
	return u
}
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./internal/infrastructure/db/...
```

Expected: エラーなし

- [ ] **Step 3: コミット**

```bash
git add internal/infrastructure/db/user_repository.go
git commit -m "feat: UserRepository sqlc実装追加"
```

---

## Task 5: VerificationTokenRepository sqlc実装

**Files:**
- Create: `internal/infrastructure/db/verification_token_repository.go`

- [ ] **Step 1: 実装**

プラン `Task 4: Step 1` のコードをそのまま使用（型は一致している）。

```go
// internal/infrastructure/db/verification_token_repository.go
package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	sqlcgen "github.com/kazumadev619-dev/fishing-api/db/generated"
)

type verificationTokenRepository struct {
	queries *sqlcgen.Queries
}

func NewVerificationTokenRepository(pool *pgxpool.Pool) *verificationTokenRepository {
	return &verificationTokenRepository{queries: sqlcgen.New(pool)}
}

func (r *verificationTokenRepository) Create(ctx context.Context, token *entity.VerificationToken) (*entity.VerificationToken, error) {
	row, err := r.queries.CreateVerificationToken(ctx, sqlcgen.CreateVerificationTokenParams{
		ID:        token.ID,
		Email:     token.Email,
		Token:     token.Token,
		ExpiresAt: token.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}
	return &entity.VerificationToken{
		ID:        row.ID,
		Email:     row.Email,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *verificationTokenRepository) FindByToken(ctx context.Context, token string) (*entity.VerificationToken, error) {
	row, err := r.queries.FindVerificationToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &entity.VerificationToken{
		ID:        row.ID,
		Email:     row.Email,
		Token:     row.Token,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *verificationTokenRepository) DeleteByEmail(ctx context.Context, email string) error {
	return r.queries.DeleteVerificationTokensByEmail(ctx, email)
}
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./internal/infrastructure/db/...
```

Expected: エラーなし

- [ ] **Step 3: コミット**

```bash
git add internal/infrastructure/db/verification_token_repository.go
git commit -m "feat: VerificationTokenRepository sqlc実装追加"
```

---

## Task 6: メール送信（Resend API）

**Files:**
- Create: `internal/infrastructure/email/email.go`

- [ ] **Step 1: 実装**

プラン `Task 5: Step 2` のコードをそのまま使用。

```go
// internal/infrastructure/email/email.go
package email

import (
	"fmt"

	"github.com/resend/resend-go/v2"
)

type EmailClient struct {
	client      *resend.Client
	fromAddress string
}

func NewEmailClient(apiKey, fromAddress string) *EmailClient {
	return &EmailClient{
		client:      resend.NewClient(apiKey),
		fromAddress: fromAddress,
	}
}

func (e *EmailClient) SendVerificationEmail(toEmail, token, appBaseURL string) error {
	verifyURL := fmt.Sprintf("%s/api/auth/verify-email?token=%s", appBaseURL, token)

	params := &resend.SendEmailRequest{
		From:    e.fromAddress,
		To:      []string{toEmail},
		Subject: "【釣りコンディションApp】メールアドレスの確認",
		Html: fmt.Sprintf(`
			<h2>メールアドレスの確認</h2>
			<p>以下のリンクをクリックしてメールアドレスを確認してください。</p>
			<p>このリンクは1時間有効です。</p>
			<a href="%s" style="background:#0066cc;color:white;padding:12px 24px;text-decoration:none;border-radius:4px;">
				メールアドレスを確認する
			</a>
			<p>リンクが機能しない場合は以下のURLをブラウザに貼り付けてください：</p>
			<p>%s</p>
		`, verifyURL, verifyURL),
	}

	_, err := e.client.Emails.Send(params)
	return err
}
```

- [ ] **Step 2: ビルド確認**

```bash
go build ./internal/infrastructure/email/...
```

Expected: エラーなし

- [ ] **Step 3: コミット**

```bash
git add internal/infrastructure/email/
git commit -m "feat: Resend APIメール送信クライアント追加"
```

---

## Task 7: 認証ユースケース実装

**Files:**
- Create: `internal/usecase/auth/auth.go`
- Create: `internal/usecase/auth/auth_test.go`

- [ ] **Step 1: テストを書く**

プラン `Task 6: Step 2` のコードをそのまま使用。

```go
// internal/usecase/auth/auth_test.go
package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	jwtpkg "github.com/kazumadev619-dev/fishing-api/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockUserRepository struct{ mock.Mock }

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}
func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}
func (m *MockUserRepository) Create(ctx context.Context, user *entity.User) (*entity.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}
func (m *MockUserRepository) UpdateEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) (*entity.User, error) {
	args := m.Called(ctx, id, verifiedAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

type MockVerificationTokenRepository struct{ mock.Mock }

func (m *MockVerificationTokenRepository) Create(ctx context.Context, token *entity.VerificationToken) (*entity.VerificationToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.VerificationToken), args.Error(1)
}
func (m *MockVerificationTokenRepository) FindByToken(ctx context.Context, token string) (*entity.VerificationToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.VerificationToken), args.Error(1)
}
func (m *MockVerificationTokenRepository) DeleteByEmail(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

type MockEmailClient struct{ mock.Mock }

func (m *MockEmailClient) SendVerificationEmail(toEmail, token, appBaseURL string) error {
	args := m.Called(toEmail, token, appBaseURL)
	return args.Error(0)
}

func TestAuthUsecase_Login_WrongPassword(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	jwtManager := jwtpkg.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")

	// bcryptで生成したハッシュ（"correctpassword"のハッシュ）
	hash := "$2a$10$somehashedpassword..."
	name := "Test User"
	userID := uuid.New()
	verifiedAt := time.Now()
	user := &entity.User{
		ID:              userID,
		Email:           "test@example.com",
		PasswordHash:    &hash,
		Name:            &name,
		EmailVerifiedAt: &verifiedAt,
	}

	userRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, jwtManager, "http://localhost:3000")
	_, err := uc.Login(context.Background(), "test@example.com", "wrongpassword")
	assert.Error(t, err)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_Register_DuplicateEmail(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	jwtManager := jwtpkg.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")

	name := "Existing User"
	existingUser := &entity.User{ID: uuid.New(), Email: "exists@example.com", Name: &name}
	userRepo.On("FindByEmail", mock.Anything, "exists@example.com").Return(existingUser, nil)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, jwtManager, "http://localhost:3000")
	err := uc.Register(context.Background(), "exists@example.com", "password123", "New User")
	assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_VerifyEmail_InvalidToken(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	jwtManager := jwtpkg.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")

	tokenRepo.On("FindByToken", mock.Anything, "invalid-token").Return(nil, domain.ErrNotFound)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, jwtManager, "http://localhost:3000")
	err := uc.VerifyEmail(context.Background(), "invalid-token")
	assert.ErrorIs(t, err, domain.ErrInvalidToken)
	tokenRepo.AssertExpectations(t)
}

func TestAuthUsecase_Register_NewUser(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	jwtManager := jwtpkg.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")

	userRepo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, domain.ErrNotFound)
	name := "New User"
	newUser := &entity.User{ID: uuid.New(), Email: "new@example.com", Name: &name}
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(newUser, nil)
	tokenRepo.On("DeleteByEmail", mock.Anything, "new@example.com").Return(nil)
	tokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.VerificationToken")).Return(
		&entity.VerificationToken{ID: uuid.New(), Email: "new@example.com", Token: "tok", ExpiresAt: time.Now().Add(time.Hour)},
		nil,
	)
	emailClient.On("SendVerificationEmail", "new@example.com", mock.AnythingOfType("string"), "http://localhost:3000").Return(nil)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, jwtManager, "http://localhost:3000")
	err := uc.Register(context.Background(), "new@example.com", "password123", "New User")
	require.NoError(t, err)
	userRepo.AssertExpectations(t)
	tokenRepo.AssertExpectations(t)
	emailClient.AssertExpectations(t)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/usecase/auth/... -v
```

Expected: FAIL

- [ ] **Step 3: 実装**

プラン `Task 6: Step 4` のコードをそのまま使用。

```go
// internal/usecase/auth/auth.go
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/repository"
	jwtpkg "github.com/kazumadev619-dev/fishing-api/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type EmailSender interface {
	SendVerificationEmail(toEmail, token, appBaseURL string) error
}

type AuthUsecase struct {
	userRepo    repository.UserRepository
	tokenRepo   repository.VerificationTokenRepository
	emailSender EmailSender
	jwtManager  *jwtpkg.Manager
	appBaseURL  string
}

func NewAuthUsecase(
	userRepo repository.UserRepository,
	tokenRepo repository.VerificationTokenRepository,
	emailSender EmailSender,
	jwtManager *jwtpkg.Manager,
	appBaseURL string,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		emailSender: emailSender,
		jwtManager:  jwtManager,
		appBaseURL:  appBaseURL,
	}
}

func (a *AuthUsecase) Register(ctx context.Context, email, password, name string) error {
	existing, err := a.userRepo.FindByEmail(ctx, email)
	if err != nil && err != domain.ErrNotFound {
		return err
	}
	if existing != nil {
		return domain.ErrAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	hashStr := string(hash)
	user := &entity.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: &hashStr,
		Name:         &name,
		IsSSO:        false,
	}

	created, err := a.userRepo.Create(ctx, user)
	if err != nil {
		return err
	}

	return a.sendVerificationEmail(ctx, created.Email)
}

func (a *AuthUsecase) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := a.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	if user.PasswordHash == nil {
		return nil, domain.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	if user.EmailVerifiedAt == nil {
		return nil, domain.ErrUnauthorized
	}

	return a.generateTokenPair(user.ID)
}

func (a *AuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := a.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	_, err = a.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	return a.generateTokenPair(claims.UserID)
}

func (a *AuthUsecase) VerifyEmail(ctx context.Context, token string) error {
	vToken, err := a.tokenRepo.FindByToken(ctx, token)
	if err != nil {
		return domain.ErrInvalidToken
	}

	if time.Now().After(vToken.ExpiresAt) {
		return domain.ErrInvalidToken
	}

	user, err := a.userRepo.FindByEmail(ctx, vToken.Email)
	if err != nil {
		return err
	}

	now := time.Now()
	if _, err := a.userRepo.UpdateEmailVerified(ctx, user.ID, now); err != nil {
		return err
	}

	return a.tokenRepo.DeleteByEmail(ctx, vToken.Email)
}

func (a *AuthUsecase) sendVerificationEmail(ctx context.Context, email string) error {
	if err := a.tokenRepo.DeleteByEmail(ctx, email); err != nil {
		return err
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	tokenStr := hex.EncodeToString(tokenBytes)

	vToken := &entity.VerificationToken{
		ID:        uuid.New(),
		Email:     email,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if _, err := a.tokenRepo.Create(ctx, vToken); err != nil {
		return err
	}

	return a.emailSender.SendVerificationEmail(email, tokenStr, a.appBaseURL)
}

func (a *AuthUsecase) generateTokenPair(userID uuid.UUID) (*TokenPair, error) {
	accessToken, err := a.jwtManager.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := a.jwtManager.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/usecase/auth/... -v
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add internal/usecase/auth/
git commit -m "feat: 認証ユースケース実装（登録・ログイン・メール確認・リフレッシュ）"
```

---

## Task 8: JWTミドルウェア

**Files:**
- Create: `internal/interface/middleware/auth.go`
- Create: `internal/interface/middleware/auth_test.go`

- [ ] **Step 1: テストを書く**

プラン `Task 7: Step 1` のコードをそのまま使用。

```go
// internal/interface/middleware/auth_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	jwtpkg "github.com/kazumadev619-dev/fishing-api/pkg/jwt"
	"github.com/stretchr/testify/assert"
)

func TestJWTAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := jwtpkg.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")
	userID := uuid.New()

	token, _ := manager.GenerateAccessToken(userID)

	router := gin.New()
	router.Use(JWTAuth(manager))
	router.GET("/protected", func(c *gin.Context) {
		id := c.MustGet("userID").(uuid.UUID)
		c.JSON(http.StatusOK, gin.H{"user_id": id})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTAuth_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := jwtpkg.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")

	router := gin.New()
	router.Use(JWTAuth(manager))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := jwtpkg.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")

	router := gin.New()
	router.Use(JWTAuth(manager))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/interface/middleware/... -v
```

Expected: FAIL

- [ ] **Step 3: 実装**

プラン `Task 7: Step 3` のコードをそのまま使用。

```go
// internal/interface/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	jwtpkg "github.com/kazumadev619-dev/fishing-api/pkg/jwt"
)

func JWTAuth(jwtManager *jwtpkg.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":  "authorization header is required",
				"code":   "UNAUTHORIZED",
				"status": 401,
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":  "invalid authorization header format",
				"code":   "UNAUTHORIZED",
				"status": 401,
			})
			return
		}

		claims, err := jwtManager.ValidateAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":  "invalid or expired token",
				"code":   "UNAUTHORIZED",
				"status": 401,
			})
			return
		}

		c.Set("userID", claims.UserID)
		c.Next()
	}
}
```

- [ ] **Step 4: テストが通ることを確認**

```bash
go test ./internal/interface/middleware/... -v
```

Expected: PASS

- [ ] **Step 5: コミット**

```bash
git add internal/interface/middleware/
git commit -m "feat: JWT認証ミドルウェア追加"
```

---

## Task 9: 認証ハンドラー + ルーター + main.go 更新

**Files:**
- Create: `internal/interface/handler/auth_handler.go`
- Create: `internal/interface/handler/auth_handler_test.go`
- Modify: `internal/interface/router/router.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: テストを書く**

プラン `Task 8: Step 1` のコードをそのまま使用（`MockAuthUsecase` の ctx は `interface{}` でなく `context.Context` に変更）。

```go
// internal/interface/handler/auth_handler_test.go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAuthUsecase struct{ mock.Mock }

func (m *MockAuthUsecase) Register(ctx context.Context, email, password, name string) error {
	args := m.Called(ctx, email, password, name)
	return args.Error(0)
}
func (m *MockAuthUsecase) Login(ctx context.Context, email, password string) (*auth.TokenPair, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}
func (m *MockAuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.TokenPair), args.Error(1)
}
func (m *MockAuthUsecase) VerifyEmail(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := &MockAuthUsecase{}
	mockUC.On("Register", mock.Anything, "new@example.com", "password123", "New User").Return(nil)

	router := gin.New()
	h := NewAuthHandler(mockUC)
	router.POST("/api/auth/register", h.Register)

	body, _ := json.Marshal(map[string]string{
		"email":    "new@example.com",
		"password": "password123",
		"name":     "New User",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_Register_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewAuthHandler(&MockAuthUsecase{})
	router.POST("/api/auth/register", h.Register)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
```

- [ ] **Step 2: テストが失敗することを確認**

```bash
go test ./internal/interface/handler/... -v -run TestAuthHandler
```

Expected: FAIL

- [ ] **Step 3: auth_handler.go を実装する**

プラン `Task 8: Step 3` のコードをそのまま使用。

```go
// internal/interface/handler/auth_handler.go
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/auth"
)

type AuthUsecaseInterface interface {
	Register(ctx context.Context, email, password, name string) error
	Login(ctx context.Context, email, password string) (*auth.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error)
	VerifyEmail(ctx context.Context, token string) error
}

type AuthHandler struct {
	usecase AuthUsecaseInterface
}

func NewAuthHandler(uc AuthUsecaseInterface) *AuthHandler {
	return &AuthHandler{usecase: uc}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "INVALID_PARAMS", "status": 400})
		return
	}

	if err := h.usecase.Register(c.Request.Context(), req.Email, req.Password, req.Name); err != nil {
		if err == domain.ErrAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered", "code": "ALREADY_EXISTS", "status": 409})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "registration successful. please check your email."})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "INVALID_PARAMS", "status": 400})
		return
	}

	tokens, err := h.usecase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err == domain.ErrUnauthorized {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials", "code": "UNAUTHORIZED", "status": 401})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "INVALID_PARAMS", "status": 400})
		return
	}

	tokens, err := h.usecase.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token", "code": "UNAUTHORIZED", "status": 401})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required", "code": "INVALID_PARAMS", "status": 400})
		return
	}

	if err := h.usecase.VerifyEmail(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token", "code": "INVALID_TOKEN", "status": 400})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email verified successfully"})
}
```

- [ ] **Step 4: router.go を更新する**

現在の `internal/interface/router/router.go` を以下に置き換える。

```go
// internal/interface/router/router.go
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/middleware"
	jwtpkg "github.com/kazumadev619-dev/fishing-api/pkg/jwt"
)

type Handlers struct {
	Auth *handler.AuthHandler
}

func New(handlers *Handlers, jwtManager *jwtpkg.Manager) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/health", handler.HealthCheck)

	api := r.Group("/api")
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", handlers.Auth.Register)
			authGroup.POST("/login", handlers.Auth.Login)
			authGroup.POST("/refresh", handlers.Auth.RefreshToken)
			authGroup.GET("/verify-email", handlers.Auth.VerifyEmail)
		}

		// 認証が必要なルート（Phase 3以降で追加）
		_ = api.Group("").Use(middleware.JWTAuth(jwtManager))
	}

	return r
}
```

- [ ] **Step 5: main.go を更新する（slogを維持）**

現在の `cmd/server/main.go` を以下に置き換える（`slog` を維持）。

```go
// cmd/server/main.go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/kazumadev619-dev/fishing-api/config"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/cache"
	infradb "github.com/kazumadev619-dev/fishing-api/internal/infrastructure/db"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/email"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/router"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/auth"
	jwtpkg "github.com/kazumadev619-dev/fishing-api/pkg/jwt"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := infradb.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	cacheClient, err := cache.NewCacheClient(ctx, cfg.Redis.URL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	_ = cacheClient // Phase 3以降で使用

	// JWT
	jwtManager := jwtpkg.NewManager(cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret)

	// Repositories
	userRepo := infradb.NewUserRepository(pool)
	tokenRepo := infradb.NewVerificationTokenRepository(pool)

	// Infrastructure
	emailClient := email.NewEmailClient(cfg.Email.ResendAPIKey, cfg.Email.FromAddress)

	// Usecases
	authUC := auth.NewAuthUsecase(userRepo, tokenRepo, emailClient, jwtManager, cfg.Server.AppBaseURL())

	// Handlers
	handlers := &router.Handlers{
		Auth: handler.NewAuthHandler(authUC),
	}

	r := router.New(handlers, jwtManager)

	slog.Info("server starting", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
```

**注意:** `cfg.Server.AppBaseURL()` が存在しない場合は `"http://localhost:3000"` のリテラルを使用する（または `config.go` に `AppBaseURL string` フィールドを追加）。

- [ ] **Step 6: config.go に APP_BASE_URL を追加（必要な場合）**

`config/config.go` の `ServerConfig` に `AppBaseURL` を追加：

```go
type ServerConfig struct {
	Port       string
	AppBaseURL string
}
```

`Load()` 内：
```go
Server: ServerConfig{
    Port:       getEnv("PORT", "8080"),
    AppBaseURL: getEnv("APP_BASE_URL", "http://localhost:3000"),
},
```

`main.go` で使用：
```go
authUC := auth.NewAuthUsecase(userRepo, tokenRepo, emailClient, jwtManager, cfg.Server.AppBaseURL)
```

- [ ] **Step 7: テストが通ることを確認**

```bash
go test ./internal/interface/handler/... -v -run TestAuthHandler
go test ./internal/interface/middleware/... -v
```

Expected: PASS

- [ ] **Step 8: ビルド確認**

```bash
go build ./...
```

Expected: エラーなし

- [ ] **Step 9: コミット**

```bash
git add internal/interface/handler/ internal/interface/middleware/ internal/interface/router/router.go cmd/server/main.go config/config.go
git commit -m "feat: 認証ハンドラー・ミドルウェア・ルーター・main.go 更新"
```

---

## 検証

```bash
# 全テスト実行
go test ./... -v

# サーバー起動
export $(cat .env | xargs) && go run ./cmd/server/

# 別ターミナルで動作確認
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","name":"Test User"}'

# Expected: {"message":"registration successful. please check your email."}

curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Expected: {"access_token":"...","refresh_token":"..."}
```
