package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// verifiedUser は EmailVerifiedAt 済みのパスワードユーザーを生成する
func verifiedUser(t *testing.T, email, password string) *entity.User {
	t.Helper()
	rawHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	hash := string(rawHash)
	name := "Test User"
	verifiedAt := time.Now()
	return &entity.User{
		ID:              uuid.New(),
		Email:           email,
		PasswordHash:    &hash,
		Name:            &name,
		EmailVerifiedAt: &verifiedAt,
	}
}

func TestAuthUsecase_Login_Success(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	mockJWT := &MockJWTManager{}

	user := verifiedUser(t, "ok@example.com", "correctpassword")
	userRepo.On("FindByEmail", mock.Anything, "ok@example.com").Return(user, nil)
	mockJWT.On("GenerateAccessToken", user.ID).Return("access-tok", nil)
	mockJWT.On("GenerateRefreshToken", user.ID).Return("refresh-tok", nil)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, mockJWT, "http://localhost:3000")
	pair, err := uc.Login(context.Background(), "ok@example.com", "correctpassword")
	require.NoError(t, err)
	assert.Equal(t, "access-tok", pair.AccessToken)
	assert.Equal(t, "refresh-tok", pair.RefreshToken)
	userRepo.AssertExpectations(t)
	mockJWT.AssertExpectations(t)
}

