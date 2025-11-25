package db

import (
	"database/sql"
	"fmt"
	"time"
)

// SyncType represents the type of sync record.
type SyncType string

const (
	SyncTypeDeal    SyncType = "deal"
	SyncTypeJournal SyncType = "journal"
)

// SyncRecord represents a sync history record.
type SyncRecord struct {
	ID            int64
	SyncType      SyncType
	FreeeID       int64
	IssueDate     string
	Amount        int64
	BeancountFile string
	SyncedAt      time.Time
}

// DocumentAttachment represents a document attachment record.
type DocumentAttachment struct {
	ID              int64
	TransactionDate string
	RefNumber       sql.NullString
	DealID          sql.NullInt64
	DocumentPath    string
	AttachedAt      time.Time
}

// SyncHistory manages sync history operations.
type SyncHistory struct {
	conn *Connection
}

// NewSyncHistory creates a new SyncHistory instance.
func NewSyncHistory(conn *Connection) *SyncHistory {
	return &SyncHistory{conn: conn}
}

// RecordSync records a sync operation.
// If the record already exists (same sync_type + freee_id), it updates it.
func (s *SyncHistory) RecordSync(record SyncRecord) error {
	query := `
		INSERT INTO sync_history (sync_type, freee_id, issue_date, amount, beancount_file)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(sync_type, freee_id) DO UPDATE SET
			issue_date = excluded.issue_date,
			amount = excluded.amount,
			beancount_file = excluded.beancount_file,
			synced_at = CURRENT_TIMESTAMP
	`

	_, err := s.conn.Exec(query,
		string(record.SyncType),
		record.FreeeID,
		record.IssueDate,
		record.Amount,
		record.BeancountFile,
	)

	if err != nil {
		return fmt.Errorf("failed to record sync: %w", err)
	}

	return nil
}

// IsSynced checks if a deal/journal has been synced.
func (s *SyncHistory) IsSynced(syncType SyncType, freeeID int64) (bool, error) {
	query := `
		SELECT COUNT(*) as count FROM sync_history
		WHERE sync_type = ? AND freee_id = ?
	`

	var count int
	err := s.conn.QueryRow(query, string(syncType), freeeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if synced: %w", err)
	}

	return count > 0, nil
}

// GetSyncRecord retrieves a sync record by freee ID.
func (s *SyncHistory) GetSyncRecord(syncType SyncType, freeeID int64) (*SyncRecord, error) {
	query := `
		SELECT id, sync_type, freee_id, issue_date, amount, beancount_file, synced_at
		FROM sync_history
		WHERE sync_type = ? AND freee_id = ?
	`

	var record SyncRecord
	var syncTypeStr string

	err := s.conn.QueryRow(query, string(syncType), freeeID).Scan(
		&record.ID,
		&syncTypeStr,
		&record.FreeeID,
		&record.IssueDate,
		&record.Amount,
		&record.BeancountFile,
		&record.SyncedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync record: %w", err)
	}

	record.SyncType = SyncType(syncTypeStr)
	return &record, nil
}

// GetSyncRecordsByType retrieves all sync records for a specific type.
func (s *SyncHistory) GetSyncRecordsByType(syncType SyncType) ([]SyncRecord, error) {
	query := `
		SELECT id, sync_type, freee_id, issue_date, amount, beancount_file, synced_at
		FROM sync_history
		WHERE sync_type = ?
		ORDER BY issue_date DESC
	`

	rows, err := s.conn.Query(query, string(syncType))
	if err != nil {
		return nil, fmt.Errorf("failed to get sync records by type: %w", err)
	}
	defer rows.Close()

	var records []SyncRecord
	for rows.Next() {
		var record SyncRecord
		var syncTypeStr string

		if err := rows.Scan(
			&record.ID,
			&syncTypeStr,
			&record.FreeeID,
			&record.IssueDate,
			&record.Amount,
			&record.BeancountFile,
			&record.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sync record: %w", err)
		}

		record.SyncType = SyncType(syncTypeStr)
		records = append(records, record)
	}

	return records, nil
}

