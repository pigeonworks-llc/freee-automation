# freee API Automation Guide

Guide for automating journal entry creation from credit card statements using freee API.

## Overview

This guide explains how to automate the process of creating journal entries (vouchers) from credit card statement data using the official freee accounting API. The implementation is designed for local development environments using Go.

### Key Topics

- OAuth2.0 authentication for freee API
- API endpoints for retrieving credit card statements
- API endpoints for creating journal entries
- Mapping credit card transactions to appropriate accounts
- Best practices for automation

---

## 1. OAuth2.0 Authentication

### 1.1 App Registration

1. Register your application at [freee Developers](https://developer.freee.co.jp/)
2. Obtain Client ID and Client Secret
3. Enable accounting API access permissions (transactions read/write, voucher creation)

### 1.2 Redirect URL Configuration

For local development, freee supports two approaches:

**Method 1: Out-of-Band (OOB) Redirect**

Set redirect URI to `urn:ietf:wg:oauth:2.0:oob`. After authorization, the code displays directly in the browser for manual copy/paste.

**Method 2: Local HTTP Server**

Start a local HTTP server and use `http://localhost:<port>/callback` as the redirect URL for automated code capture.

### 1.3 OAuth2 Endpoints

| Endpoint | URL |
|----------|-----|
| Authorization | `https://accounts.secure.freee.co.jp/public_api/authorize` |
| Token | `https://accounts.secure.freee.co.jp/public_api/token` |

### 1.4 Go Code Example

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"

    "golang.org/x/oauth2"
)

var (
    clientID     = "YOUR_CLIENT_ID"
    clientSecret = "YOUR_CLIENT_SECRET"
    redirectURL  = "urn:ietf:wg:oauth:2.0:oob"
)

