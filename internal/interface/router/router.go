package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/middleware"
	jwtutil "github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"
)

type Handlers struct {
	Auth     *handler.AuthHandler
	Weather  *handler.WeatherHandler
	Tide     *handler.TideHandler
	Location *handler.LocationHandler
	Favorite *handler.FavoriteHandler
}

func New(handlers *Handlers, jwtManager *jwtutil.Manager) *gin.Engine {
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

		api.GET("/weather", handlers.Weather.Get)
		api.GET("/conditions/tide", handlers.Tide.Get)
		api.GET("/locations/search", handlers.Location.Search)

		protected := api.Group("").Use(middleware.JWTAuth(jwtManager))
		{
			protected.GET("/favorites", handlers.Favorite.GetList)
			protected.POST("/favorites", handlers.Favorite.Add)
			protected.DELETE("/favorites/:id", handlers.Favorite.Delete)
		}
	}

	return r
}