// GetSyncedIDs retrieves all synced freee IDs for a specific type.
// This is useful for bulk filtering.
func (s *SyncHistory) GetSyncedIDs(syncType SyncType) ([]int64, error) {
	query := `
		SELECT freee_id FROM sync_history WHERE sync_type = ?
	`

	rows, err := s.conn.Query(query, string(syncType))
	if err != nil {
		return nil, fmt.Errorf("failed to get synced IDs: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan freee ID: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// DeleteSyncRecord deletes a sync record.
// Use case: Force re-sync of a specific deal/journal.
func (s *SyncHistory) DeleteSyncRecord(syncType SyncType, freeeID int64) (bool, error) {
	query := `DELETE FROM sync_history WHERE sync_type = ? AND freee_id = ?`

	result, err := s.conn.Exec(query, string(syncType), freeeID)
	if err != nil {
		return false, fmt.Errorf("failed to delete sync record: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rows > 0, nil
}

// RecordDocumentAttachment records a document attachment.
func (s *SyncHistory) RecordDocumentAttachment(attachment DocumentAttachment) error {
	query := `
		INSERT INTO document_attachments (transaction_date, ref_number, deal_id, document_path)
		VALUES (?, ?, ?, ?)
	`

	_, err := s.conn.Exec(query,
		attachment.TransactionDate,
		attachment.RefNumber,
		attachment.DealID,
		attachment.DocumentPath,
	)

	if err != nil {
		return fmt.Errorf("failed to record document attachment: %w", err)
	}

	return nil
}

// GetDocumentAttachments retrieves document attachments for a deal.
func (s *SyncHistory) GetDocumentAttachments(dealID int64) ([]DocumentAttachment, error) {
	query := `
		SELECT id, transaction_date, ref_number, deal_id, document_path, attached_at
		FROM document_attachments
		WHERE deal_id = ?
		ORDER BY attached_at DESC
	`

	rows, err := s.conn.Query(query, dealID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document attachments: %w", err)
	}
	defer rows.Close()

	var attachments []DocumentAttachment
	for rows.Next() {
		var attachment DocumentAttachment

		if err := rows.Scan(
			&attachment.ID,
			&attachment.TransactionDate,
			&attachment.RefNumber,
			&attachment.DealID,
			&attachment.DocumentPath,
			&attachment.AttachedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document attachment: %w", err)
		}

		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// IsDocumentAttached checks if a document has been attached.
func (s *SyncHistory) IsDocumentAttached(documentPath string) (bool, error) {
	query := `
		SELECT COUNT(*) as count FROM document_attachments
		WHERE document_path = ?
	`

	var count int
	err := s.conn.QueryRow(query, documentPath).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if document attached: %w", err)
	}

	return count > 0, nil
}

// Stats represents sync statistics.
type Stats struct {
	TotalDeals     int
	TotalJournals  int
	TotalDocuments int
	LastSync       sql.NullString
}

// GetStats retrieves sync statistics.
func (s *SyncHistory) GetStats() (*Stats, error) {
	var stats Stats

	// Get deal count
	err := s.conn.QueryRow(`SELECT COUNT(*) FROM sync_history WHERE sync_type = 'deal'`).Scan(&stats.TotalDeals)
	if err != nil {
		return nil, fmt.Errorf("failed to get deal count: %w", err)
	}

	// Get journal count
	err = s.conn.QueryRow(`SELECT COUNT(*) FROM sync_history WHERE sync_type = 'journal'`).Scan(&stats.TotalJournals)
	if err != nil {
		return nil, fmt.Errorf("failed to get journal count: %w", err)
	}

	// Get document count
	err = s.conn.QueryRow(`SELECT COUNT(*) FROM document_attachments`).Scan(&stats.TotalDocuments)
	if err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}

	// Get last sync time
	err = s.conn.QueryRow(`SELECT MAX(synced_at) FROM sync_history`).Scan(&stats.LastSync)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get last sync time: %w", err)
	}

	return &stats, nil
}

// GetMetadata retrieves a metadata value.
func (s *SyncHistory) GetMetadata(key string) (string, error) {
	query := `SELECT value FROM sync_metadata WHERE key = ?`

	var value string
	err := s.conn.QueryRow(query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get metadata: %w", err)
	}

	return value, nil
}

// SetMetadata sets a metadata value.
func (s *SyncHistory) SetMetadata(key, value string) error {
	query := `
		INSERT INTO sync_metadata (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := s.conn.Exec(query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}
