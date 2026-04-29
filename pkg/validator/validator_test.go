package validator

import (
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
