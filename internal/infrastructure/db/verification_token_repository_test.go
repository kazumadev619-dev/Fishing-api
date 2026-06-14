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

func newVerificationToken(email, token string) *entity.VerificationToken {
	return &entity.VerificationToken{
		ID:        uuid.New(),
		Email:     email,
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
}

func TestVerificationTokenRepository_Create_FindByToken(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewVerificationTokenRepository(db)

	email := "verify-" + uuid.New().String() + "@example.com"
	tokenStr := "token-" + uuid.New().String()

	created, err := repo.Create(ctx, newVerificationToken(email, tokenStr))
	require.NoError(t, err)
	assert.Equal(t, email, created.Email)
	assert.Equal(t, tokenStr, created.Token)

	found, err := repo.FindByToken(ctx, tokenStr)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, email, found.Email)
}

func TestVerificationTokenRepository_FindByToken_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewVerificationTokenRepository(db)

	_, err := repo.FindByToken(ctx, "nonexistent-token")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestVerificationTokenRepository_DeleteByEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewVerificationTokenRepository(db)

	email := "delete-" + uuid.New().String() + "@example.com"
	tokenStr := "token-" + uuid.New().String()

	_, err := repo.Create(ctx, newVerificationToken(email, tokenStr))
	require.NoError(t, err)

	// 存在するトークンを削除できるのだ。
	require.NoError(t, repo.DeleteByEmail(ctx, email))

	_, err = repo.FindByToken(ctx, tokenStr)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// 0 件でもエラーにならない (idempotent) のだ。
	require.NoError(t, repo.DeleteByEmail(ctx, email))
}
