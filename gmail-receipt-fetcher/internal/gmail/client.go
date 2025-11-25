// Package gmail provides Gmail API client with OAuth2 authentication.
package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client wraps Gmail API service with authentication.
type Client struct {
	service *gmail.Service
	userID  string
}

// Config holds OAuth2 configuration paths.
type Config struct {
	CredentialsPath string // Path to credentials.json from Google Cloud Console
	TokenPath       string // Path to store/load OAuth2 token
	APIEndpoint     string // Custom API endpoint (for emulator, e.g., "http://localhost:8081")
	AccessToken     string // Direct access token (for emulator, bypasses OAuth flow)
}

// NewClient creates a new Gmail API client with OAuth2 authentication.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	// If custom endpoint is specified (emulator mode), use simplified client
	if cfg.APIEndpoint != "" {
		return newEmulatorClient(ctx, cfg)
	}

	// Read credentials file
	credBytes, err := os.ReadFile(cfg.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %w", err)
	}

	// Parse OAuth2 config
	oauthConfig, err := google.ConfigFromJSON(credBytes, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	// Get OAuth2 token
	token, err := getToken(ctx, oauthConfig, cfg.TokenPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %w", err)
	}

	// Create Gmail service
	client := oauthConfig.Client(ctx, token)
	service, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Gmail service: %w", err)
	}

	return &Client{
		service: service,
		userID:  "me",
	}, nil
}

// newEmulatorClient creates a client for Gmail emulator (no OAuth required).
func newEmulatorClient(ctx context.Context, cfg Config) (*Client, error) {
	var opts []option.ClientOption

	// Set custom endpoint
	opts = append(opts, option.WithEndpoint(cfg.APIEndpoint))

	// Use access token if provided, otherwise no auth
	if cfg.AccessToken != "" {
		opts = append(opts, option.WithAPIKey(cfg.AccessToken))
	} else {
		// For emulator without auth, use a no-op HTTP client
		opts = append(opts, option.WithHTTPClient(&http.Client{}))
		opts = append(opts, option.WithoutAuthentication())
	}

	service, err := gmail.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create Gmail service: %w", err)
	}

	return &Client{
		service: service,
		userID:  "me",
	}, nil
}

// getToken retrieves OAuth2 token from file or initiates OAuth flow.
func getToken(ctx context.Context, config *oauth2.Config, tokenPath string) (*oauth2.Token, error) {
	// Try to load existing token
	token, err := loadToken(tokenPath)
	if err == nil {
		return token, nil
	}

	// No valid token, need to authenticate
	token, err = getTokenFromWeb(ctx, config)
	if err != nil {
		return nil, err
	}

	// Save token for future use
	if err := saveToken(tokenPath, token); err != nil {
		fmt.Printf("Warning: unable to save token: %v\n", err)
	}

	return token, nil
}

// getTokenFromWeb initiates OAuth2 flow via browser.
func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Use localhost callback
	config.RedirectURL = "http://localhost:8090/callback"

	// Generate auth URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Printf("Go to the following link in your browser:\n%v\n\n", authURL)

	// Start local server to receive callback
	codeChan := make(chan string)
	errChan := make(chan error)

	server := &http.Server{Addr: ":8090"}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no code in callback")
			return
		}
		fmt.Fprintf(w, "Authentication successful! You can close this window.")
		codeChan <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	var code string
	select {
	case code = <-codeChan:
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	server.Shutdown(ctx)

	// Exchange code for token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("unable to exchange code: %w", err)
	}

	return token, nil
}

// loadToken loads token from file.
func loadToken(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, err
	}

	return token, nil
}

// saveToken saves token to file.
func saveToken(path string, token *oauth2.Token) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

// Service returns the underlying Gmail service for advanced operations.
func (c *Client) Service() *gmail.Service {
	return c.service
}

// UserID returns the user ID (typically "me").
func (c *Client) UserID() string {
	return c.userID
}
