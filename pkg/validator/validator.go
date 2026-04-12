package validator

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	uuidRegex  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func IsValidUUID(id string) bool {
	return uuidRegex.MatchString(id)
}

func RoundCoordinate(value float64, precision int) float64 {
	p := math.Pow(10, float64(precision))
	return math.Round(value*p) / p
}

func ParseAndValidateCoordinates(latStr, lonStr string) (float64, float64, error) {
	if latStr == "" || lonStr == "" {
		return 0, 0, fmt.Errorf("lat and lon are required")
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lat: %w", err)
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lon: %w", err)
	}

	if lat < -90 || lat > 90 {
		return 0, 0, fmt.Errorf("lat must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return 0, 0, fmt.Errorf("lon must be between -180 and 180")
	}

	return lat, lon, nil
}
