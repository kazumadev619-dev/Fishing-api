package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"
	"github.com/stretchr/testify/assert"
)

func TestJWTAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := jwtutil.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!!")
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
	manager := jwtutil.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!!")

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
	manager := jwtutil.NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!!")

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
