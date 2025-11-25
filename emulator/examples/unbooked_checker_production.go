// unbooked_checker_production は本番環境対応の未仕訳チェッカーです。
//
// 機能:
// - OAuth2認証（自動トークンリフレッシュ）
// - トークンの永続化（~/.freee/token.json）
// - リトライロジック
// - 構造化ログ（JSON形式）
// - Google Chat通知
//
// セットアップ（初回のみ）:
//  1. freee OAuth2アプリを作成: https://developer.freee.co.jp/
//  2. 環境変数を設定:
//     export FREEE_CLIENT_ID="your_client_id"
//     export FREEE_CLIENT_SECRET="your_client_secret"
//     export FREEE_COMPANY_ID="123456"
//  3. 認証ツールを実行してトークンを取得:
//     go run examples/freee_oauth_setup.go
//
// 実行:
//   go run examples/unbooked_checker_production.go
//
// 環境変数:
//   FREEE_CLIENT_ID       - OAuth2 Client ID（必須：本番モード）
//   FREEE_CLIENT_SECRET   - OAuth2 Client Secret（必須：本番モード）
//   FREEE_COMPANY_ID      - 事業所ID（必須）
//   FREEE_REDIRECT_URL    - Redirect URL（デフォルト: urn:ietf:wg:oauth:2.0:oob）
//   FREEE_TOKEN_FILE      - トークンファイルパス（デフォルト: ~/.freee/token.json）
//   FREEE_API_URL         - APIベースURL（デフォルト: https://api.freee.co.jp）
//   GOOGLE_CHAT_WEBHOOK   - Google Chat Webhook URL（オプション）
//
// エミュレータモード（開発・テスト用）:
//   FREEE_CLIENT_IDとFREEE_CLIENT_SECRETを設定しない場合、エミュレータモードで動作します。
//   この場合、FREEE_ACCESS_TOKENとFREEE_API_URLを設定してください。
//
// 自動再認証:
//   FREEE_AUTO_REAUTH=true を設定すると、トークン期限切れ時に自動的に再認証します。
//   その場合、FREEE_LOGIN_EMAIL, FREEE_LOGIN_PASSWORD, FREEE_TOTP_SECRET も必要です。
//
package main

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

