// freee_oauth_auto.go は Playwright を使って完全自動でOAuth2認証を行います。
//
// 機能:
// - Playwrightでブラウザ自動操作
// - freeeログイン自動化
// - totp-cliを使った2FA自動入力
// - 認証コード自動取得
// - トークン保存
//
// 必要な準備:
//  1. Playwrightのインストール:
//     go run github.com/playwright-community/playwright-go/cmd/playwright@latest install
//  2. totp-cliのインストール:
//     brew install totp-cli  (macOS)
//     または https://github.com/yitsushi/totp-cli
//  3. 環境変数の設定:
//     export FREEE_CLIENT_ID="your_client_id"
//     export FREEE_CLIENT_SECRET="your_client_secret"
//     export FREEE_LOGIN_EMAIL="your@email.com"
//     export FREEE_LOGIN_PASSWORD="your_password"
//     export FREEE_TOTP_SECRET="BASE32_SECRET"
//
// 使い方:
//   go run examples/freee_oauth_auto.go
//
// 環境変数:
//   FREEE_CLIENT_ID       - OAuth2 Client ID（必須）
//   FREEE_CLIENT_SECRET   - OAuth2 Client Secret（必須）
//   FREEE_LOGIN_EMAIL     - freeeログインメールアドレス（必須）
//   FREEE_LOGIN_PASSWORD  - freeeログインパスワード（必須）
//   FREEE_TOTP_SECRET     - TOTP Secret (Base32)（2FA有効時は必須）
//   FREEE_REDIRECT_URL    - Redirect URL（デフォルト: urn:ietf:wg:oauth:2.0:oob）
//   FREEE_TOKEN_FILE      - トークンファイルパス（デフォルト: ~/.freee/token.json）
//   HEADLESS              - ヘッドレスモード（デフォルト: true, false=ブラウザ表示）
//
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"golang.org/x/oauth2"
)

// SavedToken represents the token saved to disk
type SavedToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

type AutoAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	LoginEmail   string
	LoginPwd     string
	TOTPSecret   string
	TokenFile    string
	Headless     bool
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("freee OAuth2 automatic authentication tool")

	cfg, err := loadAutoAuthConfig()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	logger.Info("configuration loaded",
		"client_id", cfg.ClientID,
		"email", cfg.LoginEmail,
		"has_totp", cfg.TOTPSecret != "",
		"headless", cfg.Headless,
		"token_file", cfg.TokenFile)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// OAuth2認証を自動実行
	token, err := performAutoAuth(ctx, cfg, logger)
	if err != nil {
		logger.Error("auto authentication failed", "error", err)
		os.Exit(1)
	}

	// トークンを保存
	if err := saveToken(cfg.TokenFile, token, logger); err != nil {
		logger.Error("failed to save token", "error", err)
		os.Exit(1)
	}

	logger.Info("authentication completed successfully",
		"token_file", cfg.TokenFile,
		"expires_at", token.Expiry)

	fmt.Println("\n==============================================")
	fmt.Println("✅ 自動認証完了！")
	fmt.Println("==============================================")
	fmt.Printf("\nトークンファイル: %s\n", cfg.TokenFile)
	fmt.Printf("有効期限: %s\n", token.Expiry.Format(time.RFC3339))
	fmt.Println("\nこれで unbooked_checker_production を実行できます。")
}

func loadAutoAuthConfig() (*AutoAuthConfig, error) {
	cfg := &AutoAuthConfig{
		ClientID:     os.Getenv("FREEE_CLIENT_ID"),
		ClientSecret: os.Getenv("FREEE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("FREEE_REDIRECT_URL"),
		LoginEmail:   os.Getenv("FREEE_LOGIN_EMAIL"),
		LoginPwd:     os.Getenv("FREEE_LOGIN_PASSWORD"),
		TOTPSecret:   os.Getenv("FREEE_TOTP_SECRET"),
		TokenFile:    os.Getenv("FREEE_TOKEN_FILE"),
		Headless:     os.Getenv("HEADLESS") != "false",
	}

	if cfg.RedirectURL == "" {
		cfg.RedirectURL = "urn:ietf:wg:oauth:2.0:oob"
	}

	if cfg.TokenFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cfg.TokenFile = homeDir + "/.freee/token.json"
	}

	// 必須パラメータチェック
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("FREEE_CLIENT_ID and FREEE_CLIENT_SECRET are required")
	}
	if cfg.LoginEmail == "" || cfg.LoginPwd == "" {
		return nil, fmt.Errorf("FREEE_LOGIN_EMAIL and FREEE_LOGIN_PASSWORD are required")
	}

	return cfg, nil
}

func performAutoAuth(ctx context.Context, cfg *AutoAuthConfig, logger *slog.Logger) (*oauth2.Token, error) {
	// OAuth2設定
	oauthConf := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.secure.freee.co.jp/public_api/authorize",
			TokenURL: "https://accounts.secure.freee.co.jp/public_api/token",
		},
		RedirectURL: cfg.RedirectURL,
		Scopes:      []string{"read", "write"},
	}

	authURL := oauthConf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	logger.Info("generated auth URL", "url", authURL)

	// Playwrightでブラウザ自動操作
	authCode, err := getAuthCodeWithPlaywright(ctx, authURL, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth code: %w", err)
	}

	logger.Info("auth code obtained", "code_length", len(authCode))

	// 認証コードをトークンに交換
	token, err := oauthConf.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange auth code: %w", err)
	}

	logger.Info("token obtained",
		"expires_at", token.Expiry,
		"has_refresh_token", token.RefreshToken != "")

	return token, nil
}

