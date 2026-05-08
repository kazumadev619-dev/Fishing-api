package favorite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock ---

type MockFavoriteRepository struct{ mock.Mock }

func (m *MockFavoriteRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Location), args.Error(1)
}

func (m *MockFavoriteRepository) Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	args := m.Called(ctx, userID, locationID)
	return args.Error(0)
}

func (m *MockFavoriteRepository) Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	args := m.Called(ctx, userID, locationID)
	return args.Error(0)
}

func (m *MockFavoriteRepository) Exists(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, locationID)
	return args.Bool(0), args.Error(1)
}

// --- Tests ---

func TestFavoriteUsecase_GetList(t *testing.T) {
	tests := []struct {
		name       string
		userID     uuid.UUID
		mockReturn []*entity.Location
		mockErr    error
		wantErr    bool
		wantLen    int
	}{
		{
			name:   "正常取得",
			userID: uuid.New(),
			mockReturn: []*entity.Location{
				{
					ID:           uuid.New(),
					Name:         "港南大橋",
					Latitude:     35.433,
					Longitude:    139.650,
					LocationType: entity.LocationTypePort,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				},
				{
					ID:           uuid.New(),
					Name:         "横須賀海岸",
					Latitude:     35.280,
					Longitude:    139.670,
					LocationType: entity.LocationTypeShore,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				},
			},
			mockErr: nil,
			wantErr: false,
			wantLen: 2,
		},
		{
			name:       "お気に入りなし（空リスト）",
			userID:     uuid.New(),
			mockReturn: []*entity.Location{},
			mockErr:    nil,
			wantErr:    false,
			wantLen:    0,
		},
		{
			name:       "リポジトリエラー",
			userID:     uuid.New(),
			mockReturn: nil,
			mockErr:    errors.New("db error"),
			wantErr:    true,
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockFavoriteRepository{}
			repo.On("FindByUserID", mock.Anything, tt.userID).Return(tt.mockReturn, tt.mockErr)

			uc := NewFavoriteUsecase(repo)
			result, err := uc.GetList(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.wantLen)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestFavoriteUsecase_Add_Success(t *testing.T) {
	repo := &MockFavoriteRepository{}
	userID := uuid.New()
	locationID := uuid.New()

	repo.On("Exists", mock.Anything, userID, locationID).Return(false, nil)
	repo.On("Add", mock.Anything, userID, locationID).Return(nil)

	uc := NewFavoriteUsecase(repo)
	err := uc.Add(context.Background(), userID, locationID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestFavoriteUsecase_Add_AlreadyExists(t *testing.T) {
	repo := &MockFavoriteRepository{}
	userID := uuid.New()
	locationID := uuid.New()

	repo.On("Exists", mock.Anything, userID, locationID).Return(true, nil)

	uc := NewFavoriteUsecase(repo)
	err := uc.Add(context.Background(), userID, locationID)

	assert.True(t, errors.Is(err, domain.ErrAlreadyExists))
	// Add は呼ばれてはいけない
	repo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestFavoriteUsecase_Add_ExistsError(t *testing.T) {
	repo := &MockFavoriteRepository{}
	userID := uuid.New()
	locationID := uuid.New()
	dbErr := errors.New("db connection error")

	repo.On("Exists", mock.Anything, userID, locationID).Return(false, dbErr)

	uc := NewFavoriteUsecase(repo)
	err := uc.Add(context.Background(), userID, locationID)

	assert.ErrorIs(t, err, dbErr)
	repo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestFavoriteUsecase_Delete(t *testing.T) {
	tests := []struct {
		name       string
		userID     uuid.UUID
		locationID uuid.UUID
		mockErr    error
		wantErr    bool
	}{
		{
			name:       "正常削除",
			userID:     uuid.New(),
			locationID: uuid.New(),
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:       "リポジトリエラー",
			userID:     uuid.New(),
			locationID: uuid.New(),
			mockErr:    errors.New("db error"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockFavoriteRepository{}
			repo.On("Delete", mock.Anything, tt.userID, tt.locationID).Return(tt.mockErr)

			uc := NewFavoriteUsecase(repo)
			err := uc.Delete(context.Background(), tt.userID, tt.locationID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}
