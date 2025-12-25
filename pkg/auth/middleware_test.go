package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_JWT_Valid(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", time.Hour)
	apiKeyManager := NewAPIKeyManager()
	middleware := NewAuthMiddleware(jwtManager, apiKeyManager, false)

	// Generate valid token
	token, err := jwtManager.Generate("user123", "user@example.com", "admin")
	require.NoError(t, err)

	// Create test handler
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r)
		assert.True(t, ok)
		assert.Equal(t, "user123", userID)

		email, ok := GetUserEmail(r)
		assert.True(t, ok)
		assert.Equal(t, "user@example.com", email)

		role, ok := GetUserRole(r)
		assert.True(t, ok)
		assert.Equal(t, "admin", role)

		w.WriteHeader(http.StatusOK)
	}))

	// Create request with Bearer token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute request
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_JWT_Invalid(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", time.Hour)
	apiKeyManager := NewAPIKeyManager()
	middleware := NewAuthMiddleware(jwtManager, apiKeyManager, false)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called for invalid token")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthMiddleware_APIKey_Valid(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", time.Hour)
	apiKeyManager := NewAPIKeyManager()
	middleware := NewAuthMiddleware(jwtManager, apiKeyManager, false)

	// Generate valid API key
	apiKey, err := apiKeyManager.Generate("user456", "Test Key", nil)
	require.NoError(t, err)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserID(r)
		assert.True(t, ok)
		assert.Equal(t, "user456", userID)

		method, ok := GetAuthMethod(r)
		assert.True(t, ok)
		assert.Equal(t, "apikey", method)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-API-Key", apiKey.Key)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthMiddleware_APIKey_Invalid(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", time.Hour)
	apiKeyManager := NewAPIKeyManager()
	middleware := NewAuthMiddleware(jwtManager, apiKeyManager, false)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called for invalid API key")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-API-Key", "invalid-key")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthMiddleware_NoAuth_Required(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", time.Hour)
	apiKeyManager := NewAPIKeyManager()
	middleware := NewAuthMiddleware(jwtManager, apiKeyManager, false)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called without authentication")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthMiddleware_NoAuth_Optional(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", time.Hour)
	apiKeyManager := NewAPIKeyManager()
	middleware := NewAuthMiddleware(jwtManager, apiKeyManager, true) // Optional auth

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should be called even without auth
		userID, ok := GetUserID(r)
		assert.False(t, ok)
		assert.Empty(t, userID)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/public", nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireRole(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", time.Hour)
	apiKeyManager := NewAPIKeyManager()
	authMiddleware := NewAuthMiddleware(jwtManager, apiKeyManager, false)

	// Generate token with admin role
	adminToken, err := jwtManager.Generate("admin123", "admin@example.com", "admin")
	require.NoError(t, err)

	// Generate token with user role
	userToken, err := jwtManager.Generate("user123", "user@example.com", "user")
	require.NoError(t, err)

	// Create handler that requires admin role
	handler := authMiddleware.Handler(
		RequireRole("admin")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		),
	)

	// Test with admin token - should succeed
	t.Run("admin access", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	// Test with user token - should fail
	t.Run("user denied", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}
