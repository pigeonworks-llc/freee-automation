package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pigeonworks-llc/freee-emulator/internal/models"
)

// CreateWalletTxn creates a new wallet transaction in the database.
func (s *Store) CreateWalletTxn(req *models.CreateWalletTxnRequest) (*models.WalletTxn, error) {
	id, err := s.NextID(BucketWalletTxns)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	now := time.Now()
	walletTxn := &models.WalletTxn{
		ID:             id,
		CompanyID:      req.CompanyID,
		Date:           req.Date,
		Amount:         req.Amount,
		EntrySide:      req.EntrySide,
		WalletableType: req.WalletableType,
		WalletableID:   req.WalletableID,
		Description:    req.Description,
		Status:         "unbooked", // Default status is unbooked (未仕訳)
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.Put(BucketWalletTxns, id, walletTxn); err != nil {
		return nil, fmt.Errorf("failed to save wallet transaction: %w", err)
	}

	return walletTxn, nil
}

// GetWalletTxn retrieves a wallet transaction by ID.
func (s *Store) GetWalletTxn(id int64) (*models.WalletTxn, error) {
	var walletTxn models.WalletTxn
	if err := s.Get(BucketWalletTxns, id, &walletTxn); err != nil {
		return nil, err
	}
	return &walletTxn, nil
}

// ListWalletTxns retrieves all wallet transactions, optionally filtered by company ID and status.
func (s *Store) ListWalletTxns(companyID *int64, status *string) ([]*models.WalletTxn, error) {
	filter := func(data []byte) bool {
		var txn models.WalletTxn
		if err := json.Unmarshal(data, &txn); err != nil {
			return false
		}

		if companyID != nil && txn.CompanyID != *companyID {
			return false
		}

		if status != nil && txn.Status != *status {
			return false
		}

		return true
	}

	results, err := s.List(BucketWalletTxns, filter)
	if err != nil {
		return nil, err
	}

	walletTxns := make([]*models.WalletTxn, 0, len(results))
	for _, data := range results {
		var txn models.WalletTxn
		if err := json.Unmarshal(data, &txn); err != nil {
			return nil, fmt.Errorf("failed to unmarshal wallet transaction: %w", err)
		}
		walletTxns = append(walletTxns, &txn)
	}

	return walletTxns, nil
}

// UpdateWalletTxn updates an existing wallet transaction.
func (s *Store) UpdateWalletTxn(id int64, req *models.UpdateWalletTxnRequest) (*models.WalletTxn, error) {
	txn, err := s.GetWalletTxn(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Status != nil {
		txn.Status = *req.Status
	}
	if req.DealID != nil {
		txn.DealID = req.DealID
	}
	if req.Description != nil {
		txn.Description = *req.Description
	}

	txn.UpdatedAt = time.Now()

	if err := s.Put(BucketWalletTxns, id, txn); err != nil {
		return nil, fmt.Errorf("failed to update wallet transaction: %w", err)
	}

	return txn, nil
}

// DeleteWalletTxn deletes a wallet transaction by ID.
func (s *Store) DeleteWalletTxn(id int64) error {
	return s.Delete(BucketWalletTxns, id)
}

// FindMatchingUnbookedWalletTxn finds an unbooked wallet transaction matching the given criteria.
// This is used to link deals with wallet transactions.
func (s *Store) FindMatchingUnbookedWalletTxn(companyID int64, date string, amount int64) (*models.WalletTxn, error) {
	unbookedStatus := "unbooked"
	txns, err := s.ListWalletTxns(&companyID, &unbookedStatus)
	if err != nil {
		return nil, err
	}

	// Find a matching transaction by date and amount
	for _, txn := range txns {
		// Match by date and absolute amount
		if txn.Date == date && abs(txn.Amount) == abs(amount) {
			return txn, nil
		}
	}

	return nil, nil // No match found (not an error)
}

// FindWalletTxnByPayment finds an unbooked wallet transaction matching payment details.
// This is used when explicit payments are provided in deal creation.
func (s *Store) FindWalletTxnByPayment(companyID int64, walletableType string, walletableID int64, date string, amount int64) (*models.WalletTxn, error) {
	unbookedStatus := "unbooked"
	txns, err := s.ListWalletTxns(&companyID, &unbookedStatus)
	if err != nil {
		return nil, err
	}

	// Find a matching transaction by walletable, date, and amount
	for _, txn := range txns {
		if txn.WalletableType == walletableType &&
			txn.WalletableID == walletableID &&
			txn.Date == date &&
			abs(txn.Amount) == abs(amount) {
			return txn, nil
		}
	}

	return nil, nil // No match found (not an error)
}

// LinkWalletTxnToDeal links a wallet transaction to a deal by updating its status and deal_id.
func (s *Store) LinkWalletTxnToDeal(walletTxnID int64, dealID int64) error {
	status := "settled"
	_, err := s.UpdateWalletTxn(walletTxnID, &models.UpdateWalletTxnRequest{
		Status: &status,
		DealID: &dealID,
	})
	return err
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
