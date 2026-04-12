package jwtutil

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	manager := NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")
	userID := uuid.New()

	token, err := manager.GenerateAccessToken(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), claims.ExpiresAt.Time, 5*time.Second)
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	manager := NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")
	userID := uuid.New()

	token, err := manager.GenerateRefreshToken(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	manager := NewManager("access-secret-32chars-minimum!!", "refresh-secret-32chars-minimum!")
	_, err := manager.ValidateAccessToken("invalid.token.here")
	assert.Error(t, err)
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	manager1 := NewManager("secret-one-32chars-minimum!!!!", "refresh-secret")
	manager2 := NewManager("secret-two-32chars-minimum!!!!", "refresh-secret")
	userID := uuid.New()

	token, err := manager1.GenerateAccessToken(userID)
	require.NoError(t, err)

	_, err = manager2.ValidateAccessToken(token)
	assert.Error(t, err)
}
