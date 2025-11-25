// checker-server は Cloud Run 用の HTTP サーバー版未仕訳チェッカーです
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/pigeonworks-llc/freee-emulator/internal/checker"
)

func main() {
	// JSON構造化ログ設定
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/check", checkHandler)

	logger.Info("starting checker server", "port", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	logger := slog.Default()

	// Cloud Scheduler からのリクエストのみ許可（オプション）
	// Cloud Scheduler が設定する User-Agent をチェック
	if r.Header.Get("User-Agent") != "" && r.Header.Get("X-CloudScheduler") == "" {
		// 本番環境では User-Agent や IP アドレスでフィルタリング可能
	}

	logger.Info("check request received",
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// チェッカー実行
	result, err := checker.RunCheck(ctx, logger)
	if err != nil {
		logger.Error("check failed", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"error":   err.Error(),
			"time":    time.Now().Format(time.RFC3339),
			"success": false,
		})
		return
	}

	// 成功レスポンス
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "success",
		"unbooked_count": result.UnbookedCount,
		"transactions":   result.Transactions,
		"time":           time.Now().Format(time.RFC3339),
		"success":        true,
		"notification_sent": result.NotificationSent,
	})

	logger.Info("check completed successfully",
		"unbooked_count", result.UnbookedCount,
		"notification_sent", result.NotificationSent)
}
