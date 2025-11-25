package freee

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ClientConfig represents the configuration for freee API client.
type ClientConfig struct {
	APIURL       string
	ClientID     string
	ClientSecret string
	AccessToken  string
	CompanyID    int64
	Timeout      time.Duration // Default: 30 seconds
}

// Client is a freee Accounting API client.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	accessToken  string
	clientID     string
	clientSecret string
	companyID    int64
}

// NewClient creates a new freee API client.
func NewClient(config ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL:      config.APIURL,
		accessToken:  config.AccessToken,
		clientID:     config.ClientID,
		clientSecret: config.ClientSecret,
		companyID:    config.CompanyID,
	}
}

// SetAccessToken sets the access token for API requests.
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// GetAccessToken obtains an OAuth2 access token.
func (c *Client) GetAccessToken() (string, error) {
	tokenURL := fmt.Sprintf("%s/oauth/token", c.baseURL)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.parseError(resp)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	return c.accessToken, nil
}

// ListDeals lists deals with optional parameters.
func (c *Client) ListDeals(params map[string]string) ([]Deal, error) {
	endpoint := fmt.Sprintf("%s/api/1/deals", c.baseURL)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("company_id", fmt.Sprintf("%d", c.companyID))
	for k, v := range params {
		queryParams.Set(k, v)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", endpoint, queryParams.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var dealsResp DealsResponse
	if err := json.NewDecoder(resp.Body).Decode(&dealsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return dealsResp.Deals, nil
}

// FetchAllDeals fetches all deals in a date range with pagination.
func (c *Client) FetchAllDeals(dateFrom, dateTo string) ([]Deal, error) {
	var allDeals []Deal
	offset := 0
	limit := 100

	for {
		params := map[string]string{
			"issue_date_from": dateFrom,
			"issue_date_to":   dateTo,
			"limit":           fmt.Sprintf("%d", limit),
			"offset":          fmt.Sprintf("%d", offset),
		}

		deals, err := c.ListDeals(params)
		if err != nil {
			return nil, fmt.Errorf("failed to list deals (offset=%d): %w", offset, err)
		}

		if len(deals) == 0 {
			break
		}

		allDeals = append(allDeals, deals...)

		if len(deals) < limit {
			break
		}

		offset += limit
	}

	return allDeals, nil
}

// ListJournals lists journals with optional parameters.
func (c *Client) ListJournals(params map[string]string) ([]Journal, error) {
	endpoint := fmt.Sprintf("%s/api/1/journals", c.baseURL)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("company_id", fmt.Sprintf("%d", c.companyID))
	for k, v := range params {
		queryParams.Set(k, v)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", endpoint, queryParams.Encode()), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var journalsResp JournalsResponse
	if err := json.NewDecoder(resp.Body).Decode(&journalsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return journalsResp.Journals, nil
}

// FetchAllJournals fetches all journals in a date range with pagination.
func (c *Client) FetchAllJournals(dateFrom, dateTo string) ([]Journal, error) {
	var allJournals []Journal
	offset := 0
	limit := 100

	for {
		params := map[string]string{
			"issue_date_from": dateFrom,
			"issue_date_to":   dateTo,
			"limit":           fmt.Sprintf("%d", limit),
			"offset":          fmt.Sprintf("%d", offset),
		}

		journals, err := c.ListJournals(params)
		if err != nil {
			return nil, fmt.Errorf("failed to list journals (offset=%d): %w", offset, err)
		}

		if len(journals) == 0 {
			break
		}

		allJournals = append(allJournals, journals...)

		if len(journals) < limit {
			break
		}

		offset += limit
	}

	return allJournals, nil
}

// parseError parses an error response from freee API.
func (c *Client) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("freee API error (status %d): failed to read error response", resp.StatusCode)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("freee API error (status %d): %s", resp.StatusCode, string(body))
	}

	if errResp.ErrorDescription != "" {
		return fmt.Errorf("freee API error: %s - %s", errResp.Error, errResp.ErrorDescription)
	}

	return fmt.Errorf("freee API error: %s", errResp.Error)
}
