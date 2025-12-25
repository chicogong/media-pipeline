package auth

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
	// UserEmailKey is the context key for user email
	UserEmailKey contextKey = "user_email"
	// UserRoleKey is the context key for user role
	UserRoleKey contextKey = "user_role"
	// AuthMethodKey is the context key for authentication method
	AuthMethodKey contextKey = "auth_method"
)

// AuthMiddleware provides HTTP middleware for authentication
type AuthMiddleware struct {
	jwtManager    *JWTManager
	apiKeyManager *APIKeyManager
	optional      bool // If true, authentication is optional
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtManager *JWTManager, apiKeyManager *APIKeyManager, optional bool) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager:    jwtManager,
		apiKeyManager: apiKeyManager,
		optional:      optional,
	}
}

// Handler returns the HTTP middleware handler
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to authenticate from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// Try Bearer token (JWT)
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if m.authenticateJWT(w, r, token) {
					next.ServeHTTP(w, r)
					return
				}
				if !m.optional {
					return // Error already written
				}
			}
		}

		// Try API-Key header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			if m.authenticateAPIKey(w, r, apiKey) {
				next.ServeHTTP(w, r)
				return
			}
			if !m.optional {
				return // Error already written
			}
		}

		// If optional and no valid auth, continue without authentication
		if m.optional {
			next.ServeHTTP(w, r)
			return
		}

		// No valid authentication found and auth is required
		http.Error(w, "Unauthorized: No valid authentication provided", http.StatusUnauthorized)
	})
}

// authenticateJWT validates JWT token and sets user context
func (m *AuthMiddleware) authenticateJWT(w http.ResponseWriter, r *http.Request, token string) bool {
	claims, err := m.jwtManager.Verify(token)
	if err != nil {
		http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
		return false
	}

	// Set user context
	ctx := r.Context()
	ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
	ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
	ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
	ctx = context.WithValue(ctx, AuthMethodKey, "jwt")
	*r = *r.WithContext(ctx)

	return true
}

// authenticateAPIKey validates API key and sets user context
func (m *AuthMiddleware) authenticateAPIKey(w http.ResponseWriter, r *http.Request, key string) bool {
	apiKey, err := m.apiKeyManager.Verify(key)
	if err != nil {
		http.Error(w, "Unauthorized: Invalid or revoked API key", http.StatusUnauthorized)
		return false
	}

	// Set user context
	ctx := r.Context()
	ctx = context.WithValue(ctx, UserIDKey, apiKey.UserID)
	ctx = context.WithValue(ctx, AuthMethodKey, "apikey")
	*r = *r.WithContext(ctx)

	return true
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	return userID, ok
}

// GetUserEmail extracts user email from request context
func GetUserEmail(r *http.Request) (string, bool) {
	email, ok := r.Context().Value(UserEmailKey).(string)
	return email, ok
}

// GetUserRole extracts user role from request context
func GetUserRole(r *http.Request) (string, bool) {
	role, ok := r.Context().Value(UserRoleKey).(string)
	return role, ok
}

// GetAuthMethod extracts authentication method from request context
func GetAuthMethod(r *http.Request) (string, bool) {
	method, ok := r.Context().Value(AuthMethodKey).(string)
	return method, ok
}

// RequireRole is a middleware that requires a specific role
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := GetUserRole(r)
			if !ok || userRole != role {
				http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
