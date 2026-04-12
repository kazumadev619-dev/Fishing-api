package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	jwtutil "github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
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
	jwtManager := jwtutil.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")

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

	uc := NewAuthUsecase(userRepo, tokenRepo, emailClient, jwtManager, "http://localhost:3000")
	_, loginErr := uc.Login(context.Background(), "test@example.com", "wrongpassword")
	assert.ErrorIs(t, loginErr, domain.ErrUnauthorized)
	userRepo.AssertExpectations(t)
}

func TestAuthUsecase_Register_DuplicateEmail(t *testing.T) {
	userRepo := &MockUserRepository{}
	tokenRepo := &MockVerificationTokenRepository{}
	emailClient := &MockEmailClient{}
	jwtManager := jwtutil.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")

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
	jwtManager := jwtutil.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")

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
	jwtManager := jwtutil.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")

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
