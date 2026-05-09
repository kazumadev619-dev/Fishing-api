package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/kazumadev619-dev/fishing-api/config"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/cache"
	infradb "github.com/kazumadev619-dev/fishing-api/internal/infrastructure/db"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/email"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/external"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/router"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/auth"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/favorite"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/location"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/tide"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/weather"
	"github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"
)

// jwtManagerAdapter は *jwtutil.Manager を auth.JWTManager インターフェースに適合させる
type jwtManagerAdapter struct{ m *jwtutil.Manager }

func (a *jwtManagerAdapter) GenerateAccessToken(userID uuid.UUID) (string, error) {
	return a.m.GenerateAccessToken(userID)
}

func (a *jwtManagerAdapter) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	return a.m.GenerateRefreshToken(userID)
}

func (a *jwtManagerAdapter) ValidateRefreshToken(tokenStr string) (uuid.UUID, error) {
	claims, err := a.m.ValidateRefreshToken(tokenStr)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}

func main() {
	if err := run(); err != nil {
		slog.Error("server fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx := context.Background()

	pool, err := infradb.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close db", "error", err)
		}
	}()

	cacheClient, err := cache.NewCacheClient(ctx, cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("connecting to redis: %w", err)
	}

	// JWT
	jwtManager := jwtutil.NewManager(cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret)

	// Repositories
	userRepo := infradb.NewUserRepository(db)
	tokenRepo := infradb.NewVerificationTokenRepository(db)
	favoriteRepo := infradb.NewFavoriteRepository(db)

	// External clients
	weatherClient := external.NewWeatherClient(cfg.External.OpenWeatherAPIKey)
	tideClient := external.NewTideClient()
	mapsClient := external.NewMapsClient(cfg.External.GoogleMapsAPIKey)
	emailClient := email.NewEmailClient(cfg.Email.ResendAPIKey, cfg.Email.FromAddress)

	// Usecases
	authUC := auth.NewAuthUsecase(userRepo, tokenRepo, emailClient, &jwtManagerAdapter{m: jwtManager}, cfg.Server.AppBaseURL)
	weatherUC := weather.NewWeatherUsecase(weatherClient, cacheClient)
	tideUC := tide.NewTideUsecase(tideClient, cacheClient)
	locationUC := location.NewLocationUsecase(mapsClient, cacheClient)
	favoriteUC := favorite.NewFavoriteUsecase(favoriteRepo)

	// Handlers
	handlers := &router.Handlers{
		Auth:     handler.NewAuthHandler(authUC),
		Weather:  handler.NewWeatherHandler(weatherUC),
		Tide:     handler.NewTideHandler(tideUC),
		Location: handler.NewLocationHandler(locationUC),
		Favorite: handler.NewFavoriteHandler(favoriteUC),
	}

	r := router.New(handlers, jwtManager)

	slog.Info("server starting", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		return fmt.Errorf("running server: %w", err)
	}
	return nil
}
