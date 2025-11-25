package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pigeonworks-llc/freee-emulator/internal/api"
	"github.com/pigeonworks-llc/freee-emulator/internal/models"
	"github.com/pigeonworks-llc/freee-emulator/internal/oauth"
	"github.com/pigeonworks-llc/freee-emulator/internal/store"
	"github.com/pigeonworks-llc/go-portalloc/pkg/ports"
)

type parallelTestClient struct {
	baseURL string
	token   string
	closer  func()
}

func setupParallelTestServer(t *testing.T) *parallelTestClient {
	t.Helper()

	// Allocate a free port using go-portalloc
	allocator := ports.NewAllocator(nil)
	port, err := allocator.AllocateRange(1)
	if err != nil {
		t.Fatalf("Failed to allocate port: %v", err)
	}

	// Create temporary database with unique name
	dbPath := fmt.Sprintf("/tmp/freee-test-parallel-%d-%d.db", os.Getpid(), port)
	t.Cleanup(func() {
		_ = os.Remove(dbPath)
	})

	// Initialize store
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	// Initialize handlers
	tokenManager := oauth.NewTokenManager(st)
	oauthHandler := oauth.NewHandler(tokenManager)
	dealsHandler := api.NewDealsHandler(st)
	journalsHandler := api.NewJournalsHandler(st)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/oauth/token", oauthHandler.HandleToken)

	r.Route("/api/1", func(r chi.Router) {
		r.Use(api.AuthMiddleware(tokenManager))

		r.Route("/deals", func(r chi.Router) {
			r.Get("/", dealsHandler.List)
			r.Post("/", dealsHandler.Create)
			r.Get("/{id}", dealsHandler.Get)
			r.Put("/{id}", dealsHandler.Update)
			r.Delete("/{id}", dealsHandler.Delete)
		})

		r.Route("/journals", func(r chi.Router) {
			r.Get("/", journalsHandler.List)
			r.Post("/", journalsHandler.Create)
			r.Get("/{id}", journalsHandler.Get)
		})
	})

	// Start server in background
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	go func() {
		_ = server.ListenAndServe()
	}()

	// Wait for server to be ready
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(baseURL + "/oauth/token")
		if err == nil {
			resp.Body.Close()
			break
		}
		if i == maxRetries-1 {
			st.Close()
			t.Fatalf("Server did not start: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	closer := func() {
		_ = server.Close()
		_ = st.Close()
	}
	t.Cleanup(closer)

	return &parallelTestClient{
		baseURL: baseURL,
		closer:  closer,
	}
}

func (c *parallelTestClient) getToken(t *testing.T) string {
	t.Helper()

	if c.token != "" {
		return c.token
	}

	resp, err := http.Post(
		c.baseURL+"/oauth/token",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString("grant_type=client_credentials"),
	)
	if err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("Failed to decode token response: %v", err)
	}

	c.token = tokenResp.AccessToken
	return c.token
}

func (c *parallelTestClient) request(t *testing.T, method, path string, body interface{}) *http.Response {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.getToken(t))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	return resp
}

func TestParallelOAuth2(t *testing.T) {
	t.Parallel()

	client := setupParallelTestServer(t)

	t.Run("Get access token", func(t *testing.T) {
		token := client.getToken(t)
		if token == "" {
			t.Fatal("Expected non-empty token")
		}
	})

	t.Run("Use token for API call", func(t *testing.T) {
		resp := client.request(t, "GET", "/api/1/deals?company_id=1", nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestParallelDealOperations(t *testing.T) {
	t.Parallel()

	client := setupParallelTestServer(t)

	t.Run("Create and retrieve deal", func(t *testing.T) {
		// Create deal
		req := models.CreateDealRequest{
			CompanyID: 1,
			IssueDate: "2025-03-01",
			Type:      "income",
			Details: []models.CreateDetailRequest{
				{
					AccountItemID: 400,
					TaxCode:       1,
					Amount:        100000,
				},
			},
		}

		resp := client.request(t, "POST", "/api/1/deals", req)
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			Deal models.Deal `json:"deal"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		dealID := result.Deal.ID

		// Retrieve deal
		resp = client.request(t, "GET", fmt.Sprintf("/api/1/deals/%d", dealID), nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestParallelJournalOperations(t *testing.T) {
	t.Parallel()

	client := setupParallelTestServer(t)

	t.Run("Create journal entry", func(t *testing.T) {
		req := models.CreateJournalRequest{
			CompanyID: 1,
			IssueDate: "2025-03-01",
			Details: []models.CreateJournalDetailRequest{
				{
					EntryType:     "debit",
					AccountItemID: 135,
					TaxCode:       0,
					Amount:        50000,
					Vat:           0,
				},
				{
					EntryType:     "credit",
					AccountItemID: 400,
					TaxCode:       1,
					Amount:        45454,
					Vat:           4546,
				},
			},
		}

		resp := client.request(t, "POST", "/api/1/journals", req)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(body))
		}
	})
}

func TestParallelMultipleDeals(t *testing.T) {
	t.Parallel()

	client := setupParallelTestServer(t)

	// Create multiple deals concurrently
	t.Run("Create multiple deals", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			i := i
			t.Run(fmt.Sprintf("Deal_%d", i), func(t *testing.T) {
				t.Parallel()

				req := models.CreateDealRequest{
					CompanyID: 1,
					IssueDate: "2025-03-01",
					Type:      "income",
					Details: []models.CreateDetailRequest{
						{
							AccountItemID: int64(400 + i),
							TaxCode:       1,
							Amount:        int64(10000 * (i + 1)),
						},
					},
				}

				resp := client.request(t, "POST", "/api/1/deals", req)
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusCreated {
					body, _ := io.ReadAll(resp.Body)
					t.Errorf("Deal %d: Expected status 201, got %d: %s", i, resp.StatusCode, string(body))
				}
			})
		}
	})

	// Verify all deals were created
	t.Run("List all deals", func(t *testing.T) {
		resp := client.request(t, "GET", "/api/1/deals?company_id=1", nil)
		defer resp.Body.Close()

		var result struct {
			Deals []*models.Deal `json:"deals"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		if len(result.Deals) != 5 {
			t.Errorf("Expected 5 deals, got %d", len(result.Deals))
		}
	})
}

func TestParallelStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Parallel()

	client := setupParallelTestServer(t)

	// Run 20 concurrent operations
	t.Run("Stress test with 20 operations", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			i := i
			t.Run(fmt.Sprintf("Operation_%d", i), func(t *testing.T) {
				t.Parallel()

				if i%2 == 0 {
					// Create deal
					req := models.CreateDealRequest{
						CompanyID: 1,
						IssueDate: "2025-03-01",
						Type:      "income",
						Details: []models.CreateDetailRequest{
							{
								AccountItemID: 400,
								TaxCode:       1,
								Amount:        int64(1000 * (i + 1)),
							},
						},
					}

					resp := client.request(t, "POST", "/api/1/deals", req)
					resp.Body.Close()
				} else {
					// Create journal
					req := models.CreateJournalRequest{
						CompanyID: 1,
						IssueDate: "2025-03-01",
						Details: []models.CreateJournalDetailRequest{
							{
								EntryType:     "debit",
								AccountItemID: 135,
								TaxCode:       0,
								Amount:        int64(1000 * (i + 1)),
								Vat:           0,
							},
							{
								EntryType:     "credit",
								AccountItemID: 400,
								TaxCode:       0,
								Amount:        int64(1000 * (i + 1)),
								Vat:           0,
							},
						},
					}

					resp := client.request(t, "POST", "/api/1/journals", req)
					resp.Body.Close()
				}
			})
		}
	})
}
