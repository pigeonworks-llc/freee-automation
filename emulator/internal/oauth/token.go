package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pigeonworks-llc/freee-emulator/internal/store"
)

const (
	tokenLength     = 32
	tokenTTL        = 3600       // 1 hour in seconds
	refreshTokenTTL = 2592000    // 30 days in seconds
)

// TokenManager manages OAuth2 access tokens.
type TokenManager struct {
	store *store.Store
}

// NewTokenManager creates a new TokenManager.
func NewTokenManager(s *store.Store) *TokenManager {
	return &TokenManager{store: s}
}

// GenerateToken generates a new access token and stores it.
func (tm *TokenManager) GenerateToken() (string, error) {
	token, err := generateRandomToken(tokenLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Store token with expiration time.
	expiresAt := time.Now().Add(tokenTTL * time.Second).Unix()
	if err := tm.store.PutString(store.BucketTokens, token, fmt.Sprintf("%d", expiresAt)); err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	return token, nil
}

// GenerateRefreshToken generates a new refresh token and stores it.
func (tm *TokenManager) GenerateRefreshToken() (string, error) {
	token, err := generateRandomToken(tokenLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token with longer expiration time.
	expiresAt := time.Now().Add(refreshTokenTTL * time.Second).Unix()
	if err := tm.store.PutString(store.BucketTokens, "refresh:"+token, fmt.Sprintf("%d", expiresAt)); err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return token, nil
}

// ValidateToken validates an access token.
func (tm *TokenManager) ValidateToken(token string) (bool, error) {
	expiresAtStr, err := tm.store.GetString(store.BucketTokens, token)
	if err != nil {
		if err == store.ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to get token: %w", err)
	}

	var expiresAt int64
	if _, err := fmt.Sscanf(expiresAtStr, "%d", &expiresAt); err != nil {
		return false, fmt.Errorf("failed to parse expiration time: %w", err)
	}

	// Check if token is expired.
	if time.Now().Unix() > expiresAt {
		// Delete expired token.
		_ = tm.store.DeleteString(store.BucketTokens, token)
		return false, nil
	}

	return true, nil
}

// RevokeToken revokes an access token.
func (tm *TokenManager) RevokeToken(token string) error {
	return tm.store.DeleteString(store.BucketTokens, token)
}

// generateRandomToken generates a random token string.
func generateRandomToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
