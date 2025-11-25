// Package db provides SQLite database management for sync history and metadata.
package db

// Schema defines the SQL statements to create database tables.
const Schema = `
-- Sync history table
-- Tracks which freee deals/journals have been synced to Beancount
CREATE TABLE IF NOT EXISTS sync_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_type TEXT NOT NULL,           -- 'deal' or 'journal'
    freee_id INTEGER NOT NULL,         -- ID from freee API
    issue_date TEXT NOT NULL,          -- YYYY-MM-DD
    amount INTEGER NOT NULL,           -- Amount in JPY (integer)
    beancount_file TEXT NOT NULL,      -- Path to Beancount file
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(sync_type, freee_id)
);

CREATE INDEX IF NOT EXISTS idx_sync_history_type_id
    ON sync_history(sync_type, freee_id);

CREATE INDEX IF NOT EXISTS idx_sync_history_date
    ON sync_history(issue_date);

-- Document attachments table
-- Tracks which documents (receipts, invoices) have been attached to transactions
CREATE TABLE IF NOT EXISTS document_attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_date TEXT NOT NULL,    -- YYYY-MM-DD
    ref_number TEXT,                   -- Reference number from freee
    deal_id INTEGER,                   -- Deal ID from freee (optional)
    document_path TEXT NOT NULL,       -- Path to the document file
    attached_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_doc_attachments_deal
    ON document_attachments(deal_id);

CREATE INDEX IF NOT EXISTS idx_doc_attachments_path
    ON document_attachments(document_path);

-- Sync metadata table
-- Stores key-value metadata about sync operations
CREATE TABLE IF NOT EXISTS sync_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

// InitializeSchema initializes the database schema.
// It creates all tables if they don't exist.
func InitializeSchema(conn *Connection) error {
	if _, err := conn.Exec(Schema); err != nil {
		return err
	}
	return nil
}
