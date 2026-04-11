package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Success(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/fishing")
	t.Setenv("JWT_ACCESS_SECRET", "access-secret-32chars-minimum!!")
	t.Setenv("JWT_REFRESH_SECRET", "refresh-secret-32chars-minimum!")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "postgres://localhost/fishing", cfg.Database.URL)
	assert.Equal(t, "redis://localhost:6379", cfg.Redis.URL)
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("JWT_ACCESS_SECRET", "access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "refresh-secret")

	_, err := Load()
	assert.ErrorContains(t, err, "DATABASE_URL")
}

func TestLoad_CustomPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/fishing")
	t.Setenv("JWT_ACCESS_SECRET", "access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "refresh-secret")
	t.Setenv("PORT", "9090")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Server.Port)
}
