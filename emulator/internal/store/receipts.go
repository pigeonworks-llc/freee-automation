package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pigeonworks-llc/freee-emulator/internal/models"
	bolt "go.etcd.io/bbolt"
)

// CreateReceipt creates a new receipt
func (s *Store) CreateReceipt(req *models.CreateReceiptRequest) (*models.Receipt, error) {
	id, err := s.NextID(BucketReceipts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	now := time.Now()
	receipt := &models.Receipt{
		ID:          id,
		CompanyID:   req.CompanyID,
		IssueDate:   req.IssueDate,
		Description: req.Description,
		Status:      "unconfirmed",
		FileName:    req.FileName,
		FilePath:    req.FilePath,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.Put(BucketReceipts, id, receipt); err != nil {
		return nil, fmt.Errorf("failed to save receipt: %w", err)
	}

	return receipt, nil
}

// GetReceipt retrieves a receipt by ID
func (s *Store) GetReceipt(id int64) (*models.Receipt, error) {
	var receipt models.Receipt
	if err := s.Get(BucketReceipts, id, &receipt); err != nil {
		return nil, err
	}
	return &receipt, nil
}

// ListReceipts retrieves all receipts, optionally filtered by company ID
func (s *Store) ListReceipts(companyID *int64) ([]*models.Receipt, error) {
	filter := func(data []byte) bool {
		if companyID == nil {
			return true
		}
		var receipt models.Receipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			return false
		}
		return receipt.CompanyID == *companyID
	}

	results, err := s.List(BucketReceipts, filter)
	if err != nil {
		return nil, err
	}

	receipts := make([]*models.Receipt, 0, len(results))
	for _, data := range results {
		var receipt models.Receipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			return nil, fmt.Errorf("failed to unmarshal receipt: %w", err)
		}
		receipts = append(receipts, &receipt)
	}

	return receipts, nil
}

// DeleteReceipt deletes a receipt by ID
func (s *Store) DeleteReceipt(id int64) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketReceipts))
		if b == nil {
			return fmt.Errorf("bucket %s not found", BucketReceipts)
		}

		key := itob(id)
		if b.Get(key) == nil {
			return ErrNotFound
		}

		return b.Delete(key)
	})
}
