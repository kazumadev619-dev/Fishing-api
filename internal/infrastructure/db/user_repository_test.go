package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:17"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := NewPool(ctx, connStr)
	require.NoError(t, err)

	db := stdlib.OpenDBFromPool(pool)

	// Apply schema
	schema, err := os.ReadFile("../../../db/schema.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(schema))
	require.NoError(t, err)

	return db, func() {
		db.Close()
		_ = container.Terminate(ctx)
	}
}

func TestUserRepository_CreateAndFind(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewUserRepository(db)

	hash := "hashed-password"
	name := "Test User"
	user := &entity.User{
		ID:           uuid.New(),
		Email:        "test-" + uuid.New().String() + "@example.com",
		PasswordHash: &hash,
		Name:         &name,
		IsSSO:        false,
	}

	created, err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.Equal(t, user.Email, created.Email)
	assert.Equal(t, hash, *created.PasswordHash)

	found, err := repo.FindByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)

	foundByID, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.Email, foundByID.Email)
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewUserRepository(db)

	_, err := repo.FindByEmail(ctx, "nonexistent@example.com")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
