package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createUser はテスト用のユーザーを 1 件作成し、その ID を返すのだ。
func createUser(t *testing.T, repo repository.UserRepository) uuid.UUID {
	t.Helper()
	hash := "hashed-password"
	user := &entity.User{
		ID:           uuid.New(),
		Email:        "fav-" + uuid.New().String() + "@example.com",
		PasswordHash: &hash,
		IsSSO:        false,
	}
	created, err := repo.Create(context.Background(), user)
	require.NoError(t, err)
	return created.ID
}

func TestFavoriteRepository_Add_FindByUserID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := NewUserRepository(db)
	favRepo := NewFavoriteRepository(db)

	userID := createUser(t, userRepo)
	locID := seedLocation(t, db, "Tokyo Bay")

	err := favRepo.Add(ctx, userID, locID)
	require.NoError(t, err)

	locations, err := favRepo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, locations, 1)
	assert.Equal(t, locID, locations[0].ID)
	assert.Equal(t, "Tokyo Bay", locations[0].Name)
}

func TestFavoriteRepository_FindByUserID_FullLocation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := NewUserRepository(db)
	favRepo := NewFavoriteRepository(db)

	userID := createUser(t, userRepo)
	portID := seedPort(t, db, "Tokyo Port")
	locID := seedFullLocation(t, db, "Full Location", portID)

	require.NoError(t, favRepo.Add(ctx, userID, locID))

	locations, err := favRepo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, locations, 1)

	loc := locations[0]
	assert.Equal(t, locID, loc.ID)
	require.NotNil(t, loc.Region)
	assert.Equal(t, "Kanto", *loc.Region)
	require.NotNil(t, loc.Prefecture)
	assert.Equal(t, "Tokyo", *loc.Prefecture)
	require.NotNil(t, loc.PortID)
	assert.Equal(t, portID, *loc.PortID)
}

func TestFavoriteRepository_FindByUserID_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := NewUserRepository(db)
	favRepo := NewFavoriteRepository(db)

	userID := createUser(t, userRepo)

	locations, err := favRepo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, locations)
}

func TestFavoriteRepository_Exists_TrueFalse(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := NewUserRepository(db)
	favRepo := NewFavoriteRepository(db)

	userID := createUser(t, userRepo)
	locID := seedLocation(t, db, "Sagami Bay")

	// Add 前は存在しないのだ。
	exists, err := favRepo.Exists(ctx, userID, locID)
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, favRepo.Add(ctx, userID, locID))

	// Add 後は存在するのだ。
	exists, err = favRepo.Exists(ctx, userID, locID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestFavoriteRepository_Delete_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := NewUserRepository(db)
	favRepo := NewFavoriteRepository(db)

	userID := createUser(t, userRepo)
	locID := seedLocation(t, db, "Suruga Bay")

	require.NoError(t, favRepo.Add(ctx, userID, locID))

	err := favRepo.Delete(ctx, userID, locID)
	require.NoError(t, err)

	exists, err := favRepo.Exists(ctx, userID, locID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestFavoriteRepository_Delete_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := NewUserRepository(db)
	favRepo := NewFavoriteRepository(db)

	userID := createUser(t, userRepo)
	locID := seedLocation(t, db, "Ise Bay")

	// 一度も Add していない favorite を Delete すると ErrNotFound なのだ。
	err := favRepo.Delete(ctx, userID, locID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