func TestAuthUsecase_Login_UserNotFound(t *testing.T) {
	userRepo := &MockUserRepository{}
	userRepo.On("FindByEmail", mock.Anything, "missing@example.com").Return(nil, domain.ErrNotFound)

	uc := NewAuthUsecase(userRepo, &MockVerificationTokenRepository{}, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	_, err := uc.Login(context.Background(), "missing@example.com", "whatever")
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_Login_RepoError(t *testing.T) {
	userRepo := &MockUserRepository{}
	repoErr := errors.New("db down")
	userRepo.On("FindByEmail", mock.Anything, "err@example.com").Return(nil, repoErr)

	uc := NewAuthUsecase(userRepo, &MockVerificationTokenRepository{}, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	_, err := uc.Login(context.Background(), "err@example.com", "whatever")
	assert.ErrorIs(t, err, repoErr)
	assert.NotErrorIs(t, err, domain.ErrUnauthorized)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_Login_SSOUserNoPassword(t *testing.T) {
	userRepo := &MockUserRepository{}
	verifiedAt := time.Now()
	ssoUser := &entity.User{
		ID:              uuid.New(),
		Email:           "sso@example.com",
		PasswordHash:    nil, // SSO ユーザーはパスワードを持たない
		IsSSO:           true,
		EmailVerifiedAt: &verifiedAt,
	}
	userRepo.On("FindByEmail", mock.Anything, "sso@example.com").Return(ssoUser, nil)

	uc := NewAuthUsecase(userRepo, &MockVerificationTokenRepository{}, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	_, err := uc.Login(context.Background(), "sso@example.com", "anything")
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_Login_TokenGenerationError(t *testing.T) {
	userRepo := &MockUserRepository{}
	mockJWT := &MockJWTManager{}
	user := verifiedUser(t, "tokfail@example.com", "correctpassword")
	tokenErr := errors.New("signing failed")

	userRepo.On("FindByEmail", mock.Anything, "tokfail@example.com").Return(user, nil)
	mockJWT.On("GenerateAccessToken", user.ID).Return("", tokenErr)

	uc := NewAuthUsecase(userRepo, &MockVerificationTokenRepository{}, &MockEmailClient{}, mockJWT, "http://localhost:3000")
	_, err := uc.Login(context.Background(), "tokfail@example.com", "correctpassword")
	assert.ErrorIs(t, err, tokenErr)
	userRepo.AssertExpectations(t)
	mockJWT.AssertExpectations(t)
}

func TestAuthUsecase_Register_FindError(t *testing.T) {
	userRepo := &MockUserRepository{}
	repoErr := errors.New("connection reset")
	userRepo.On("FindByEmail", mock.Anything, "x@example.com").Return(nil, repoErr)

	uc := NewAuthUsecase(userRepo, &MockVerificationTokenRepository{}, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	err := uc.Register(context.Background(), "x@example.com", "password123", "X")
	assert.ErrorIs(t, err, repoErr)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_Register_CreateError(t *testing.T) {
	userRepo := &MockUserRepository{}
	createErr := errors.New("unique violation")
	userRepo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, domain.ErrNotFound)
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil, createErr)

	uc := NewAuthUsecase(userRepo, &MockVerificationTokenRepository{}, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	err := uc.Register(context.Background(), "new@example.com", "password123", "New")
	assert.ErrorIs(t, err, createErr)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_Register_EmailSendError(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	sendErr := errors.New("resend 500")

	name := "New"
	userRepo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, domain.ErrNotFound)
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).
		Return(&entity.User{ID: uuid.New(), Email: "new@example.com", Name: &name}, nil)
	tokenRepo.On("DeleteByEmail", mock.Anything, "new@example.com").Return(nil)
	tokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.VerificationToken")).
		Return(&entity.VerificationToken{ID: uuid.New(), Email: "new@example.com"}, nil)
	emailClient.On("SendVerificationEmail", "new@example.com", mock.AnythingOfType("string"), "http://localhost:3000").Return(sendErr)

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, &MockJWTManager{}, "http://localhost:3000")
	err := uc.Register(context.Background(), "new@example.com", "password123", "New")
	assert.ErrorIs(t, err, sendErr)
	emailClient.AssertExpectations(t)
}

func TestAuthUsecase_Register_DeleteOldTokenError(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	delErr := errors.New("delete failed")

	name := "New"
	userRepo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, domain.ErrNotFound)
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).
		Return(&entity.User{ID: uuid.New(), Email: "new@example.com", Name: &name}, nil)
	tokenRepo.On("DeleteByEmail", mock.Anything, "new@example.com").Return(delErr)

	uc := NewAuthUsecase(userRepo, tokenRepo, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	err := uc.Register(context.Background(), "new@example.com", "password123", "New")
	assert.ErrorIs(t, err, delErr)
	tokenRepo.AssertExpectations(t)
}

func TestAuthUsecase_Register_TokenCreateError(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	createErr := errors.New("token insert failed")

	name := "New"
	userRepo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, domain.ErrNotFound)
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).
		Return(&entity.User{ID: uuid.New(), Email: "new@example.com", Name: &name}, nil)
	tokenRepo.On("DeleteByEmail", mock.Anything, "new@example.com").Return(nil)
	tokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.VerificationToken")).Return(nil, createErr)

	uc := NewAuthUsecase(userRepo, tokenRepo, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	err := uc.Register(context.Background(), "new@example.com", "password123", "New")
	assert.ErrorIs(t, err, createErr)
	tokenRepo.AssertExpectations(t)
}

func TestAuthUsecase_RefreshToken_UserNotFound(t *testing.T) {
	userRepo := &MockUserRepository{}
	mockJWT := &MockJWTManager{}
	userID := uuid.New()

	mockJWT.On("ValidateRefreshToken", "valid-token").Return(userID, nil)
	userRepo.On("FindByID", mock.Anything, userID).Return(nil, domain.ErrNotFound)

	uc := NewAuthUsecase(userRepo, &MockVerificationTokenRepository{}, &MockEmailClient{}, mockJWT, "http://localhost:3000")
	_, err := uc.RefreshToken(context.Background(), "valid-token")
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
	mockJWT.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_VerifyEmail_Expired(t *testing.T) {
	tokenRepo := &MockVerificationTokenRepository{}
	vToken := &entity.VerificationToken{
		Email:     "user@example.com",
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Minute), // 期限切れ
	}
	tokenRepo.On("FindByToken", mock.Anything, "expired-token").Return(vToken, nil)

	uc := NewAuthUsecase(&MockUserRepository{}, tokenRepo, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	err := uc.VerifyEmail(context.Background(), "expired-token")
	assert.ErrorIs(t, err, domain.ErrInvalidToken)
	tokenRepo.AssertExpectations(t)
}

func TestAuthUsecase_VerifyEmail_FindUserError(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	findErr := errors.New("db error")

	vToken := &entity.VerificationToken{Email: "user@example.com", Token: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	tokenRepo.On("FindByToken", mock.Anything, "tok").Return(vToken, nil)
	userRepo.On("FindByEmail", mock.Anything, "user@example.com").Return(nil, findErr)

	uc := NewAuthUsecase(userRepo, tokenRepo, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	err := uc.VerifyEmail(context.Background(), "tok")
	assert.ErrorIs(t, err, findErr)
	userRepo.AssertExpectations(t)
	tokenRepo.AssertExpectations(t)
}

func TestAuthUsecase_VerifyEmail_UpdateError(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	updateErr := errors.New("update failed")
	userID := uuid.New()

	user := &entity.User{ID: userID, Email: "user@example.com", EmailVerifiedAt: nil}
	vToken := &entity.VerificationToken{Email: "user@example.com", Token: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	tokenRepo.On("FindByToken", mock.Anything, "tok").Return(vToken, nil)
	userRepo.On("FindByEmail", mock.Anything, "user@example.com").Return(user, nil)
	userRepo.On("UpdateEmailVerified", mock.Anything, userID, mock.Anything).Return(nil, updateErr)

	uc := NewAuthUsecase(userRepo, tokenRepo, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	err := uc.VerifyEmail(context.Background(), "tok")
	assert.ErrorIs(t, err, updateErr)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_VerifyEmail_DeleteTokenError(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	delErr := errors.New("delete failed")
	userID := uuid.New()

	user := &entity.User{ID: userID, Email: "user@example.com", EmailVerifiedAt: nil}
	vToken := &entity.VerificationToken{Email: "user@example.com", Token: "tok", ExpiresAt: time.Now().Add(time.Hour)}
	tokenRepo.On("FindByToken", mock.Anything, "tok").Return(vToken, nil)
	userRepo.On("FindByEmail", mock.Anything, "user@example.com").Return(user, nil)
	userRepo.On("UpdateEmailVerified", mock.Anything, userID, mock.Anything).
		Return(&entity.User{ID: userID, Email: "user@example.com"}, nil)
	tokenRepo.On("DeleteByEmail", mock.Anything, "user@example.com").Return(delErr)

	uc := NewAuthUsecase(userRepo, tokenRepo, &MockEmailClient{}, &MockJWTManager{}, "http://localhost:3000")
	err := uc.VerifyEmail(context.Background(), "tok")
	assert.ErrorIs(t, err, delErr)
	tokenRepo.AssertExpectations(t)
}
