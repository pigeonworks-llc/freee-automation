// Package main freee API Emulator Server
//
// @title freee API Emulator
// @version 1.0
// @description Local emulator for freee accounting API for development and testing
// @termsOfService http://swagger.io/terms/
//
// @contact.name API Support
// @contact.email support@example.com
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:8080
// @BasePath /api/1
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the access token
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pigeonworks-llc/freee-emulator/internal/api"
	"github.com/pigeonworks-llc/freee-emulator/internal/oauth"
	"github.com/pigeonworks-llc/freee-emulator/internal/store"
)

const (
	defaultPort      = "8080"
	defaultDBPath    = "./data/freee.db"
	defaultUploadDir = "./data/receipts"
)

func main() {
	// Setup structured JSON logging.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Get configuration from environment variables.
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = defaultUploadDir
	}

	// Initialize store.
	st, err := store.New(dbPath)
	if err != nil {
		slog.Error("failed to initialize store", "error", err, "db_path", dbPath)
		os.Exit(1)
	}
	defer func() {
		if err := st.Close(); err != nil {
			slog.Error("failed to close store", "error", err)
		}
	}()

	slog.Info("database initialized", "db_path", dbPath)

	// Initialize OAuth2 token manager.
	tokenManager := oauth.NewTokenManager(st)

	// Initialize handlers.
	oauthHandler := oauth.NewHandler(tokenManager)
	companiesHandler := api.NewCompaniesHandler()
	accountItemsHandler := api.NewAccountItemsHandler()
	walletablesHandler := api.NewWalletablesHandler()
	dealsHandler := api.NewDealsHandler(st)
	journalsHandler := api.NewJournalsHandler(st)
	walletTxnsHandler := api.NewWalletTxnsHandler(st)
	receiptsHandler := api.NewReceiptsHandler(st, uploadDir)

	// Setup router.
	r := chi.NewRouter()

	// Middleware.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// OAuth2 endpoints (no authentication required).
	r.Post("/oauth/token", oauthHandler.HandleToken)
	r.Get("/oauth/authorize", oauthHandler.HandleLoginPage)
	r.Post("/oauth/authorize/login", oauthHandler.HandleLogin)
	r.Post("/oauth/authorize/2fa", oauthHandler.Handle2FA)
	r.Post("/oauth/authorize/confirm", oauthHandler.HandleAuthorizeConfirm)

	// API endpoints (authentication required).
	r.Route("/api/1", func(r chi.Router) {
		r.Use(api.AuthMiddleware(tokenManager))

		// Companies endpoint.
		r.Get("/companies", companiesHandler.List)

		// Account Items endpoint.
		r.Get("/account_items", accountItemsHandler.List)

		// Walletables endpoint.
		r.Get("/walletables", walletablesHandler.List)

		// Deals endpoints.
		r.Route("/deals", func(r chi.Router) {
			r.Get("/", dealsHandler.List)
			r.Post("/", dealsHandler.Create)
			r.Get("/{id}", dealsHandler.Get)
			r.Put("/{id}", dealsHandler.Update)
			r.Delete("/{id}", dealsHandler.Delete)
		})

		// Journals endpoints.
		r.Route("/journals", func(r chi.Router) {
			r.Get("/", journalsHandler.List)
			r.Post("/", journalsHandler.Create)
			r.Get("/{id}", journalsHandler.Get)
		})

		// Wallet Transactions endpoints.
		r.Route("/wallet_txns", func(r chi.Router) {
			r.Get("/", walletTxnsHandler.List)
			r.Post("/", walletTxnsHandler.Create)
			r.Get("/{id}", walletTxnsHandler.Get)
			r.Put("/{id}", walletTxnsHandler.Update)
			r.Delete("/{id}", walletTxnsHandler.Delete)
		})

		// Receipts endpoints.
		r.Route("/receipts", func(r chi.Router) {
			r.Get("/", receiptsHandler.List)
			r.Post("/", receiptsHandler.Create)
			r.Get("/{id}", receiptsHandler.Get)
			r.Delete("/{id}", receiptsHandler.Delete)
		})
	})

	// Health check endpoint.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Start server.
	addr := fmt.Sprintf(":%s", port)
	slog.Info("starting freee API emulator", "addr", addr, "port", port)

	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		slog.Info("shutting down server")
		if err := server.Close(); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
