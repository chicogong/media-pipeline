package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// APIKey represents an API key
type APIKey struct {
	Key       string    `json:"key"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"` // Friendly name for the key
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Revoked   bool      `json:"revoked"`
}

// APIKeyManager manages API keys
type APIKeyManager struct {
	keys map[string]*APIKey // key -> APIKey
	mu   sync.RWMutex
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager() *APIKeyManager {
	return &APIKeyManager{
		keys: make(map[string]*APIKey),
	}
}

// Generate creates a new API key
func (m *APIKeyManager) Generate(userID, name string, expiresAt *time.Time) (*APIKey, error) {
	// Generate random 32-byte key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	key := "sk_" + base64.URLEncoding.EncodeToString(keyBytes)

	apiKey := &APIKey{
		Key:       key,
		UserID:    userID,
		Name:      name,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		Revoked:   false,
	}

	m.mu.Lock()
	m.keys[key] = apiKey
	m.mu.Unlock()

	return apiKey, nil
}

// Verify checks if an API key is valid
func (m *APIKeyManager) Verify(key string) (*APIKey, error) {
	m.mu.RLock()
	apiKey, exists := m.keys[key]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid API key")
	}

	if apiKey.Revoked {
		return nil, fmt.Errorf("API key has been revoked")
	}

	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	return apiKey, nil
}

// Revoke marks an API key as revoked
func (m *APIKeyManager) Revoke(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	apiKey, exists := m.keys[key]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	apiKey.Revoked = true
	return nil
}

// Delete removes an API key
func (m *APIKeyManager) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.keys[key]; !exists {
		return fmt.Errorf("API key not found")
	}

	delete(m.keys, key)
	return nil
}

// List returns all API keys for a user
func (m *APIKeyManager) List(userID string) []*APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []*APIKey
	for _, apiKey := range m.keys {
		if apiKey.UserID == userID {
			keys = append(keys, apiKey)
		}
	}

	return keys
}

// Count returns the total number of active keys
func (m *APIKeyManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, apiKey := range m.keys {
		if !apiKey.Revoked {
			count++
		}
	}

	return count
}
