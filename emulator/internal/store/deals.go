package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pigeonworks-llc/freee-emulator/internal/models"
)

// CreateDeal creates a new deal in the database.
func (s *Store) CreateDeal(req *models.CreateDealRequest) (*models.Deal, error) {
	id, err := s.NextID(BucketDeals)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	now := time.Now()
	deal := &models.Deal{
		ID:        id,
		CompanyID: req.CompanyID,
		IssueDate: req.IssueDate,
		DueDate:   req.DueDate,
		Type:      req.Type,
		RefNumber: req.RefNumber,
		PartnerID: req.PartnerID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Convert CreateDetailRequest to Detail.
	var totalAmount int64
	details := make([]models.Detail, len(req.Details))
	for i, d := range req.Details {
		detailID, err := s.NextID(BucketDeals)
		if err != nil {
			return nil, fmt.Errorf("failed to generate detail ID: %w", err)
		}

		// Calculate VAT (simplified: 10% if tax_code != 0).
		vat := int64(0)
		if d.TaxCode != 0 {
			vat = d.Amount / 10
		}

		details[i] = models.Detail{
			ID:            detailID,
			AccountItemID: d.AccountItemID,
			AccountItemName: fmt.Sprintf("Account Item %d", d.AccountItemID),
			TaxCode:       d.TaxCode,
			Amount:        d.Amount,
			Vat:           vat,
			Description:   d.Description,
			ItemID:        d.ItemID,
			SectionID:     d.SectionID,
		}
		totalAmount += d.Amount + vat
	}

	deal.Details = details
	deal.Amount = totalAmount

	// Process payments and create Payment objects.
	if len(req.Payments) > 0 {
		payments := make([]models.Payment, len(req.Payments))
		for i, p := range req.Payments {
			paymentID, err := s.NextID(BucketDeals)
			if err != nil {
				return nil, fmt.Errorf("failed to generate payment ID: %w", err)
			}
			payments[i] = models.Payment{
				ID:                 paymentID,
				Date:               p.Date,
				Amount:             p.Amount,
				FromWalletableType: p.FromWalletableType,
				FromWalletableID:   p.FromWalletableID,
			}
		}
		deal.Payments = payments
	}

	if err := s.Put(BucketDeals, id, deal); err != nil {
		return nil, fmt.Errorf("failed to save deal: %w", err)
	}

	// Link wallet transactions based on payments or auto-match.
	if len(req.Payments) > 0 {
		// Explicit payments: link matching wallet_txns by walletable_id, date, and amount.
		for _, p := range req.Payments {
			txn, err := s.FindWalletTxnByPayment(req.CompanyID, p.FromWalletableType, p.FromWalletableID, p.Date, p.Amount)
			if err != nil {
				// Log error but don't fail deal creation
			} else if txn != nil {
				if err := s.LinkWalletTxnToDeal(txn.ID, id); err != nil {
					// Log error but don't fail deal creation
				}
			}
		}
	} else {
		// Auto-link matching unbooked wallet transactions.
		// This implements the "自動で経理" (automatic bookkeeping) feature.
		matchingTxn, err := s.FindMatchingUnbookedWalletTxn(req.CompanyID, req.IssueDate, totalAmount)
		if err != nil {
			// Log error but don't fail deal creation
		} else if matchingTxn != nil {
			// Found a matching wallet transaction, link it to this deal
			if err := s.LinkWalletTxnToDeal(matchingTxn.ID, id); err != nil {
				// Log error but don't fail deal creation
			}
		}
	}

	return deal, nil
}

// GetDeal retrieves a deal by ID.
func (s *Store) GetDeal(id int64) (*models.Deal, error) {
	var deal models.Deal
	if err := s.Get(BucketDeals, id, &deal); err != nil {
		return nil, err
	}
	return &deal, nil
}

// ListDeals retrieves all deals, optionally filtered by company ID.
func (s *Store) ListDeals(companyID *int64) ([]*models.Deal, error) {
	filter := func(data []byte) bool {
		if companyID == nil {
			return true
		}

		var deal models.Deal
		if err := json.Unmarshal(data, &deal); err != nil {
			return false
		}
		return deal.CompanyID == *companyID
	}

	results, err := s.List(BucketDeals, filter)
	if err != nil {
		return nil, err
	}

	deals := make([]*models.Deal, 0, len(results))
	for _, data := range results {
		var deal models.Deal
		if err := json.Unmarshal(data, &deal); err != nil {
			return nil, fmt.Errorf("failed to unmarshal deal: %w", err)
		}
		deals = append(deals, &deal)
	}

	return deals, nil
}

// UpdateDeal updates an existing deal.
func (s *Store) UpdateDeal(id int64, req *models.UpdateDealRequest) (*models.Deal, error) {
	deal, err := s.GetDeal(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided.
	if req.IssueDate != nil {
		deal.IssueDate = *req.IssueDate
	}
	if req.DueDate != nil {
		deal.DueDate = req.DueDate
	}
	if req.RefNumber != nil {
		deal.RefNumber = req.RefNumber
	}
	if req.PartnerID != nil {
		deal.PartnerID = req.PartnerID
	}

	// Update details if provided.
	if len(req.Details) > 0 {
		var totalAmount int64
		details := make([]models.Detail, len(req.Details))
		for i, d := range req.Details {
			detailID, err := s.NextID(BucketDeals)
			if err != nil {
				return nil, fmt.Errorf("failed to generate detail ID: %w", err)
			}

			vat := int64(0)
			if d.TaxCode != 0 {
				vat = d.Amount / 10
			}

			details[i] = models.Detail{
				ID:            detailID,
				AccountItemID: d.AccountItemID,
				AccountItemName: fmt.Sprintf("Account Item %d", d.AccountItemID),
				TaxCode:       d.TaxCode,
				Amount:        d.Amount,
				Vat:           vat,
				Description:   d.Description,
				ItemID:        d.ItemID,
				SectionID:     d.SectionID,
			}
			totalAmount += d.Amount + vat
		}
		deal.Details = details
		deal.Amount = totalAmount
	}

	deal.UpdatedAt = time.Now()

	if err := s.Put(BucketDeals, id, deal); err != nil {
		return nil, fmt.Errorf("failed to update deal: %w", err)
	}

	return deal, nil
}

// DeleteDeal deletes a deal by ID.
func (s *Store) DeleteDeal(id int64) error {
	return s.Delete(BucketDeals, id)
}
