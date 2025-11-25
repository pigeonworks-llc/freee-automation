package models

import "time"

// WalletTxn represents a wallet transaction (明細) in freee accounting API.
type WalletTxn struct {
	ID                 int64     `json:"id"`
	CompanyID          int64     `json:"company_id"`
	Date               string    `json:"date"` // YYYY-MM-DD
	Amount             int64     `json:"amount"`
	Balance            *int64    `json:"balance,omitempty"`
	EntrySide          string    `json:"entry_side"` // income or expense
	WalletableType     string    `json:"walletable_type"` // bank_account or credit_card
	WalletableID       int64     `json:"walletable_id"`
	Description        string    `json:"description"`
	Status             string    `json:"status"` // settled, unbooked, passed
	DealID             *int64    `json:"deal_id,omitempty"`
	DealBalance        *int64    `json:"deal_balance,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// CreateWalletTxnRequest represents the request to create a wallet transaction.
type CreateWalletTxnRequest struct {
	CompanyID      int64  `json:"company_id"`
	Date           string `json:"date"`
	Amount         int64  `json:"amount"`
	EntrySide      string `json:"entry_side"`
	WalletableType string `json:"walletable_type"`
	WalletableID   int64  `json:"walletable_id"`
	Description    string `json:"description"`
}

// UpdateWalletTxnRequest represents the request to update a wallet transaction.
type UpdateWalletTxnRequest struct {
	Status      *string `json:"status,omitempty"`
	DealID      *int64  `json:"deal_id,omitempty"`
	Description *string `json:"description,omitempty"`
}
