package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/kazumadev619-dev/fishing-api/config"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/cache"
	infradb "github.com/kazumadev619-dev/fishing-api/internal/infrastructure/db"
	"github.com/kazumadev619-dev/fishing-api/internal/infrastructure/email"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/router"
	"github.com/kazumadev619-dev/fishing-api/internal/usecase/auth"
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

	// *sql.DB を1回だけ生成してコネクションプールを共有
	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close db", "error", err)
		}
	}()

	cacheClient, err := cache.NewCacheClient(ctx, cfg.Redis.URL)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	_ = cacheClient // Phase 3以降で使用

	// JWT
	jwtManager := jwtutil.NewManager(cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret)

	// Repositories（*sql.DB を共有）
	userRepo := infradb.NewUserRepository(db)
	tokenRepo := infradb.NewVerificationTokenRepository(db)

	// Infrastructure
	emailClient := email.NewEmailClient(cfg.Email.ResendAPIKey, cfg.Email.FromAddress)

	// Usecases（JWTManagerAdapter 経由で auth.JWTManager を満たす）
	authUC := auth.NewAuthUsecase(userRepo, tokenRepo, emailClient, &jwtManagerAdapter{m: jwtManager}, cfg.Server.AppBaseURL)

	// Handlers
	handlers := &router.Handlers{
		Auth: handler.NewAuthHandler(authUC),
	}

	r := router.New(handlers)

	slog.Info("server starting", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
