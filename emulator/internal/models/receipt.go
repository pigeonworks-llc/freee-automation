package models

import "time"

// Receipt represents a receipt in the freee accounting system
type Receipt struct {
	ID          int64     `json:"id"`
	CompanyID   int64     `json:"company_id"`
	IssueDate   string    `json:"issue_date"` // YYYY-MM-DD
	Description string    `json:"description"`
	Status      string    `json:"status"` // "unconfirmed", "confirmed"
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateReceiptRequest represents the request to create a receipt
// Note: File upload is handled separately via multipart/form-data
type CreateReceiptRequest struct {
	CompanyID   int64  `json:"company_id"`
	IssueDate   string `json:"issue_date"`
	Description string `json:"description"`
	FileName    string `json:"file_name"`
	FilePath    string `json:"file_path"`
}
