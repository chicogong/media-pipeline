package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyManager_Generate(t *testing.T) {
	manager := NewAPIKeyManager()

	apiKey, err := manager.Generate("user123", "Test Key", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, apiKey.Key)
	assert.Equal(t, "user123", apiKey.UserID)
	assert.Equal(t, "Test Key", apiKey.Name)
	assert.False(t, apiKey.Revoked)
	assert.True(t, apiKey.Key[:3] == "sk_") // Check prefix
}

func TestAPIKeyManager_Generate_WithExpiry(t *testing.T) {
	manager := NewAPIKeyManager()

	expiresAt := time.Now().Add(time.Hour)
	apiKey, err := manager.Generate("user123", "Test Key", &expiresAt)
	require.NoError(t, err)
	assert.NotNil(t, apiKey.ExpiresAt)
	assert.Equal(t, expiresAt.Unix(), apiKey.ExpiresAt.Unix())
}

func TestAPIKeyManager_Verify_ValidKey(t *testing.T) {
	manager := NewAPIKeyManager()

	apiKey, err := manager.Generate("user123", "Test Key", nil)
	require.NoError(t, err)

	verified, err := manager.Verify(apiKey.Key)
	require.NoError(t, err)
	assert.Equal(t, apiKey.UserID, verified.UserID)
	assert.Equal(t, apiKey.Name, verified.Name)
}

func TestAPIKeyManager_Verify_InvalidKey(t *testing.T) {
	manager := NewAPIKeyManager()

	_, err := manager.Verify("invalid-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
}

func TestAPIKeyManager_Verify_RevokedKey(t *testing.T) {
	manager := NewAPIKeyManager()

	apiKey, err := manager.Generate("user123", "Test Key", nil)
	require.NoError(t, err)

	// Revoke the key
	err = manager.Revoke(apiKey.Key)
	require.NoError(t, err)

	// Verify should fail
	_, err = manager.Verify(apiKey.Key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")
}

func TestAPIKeyManager_Verify_ExpiredKey(t *testing.T) {
	manager := NewAPIKeyManager()

	// Create key that expires immediately
	expiresAt := time.Now().Add(-time.Hour)
	apiKey, err := manager.Generate("user123", "Test Key", &expiresAt)
	require.NoError(t, err)

	// Verify should fail
	_, err = manager.Verify(apiKey.Key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestAPIKeyManager_Revoke(t *testing.T) {
	manager := NewAPIKeyManager()

	apiKey, err := manager.Generate("user123", "Test Key", nil)
	require.NoError(t, err)

	err = manager.Revoke(apiKey.Key)
	require.NoError(t, err)

	// Verify key is revoked
	verified, err := manager.Verify(apiKey.Key)
	assert.Error(t, err)
	assert.Nil(t, verified)
}

func TestAPIKeyManager_Revoke_NotFound(t *testing.T) {
	manager := NewAPIKeyManager()

	err := manager.Revoke("non-existent-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAPIKeyManager_Delete(t *testing.T) {
	manager := NewAPIKeyManager()

	apiKey, err := manager.Generate("user123", "Test Key", nil)
	require.NoError(t, err)

	err = manager.Delete(apiKey.Key)
	require.NoError(t, err)

	// Verify key no longer exists
	_, err = manager.Verify(apiKey.Key)
	assert.Error(t, err)
}

func TestAPIKeyManager_List(t *testing.T) {
	manager := NewAPIKeyManager()

	// Generate keys for different users
	_, err := manager.Generate("user1", "Key 1", nil)
	require.NoError(t, err)
	_, err = manager.Generate("user1", "Key 2", nil)
	require.NoError(t, err)
	_, err = manager.Generate("user2", "Key 3", nil)
	require.NoError(t, err)

	// List keys for user1
	keys := manager.List("user1")
	assert.Len(t, keys, 2)

	// List keys for user2
	keys = manager.List("user2")
	assert.Len(t, keys, 1)

	// List keys for non-existent user
	keys = manager.List("user3")
	assert.Len(t, keys, 0)
}

func TestAPIKeyManager_Count(t *testing.T) {
	manager := NewAPIKeyManager()

	assert.Equal(t, 0, manager.Count())

	key1, err := manager.Generate("user1", "Key 1", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, manager.Count())

	_, err = manager.Generate("user1", "Key 2", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, manager.Count())

	// Revoke a key
	err = manager.Revoke(key1.Key)
	require.NoError(t, err)
	assert.Equal(t, 1, manager.Count()) // Count should decrease
}
