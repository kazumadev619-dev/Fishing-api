package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/kazumadev619-dev/fishing-api/config"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/cache"
	infradb "github.com/kazumadev619-dev/fishing-api/internal/infrastructure/db"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/email"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/router"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/auth"
	jwtutil "github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := infradb.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	cacheClient, err := cache.NewCacheClient(ctx, cfg.Redis.URL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	_ = cacheClient // Phase 3以降で使用

	// JWT
	jwtManager := jwtutil.NewManager(cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret)

	// Repositories
	userRepo := infradb.NewUserRepository(pool)
	tokenRepo := infradb.NewVerificationTokenRepository(pool)

	// Infrastructure
	emailClient := email.NewEmailClient(cfg.Email.ResendAPIKey, cfg.Email.FromAddress)

	// Usecases
	authUC := auth.NewAuthUsecase(userRepo, tokenRepo, emailClient, jwtManager, cfg.Server.AppBaseURL)

	// Handlers
	handlers := &router.Handlers{
		Auth: handler.NewAuthHandler(authUC),
	}

	r := router.New(handlers, jwtManager)

	slog.Info("server starting", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
