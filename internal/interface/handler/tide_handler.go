package handler

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type TideFetcher interface {
	GetTideData(ctx context.Context, prefCode, portCode, date string) (*entity.TideData, error)
}

type TideHandler struct {
	usecase TideFetcher
}

func NewTideHandler(uc TideFetcher) *TideHandler {
	return &TideHandler{usecase: uc}
}

var (
	prefCodeRegex = regexp.MustCompile(`^[0-9]{1,2}$`)
	portCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	dateRegex     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func (h *TideHandler) Get(c *gin.Context) {
	prefCode := c.Query("prefectureCode")
	portCode := c.Query("portCode")
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	if prefCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prefectureCode is required", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if portCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "portCode is required", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if !prefCodeRegex.MatchString(prefCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prefectureCode format", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if !portCodeRegex.MatchString(portCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid portCode format", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if !dateRegex.MatchString(date) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format. Use YYYY-MM-DD", "code": "INVALID_PARAMS", "status": 400})
		return
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date", "code": "INVALID_PARAMS", "status": 400})
		return
	}

	data, err := h.usecase.GetTideData(c.Request.Context(), prefCode, portCode, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tide data", "code": "INTERNAL_ERROR", "status": 500})
		return
	}

	c.JSON(http.StatusOK, data)
}