func getAuthCodeWithPlaywright(ctx context.Context, authURL string, cfg *AutoAuthConfig, logger *slog.Logger) (string, error) {
	logger.Info("starting playwright browser automation")

	// Playwrightを初期化
	pw, err := playwright.Run()
	if err != nil {
		return "", fmt.Errorf("failed to start playwright: %w", err)
	}
	defer pw.Stop()

	// ブラウザを起動
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(cfg.Headless),
	})
	if err != nil {
		return "", fmt.Errorf("failed to launch browser: %w", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		return "", fmt.Errorf("failed to create page: %w", err)
	}

	logger.Info("navigating to auth URL")

	// 認証URLを開く
	if _, err := page.Goto(authURL); err != nil {
		return "", fmt.Errorf("failed to navigate: %w", err)
	}

	// ログインページでメール入力を待つ
	logger.Info("waiting for login form")
	if err := page.Locator("input[type='email'], input[name='email']").WaitFor(); err != nil {
		return "", fmt.Errorf("login form not found: %w", err)
	}

	// メールアドレス入力
	logger.Info("entering email")
	if err := page.Locator("input[type='email'], input[name='email']").Fill(cfg.LoginEmail); err != nil {
		return "", fmt.Errorf("failed to enter email: %w", err)
	}

	// パスワード入力
	logger.Info("entering password")
	if err := page.Locator("input[type='password'], input[name='password']").Fill(cfg.LoginPwd); err != nil {
		return "", fmt.Errorf("failed to enter password: %w", err)
	}

	// ログインボタンをクリック
	logger.Info("clicking login button")
	if err := page.Locator("button[type='submit'], input[type='submit']").Click(); err != nil {
		return "", fmt.Errorf("failed to click login button: %w", err)
	}

	// 2FAページが表示されるか確認（3秒待機）
	time.Sleep(3 * time.Second)

	// 2FAコード入力フィールドが存在するかチェック
	totpInput := page.Locator("input[name='otp'], input[name='code'], input[type='text'][placeholder*='認証']")
	if count, _ := totpInput.Count(); count > 0 && cfg.TOTPSecret != "" {
		logger.Info("2FA detected, generating TOTP code")

		// totp-cliでコード生成
		totpCode, err := generateTOTPCode(cfg.TOTPSecret)
		if err != nil {
			return "", fmt.Errorf("failed to generate TOTP code: %w", err)
		}

		logger.Info("entering TOTP code")
		if err := totpInput.Fill(totpCode); err != nil {
			return "", fmt.Errorf("failed to enter TOTP code: %w", err)
		}

		// 2FA送信ボタンをクリック
		if err := page.Locator("button[type='submit'], input[type='submit']").Click(); err != nil {
			return "", fmt.Errorf("failed to click 2FA submit button: %w", err)
		}
	}

	// 「許可する」ボタンを待つ
	logger.Info("waiting for authorization page")
	time.Sleep(2 * time.Second)

	// 許可ボタンをクリック（様々なパターンに対応）
	logger.Info("clicking authorize button")
	authorizeBtn := page.Locator("button:has-text('許可'), button:has-text('承認'), input[value='許可'], input[value='承認']")
	if err := authorizeBtn.Click(); err != nil {
		return "", fmt.Errorf("failed to click authorize button: %w", err)
	}

	// 認証コードが表示されるのを待つ
	logger.Info("waiting for auth code")
	time.Sleep(2 * time.Second)

	// 認証コードを取得（複数のパターンを試す）
	var authCode string

	// パターン1: codeパラメータ付きURLにリダイレクト
	currentURL := page.URL()
	if strings.Contains(currentURL, "code=") {
		parts := strings.Split(currentURL, "code=")
		if len(parts) > 1 {
			authCode = strings.Split(parts[1], "&")[0]
			logger.Info("auth code found in URL", "code_length", len(authCode))
		}
	}

	// パターン2: ページ上のテキストから取得
	if authCode == "" {
		codeElement := page.Locator("code, pre, .auth-code, #auth-code")
		if text, err := codeElement.TextContent(); err == nil && text != "" {
			authCode = strings.TrimSpace(text)
			logger.Info("auth code found in page element", "code_length", len(authCode))
		}
	}

	// パターン3: input要素から取得
	if authCode == "" {
		codeInput := page.Locator("input[readonly], input[name='code']")
		if val, err := codeInput.InputValue(); err == nil && val != "" {
			authCode = strings.TrimSpace(val)
			logger.Info("auth code found in input field", "code_length", len(authCode))
		}
	}

	if authCode == "" {
		return "", fmt.Errorf("auth code not found in page")
	}

	return authCode, nil
}

func generateTOTPCode(secret string) (string, error) {
	// totp-cli を使ってコード生成
	cmd := exec.Command("totp-cli", "generate", secret)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("totp-cli failed: %w", err)
	}

	code := strings.TrimSpace(string(output))
	if len(code) != 6 {
		return "", fmt.Errorf("invalid TOTP code length: %d", len(code))
	}

	return code, nil
}

func saveToken(tokenFile string, token *oauth2.Token, logger *slog.Logger) error {
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
	dir := tokenFile[:len(tokenFile)-len("/token.json")]
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// ファイルに書き込み
	if err := os.WriteFile(tokenFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	logger.Info("token saved successfully", "token_file", tokenFile)
	return nil
}