func main() {
    conf := &oauth2.Config{
        ClientID:     clientID,
        ClientSecret: clientSecret,
        RedirectURL:  redirectURL,
        Scopes:       []string{}, // freee manages scopes via app permissions
        Endpoint: oauth2.Endpoint{
            AuthURL:  "https://accounts.secure.freee.co.jp/public_api/authorize",
            TokenURL: "https://accounts.secure.freee.co.jp/public_api/token",
        },
    }

    // 1. Generate and display authorization URL
    state := "example-state" // Random string for CSRF protection
    authURL := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
    fmt.Println("Open this URL in your browser:")
    fmt.Println(authURL)

    // 2. Get authorization code from user
    fmt.Print("Enter the authorization code: ")
    scanner := bufio.NewScanner(os.Stdin)
    scanner.Scan()
    code := scanner.Text()

    // 3. Exchange code for access token
    token, err := conf.Exchange(context.Background(), code)
    if err != nil {
        log.Fatalf("Token exchange error: %v\n", err)
    }
    fmt.Printf("Access Token: %s\nRefresh Token: %s\n",
               token.AccessToken, token.RefreshToken)
}
```

### 1.5 Token Refresh

- Access tokens expire in approximately 6 hours
- Use refresh tokens for automatic renewal
- The `oauth2.TokenSource` handles automatic refresh

---

## 2. API Endpoints

### 2.1 Companies (Organizations)

Retrieve the list of accessible companies/organizations.

```
GET /api/1/companies
```

**Response:**
```json
{
  "companies": [
    {
      "id": 1,
      "display_name": "Pigeonworks LLC",
      "name": "Pigeonworks LLC",
      "name_kana": "Pigeonworks"
    }
  ]
}
```

### 2.2 Account Items

Retrieve account items (chart of accounts) for a company.

```
GET /api/1/account_items?company_id={company_id}
```

**Response:**
```json
{
  "account_items": [
    {
      "id": 101,
      "name": "Cash",
      "account_category": "asset",
      "default_tax_code": 0
    },
    {
      "id": 502,
      "name": "Books & Publications",
      "account_category": "expense",
      "default_tax_code": 136
    }
  ]
}
```

### 2.3 Walletables (Accounts/Cards)

Retrieve registered bank accounts, credit cards, and other walletables.

```
GET /api/1/walletables?company_id={company_id}
```

**Walletable Types:**
- `bank_account` - Bank accounts
- `credit_card` - Credit cards
- `wallet` - E-money/digital wallets

### 2.4 Wallet Transactions (Card Statements)

Retrieve credit card or bank statement transactions.

```
GET /api/1/wallet_txns?company_id={company_id}&walletable_type={type}&walletable_id={id}
```

**Parameters:**
- `company_id` (required) - Company ID
- `walletable_type` - Filter by type (e.g., `credit_card`)
- `walletable_id` - Filter by specific account ID (requires `walletable_type`)

**Response Fields:**

| Field | Description |
|-------|-------------|
| `id` | Transaction ID |
| `date` | Transaction date (yyyy-mm-dd) |
| `amount` | Amount (positive for expenses) |
| `entry_side` | `expense` or `income` |
| `walletable_type` | Account type |
| `walletable_id` | Account ID |
| `description` | Transaction description/merchant |
| `status` | `unbooked` (pending), `settled` (booked), `ignored`, `matching` |

### 2.5 Deals (Transactions)

Create income/expense transactions linked to wallet transactions.

```
POST /api/1/deals
```

**Request Body:**
```json
{
  "company_id": 1,
  "issue_date": "2025-11-24",
  "type": "expense",
  "due_date": "2025-12-24",
  "details": [
    {
      "account_item_id": 502,
      "tax_code": 136,
      "amount": 1650,
      "description": "Amazon - Technical Book"
    }
  ],
  "payments": [
    {
      "date": "2025-11-24",
      "from_walletable_type": "credit_card",
      "from_walletable_id": 123,
      "amount": 1650
    }
  ]
}
```

**Key Points:**
- `type`: `income` or `expense`
- `details`: Array of expense/income line items
- `payments`: Array of payment methods (links to wallet transactions)

### 2.6 Manual Journals (Vouchers)

Create journal entries directly without linking to wallet transactions.

```
POST /api/1/manual_journals
```

**Request Body:**
```json
{
  "company_id": 1,
  "issue_date": "2025-11-24",
  "adjustment": false,
  "details": [
    {
      "account_item_id": 502,
      "tax_code": 136,
      "amount": 1500,
      "entry_side": "debit",
      "description": "Office supplies"
    },
    {
      "account_item_id": 203,
      "tax_code": 0,
      "amount": 1500,
      "entry_side": "credit",
      "description": "Credit card payment"
    }
  ]
}
```

**Fields:**
- `adjustment`: Set `true` for closing entries
- `entry_side`: `debit` or `credit`

---

## 3. Automation Workflow

### 3.1 Process Flow

```
1. Get Companies       GET /api/1/companies
         |
         v
2. Get Account Items   GET /api/1/account_items?company_id=X
         |
         v
3. Get Walletables     GET /api/1/walletables?company_id=X
         |
         v
4. Get Wallet Txns     GET /api/1/wallet_txns?company_id=X&status=unbooked
         |
         v
5. Map to Accounts     (rule-based mapping)
         |
         v
6. Create Deals        POST /api/1/deals (auto-links wallet_txns)
```

### 3.2 Account Mapping Example

```go
// Map description to account item ID and tax code
func mapDescriptionToAccount(description string) (accountID int64, taxCode int64) {
    rules := []struct {
        keywords  []string
        accountID int64
        taxCode   int64
    }{
        {[]string{"Electric", "Gas", "Water"}, 512, 136},  // Utilities
        {[]string{"Taxi", "Train", "Transit"}, 507, 136},  // Transportation
        {[]string{"Amazon", "Book"}, 502, 136},            // Books & Publications
        {[]string{"AWS", "Cloud"}, 505, 136},              // Communication
        {[]string{"Hotel", "Lodging"}, 507, 136},          // Travel
    }

    descLower := strings.ToLower(description)
    for _, rule := range rules {
        for _, keyword := range rule.keywords {
            if strings.Contains(descLower, strings.ToLower(keyword)) {
                return rule.accountID, rule.taxCode
            }
        }
    }

    // Default: Miscellaneous expense
    return 509, 136
}
```

### 3.3 Creating Deals with Payment Linking

```go
type DealRequest struct {
    CompanyID int64            `json:"company_id"`
    IssueDate string           `json:"issue_date"`
    Type      string           `json:"type"`
    Details   []DealDetail     `json:"details"`
    Payments  []DealPayment    `json:"payments,omitempty"`
}