type WalletTxn struct {
	ID          int64  `json:"id"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Amount      int64  `json:"amount"`
}

type Response struct {
	WalletTxns []WalletTxn `json:"wallet_txns"`
}

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	CompanyID    string
	WebhookURL   string
	BaseURL      string
	TokenFile    string
}

// TokenStore manages OAuth2 token persistence
type TokenStore struct {
	filePath string
}

// SavedToken represents the token saved to disk
type SavedToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		ClientID:     os.Getenv("FREEE_CLIENT_ID"),
		ClientSecret: os.Getenv("FREEE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("FREEE_REDIRECT_URL"),
		CompanyID:    os.Getenv("FREEE_COMPANY_ID"),
		WebhookURL:   os.Getenv("GOOGLE_CHAT_WEBHOOK"),
		BaseURL:      os.Getenv("FREEE_API_URL"),
		TokenFile:    os.Getenv("FREEE_TOKEN_FILE"),
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.freee.co.jp" // 本番環境
	}

	if cfg.TokenFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cfg.TokenFile = homeDir + "/.freee/token.json"
	}

	// 必須パラメータのチェック
	if cfg.CompanyID == "" {
		return nil, fmt.Errorf("FREEE_COMPANY_ID is required")
	}

	return cfg, nil
}

func main() {
	// JSON形式の構造化ログを設定
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting unbooked transaction checker")

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	logger.Info("configuration loaded",
		"base_url", cfg.BaseURL,
		"company_id", cfg.CompanyID,
		"webhook_enabled", cfg.WebhookURL != "")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// OAuth2クライアント作成
	client, authMode, err := createHTTPClient(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to create HTTP client", "error", err)
		os.Exit(1)
	}

	logger.Info("HTTP client created", "auth_mode", authMode)

	// 未仕訳明細を取得
	unclassifiedCount, txns, err := fetchUnclassifiedTransactions(ctx, client, cfg, logger)
	if err != nil {
		logger.Error("failed to fetch transactions", "error", err)
		os.Exit(1)
	}

	// トランザクションIDと金額を収集
	txnIDs := make([]int64, len(txns))
	txnAmounts := make([]int64, len(txns))
	for i, txn := range txns {
		txnIDs[i] = txn.ID
		txnAmounts[i] = txn.Amount
	}

	logger.Info("unbooked transactions found",
		"count", unclassifiedCount,
		"transaction_ids", txnIDs,
		"amounts", txnAmounts)

	// 詳細をログ出力
	for _, txn := range txns {
		logger.Info("unbooked transaction detail",
			"id", txn.ID,
			"amount", txn.Amount,
			"description", txn.Description,
			"status", txn.Status)
	}

	// 通知送信
	if unclassifiedCount > 0 && cfg.WebhookURL != "" {
		if err := sendNotification(ctx, cfg.WebhookURL, unclassifiedCount, txns, logger); err != nil {
			logger.Warn("failed to send notification", "error", err)
		} else {
			logger.Info("notification sent successfully")
		}
	}

	// 終了コード設定（監視システム用）
	if unclassifiedCount > 10 {
		logger.Error("too many unclassified transactions",
			"count", unclassifiedCount,
			"threshold", 10)
		os.Exit(1) // アラート用
	}

	logger.Info("checker completed successfully", "unbooked_count", unclassifiedCount)
}

func createHTTPClient(ctx context.Context, cfg *Config, logger *slog.Logger) (*http.Client, string, error) {
	// エミュレータモード（開発・テスト用）
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		logger.Info("using emulator mode", "token_env_set", os.Getenv("FREEE_ACCESS_TOKEN") != "")
		return &http.Client{
			Timeout: 30 * time.Second,
			Transport: &tokenTransport{
				base:  http.DefaultTransport,
				token: os.Getenv("FREEE_ACCESS_TOKEN"),
			},
		}, "emulator", nil
	}

	// 本番モード（OAuth2）
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

	// 保存されたトークンを読み込む
	token, err := tokenStore.Load()
	if err != nil {
		return nil, "", fmt.Errorf("failed to load token: %w", err)
	}

	if token == nil {
		logger.Warn("no saved token found", "token_file", cfg.TokenFile)

		// 自動再認証が有効な場合は試みる
		if os.Getenv("FREEE_AUTO_REAUTH") == "true" {
			logger.Info("attempting automatic re-authentication")
			if err := runAutoAuth(logger); err != nil {
				return nil, "", fmt.Errorf("auto re-authentication failed: %w", err)
			}

			// 再度トークンを読み込む
			token, err = tokenStore.Load()
			if err != nil || token == nil {
				return nil, "", fmt.Errorf("failed to load token after auto re-auth: %w", err)
			}
			logger.Info("token obtained via auto re-authentication")
		} else {
			return nil, "", fmt.Errorf("no saved token found. Please run freee_oauth_setup.go or freee_oauth_auto.go")
		}
	}

	logger.Info("loaded saved token", "expires_at", token.Expiry, "has_refresh_token", token.RefreshToken != "")

	// トークンソースを作成（自動リフレッシュ付き）
	tokenSource := conf.TokenSource(ctx, token)

	// 有効期限チェック＆必要に応じてリフレッシュ
	newToken, err := tokenSource.Token()
	if err != nil {
		logger.Error("token refresh failed", "error", err)

		// 自動再認証が有効な場合は試みる
		if os.Getenv("FREEE_AUTO_REAUTH") == "true" {
			logger.Info("attempting automatic re-authentication due to refresh failure")
			if err := runAutoAuth(logger); err != nil {
				return nil, "", fmt.Errorf("auto re-authentication failed: %w", err)
			}

			// 再度トークンを読み込んでリトライ
			token, err = tokenStore.Load()
			if err != nil {
				return nil, "", fmt.Errorf("failed to load token after auto re-auth: %w", err)
			}

			tokenSource = conf.TokenSource(ctx, token)
			newToken, err = tokenSource.Token()
			if err != nil {
				return nil, "", fmt.Errorf("failed to refresh token after auto re-auth: %w", err)
			}
			logger.Info("token refreshed successfully after auto re-authentication")
		} else {
			return nil, "", fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	// トークンが更新された場合は保存
	if newToken.AccessToken != token.AccessToken {
		logger.Info("token refreshed", "new_expiry", newToken.Expiry)
		if err := tokenStore.Save(newToken); err != nil {
			logger.Warn("failed to save refreshed token", "error", err)
		} else {
			logger.Info("refreshed token saved", "token_file", cfg.TokenFile)
		}
	}

	return oauth2.NewClient(ctx, tokenSource), "oauth2", nil
}

// tokenTransport はシンプルなBearer Token認証用のTransport
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

	logger.Info("fetching wallet transactions", "url", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// リトライロジック（簡易版）
	var resp *http.Response
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			break
		}
		if i < maxRetries-1 {
			logger.Warn("retrying API request",
				"attempt", i+1,
				"max_retries", maxRetries,
				"error", err,
				"status_code", func() int {
					if resp != nil {
						return resp.StatusCode
					}
					return 0
				}())
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}

	if err != nil {
		return 0, nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	logger.Info("API response received", "status_code", resp.StatusCode)

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

	logger.Info("parsed wallet transactions", "total_count", len(data.WalletTxns))

	// 未仕訳をカウント＆収集
	unclassifiedCount := 0
	var unclassifiedTxns []WalletTxn
	for _, txn := range data.WalletTxns {
		if txn.Status == "unbooked" {
			unclassifiedCount++
			unclassifiedTxns = append(unclassifiedTxns, txn)
		}
	}

	return unclassifiedCount, unclassifiedTxns, nil
}

func sendNotification(ctx context.Context, webhookURL string, count int, txns []WalletTxn, logger *slog.Logger) error {
	logger.Info("sending notification", "count", count, "webhook_url_set", webhookURL != "")

	// 詳細メッセージ作成
	message := fmt.Sprintf("⚠️ 未仕分け明細: %d件\n\n", count)
	for i, txn := range txns {
		if i >= 5 { // 最大5件まで表示
			message += fmt.Sprintf("...他%d件", count-5)
			break
		}
		message += fmt.Sprintf("• ID:%d 金額:¥%d %s\n", txn.ID, txn.Amount, txn.Description)
	}

	payload := map[string]string{"text": message}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

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

	logger.Info("notification response", "status_code", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("notification failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
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
			return nil, nil // トークンファイルが存在しない場合はnilを返す
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

	// ディレクトリを作成
	dir := ts.filePath[:len(ts.filePath)-len("/token.json")]
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// ファイルに書き込み（パーミッション 0600 で保護）
	if err := os.WriteFile(ts.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// runAutoAuth executes the automatic OAuth authentication script
func runAutoAuth(logger *slog.Logger) error {
	logger.Info("running automatic OAuth authentication")

	// freee_oauth_auto.go のパスを取得
	// カレントディレクトリからの相対パスまたは絶対パスを試す
	possiblePaths := []string{
		"examples/freee_oauth_auto.go",
		"freee_oauth_auto.go",
		filepath.Join(os.Getenv("GOPATH"), "src/github.com/pigeonworks-llc/freee-emulator/examples/freee_oauth_auto.go"),
	}

	var autoAuthPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			autoAuthPath = path
			break
		}
	}

	if autoAuthPath == "" {
		return fmt.Errorf("freee_oauth_auto.go not found in common locations")
	}

	// go run で実行
	cmd := exec.Command("go", "run", autoAuthPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 環境変数を継承
	cmd.Env = os.Environ()

	logger.Info("executing auto auth script", "path", autoAuthPath)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute auto auth script: %w", err)
	}

	logger.Info("auto auth script completed successfully")
	return nil
}
