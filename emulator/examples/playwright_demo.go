// playwright_demo は freee ログインフローを Playwright で自動化するデモです
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

	logger.Info("🎭 Playwright Demo - freee Login Flow Automation")

	// Playwrightを初期化
	pw, err := playwright.Run()
	if err != nil {
		logger.Error("failed to start playwright", "error", err)
		os.Exit(1)
	}
	defer pw.Stop()

	logger.Info("launching browser", "headless", false)

	// ブラウザを起動（ヘッドフルモード - SlowMoで動作を遅くして見やすく）
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		SlowMo:   playwright.Float(800), // 800msごとに1アクション
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

	logger.Info("📱 Step 1/3: Opening login page")
	if _, err := page.Goto("http://localhost:9090/"); err != nil {
		logger.Error("failed to navigate", "error", err)
		os.Exit(1)
	}

	time.Sleep(1 * time.Second)

	logger.Info("✍️  Entering email address")
	if err := page.Locator("input[type='email']").Fill("test@example.com"); err != nil {
		logger.Error("failed to enter email", "error", err)
		os.Exit(1)
	}

	logger.Info("🔒 Entering password")
	if err := page.Locator("input[type='password']").Fill("password"); err != nil {
		logger.Error("failed to enter password", "error", err)
		os.Exit(1)
	}

	logger.Info("👆 Clicking login button")
	if err := page.Locator("button[type='submit']").Click(); err != nil {
		logger.Error("failed to click login", "error", err)
		os.Exit(1)
	}

	time.Sleep(2 * time.Second)

	logger.Info("📱 Step 2/3: Entering 2FA code")
	logger.Info("🔢 Entering TOTP code: 123456")
	if err := page.Locator("input[name='otp']").Fill("123456"); err != nil {
		logger.Error("failed to enter OTP", "error", err)
		os.Exit(1)
	}

	logger.Info("👆 Submitting 2FA")
	if err := page.Locator("button[type='submit']").Click(); err != nil {
		logger.Error("failed to submit 2FA", "error", err)
		os.Exit(1)
	}

	time.Sleep(2 * time.Second)

	logger.Info("📱 Step 3/3: Authorizing application")
	logger.Info("✅ Clicking 'Authorize' button")
	if err := page.Locator("button.btn-auth").Click(); err != nil {
		logger.Error("failed to authorize", "error", err)
		os.Exit(1)
	}

	time.Sleep(2 * time.Second)

	logger.Info("🎉 Getting authorization code")
	authCode, err := page.Locator("#auth-code").TextContent()
	if err != nil {
		logger.Error("failed to get auth code", "error", err)
		os.Exit(1)
	}

	logger.Info("authorization code obtained", "code", authCode)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ Playwright Automation Demo 完了！")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\n認証コード: %s\n\n", authCode)
	fmt.Println("フロー:")
	fmt.Println("  1. ログインページ → メール・パスワード入力")
	fmt.Println("  2. 2FA認証 → TOTPコード入力")
	fmt.Println("  3. アプリ認証 → 許可ボタンクリック")
	fmt.Println("  4. 認証コード取得 ✓")
	fmt.Println("\nブラウザは5秒後に自動的に閉じます...")

	time.Sleep(5 * time.Second)
}

func strings_Repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

var strings = struct {
	Repeat func(string, int) string
}{
	Repeat: strings_Repeat,
}