type DealDetail struct {
    AccountItemID int64  `json:"account_item_id"`
    TaxCode       int64  `json:"tax_code"`
    Amount        int64  `json:"amount"`
    Description   string `json:"description"`
}

type DealPayment struct {
    Date               string `json:"date"`
    FromWalletableType string `json:"from_walletable_type"`
    FromWalletableID   int64  `json:"from_walletable_id"`
    Amount             int64  `json:"amount"`
}

func createDealFromWalletTxn(txn WalletTxn, accountID, taxCode int64) DealRequest {
    return DealRequest{
        CompanyID: txn.CompanyID,
        IssueDate: txn.Date,
        Type:      "expense",
        Details: []DealDetail{
            {
                AccountItemID: accountID,
                TaxCode:       taxCode,
                Amount:        txn.Amount,
                Description:   txn.Description,
            },
        },
        Payments: []DealPayment{
            {
                Date:               txn.Date,
                FromWalletableType: txn.WalletableType,
                FromWalletableID:   txn.WalletableID,
                Amount:             txn.Amount,
            },
        },
    }
}
```

---

## 4. Best Practices

### 4.1 Rate Limiting

- **Limit:** 300 requests per minute per company
- **Recommendation:** Add delays between API calls when processing many transactions
- **Handling:** Check for HTTP 429/403 errors and implement exponential backoff

### 4.2 Error Handling

| Status Code | Meaning | Action |
|-------------|---------|--------|
| 401 | Unauthorized | Refresh access token |
| 403 | Forbidden / Rate limited | Check permissions or wait |
| 422 | Validation error | Check request parameters |

**Response header `X-Freee-Request-ID`** can be used for debugging and support inquiries.

### 4.3 Handling Unclassified Transactions

For transactions that cannot be automatically mapped:

1. **Skip and log** - Don't create deals, report for manual review
2. **Use placeholder account** - Map to "Miscellaneous" and flag for review
3. **Queue for review** - Store in pending state for human decision

### 4.4 Auto-Registration Rules

freee supports auto-registration rules that automatically classify transactions based on patterns. When adding transactions via API, these rules may apply automatically.

### 4.5 Transaction Status Management

When creating deals via the API with payment information that references wallet transactions, the wallet transaction status automatically changes from `unbooked` to `settled`.

---

## 5. Common Tax Codes

| Code | Description |
|------|-------------|
| 0 | Non-taxable |
| 21 | Taxable sales (10%) |
| 136 | Taxable purchase (10%) |
| 2 | Taxable sales (8% - reduced rate) |
| 103 | Taxable purchase (8% - reduced rate) |

For the full list, use `GET /api/1/taxes?company_id={company_id}`.

---

## 6. Testing with freee-emulator

For local development and testing, use the freee-emulator:

```bash
# Start the emulator
PORT=8080 ./bin/freee-emulator

# Get OAuth token
curl -X POST http://localhost:8080/oauth/token \
  -d "grant_type=client_credentials&client_id=test&client_secret=test"

# Test API endpoints
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/1/companies

curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/1/account_items?company_id=1"
```

---

## 7. References

- [freee Developers](https://developer.freee.co.jp/) - Official documentation
- [freee API Reference](https://developer.freee.co.jp/reference/) - API specifications
- [freee SDK (Go)](https://github.com/freee/freee-accounting-sdk-go) - Official Go SDK
