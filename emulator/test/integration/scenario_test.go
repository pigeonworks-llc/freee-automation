package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pigeonworks-llc/freee-emulator/internal/api"
	"github.com/pigeonworks-llc/freee-emulator/internal/models"
	"github.com/pigeonworks-llc/freee-emulator/internal/oauth"
	"github.com/pigeonworks-llc/freee-emulator/internal/store"
)

type testClient struct {
	server *httptest.Server
	token  string
}

func setupTestServer(t *testing.T) *testClient {
	t.Helper()

	// Create temporary database
	dbPath := fmt.Sprintf("/tmp/freee-test-%d.db", os.Getpid())
	t.Cleanup(func() {
		_ = os.Remove(dbPath)
	})

	// Initialize store
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

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

	// Create test server
	server := httptest.NewServer(r)
	t.Cleanup(server.Close)

	return &testClient{server: server}
}

func (c *testClient) getToken(t *testing.T) string {
	t.Helper()

	if c.token != "" {
		return c.token
	}

	resp, err := http.Post(
		c.server.URL+"/oauth/token",
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

func (c *testClient) request(t *testing.T, method, path string, body interface{}) *http.Response {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, c.server.URL+path, reqBody)
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

func TestOAuth2Flow(t *testing.T) {
	client := setupTestServer(t)

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

func TestDealLifecycle(t *testing.T) {
	client := setupTestServer(t)

	var dealID int64

	t.Run("Create deal", func(t *testing.T) {
		req := models.CreateDealRequest{
			CompanyID: 1,
			IssueDate: "2025-01-15",
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
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			Deal models.Deal `json:"deal"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		dealID = result.Deal.ID
		if dealID == 0 {
			t.Fatal("Expected non-zero deal ID")
		}

		if result.Deal.Amount != 110000 { // 100000 + 10% tax
			t.Errorf("Expected amount 110000, got %d", result.Deal.Amount)
		}
	})

	t.Run("Get deal", func(t *testing.T) {
		resp := client.request(t, "GET", fmt.Sprintf("/api/1/deals/%d", dealID), nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Deal models.Deal `json:"deal"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.Deal.ID != dealID {
			t.Errorf("Expected deal ID %d, got %d", dealID, result.Deal.ID)
		}
	})

	t.Run("Update deal", func(t *testing.T) {
		issueDate := "2025-01-20"
		req := models.UpdateDealRequest{
			IssueDate: &issueDate,
		}

		resp := client.request(t, "PUT", fmt.Sprintf("/api/1/deals/%d", dealID), req)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			Deal models.Deal `json:"deal"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.Deal.IssueDate != issueDate {
			t.Errorf("Expected issue_date %s, got %s", issueDate, result.Deal.IssueDate)
		}
	})

	t.Run("List deals", func(t *testing.T) {
		resp := client.request(t, "GET", "/api/1/deals?company_id=1", nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Deals []*models.Deal `json:"deals"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(result.Deals) == 0 {
			t.Error("Expected at least one deal")
		}
	})

	t.Run("Delete deal", func(t *testing.T) {
		resp := client.request(t, "DELETE", fmt.Sprintf("/api/1/deals/%d", dealID), nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 204, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("Verify deletion", func(t *testing.T) {
		resp := client.request(t, "GET", fmt.Sprintf("/api/1/deals/%d", dealID), nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})
}

func TestJournalCreation(t *testing.T) {
	client := setupTestServer(t)

	t.Run("Create balanced journal", func(t *testing.T) {
		req := models.CreateJournalRequest{
			CompanyID: 1,
			IssueDate: "2025-02-01",
			Details: []models.CreateJournalDetailRequest{
				{
					EntryType:     "debit",
					AccountItemID: 135,
					TaxCode:       0,
					Amount:        100000,
					Vat:           0,
				},
				{
					EntryType:     "credit",
					AccountItemID: 400,
					TaxCode:       1,
					Amount:        90909,
					Vat:           9091,
				},
			},
		}

		resp := client.request(t, "POST", "/api/1/journals", req)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			Journal models.Journal `json:"journal"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(result.Journal.Details) != 2 {
			t.Errorf("Expected 2 details, got %d", len(result.Journal.Details))
		}
	})
}

func TestCompleteScenario(t *testing.T) {
	client := setupTestServer(t)

	t.Run("Complete business scenario", func(t *testing.T) {
		// Step 1: Create income deal
		t.Log("Creating income deal...")
		incomeReq := models.CreateDealRequest{
			CompanyID: 1,
			IssueDate: "2025-03-01",
			Type:      "income",
			Details: []models.CreateDetailRequest{
				{
					AccountItemID: 400,
					TaxCode:       1,
					Amount:        200000,
				},
			},
		}

		resp := client.request(t, "POST", "/api/1/deals", incomeReq)
		resp.Body.Close()

		// Step 2: Create expense deal
		t.Log("Creating expense deal...")
		expenseReq := models.CreateDealRequest{
			CompanyID: 1,
			IssueDate: "2025-03-05",
			Type:      "expense",
			Details: []models.CreateDetailRequest{
				{
					AccountItemID: 801,
					TaxCode:       1,
					Amount:        80000,
				},
			},
		}

		resp = client.request(t, "POST", "/api/1/deals", expenseReq)
		resp.Body.Close()

		// Step 3: Create journal entry for payment
		t.Log("Creating payment journal...")
		journalReq := models.CreateJournalRequest{
			CompanyID: 1,
			IssueDate: "2025-03-10",
			Details: []models.CreateJournalDetailRequest{
				{
					EntryType:     "debit",
					AccountItemID: 135,
					TaxCode:       0,
					Amount:        200000,
					Vat:           0,
				},
				{
					EntryType:     "credit",
					AccountItemID: 141,
					TaxCode:       0,
					Amount:        200000,
					Vat:           0,
				},
			},
		}

		resp = client.request(t, "POST", "/api/1/journals", journalReq)
		resp.Body.Close()

		// Step 4: Verify all data
		t.Log("Verifying created data...")
		resp = client.request(t, "GET", "/api/1/deals?company_id=1", nil)
		defer resp.Body.Close()

		var dealsResult struct {
			Deals []*models.Deal `json:"deals"`
		}
		json.NewDecoder(resp.Body).Decode(&dealsResult)

		if len(dealsResult.Deals) != 2 {
			t.Errorf("Expected 2 deals, got %d", len(dealsResult.Deals))
		}

		resp = client.request(t, "GET", "/api/1/journals?company_id=1", nil)
		defer resp.Body.Close()

		var journalsResult struct {
			Journals []*models.Journal `json:"journals"`
		}
		json.NewDecoder(resp.Body).Decode(&journalsResult)

		if len(journalsResult.Journals) != 1 {
			t.Errorf("Expected 1 journal, got %d", len(journalsResult.Journals))
		}

		t.Log("Scenario completed successfully!")
	})
}
