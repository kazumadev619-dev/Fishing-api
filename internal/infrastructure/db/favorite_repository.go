package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	sqlcgen "github.com/kazumadev619-dev/fishing-api/db/generated"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/repository"
)

type favoriteRepository struct {
	queries *sqlcgen.Queries
}

func NewFavoriteRepository(db *sql.DB) repository.FavoriteRepository {
	return &favoriteRepository{queries: sqlcgen.New(db)}
}

func (r *favoriteRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error) {
	rows, err := r.queries.FindFavoritesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("finding favorites for user %s: %w", userID, err)
	}
	locations := make([]*entity.Location, 0, len(rows))
	for _, row := range rows {
		locations = append(locations, toLocationEntity(row))
	}
	return locations, nil
}

func (r *favoriteRepository) Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	return r.queries.AddFavorite(ctx, sqlcgen.AddFavoriteParams{
		ID:         uuid.New(),
		UserID:     userID,
		LocationID: locationID,
	})
}

func (r *favoriteRepository) Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	return r.queries.DeleteFavorite(ctx, sqlcgen.DeleteFavoriteParams{
		UserID:     userID,
		LocationID: locationID,
	})
}

func (r *favoriteRepository) Exists(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) (bool, error) {
	return r.queries.FavoriteExists(ctx, sqlcgen.FavoriteExistsParams{
		UserID:     userID,
		LocationID: locationID,
	})
}

func toLocationEntity(row sqlcgen.Location) *entity.Location {
	loc := &entity.Location{
		ID:           row.ID,
		Name:         row.Name,
		Latitude:     row.Latitude,
		Longitude:    row.Longitude,
		LocationType: entity.LocationType(row.LocationType),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
	if row.Region.Valid {
		loc.Region = &row.Region.String
	}
	if row.Prefecture.Valid {
		loc.Prefecture = &row.Prefecture.String
	}
	if row.PortID.Valid {
		loc.PortID = &row.PortID.UUID
	}
	return loc
}
