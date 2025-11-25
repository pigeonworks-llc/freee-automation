package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pigeonworks-llc/freee-emulator/internal/models"
)

// CreateJournal creates a new journal entry in the database.
func (s *Store) CreateJournal(req *models.CreateJournalRequest) (*models.Journal, error) {
	id, err := s.NextID(BucketJournals)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	now := time.Now()
	journal := &models.Journal{
		ID:        id,
		CompanyID: req.CompanyID,
		IssueDate: req.IssueDate,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Convert CreateJournalDetailRequest to JournalDetail.
	details := make([]models.JournalDetail, len(req.Details))
	for i, d := range req.Details {
		detailID, err := s.NextID(BucketJournals)
		if err != nil {
			return nil, fmt.Errorf("failed to generate detail ID: %w", err)
		}

		details[i] = models.JournalDetail{
			ID:              detailID,
			EntryType:       d.EntryType,
			AccountItemID:   d.AccountItemID,
			AccountItemName: fmt.Sprintf("Account Item %d", d.AccountItemID),
			TaxCode:         d.TaxCode,
			PartnerID:       d.PartnerID,
			Amount:          d.Amount,
			Vat:             d.Vat,
			Description:     d.Description,
			ItemID:          d.ItemID,
			SectionID:       d.SectionID,
		}
	}

	journal.Details = details

	if err := s.Put(BucketJournals, id, journal); err != nil {
		return nil, fmt.Errorf("failed to save journal: %w", err)
	}

	return journal, nil
}

// GetJournal retrieves a journal entry by ID.
func (s *Store) GetJournal(id int64) (*models.Journal, error) {
	var journal models.Journal
	if err := s.Get(BucketJournals, id, &journal); err != nil {
		return nil, err
	}
	return &journal, nil
}

// ListJournals retrieves all journal entries, optionally filtered by company ID.
func (s *Store) ListJournals(companyID *int64) ([]*models.Journal, error) {
	filter := func(data []byte) bool {
		if companyID == nil {
			return true
		}

		var journal models.Journal
		if err := json.Unmarshal(data, &journal); err != nil {
			return false
		}
		return journal.CompanyID == *companyID
	}

	results, err := s.List(BucketJournals, filter)
	if err != nil {
		return nil, err
	}

	journals := make([]*models.Journal, 0, len(results))
	for _, data := range results {
		var journal models.Journal
		if err := json.Unmarshal(data, &journal); err != nil {
			return nil, fmt.Errorf("failed to unmarshal journal: %w", err)
		}
		journals = append(journals, &journal)
	}

	return journals, nil
}
