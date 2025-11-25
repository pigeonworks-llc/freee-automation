// test_playwright_demo はエミュレータのログイン画面でPlaywrightの動作をデモします
package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Playwright Demo - freee Emulator Login Flow")

	baseURL := os.Getenv("FREEE_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	// OAuth authorize URL
	authURL := fmt.Sprintf("%s/oauth/authorize?client_id=test&redirect_uri=urn:ietf:wg:oauth:2.0:oob&response_type=code&state=test", baseURL)

	logger.Info("starting playwright", "headless", false)

	// Playwrightを初期化
	pw, err := playwright.Run()
	if err != nil {
		logger.Error("failed to start playwright", "error", err)
		os.Exit(1)
	}
	defer pw.Stop()

	// ブラウザを起動（ヘッドフルモード）
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		SlowMo:   playwright.Float(500), // 動作を遅くして見やすく
	})
	if err != nil {
		logger.Error("failed to launch browser", "error", err)
		os.Exit(1)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		logger.Error("failed to create page", "error", err)
		os.Exit(1)
	}

	logger.Info("navigating to auth URL", "url", authURL)

	// 認証URLを開く
	if _, err := page.Goto(authURL); err != nil {
		logger.Error("failed to navigate", "error", err)
		os.Exit(1)
	}

	logger.Info("waiting for login form")
	time.Sleep(1 * time.Second)

	// メールアドレス入力
	logger.Info("entering email")
	if err := page.Locator("input[type='email']").Fill("test@example.com"); err != nil {
		logger.Error("failed to enter email", "error", err)
		os.Exit(1)
	}

	time.Sleep(500 * time.Millisecond)

	// パスワード入力
	logger.Info("entering password")
	if err := page.Locator("input[type='password']").Fill("password"); err != nil {
		logger.Error("failed to enter password", "error", err)
		os.Exit(1)
	}

	time.Sleep(500 * time.Millisecond)

	// ログインボタンをクリック
	logger.Info("clicking login button")
	if err := page.Locator("button[type='submit']").Click(); err != nil {
		logger.Error("failed to click login button", "error", err)
		os.Exit(1)
	}

	// 2FAページを待つ
	logger.Info("waiting for 2FA page")
	time.Sleep(2 * time.Second)

	// 2FAコード入力
	logger.Info("entering TOTP code")
	if err := page.Locator("input[name='otp']").Fill("123456"); err != nil {
		logger.Error("failed to enter TOTP code", "error", err)
		os.Exit(1)
	}

	time.Sleep(500 * time.Millisecond)

	// 2FA送信
	logger.Info("submitting 2FA")
	if err := page.Locator("button[type='submit']").Click(); err != nil {
		logger.Error("failed to submit 2FA", "error", err)
		os.Exit(1)
	}

	// 許可ページを待つ
	logger.Info("waiting for authorization page")
	time.Sleep(2 * time.Second)

	// 許可ボタンをクリック
	logger.Info("clicking authorize button")
	authorizeBtn := page.Locator("button.btn-authorize")
	if err := authorizeBtn.Click(); err != nil {
		logger.Error("failed to click authorize button", "error", err)
		os.Exit(1)
	}

	// 認証コードページを待つ
	logger.Info("waiting for auth code page")
	time.Sleep(2 * time.Second)

	// 認証コードを取得
	codeElement := page.Locator("#auth-code")
	authCode, err := codeElement.TextContent()
	if err != nil {
		logger.Error("failed to get auth code", "error", err)
		os.Exit(1)
	}

	logger.Info("auth code obtained", "code", authCode)

	fmt.Println("\n==============================================")
	fmt.Println("✅ Playwright Demo 完了！")
	fmt.Println("==============================================")
	fmt.Printf("\n認証コード: %s\n", authCode)
	fmt.Println("\nブラウザは5秒後に自動的に閉じます...")

	time.Sleep(5 * time.Second)
}
