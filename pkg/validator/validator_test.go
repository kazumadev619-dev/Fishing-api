package validator

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidEmail(t *testing.T) {
	assert.True(t, IsValidEmail("user@example.com"))
	assert.True(t, IsValidEmail("user+tag@example.co.jp"))
	assert.False(t, IsValidEmail("invalid"))
	assert.False(t, IsValidEmail("@example.com"))
	assert.False(t, IsValidEmail(""))
}

func TestIsValidUUID(t *testing.T) {
	assert.True(t, IsValidUUID("550e8400-e29b-41d4-a716-446655440000"))
	assert.False(t, IsValidUUID("not-a-uuid"))
	assert.False(t, IsValidUUID(""))
}

func TestRoundCoordinate(t *testing.T) {
	assert.Equal(t, 35.6895, RoundCoordinate(35.68954321, 4))
	assert.Equal(t, 139.6917, RoundCoordinate(139.69174321, 4))
}

func TestParseAndValidateCoordinates(t *testing.T) {
	lat, lon, err := ParseAndValidateCoordinates("35.6895", "139.6917")
	assert.NoError(t, err)
	assert.Equal(t, 35.6895, lat)
	assert.Equal(t, 139.6917, lon)

	_, _, err = ParseAndValidateCoordinates("", "139.6917")
	assert.Error(t, err)

	_, _, err = ParseAndValidateCoordinates("91.0", "0")
	assert.Error(t, err)

	_, _, err = ParseAndValidateCoordinates("0", "181.0")
	assert.Error(t, err)
}

func TestValidateCoordinates(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		lon     float64
		wantErr string
	}{
		{name: "正常", lat: 35.6895, lon: 139.6917},
		{name: "境界最小", lat: -90, lon: -180},
		{name: "境界最大", lat: 90, lon: 180},
		{name: "lat 上限超過", lat: 90.1, lon: 0, wantErr: "lat must be between -90 and 90"},
		{name: "lat 下限未満", lat: -90.1, lon: 0, wantErr: "lat must be between -90 and 90"},
		{name: "lat NaN", lat: math.NaN(), lon: 0, wantErr: "lat must be between -90 and 90"},
		{name: "lon 上限超過", lat: 0, lon: 180.1, wantErr: "lon must be between -180 and 180"},
		{name: "lon 下限未満", lat: 0, lon: -180.1, wantErr: "lon must be between -180 and 180"},
		{name: "lon NaN", lat: 0, lon: math.NaN(), wantErr: "lon must be between -180 and 180"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCoordinates(tt.lat, tt.lon)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
