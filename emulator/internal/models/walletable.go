package models

// Walletable represents a bank account, credit card, or wallet in freee.
type Walletable struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"`              // bank_account, credit_card, wallet
	BankID           *int64 `json:"bank_id,omitempty"` // For bank accounts
	LastBalance      int64  `json:"last_balance"`
	WalletableBalance int64 `json:"walletable_balance"`
}

// WalletablesResponse represents the response for GET /api/1/walletables
type WalletablesResponse struct {
	Walletables []Walletable `json:"walletables"`
}
