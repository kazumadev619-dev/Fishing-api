package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
	"github.com/kazumadev619-dev/fishing-api/pkg/validator"
)

type WeatherUsecaseInterface interface {
	GetCurrent(ctx context.Context, lat, lon float64) (*entity.WeatherData, error)
	GetForecast(ctx context.Context, lat, lon float64) ([]*entity.WeatherData, error)
}

type WeatherHandler struct {
	usecase WeatherUsecaseInterface
}

func NewWeatherHandler(uc WeatherUsecaseInterface) *WeatherHandler {
	return &WeatherHandler{usecase: uc}
}

func (h *WeatherHandler) Get(c *gin.Context) {
	lat, lon, err := validator.ParseAndValidateCoordinates(
		c.Query("lat"), c.Query("lon"),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "INVALID_PARAMS", "status": 400})
		return
	}

	weatherType := c.DefaultQuery("type", "current")
	ctx := c.Request.Context()

	switch weatherType {
	case "current":
		data, err := h.usecase.GetCurrent(ctx, lat, lon)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch weather", "code": "INTERNAL_ERROR", "status": 500})
			return
		}
		c.JSON(http.StatusOK, data)
	case "forecast":
		data, err := h.usecase.GetForecast(ctx, lat, lon)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch forecast", "code": "INTERNAL_ERROR", "status": 500})
			return
		}
		c.JSON(http.StatusOK, data)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'current' or 'forecast'", "code": "INVALID_PARAMS", "status": 400})
	}
}
