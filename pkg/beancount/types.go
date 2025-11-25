// Package beancount provides repository pattern for Beancount file operations.
package beancount

// Transaction represents a Beancount transaction.
type Transaction struct {
	Date      string     // YYYY-MM-DD
	Narration string     // Transaction description
	Payee     string     // Payee name (optional)
	Tags      []string   // Tags (e.g., ["invoice-123"])
	Links     []string   // Links (optional)
	Metadata  map[string]string // Metadata key-value pairs
	Postings  []Posting  // Transaction postings
}

// Posting represents a posting in a Beancount transaction.
type Posting struct {
	Account  string  // Account name (e.g., "Assets:Bank:Checking")
	Amount   float64 // Amount (positive for debit, negative for credit)
	Currency string  // Currency code (e.g., "JPY")
	Comment  string  // Posting comment (optional)
}
