// Package freee provides token management for freee OAuth2 authentication.
package freee

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	tokenEndpoint     = "https://accounts.secure.freee.co.jp/public_api/token"
	defaultTokenPath  = ".config/gmail-fetcher/freee_token.json"
	tokenExpiryBuffer = 5 * time.Minute // Refresh 5 minutes before expiry
)

// Token represents freee OAuth2 token information.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	CompanyID    int    `json:"company_id"`
}

// TokenManager handles token persistence and refresh.
type TokenManager struct {
	tokenPath    string
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// NewTokenManager creates a new token manager.
func NewTokenManager(clientID, clientSecret, tokenPath string) *TokenManager {
	if tokenPath == "" {
		home, _ := os.UserHomeDir()
		tokenPath = filepath.Join(home, defaultTokenPath)
	}
	return &TokenManager{
		tokenPath:    tokenPath,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// LoadToken loads token from file.
func (m *TokenManager) LoadToken() (*Token, error) {
	data, err := os.ReadFile(m.tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &token, nil
}

// SaveToken saves token to file.
func (m *TokenManager) SaveToken(token *Token) error {
	// Create directory if not exists
	dir := filepath.Dir(m.tokenPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(m.tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// IsExpired checks if the token is expired or will expire soon.
func (m *TokenManager) IsExpired(token *Token) bool {
	expiresAt := time.Unix(token.ExpiresAt, 0)
	return time.Now().Add(tokenExpiryBuffer).After(expiresAt)
}

// RefreshToken refreshes the access token using the refresh token.
func (m *TokenManager) RefreshToken(token *Token) (*Token, error) {
	if m.clientID == "" || m.clientSecret == "" {
		return nil, fmt.Errorf("client_id and client_secret are required for token refresh")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", m.clientID)
	data.Set("client_secret", m.clientSecret)
	data.Set("refresh_token", token.RefreshToken)

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("token refresh failed: %s - %s", errResp.Error, errResp.Description)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		CompanyID    int    `json:"company_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	newToken := &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Unix() + tokenResp.ExpiresIn,
		CompanyID:    tokenResp.CompanyID,
	}

	return newToken, nil
}

// GetValidToken returns a valid access token, refreshing if necessary.
func (m *TokenManager) GetValidToken() (*Token, error) {
	token, err := m.LoadToken()
	if err != nil {
		return nil, err
	}

	if !m.IsExpired(token) {
		return token, nil
	}

	// Token expired, refresh it
	newToken, err := m.RefreshToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Save refreshed token
	if err := m.SaveToken(newToken); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	return newToken, nil
}

// InitializeToken saves initial token from OAuth2 authorization code exchange.
func (m *TokenManager) InitializeToken(accessToken, refreshToken string, expiresIn int64, companyID int) error {
	token := &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Unix() + expiresIn,
		CompanyID:    companyID,
	}
	return m.SaveToken(token)
}
