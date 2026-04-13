package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
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

func TestAuthHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := &MockAuthUsecase{}
	pair := &auth.TokenPair{AccessToken: "acc", RefreshToken: "ref"}
	mockUC.On("Login", mock.Anything, "user@example.com", "password123").Return(pair, nil)

	router := gin.New()
	h := NewAuthHandler(mockUC)
	router.POST("/api/auth/login", h.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_Login_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := &MockAuthUsecase{}
	mockUC.On("Login", mock.Anything, "bad@example.com", "wrongpass").Return(nil, domain.ErrUnauthorized)

	router := gin.New()
	h := NewAuthHandler(mockUC)
	router.POST("/api/auth/login", h.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "bad@example.com",
		"password": "wrongpass",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_Login_EmailNotVerified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := &MockAuthUsecase{}
	mockUC.On("Login", mock.Anything, "unverified@example.com", "password123").Return(nil, domain.ErrEmailNotVerified)

	router := gin.New()
	h := NewAuthHandler(mockUC)
	router.POST("/api/auth/login", h.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "unverified@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := &MockAuthUsecase{}
	pair := &auth.TokenPair{AccessToken: "new-acc", RefreshToken: "new-ref"}
	mockUC.On("RefreshToken", mock.Anything, "valid-refresh").Return(pair, nil)

	router := gin.New()
	h := NewAuthHandler(mockUC)
	router.POST("/api/auth/refresh", h.RefreshToken)

	body, _ := json.Marshal(map[string]string{"refresh_token": "valid-refresh"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_RefreshToken_Invalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := &MockAuthUsecase{}
	mockUC.On("RefreshToken", mock.Anything, "bad-token").Return(nil, domain.ErrInvalidToken)

	router := gin.New()
	h := NewAuthHandler(mockUC)
	router.POST("/api/auth/refresh", h.RefreshToken)

	body, _ := json.Marshal(map[string]string{"refresh_token": "bad-token"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_VerifyEmail_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := &MockAuthUsecase{}
	mockUC.On("VerifyEmail", mock.Anything, "valid-token").Return(nil)

	router := gin.New()
	h := NewAuthHandler(mockUC)
	router.GET("/api/auth/verify-email", h.VerifyEmail)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/verify-email?token=valid-token", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockUC.AssertExpectations(t)
}

func TestAuthHandler_VerifyEmail_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockUC := &MockAuthUsecase{}
	mockUC.On("VerifyEmail", mock.Anything, "bad-token").Return(domain.ErrInvalidToken)

	router := gin.New()
	h := NewAuthHandler(mockUC)
	router.GET("/api/auth/verify-email", h.VerifyEmail)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/verify-email?token=bad-token", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	mockUC.AssertExpectations(t)
}
