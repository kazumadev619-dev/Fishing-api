package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFavoriteUsecase struct{ mock.Mock }

func (m *MockFavoriteUsecase) GetList(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Location), args.Error(1)
}

func (m *MockFavoriteUsecase) Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	args := m.Called(ctx, userID, locationID)
	return args.Error(0)
}

func (m *MockFavoriteUsecase) Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error {
	args := m.Called(ctx, userID, locationID)
	return args.Error(0)
}

func setupFavoriteRouter(h *FavoriteHandler) *gin.Engine {
	router := gin.New()
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	router.GET("/api/favorites", h.GetList)
	router.POST("/api/favorites", h.Add)
	router.DELETE("/api/favorites/:id", h.Delete)
	return router
}

func TestFavoriteHandler_GetList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	locationID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		setupMock  func(*MockFavoriteUsecase)
		wantStatus int
	}{
		{
			name: "正常取得",
			setupMock: func(m *MockFavoriteUsecase) {
				m.On("GetList", mock.Anything, userID).Return([]*entity.Location{{ID: locationID, Name: "東京湾"}}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "エラー → 500",
			setupMock: func(m *MockFavoriteUsecase) {
				m.On("GetList", mock.Anything, userID).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &MockFavoriteUsecase{}
			tt.setupMock(mockUC)

			router := setupFavoriteRouter(NewFavoriteHandler(mockUC))
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/favorites", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			mockUC.AssertExpectations(t)
		})
	}
}

func TestFavoriteHandler_Add(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	locationID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		body       interface{}
		setupMock  func(*MockFavoriteUsecase)
		wantStatus int
	}{
		{
			name: "正常追加",
			body: map[string]string{"location_id": locationID.String()},
			setupMock: func(m *MockFavoriteUsecase) {
				m.On("Add", mock.Anything, userID, locationID).Return(nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "bodyなし → 400",
			body:       map[string]string{},
			setupMock:  func(m *MockFavoriteUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "無効なUUID → 400",
			body:       map[string]string{"location_id": "not-a-uuid"},
			setupMock:  func(m *MockFavoriteUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "既にお気に入り → 409",
			body: map[string]string{"location_id": locationID.String()},
			setupMock: func(m *MockFavoriteUsecase) {
				m.On("Add", mock.Anything, userID, locationID).Return(domain.ErrAlreadyExists)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "DBエラー → 500",
			body: map[string]string{"location_id": locationID.String()},
			setupMock: func(m *MockFavoriteUsecase) {
				m.On("Add", mock.Anything, userID, locationID).Return(errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &MockFavoriteUsecase{}
			tt.setupMock(mockUC)

			router := setupFavoriteRouter(NewFavoriteHandler(mockUC))
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/favorites", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			mockUC.AssertExpectations(t)
		})
	}
}

func TestFavoriteHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	locationID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		paramID    string
		setupMock  func(*MockFavoriteUsecase)
		wantStatus int
	}{
		{
			name:    "正常削除",
			paramID: locationID.String(),
			setupMock: func(m *MockFavoriteUsecase) {
				m.On("Delete", mock.Anything, userID, locationID).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "無効なUUID → 400",
			paramID:    "not-a-uuid",
			setupMock:  func(m *MockFavoriteUsecase) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "存在しない → 404",
			paramID: locationID.String(),
			setupMock: func(m *MockFavoriteUsecase) {
				m.On("Delete", mock.Anything, userID, locationID).Return(domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:    "DBエラー → 500",
			paramID: locationID.String(),
			setupMock: func(m *MockFavoriteUsecase) {
				m.On("Delete", mock.Anything, userID, locationID).Return(errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &MockFavoriteUsecase{}
			tt.setupMock(mockUC)

			router := setupFavoriteRouter(NewFavoriteHandler(mockUC))
			req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/favorites/"+tt.paramID, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			mockUC.AssertExpectations(t)
		})
	}
}
