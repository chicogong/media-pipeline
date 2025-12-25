package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager_Generate(t *testing.T) {
	manager := NewJWTManager("test-secret-key", time.Hour)

	token, err := manager.Generate("user123", "user@example.com", "admin")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTManager_Verify_ValidToken(t *testing.T) {
	manager := NewJWTManager("test-secret-key", time.Hour)

	// Generate token
	token, err := manager.Generate("user123", "user@example.com", "admin")
	require.NoError(t, err)

	// Verify token
	claims, err := manager.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "user@example.com", claims.Email)
	assert.Equal(t, "admin", claims.Role)
}

func TestJWTManager_Verify_ExpiredToken(t *testing.T) {
	manager := NewJWTManager("test-secret-key", time.Millisecond)

	// Generate token with very short duration
	token, err := manager.Generate("user123", "user@example.com", "admin")
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Verify should fail
	_, err = manager.Verify(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestJWTManager_Verify_InvalidToken(t *testing.T) {
	manager := NewJWTManager("test-secret-key", time.Hour)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"malformed token", "not.a.valid.token"},
		{"wrong secret", generateTokenWithDifferentSecret(t)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.Verify(tt.token)
			assert.Error(t, err)
		})
	}
}

func TestJWTManager_Refresh(t *testing.T) {
	manager := NewJWTManager("test-secret-key", time.Hour)

	// Generate original token
	originalToken, err := manager.Generate("user123", "user@example.com", "admin")
	require.NoError(t, err)

	// Refresh token
	newToken, err := manager.Refresh(originalToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newToken)

	// Verify new token has same user info and is valid
	claims, err := manager.Verify(newToken)
	require.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "user@example.com", claims.Email)
	assert.Equal(t, "admin", claims.Role)
}

func TestJWTManager_Refresh_InvalidToken(t *testing.T) {
	manager := NewJWTManager("test-secret-key", time.Hour)

	_, err := manager.Refresh("invalid.token")
	assert.Error(t, err)
}

// Helper function to generate a token with different secret
func generateTokenWithDifferentSecret(t *testing.T) string {
	wrongManager := NewJWTManager("wrong-secret", time.Hour)
	token, err := wrongManager.Generate("user123", "user@example.com", "admin")
	require.NoError(t, err)
	return token
}
