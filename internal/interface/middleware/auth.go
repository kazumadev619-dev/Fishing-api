package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	jwtutil "github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"
)

func JWTAuth(jwtManager *jwtutil.Manager) gin.HandlerFunc {
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
