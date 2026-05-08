package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domain "github.com/kazumadev619-dev/fishing-api/internal/domain"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type FavoriteManager interface {
	GetList(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error)
	Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error
	Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error
}

type FavoriteHandler struct {
	usecase FavoriteManager
}

func NewFavoriteHandler(uc FavoriteManager) *FavoriteHandler {
	return &FavoriteHandler{usecase: uc}
}

func (h *FavoriteHandler) GetList(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	locations, err := h.usecase.GetList(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get favorites", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorites": locations})
}

type addFavoriteRequest struct {
	LocationID string `json:"location_id" binding:"required"`
}

func (h *FavoriteHandler) Add(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req addFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "INVALID_PARAMS", "status": 400})
		return
	}

	locationID, err := uuid.Parse(req.LocationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid location_id", "code": "INVALID_PARAMS", "status": 400})
		return
	}

	if err := h.usecase.Add(c.Request.Context(), userID, locationID); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "already favorited", "code": "ALREADY_EXISTS", "status": 409})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add favorite", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "added to favorites"})
}

func (h *FavoriteHandler) Delete(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	locationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid location id", "code": "INVALID_PARAMS", "status": 400})
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), userID, locationID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "favorite not found", "code": "NOT_FOUND", "status": 404})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete favorite", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "removed from favorites"})
}
