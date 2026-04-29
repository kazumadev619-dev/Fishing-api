package handler

import (
	"context"
	"errors"
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
		if errors.Is(err, domain.ErrAlreadyExists) {
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
		if errors.Is(err, domain.ErrEmailNotVerified) {
			c.JSON(http.StatusForbidden, gin.H{"error": "email not verified", "code": "EMAIL_NOT_VERIFIED", "status": 403})
			return
		}
		if errors.Is(err, domain.ErrUnauthorized) {
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
