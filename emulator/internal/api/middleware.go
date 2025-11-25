package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pigeonworks-llc/freee-emulator/internal/oauth"
)

type contextKey string

const (
	contextKeyToken contextKey = "token"
)

// AuthMiddleware is a middleware that validates OAuth2 access tokens.
func AuthMiddleware(tokenManager *oauth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Missing Authorization header")
				return
			}

			// Parse Bearer token.
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid Authorization header format")
				return
			}

			token := parts[1]

			// Validate token.
			valid, err := tokenManager.ValidateToken(token)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to validate token")
				return
			}

			if !valid {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
				return
			}

			// Store token in context.
			ctx := context.WithValue(r.Context(), contextKeyToken, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, error, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error:            error,
		ErrorDescription: description,
	})
}
