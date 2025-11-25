// Package freee provides freee Accounting API client and types.
package freee

import "time"

// Deal represents a transaction in freee accounting API.
type Deal struct {
	ID          int64     `json:"id"`
	CompanyID   int64     `json:"company_id"`
	IssueDate   string    `json:"issue_date"` // YYYY-MM-DD
	DueDate     *string   `json:"due_date,omitempty"`
	Type        string    `json:"type"` // income or expense
	Details     []Detail  `json:"details"`
	Payments    []Payment `json:"payments,omitempty"`
	Amount      int64     `json:"amount"`
	DueAmount   *int64    `json:"due_amount,omitempty"`
	RefNumber   *string   `json:"ref_number,omitempty"`
	PartnerID   *int64    `json:"partner_id,omitempty"`
	PartnerCode *string   `json:"partner_code,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Detail represents a line item in a deal.
type Detail struct {
	ID              int64    `json:"id"`
	AccountItemID   int64    `json:"account_item_id"`
	AccountItemName string   `json:"account_item_name"`
	TaxCode         int      `json:"tax_code"`
	Amount          int64    `json:"amount"`
	Vat             int64    `json:"vat"`
	Description     *string  `json:"description,omitempty"`
	ItemID          *int64   `json:"item_id,omitempty"`
	ItemName        *string  `json:"item_name,omitempty"`
	SectionID       *int64   `json:"section_id,omitempty"`
	SectionName     *string  `json:"section_name,omitempty"`
	TagIDs          []int64  `json:"tag_ids,omitempty"`
	TagNames        []string `json:"tag_names,omitempty"`
	Segment1TagID   *int64   `json:"segment_1_tag_id,omitempty"`
	Segment1TagName *string  `json:"segment_1_tag_name,omitempty"`
	Segment2TagID   *int64   `json:"segment_2_tag_id,omitempty"`
	Segment2TagName *string  `json:"segment_2_tag_name,omitempty"`
	Segment3TagID   *int64   `json:"segment_3_tag_id,omitempty"`
	Segment3TagName *string  `json:"segment_3_tag_name,omitempty"`
}

// Payment represents payment information for a deal.
type Payment struct {
	ID                 int64  `json:"id"`
	Date               string `json:"date"` // YYYY-MM-DD
	Amount             int64  `json:"amount"`
	FromWalletableType string `json:"from_walletable_type"` // bank_account or credit_card
	FromWalletableID   int64  `json:"from_walletable_id"`
}

// Journal represents a journal entry in freee accounting API.
type Journal struct {
	ID        int64           `json:"id"`
	CompanyID int64           `json:"company_id"`
	IssueDate string          `json:"issue_date"` // YYYY-MM-DD
	Details   []JournalDetail `json:"details"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// JournalDetail represents a detail line in a journal entry.
type JournalDetail struct {
	ID              int64   `json:"id"`
	AccountItemID   int64   `json:"account_item_id"`
	AccountItemName string  `json:"account_item_name"`
	TaxCode         int     `json:"tax_code"`
	Amount          int64   `json:"amount"`
	Vat             int64   `json:"vat"`
	EntryType       string  `json:"entry_type"` // debit or credit
	Description     *string `json:"description,omitempty"`
}

// DealsResponse represents the response from /api/1/deals endpoint.
type DealsResponse struct {
	Deals []Deal `json:"deals"`
}

// JournalsResponse represents the response from /api/1/journals endpoint.
type JournalsResponse struct {
	Journals []Journal `json:"journals"`
}

// TokenResponse represents OAuth2 token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// ErrorResponse represents an error response from freee API.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}
