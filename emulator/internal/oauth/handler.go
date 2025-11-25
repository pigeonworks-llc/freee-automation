package oauth

import (
	"encoding/json"
	"net/http"
)

// TokenResponse represents the OAuth2 token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	CompanyID    int64  `json:"company_id"`
}

// ErrorResponse represents an OAuth2 error response.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// Handler handles OAuth2 endpoints.
type Handler struct {
	tokenManager *TokenManager
}

// NewHandler creates a new OAuth2 handler.
func NewHandler(tm *TokenManager) *Handler {
	return &Handler{tokenManager: tm}
}

// HandleToken handles the token endpoint.
func (h *Handler) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "invalid_request", "Method not allowed")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse form")
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType == "" {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Missing grant_type")
		return
	}

	// For simplicity, accept any grant type and generate tokens.
	accessToken, err := h.tokenManager.GenerateToken()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "server_error", "Failed to generate access token")
		return
	}

	refreshToken, err := h.tokenManager.GenerateRefreshToken()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "server_error", "Failed to generate refresh token")
		return
	}

	response := TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokenTTL,
		CompanyID:    1, // Default company ID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// writeError writes an OAuth2 error response.
func (h *Handler) writeError(w http.ResponseWriter, status int, error, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error:            error,
		ErrorDescription: description,
	})
}
