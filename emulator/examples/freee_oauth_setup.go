// freee_oauth_setup は初回のOAuth2認証を行い、トークンを保存するツールです。
//
// 使い方:
//  1. freeeアプリケーションを作成: https://developer.freee.co.jp/
//  2. 環境変数を設定:
//     export FREEE_CLIENT_ID="your_client_id"
//     export FREEE_CLIENT_SECRET="your_client_secret"
//  3. このツールを実行:
//     go run examples/freee_oauth_setup.go
//  4. ブラウザで認証URLを開き、認証コードを取得
//  5. 認証コードを入力
//
// トークンは ~/.freee/token.json に保存されます。
// その後、unbooked_checker_production がこのトークンを自動的に使用・リフレッシュします。
//
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"golang.org/x/oauth2"
)

// SavedToken represents the token saved to disk
type SavedToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("freee OAuth2 setup tool")

	// 環境変数から設定を読み込む
	clientID := os.Getenv("FREEE_CLIENT_ID")
	clientSecret := os.Getenv("FREEE_CLIENT_SECRET")
	redirectURL := os.Getenv("FREEE_REDIRECT_URL")
	tokenFile := os.Getenv("FREEE_TOKEN_FILE")

	if clientID == "" || clientSecret == "" {
		logger.Error("FREEE_CLIENT_ID and FREEE_CLIENT_SECRET are required")
		os.Exit(1)
	}

	if redirectURL == "" {
		redirectURL = "urn:ietf:wg:oauth:2.0:oob" // デフォルト：OOB flow
	}

	if tokenFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			logger.Error("failed to get home directory", "error", err)
			os.Exit(1)
		}
		tokenFile = homeDir + "/.freee/token.json"
	}

	logger.Info("configuration loaded",
		"client_id", clientID,
		"redirect_url", redirectURL,
		"token_file", tokenFile)

	// OAuth2設定
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.secure.freee.co.jp/public_api/authorize",
			TokenURL: "https://accounts.secure.freee.co.jp/public_api/token",
		},
		RedirectURL: redirectURL,
		Scopes:      []string{"read", "write"},
	}

	// 認証URLを生成
	authURL := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)

	fmt.Println("\n==============================================")
	fmt.Println("freee OAuth2 認証セットアップ")
	fmt.Println("==============================================")
	fmt.Println("\n1. 以下のURLをブラウザで開いてください:")
	fmt.Printf("\n%s\n\n", authURL)
	fmt.Println("2. freeeにログインして、アプリを認証してください")
	fmt.Println("3. 認証後に表示される認証コードを入力してください")
	fmt.Print("\n認証コード: ")

	var authCode string
	if _, err := fmt.Scanln(&authCode); err != nil {
		logger.Error("failed to read auth code", "error", err)
		os.Exit(1)
	}

	logger.Info("exchanging auth code for token")

	// 認証コードをトークンに交換
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := conf.Exchange(ctx, authCode)
	if err != nil {
		logger.Error("failed to exchange auth code", "error", err)
		os.Exit(1)
	}

	logger.Info("token obtained",
		"expires_at", token.Expiry,
		"has_refresh_token", token.RefreshToken != "")

	// トークンを保存
	saved := SavedToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		logger.Error("failed to marshal token", "error", err)
		os.Exit(1)
	}

	// ディレクトリを作成
	dir := tokenFile[:len(tokenFile)-len("/token.json")]
	if err := os.MkdirAll(dir, 0700); err != nil {
		logger.Error("failed to create token directory", "error", err)
		os.Exit(1)
	}

	// ファイルに書き込み
	if err := os.WriteFile(tokenFile, data, 0600); err != nil {
		logger.Error("failed to write token file", "error", err)
		os.Exit(1)
	}

	logger.Info("token saved successfully", "token_file", tokenFile)

	fmt.Println("\n==============================================")
	fmt.Println("✅ セットアップ完了！")
	fmt.Println("==============================================")
	fmt.Printf("\nトークンファイル: %s\n", tokenFile)
	fmt.Printf("有効期限: %s\n", token.Expiry.Format(time.RFC3339))
	fmt.Println("\nこれで unbooked_checker_production を実行できます。")
	fmt.Println("トークンは自動的にリフレッシュされます。")
}
