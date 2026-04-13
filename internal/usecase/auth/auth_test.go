package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// --- Mocks ---

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

type MockJWTManager struct{ mock.Mock }

func (m *MockJWTManager) GenerateAccessToken(userID uuid.UUID) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockJWTManager) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockJWTManager) ValidateRefreshToken(tokenStr string) (uuid.UUID, error) {
	args := m.Called(tokenStr)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// --- Tests ---

func TestAuthUsecase_Login_WrongPassword(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	mockJWT := &MockJWTManager{}

	rawHash, err := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	require.NoError(t, err)
	hash := string(rawHash)
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

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, mockJWT, "http://localhost:3000")
	_, loginErr := uc.Login(context.Background(), "test@example.com", "wrongpassword")
	assert.ErrorIs(t, loginErr, domain.ErrUnauthorized)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_Login_EmailNotVerified(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	mockJWT := &MockJWTManager{}

	rawHash, err := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	require.NoError(t, err)
	hash := string(rawHash)
	name := "Test User"
	user := &entity.User{
		ID:              uuid.New(),
		Email:           "unverified@example.com",
		PasswordHash:    &hash,
		Name:            &name,
		EmailVerifiedAt: nil,
	}

	userRepo.On("FindByEmail", mock.Anything, "unverified@example.com").Return(user, nil)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, mockJWT, "http://localhost:3000")
	_, loginErr := uc.Login(context.Background(), "unverified@example.com", "correctpassword")
	assert.ErrorIs(t, loginErr, domain.ErrEmailNotVerified)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_Register_DuplicateEmail(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	mockJWT := &MockJWTManager{}

	name := "Existing User"
	existingUser := &entity.User{ID: uuid.New(), Email: "exists@example.com", Name: &name}
	userRepo.On("FindByEmail", mock.Anything, "exists@example.com").Return(existingUser, nil)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, mockJWT, "http://localhost:3000")
	err := uc.Register(context.Background(), "exists@example.com", "password123", "New User")
	assert.ErrorIs(t, err, domain.ErrAlreadyExists)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_VerifyEmail_InvalidToken(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	mockJWT := &MockJWTManager{}

	tokenRepo.On("FindByToken", mock.Anything, "invalid-token").Return(nil, domain.ErrNotFound)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, mockJWT, "http://localhost:3000")
	err := uc.VerifyEmail(context.Background(), "invalid-token")
	assert.ErrorIs(t, err, domain.ErrInvalidToken)
	tokenRepo.AssertExpectations(t)
}

func TestAuthUsecase_VerifyEmail_AlreadyVerified(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	mockJWT := &MockJWTManager{}

	verifiedAt := time.Now().Add(-1 * time.Hour)
	user := &entity.User{
		ID:              uuid.New(),
		Email:           "verified@example.com",
		EmailVerifiedAt: &verifiedAt,
	}
	vToken := &entity.VerificationToken{
		Email:     "verified@example.com",
		Token:     "some-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	tokenRepo.On("FindByToken", mock.Anything, "some-token").Return(vToken, nil)
	userRepo.On("FindByEmail", mock.Anything, "verified@example.com").Return(user, nil)
	tokenRepo.On("DeleteByEmail", mock.Anything, "verified@example.com").Return(nil)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, mockJWT, "http://localhost:3000")
	err := uc.VerifyEmail(context.Background(), "some-token")
	require.NoError(t, err)
	// UpdateEmailVerified は呼ばれないことを確認（冪等性）
	userRepo.AssertNotCalled(t, "UpdateEmailVerified", mock.Anything, mock.Anything, mock.Anything)
	tokenRepo.AssertExpectations(t)
}

func TestAuthUsecase_Register_NewUser(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	mockJWT := &MockJWTManager{}

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

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, mockJWT, "http://localhost:3000")
	err := uc.Register(context.Background(), "new@example.com", "password123", "New User")
	require.NoError(t, err)
	userRepo.AssertExpectations(t)
	tokenRepo.AssertExpectations(t)
	emailClient.AssertExpectations(t)
}

func TestAuthUsecase_RefreshToken_Success(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	mockJWT := &MockJWTManager{}

	userID := uuid.New()
	name := "Test User"
	user := &entity.User{ID: userID, Email: "test@example.com", Name: &name}

	mockJWT.On("ValidateRefreshToken", "valid-refresh-token").Return(userID, nil)
	mockJWT.On("GenerateAccessToken", userID).Return("new-access-token", nil)
	mockJWT.On("GenerateRefreshToken", userID).Return("new-refresh-token", nil)
	userRepo.On("FindByID", mock.Anything, userID).Return(user, nil)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, mockJWT, "http://localhost:3000")
	pair, err := uc.RefreshToken(context.Background(), "valid-refresh-token")
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", pair.AccessToken)
	mockJWT.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_RefreshToken_InvalidToken(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	mockJWT := &MockJWTManager{}

	mockJWT.On("ValidateRefreshToken", "bad-token").Return(uuid.Nil, errors.New("invalid"))

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, mockJWT, "http://localhost:3000")
	_, err := uc.RefreshToken(context.Background(), "bad-token")
	assert.ErrorIs(t, err, domain.ErrInvalidToken)
	mockJWT.AssertExpectations(t)
}
