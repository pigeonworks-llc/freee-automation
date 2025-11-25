-- freee-sync Synchronization History Database
-- Purpose: Track synchronized deals/journals and prevent duplicates
-- Architecture: Hybrid approach (SQLite for history + Beancount files for data)

-- Synchronization history table
-- Stores which freee deals/journals have been synced to Beancount
CREATE TABLE IF NOT EXISTS sync_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  sync_type TEXT NOT NULL CHECK(sync_type IN ('deal', 'journal')),
  freee_id INTEGER NOT NULL,
  issue_date TEXT NOT NULL,
  amount INTEGER NOT NULL,
  beancount_file TEXT NOT NULL,
  synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(sync_type, freee_id)
);

-- Document attachment history table
-- Tracks which documents have been attached to transactions
CREATE TABLE IF NOT EXISTS document_attachments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  transaction_date TEXT NOT NULL,
  ref_number TEXT,
  deal_id INTEGER,
  document_path TEXT NOT NULL,
  attached_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sync metadata table
-- Stores global sync state (last sync time, version, etc.)
CREATE TABLE IF NOT EXISTS sync_metadata (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_sync_history_date ON sync_history(issue_date);
CREATE INDEX IF NOT EXISTS idx_sync_history_freee_id ON sync_history(freee_id);
CREATE INDEX IF NOT EXISTS idx_sync_history_type_id ON sync_history(sync_type, freee_id);
CREATE INDEX IF NOT EXISTS idx_document_date ON document_attachments(transaction_date);
CREATE INDEX IF NOT EXISTS idx_document_deal_id ON document_attachments(deal_id);

-- Initialize metadata
INSERT OR IGNORE INTO sync_metadata (key, value) VALUES ('schema_version', '1.0.0');
INSERT OR IGNORE INTO sync_metadata (key, value) VALUES ('created_at', datetime('now'));
