package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/repository"
	jwtutil "github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"
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
	jwtManager  *jwtutil.Manager
	appBaseURL  string
}

func NewAuthUsecase(
	userRepo repository.UserRepository,
	tokenRepo repository.VerificationTokenRepository,
	emailSender EmailSender,
	jwtManager *jwtutil.Manager,
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
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
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
		if errors.Is(err, domain.ErrNotFound) {
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
