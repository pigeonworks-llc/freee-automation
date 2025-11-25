package models

import "time"

// Journal represents a journal entry (仕訳) in freee accounting API.
type Journal struct {
	ID        int64       `json:"id"`
	CompanyID int64       `json:"company_id"`
	IssueDate string      `json:"issue_date"` // YYYY-MM-DD
	Details   []JournalDetail `json:"details"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// JournalDetail represents a single line in a journal entry.
type JournalDetail struct {
	ID              int64   `json:"id"`
	EntryType       string  `json:"entry_type"` // debit or credit
	AccountItemID   int64   `json:"account_item_id"`
	AccountItemName string  `json:"account_item_name"`
	TaxCode         int     `json:"tax_code"`
	PartnerID       *int64  `json:"partner_id,omitempty"`
	PartnerCode     *string `json:"partner_code,omitempty"`
	Amount          int64   `json:"amount"`
	Vat             int64   `json:"vat"`
	Description     *string `json:"description,omitempty"`
	ItemID          *int64  `json:"item_id,omitempty"`
	ItemName        *string `json:"item_name,omitempty"`
	SectionID       *int64  `json:"section_id,omitempty"`
	SectionName     *string `json:"section_name,omitempty"`
	TagIDs          []int64 `json:"tag_ids,omitempty"`
	TagNames        []string `json:"tag_names,omitempty"`
	Segment1TagID   *int64  `json:"segment_1_tag_id,omitempty"`
	Segment1TagName *string `json:"segment_1_tag_name,omitempty"`
	Segment2TagID   *int64  `json:"segment_2_tag_id,omitempty"`
	Segment2TagName *string `json:"segment_2_tag_name,omitempty"`
	Segment3TagID   *int64  `json:"segment_3_tag_id,omitempty"`
	Segment3TagName *string `json:"segment_3_tag_name,omitempty"`
}

// CreateJournalRequest represents the request to create a journal entry.
type CreateJournalRequest struct {
	CompanyID int64                   `json:"company_id"`
	IssueDate string                  `json:"issue_date"`
	Details   []CreateJournalDetailRequest `json:"details"`
}

// CreateJournalDetailRequest represents a detail in create journal request.
type CreateJournalDetailRequest struct {
	EntryType     string  `json:"entry_type"`
	AccountItemID int64   `json:"account_item_id"`
	TaxCode       int     `json:"tax_code"`
	PartnerID     *int64  `json:"partner_id,omitempty"`
	Amount        int64   `json:"amount"`
	Vat           int64   `json:"vat"`
	Description   *string `json:"description,omitempty"`
	ItemID        *int64  `json:"item_id,omitempty"`
	SectionID     *int64  `json:"section_id,omitempty"`
}
