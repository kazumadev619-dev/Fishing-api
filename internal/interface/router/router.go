package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
)

type Handlers struct {
	Auth *handler.AuthHandler
}

func New(handlers *Handlers) *gin.Engine {
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

		// 認証が必要なルートは Phase 3 以降で追加する
	}

	return r
}
