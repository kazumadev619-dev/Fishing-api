package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type LocationSearcher interface {
	Search(ctx context.Context, query string, limit int) ([]*entity.LocationSearchResult, error)
}

type LocationHandler struct {
	usecase LocationSearcher
}

func NewLocationHandler(uc LocationSearcher) *LocationHandler {
	return &LocationHandler{usecase: uc}
}

func (h *LocationHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if len(query) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query must be at least 2 characters", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if len(query) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query must be less than 200 characters", "code": "INVALID_PARAMS", "status": 400})
		return
	}

	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter", "code": "INVALID_PARAMS", "status": 400})
			return
		}
		limit = parsed
	}

	results, err := h.usecase.Search(c.Request.Context(), query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "location search failed", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}
