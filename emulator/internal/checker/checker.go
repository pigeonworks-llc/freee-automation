// Package checker provides the core unbooked transaction checking logic
package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
)

// WalletTxn represents a wallet transaction
type WalletTxn struct {
	ID          int64  `json:"id"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Amount      int64  `json:"amount"`
}

// Response represents the API response
type Response struct {
	WalletTxns []WalletTxn `json:"wallet_txns"`
}

// Config holds the checker configuration
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	CompanyID    string
	WebhookURL   string
	BaseURL      string
	TokenFile    string
	AutoReauth   bool
}

// Result represents the check result
type Result struct {
	UnbookedCount    int
	Transactions     []WalletTxn
	NotificationSent bool
}

// SavedToken represents the token saved to disk/cloud storage
type SavedToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// TokenStore manages OAuth2 token persistence
type TokenStore struct {
	filePath string
}

// NewTokenStore creates a new TokenStore
func NewTokenStore(filePath string) *TokenStore {
	return &TokenStore{filePath: filePath}
}

// Load reads the token from disk
func (ts *TokenStore) Load() (*oauth2.Token, error) {
	data, err := os.ReadFile(ts.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var saved SavedToken
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  saved.AccessToken,
		RefreshToken: saved.RefreshToken,
		TokenType:    saved.TokenType,
		Expiry:       saved.Expiry,
	}, nil
}

// Save writes the token to disk
func (ts *TokenStore) Save(token *oauth2.Token) error {
	saved := SavedToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	dir := filepath.Dir(ts.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	if err := os.WriteFile(ts.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ClientID:     os.Getenv("FREEE_CLIENT_ID"),
		ClientSecret: os.Getenv("FREEE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("FREEE_REDIRECT_URL"),
		CompanyID:    os.Getenv("FREEE_COMPANY_ID"),
		WebhookURL:   os.Getenv("GOOGLE_CHAT_WEBHOOK"),
		BaseURL:      os.Getenv("FREEE_API_URL"),
		TokenFile:    os.Getenv("FREEE_TOKEN_FILE"),
		AutoReauth:   os.Getenv("FREEE_AUTO_REAUTH") == "true",
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.freee.co.jp"
	}

	if cfg.TokenFile == "" {
		// Cloud Run では /tmp を使用
		cfg.TokenFile = "/tmp/.freee/token.json"
	}

	if cfg.RedirectURL == "" {
		cfg.RedirectURL = "urn:ietf:wg:oauth:2.0:oob"
	}

	if cfg.CompanyID == "" {
		return nil, fmt.Errorf("FREEE_COMPANY_ID is required")
	}

	return cfg, nil
}

// RunCheck executes the unbooked transaction check
func RunCheck(ctx context.Context, logger *slog.Logger) (*Result, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	logger.Info("configuration loaded",
		"base_url", cfg.BaseURL,
		"company_id", cfg.CompanyID,
		"webhook_enabled", cfg.WebhookURL != "")

	// Create HTTP client
	client, authMode, err := createHTTPClient(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	logger.Info("HTTP client created", "auth_mode", authMode)

	// Fetch unbooked transactions
	unclassifiedCount, txns, err := fetchUnclassifiedTransactions(ctx, client, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	logger.Info("unbooked transactions found", "count", unclassifiedCount)

	// Send notification if needed
	notificationSent := false
	if unclassifiedCount > 0 && cfg.WebhookURL != "" {
		if err := sendNotification(ctx, cfg.WebhookURL, unclassifiedCount, txns, logger); err != nil {
			logger.Warn("failed to send notification", "error", err)
		} else {
			logger.Info("notification sent successfully")
			notificationSent = true
		}
	}

	return &Result{
		UnbookedCount:    unclassifiedCount,
		Transactions:     txns,
		NotificationSent: notificationSent,
	}, nil
}

func createHTTPClient(ctx context.Context, cfg *Config, logger *slog.Logger) (*http.Client, string, error) {
	// エミュレータモード
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		logger.Info("using emulator mode")
		return &http.Client{
			Timeout: 30 * time.Second,
			Transport: &tokenTransport{
				base:  http.DefaultTransport,
				token: os.Getenv("FREEE_ACCESS_TOKEN"),
			},
		}, "emulator", nil
	}

	// OAuth2モード
	conf := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.secure.freee.co.jp/public_api/authorize",
			TokenURL: "https://accounts.secure.freee.co.jp/public_api/token",
		},
		RedirectURL: cfg.RedirectURL,
		Scopes:      []string{"read"},
	}

	tokenStore := NewTokenStore(cfg.TokenFile)

	token, err := tokenStore.Load()
	if err != nil {
		return nil, "", fmt.Errorf("failed to load token: %w", err)
	}

	if token == nil {
		if cfg.AutoReauth {
			logger.Info("attempting automatic re-authentication")
			if err := runAutoAuth(logger); err != nil {
				return nil, "", fmt.Errorf("auto re-authentication failed: %w", err)
			}
			token, err = tokenStore.Load()
			if err != nil || token == nil {
				return nil, "", fmt.Errorf("failed to load token after auto re-auth: %w", err)
			}
		} else {
			return nil, "", fmt.Errorf("no saved token found")
		}
	}

	tokenSource := conf.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		if cfg.AutoReauth {
			logger.Info("token refresh failed, attempting auto re-auth")
			if err := runAutoAuth(logger); err != nil {
				return nil, "", fmt.Errorf("auto re-authentication failed: %w", err)
			}
			token, _ = tokenStore.Load()
			tokenSource = conf.TokenSource(ctx, token)
			newToken, err = tokenSource.Token()
			if err != nil {
				return nil, "", fmt.Errorf("failed to refresh token after auto re-auth: %w", err)
			}
		} else {
			return nil, "", fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	if newToken.AccessToken != token.AccessToken {
		logger.Info("token refreshed")
		tokenStore.Save(newToken)
	}

	return oauth2.NewClient(ctx, tokenSource), "oauth2", nil
}

type tokenTransport struct {
	base  http.RoundTripper
	token string
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.base.RoundTrip(req)
}

func fetchUnclassifiedTransactions(ctx context.Context, client *http.Client, cfg *Config, logger *slog.Logger) (int, []WalletTxn, error) {
	url := fmt.Sprintf("%s/api/1/wallet_txns?company_id=%s", cfg.BaseURL, cfg.CompanyID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read response: %w", err)
	}

	var data Response
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var unclassifiedTxns []WalletTxn
	for _, txn := range data.WalletTxns {
		if txn.Status == "unbooked" {
			unclassifiedTxns = append(unclassifiedTxns, txn)
		}
	}

	return len(unclassifiedTxns), unclassifiedTxns, nil
}

func sendNotification(ctx context.Context, webhookURL string, count int, txns []WalletTxn, logger *slog.Logger) error {
	message := fmt.Sprintf("⚠️ 未仕分け明細: %d件\n\n", count)
	for i, txn := range txns {
		if i >= 5 {
			message += fmt.Sprintf("...他%d件", count-5)
			break
		}
		message += fmt.Sprintf("• ID:%d 金額:¥%d %s\n", txn.ID, txn.Amount, txn.Description)
	}

	payload := map[string]string{"text": message}
	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notification failed: status=%d", resp.StatusCode)
	}

	return nil
}

func runAutoAuth(logger *slog.Logger) error {
	logger.Info("running automatic OAuth authentication")

	possiblePaths := []string{
		"examples/freee_oauth_auto.go",
		"freee_oauth_auto.go",
		"/app/freee_oauth_auto.go", // Cloud Run
	}

	var autoAuthPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			autoAuthPath = path
			break
		}
	}

	if autoAuthPath == "" {
		return fmt.Errorf("freee_oauth_auto.go not found")
	}

	cmd := exec.Command("go", "run", autoAuthPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute auto auth script: %w", err)
	}

	return nil
}
