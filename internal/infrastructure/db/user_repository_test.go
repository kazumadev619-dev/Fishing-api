package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewUserRepository(db)

	_, err := repo.FindByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepository_UpdateEmailVerified(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewUserRepository(db)

	hash := "hashed-password"
	user := &entity.User{
		ID:           uuid.New(),
		Email:        "verify-" + uuid.New().String() + "@example.com",
		PasswordHash: &hash,
		IsSSO:        false,
	}

	created, err := repo.Create(ctx, user)
	require.NoError(t, err)
	require.Nil(t, created.EmailVerifiedAt, "新規作成時は未認証 (email_verified_at = NULL) のはず")

	verifiedAt := time.Now().UTC()
	updated, err := repo.UpdateEmailVerified(ctx, created.ID, verifiedAt)
	require.NoError(t, err)
	require.NotNil(t, updated.EmailVerifiedAt)

	// Refetch して永続化されていることを確認するのだ。
	refetched, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, refetched.EmailVerifiedAt)
	assert.WithinDuration(t, verifiedAt, *refetched.EmailVerifiedAt, time.Second)
}

func TestUserRepository_UpdateEmailVerified_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewUserRepository(db)

	_, err := repo.UpdateEmailVerified(ctx, uuid.New(), time.Now().UTC())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
