// Package freee provides a client for freee API to fetch unregistered transactions.
package freee

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Client provides access to freee API.
type Client struct {
	apiURL       string
	accessToken  string
	companyID    string
	httpClient   *http.Client
	tokenManager *TokenManager
}

// WalletTransaction represents an unregistered wallet transaction from freee.
type WalletTransaction struct {
	ID             int         `json:"id"`
	CompanyID      int         `json:"company_id"`
	Date           string      `json:"date"`
	Amount         int         `json:"amount"`
	DueAmount      int         `json:"due_amount"`
	Balance        int         `json:"balance"`
	EntrySide      string      `json:"entry_side"` // "income" or "expense"
	WalletableType string      `json:"walletable_type"`
	WalletableID   int         `json:"walletable_id"`
	Description    string      `json:"description"`
	Status         interface{} `json:"status"` // 1 (unbooked) or 2 (booked) in real API, "unbooked"/"booked" in emulator
}

// Transaction represents a simplified transaction for matching.
type Transaction struct {
	ID          int
	Date        time.Time
	Amount      int
	Description string
	Vendor      string // Extracted vendor name
}

// NewClient creates a new freee API client with a static access token.
func NewClient(apiURL, accessToken, companyID string) *Client {
	return &Client{
		apiURL:      strings.TrimSuffix(apiURL, "/"),
		accessToken: accessToken,
		companyID:   companyID,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// NewClientWithTokenManager creates a new freee API client with token auto-refresh.
func NewClientWithTokenManager(apiURL string, tokenManager *TokenManager) (*Client, error) {
	token, err := tokenManager.GetValidToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	return &Client{
		apiURL:       strings.TrimSuffix(apiURL, "/"),
		accessToken:  token.AccessToken,
		companyID:    fmt.Sprintf("%d", token.CompanyID),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		tokenManager: tokenManager,
	}, nil
}

// ensureValidToken refreshes the token if necessary and updates the client.
func (c *Client) ensureValidToken() error {
	if c.tokenManager == nil {
		return nil // No token manager, use static token
	}

	token, err := c.tokenManager.GetValidToken()
	if err != nil {
		return err
	}

	c.accessToken = token.AccessToken
	c.companyID = fmt.Sprintf("%d", token.CompanyID)
	return nil
}

// FetchUnregisteredTransactions fetches unregistered wallet transactions.
// Uses pagination to fetch all transactions (status=1 means 消込待ち/unbooked).
func (c *Client) FetchUnregisteredTransactions() ([]WalletTransaction, error) {
	// Ensure token is valid before making requests
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	var allTxns []WalletTransaction
	offset := 0
	limit := 100 // Maximum allowed by freee API

	for {
		url := fmt.Sprintf("%s/api/1/wallet_txns?company_id=%s&status=1&limit=%d&offset=%d",
			c.apiURL, c.companyID, limit, offset)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("API error: %s", resp.Status)
		}

		var result struct {
			WalletTxns []WalletTransaction `json:"wallet_txns"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		allTxns = append(allTxns, result.WalletTxns...)

		// If we got fewer than limit, we've fetched all transactions
		if len(result.WalletTxns) < limit {
			break
		}
		offset += limit
	}

	return allTxns, nil
}

// FilterByVendor filters transactions by vendor patterns in description.
// Only includes truly unprocessed transactions (status=1).
func FilterByVendor(txns []WalletTransaction, vendors []string) []Transaction {
	var patterns []*regexp.Regexp
	for _, v := range vendors {
		pattern := regexp.MustCompile("(?i)" + regexp.QuoteMeta(v))
		patterns = append(patterns, pattern)
	}

	var result []Transaction
	for _, txn := range txns {
		// Only expense transactions
		if txn.EntrySide != "expense" {
			continue
		}
		// Skip if status is not 1 (消込待ち/unbooked)
		if !isUnbooked(txn.Status) {
			continue
		}

		// Check if description matches any vendor pattern
		vendor := ""
		for i, pattern := range patterns {
			if pattern.MatchString(txn.Description) {
				vendor = vendors[i]
				break
			}
		}

		if vendor != "" {
			date, _ := time.Parse("2006-01-02", txn.Date)
			result = append(result, Transaction{
				ID:          txn.ID,
				Date:        date,
				Amount:      abs(txn.Amount),
				Description: txn.Description,
				Vendor:      vendor,
			})
		}
	}

	return result
}

// FilterAll returns all expense transactions without vendor filtering.
// Only includes truly unprocessed transactions (status=1).
func FilterAll(txns []WalletTransaction) []Transaction {
	var result []Transaction
	for _, txn := range txns {
		if txn.EntrySide != "expense" {
			continue
		}
		// Skip if status is not 1 (消込待ち/unbooked)
		// API may return status=2 (消込済み) even when filtered
		if !isUnbooked(txn.Status) {
			continue
		}
		date, _ := time.Parse("2006-01-02", txn.Date)
		result = append(result, Transaction{
			ID:          txn.ID,
			Date:        date,
			Amount:      abs(txn.Amount),
			Description: txn.Description,
			Vendor:      extractVendor(txn.Description),
		})
	}
	return result
}

// isUnbooked checks if the status indicates an unbooked transaction.
// Status can be int (1) or string ("unbooked") depending on API/emulator.
func isUnbooked(status interface{}) bool {
	switch v := status.(type) {
	case float64:
		return int(v) == 1
	case int:
		return v == 1
	case string:
		return v == "unbooked" || v == "1"
	}
	return false
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func extractVendor(desc string) string {
	patterns := map[*regexp.Regexp]string{
		regexp.MustCompile(`(?i)amazon`):        "amazon",
		regexp.MustCompile(`(?i)rakuten|楽天`):    "rakuten",
		regexp.MustCompile(`(?i)yodobashi|ヨドバシ`): "yodobashi",
		regexp.MustCompile(`(?i)apple`):         "apple",
		regexp.MustCompile(`(?i)google`):        "google",
	}

	for pattern, name := range patterns {
		if pattern.MatchString(desc) {
			return name
		}
	}
	return "unknown"
}

// CreateDealRequest represents a request to create a deal in freee.
type CreateDealRequest struct {
	CompanyID int64                  `json:"company_id"`
	IssueDate string                 `json:"issue_date"`
	DueDate   string                 `json:"due_date,omitempty"`
	Type      string                 `json:"type"` // "expense" or "income"
	Details   []CreateDealDetail     `json:"details"`
	Payments  []CreateDealPayment    `json:"payments,omitempty"`
}

// CreateDealDetail represents a line item in a deal creation request.
type CreateDealDetail struct {
	AccountItemID int64  `json:"account_item_id"`
	TaxCode       int    `json:"tax_code"`
	Amount        int64  `json:"amount"`
	Description   string `json:"description,omitempty"`
}

// CreateDealPayment represents payment information in a deal creation request.
type CreateDealPayment struct {
	Date               string `json:"date"`
	FromWalletableType string `json:"from_walletable_type"`
	FromWalletableID   int64  `json:"from_walletable_id"`
	Amount             int64  `json:"amount"`
}

// CreateDealResponse represents the response from deal creation.
type CreateDealResponse struct {
	Deal struct {
		ID        int64  `json:"id"`
		IssueDate string `json:"issue_date"`
		Type      string `json:"type"`
		Amount    int64  `json:"amount"`
	} `json:"deal"`
}

// CreateDeal creates a new deal (expense/income transaction) in freee.
func (c *Client) CreateDeal(req *CreateDealRequest) (*CreateDealResponse, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	url := fmt.Sprintf("%s/api/1/deals", c.apiURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var result CreateDealResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CreateDealFromTransaction creates a deal from a wallet transaction with account mapping.
func (c *Client) CreateDealFromTransaction(txn WalletTransaction, accountItemID int64, taxCode int) (*CreateDealResponse, error) {
	companyID := int64(txn.CompanyID)

	req := &CreateDealRequest{
		CompanyID: companyID,
		IssueDate: txn.Date,
		Type:      "expense",
		Details: []CreateDealDetail{
			{
				AccountItemID: accountItemID,
				TaxCode:       taxCode,
				Amount:        int64(abs(txn.Amount)),
				Description:   txn.Description,
			},
		},
		Payments: []CreateDealPayment{
			{
				Date:               txn.Date,
				FromWalletableType: txn.WalletableType,
				FromWalletableID:   int64(txn.WalletableID),
				Amount:             int64(abs(txn.Amount)),
			},
		},
	}

	return c.CreateDeal(req)
}
