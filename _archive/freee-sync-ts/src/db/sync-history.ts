import { DatabaseConnection } from './connection';

/**
 * Sync history record
 * Tracks which freee deals/journals have been synced to Beancount
 */
export interface SyncRecord {
  id: number;
  sync_type: 'deal' | 'journal';
  freee_id: number;
  issue_date: string;
  amount: number;
  beancount_file: string;
  synced_at: string;
}

/**
 * Document attachment record
 * Tracks which documents have been attached to transactions
 */
export interface DocumentAttachment {
  id: number;
  transaction_date: string;
  ref_number: string | null;
  deal_id: number | null;
  document_path: string;
  attached_at: string;
}

/**
 * Sync history manager
 *
 * Purpose:
 * - Record which freee deals/journals have been synced
 * - Prevent duplicate syncs
 * - Track document attachments
 * - Query sync metadata
 */
export class SyncHistory {
  constructor(private conn: DatabaseConnection) {}

  /**
   * Record a sync operation
   * If the record already exists (same sync_type + freee_id), update it
   */
  async recordSync(record: Omit<SyncRecord, 'id' | 'synced_at'>): Promise<void> {
    this.conn.execute(
      `INSERT INTO sync_history (sync_type, freee_id, issue_date, amount, beancount_file)
       VALUES (?, ?, ?, ?, ?)
       ON CONFLICT(sync_type, freee_id) DO UPDATE SET
         issue_date = excluded.issue_date,
         amount = excluded.amount,
         beancount_file = excluded.beancount_file,
         synced_at = CURRENT_TIMESTAMP`,
      [
        record.sync_type,
        record.freee_id,
        record.issue_date,
        record.amount,
        record.beancount_file,
      ]
    );
  }

  /**
   * Check if a deal/journal has been synced
   */
  async isSynced(syncType: 'deal' | 'journal', freeeId: number): Promise<boolean> {
    const result = this.conn.queryOne<{ count: number }>(
      `SELECT COUNT(*) as count FROM sync_history
       WHERE sync_type = ? AND freee_id = ?`,
      [syncType, freeeId]
    );
    return (result?.count ?? 0) > 0;
  }

  /**
   * Get sync record by freee ID
   */
  async getSyncRecord(
    syncType: 'deal' | 'journal',
    freeeId: number
  ): Promise<SyncRecord | null> {
    const result = this.conn.queryOne<SyncRecord>(
      `SELECT * FROM sync_history
       WHERE sync_type = ? AND freee_id = ?`,
      [syncType, freeeId]
    );
    return result || null;
  }

  /**
   * Get all sync records for a specific type
   */
  async getSyncRecordsByType(syncType: 'deal' | 'journal'): Promise<SyncRecord[]> {
    return this.conn.query<SyncRecord>(
      `SELECT * FROM sync_history
       WHERE sync_type = ?
       ORDER BY issue_date DESC`,
      [syncType]
    );
  }

  /**
   * Get the last sync time across all records
   */
  async getLastSyncTime(): Promise<Date | null> {
    const result = this.conn.queryOne<{ max_time: string }>(
      `SELECT MAX(synced_at) as max_time FROM sync_history`
    );
    return result?.max_time ? new Date(result.max_time) : null;
  }

  /**
   * Get all synced freee IDs for a specific type
   * Useful for bulk filtering
   */
  async getSyncedIds(syncType: 'deal' | 'journal'): Promise<number[]> {
    const results = this.conn.query<{ freee_id: number }>(
      `SELECT freee_id FROM sync_history WHERE sync_type = ?`,
      [syncType]
    );
    return results.map((r) => r.freee_id);
  }

  /**
   * Delete a sync record
   * Use case: Force re-sync of a specific deal/journal
   */
  async deleteSyncRecord(syncType: 'deal' | 'journal', freeeId: number): Promise<boolean> {
    const result = this.conn.execute(
      `DELETE FROM sync_history WHERE sync_type = ? AND freee_id = ?`,
      [syncType, freeeId]
    );
    return result.changes > 0;
  }

  /**
   * Record a document attachment
   */
  async recordDocumentAttachment(
    attachment: Omit<DocumentAttachment, 'id' | 'attached_at'>
  ): Promise<void> {
    this.conn.execute(
      `INSERT INTO document_attachments (transaction_date, ref_number, deal_id, document_path)
       VALUES (?, ?, ?, ?)`,
      [
        attachment.transaction_date,
        attachment.ref_number,
        attachment.deal_id,
        attachment.document_path,
      ]
    );
  }

  /**
   * Get document attachments for a deal
   */
  async getDocumentAttachments(dealId: number): Promise<DocumentAttachment[]> {
    return this.conn.query<DocumentAttachment>(
      `SELECT * FROM document_attachments
       WHERE deal_id = ?
       ORDER BY attached_at DESC`,
      [dealId]
    );
  }

  /**
   * Check if a document has been attached
   */
  async isDocumentAttached(documentPath: string): Promise<boolean> {
    const result = this.conn.queryOne<{ count: number }>(
      `SELECT COUNT(*) as count FROM document_attachments
       WHERE document_path = ?`,
      [documentPath]
    );
    return (result?.count ?? 0) > 0;
  }

  /**
   * Get sync statistics
   */
  async getStats(): Promise<{
    total_deals: number;
    total_journals: number;
    total_documents: number;
    last_sync: string | null;
  }> {
    const dealCount = this.conn.queryOne<{ count: number }>(
      `SELECT COUNT(*) as count FROM sync_history WHERE sync_type = 'deal'`
    );

    const journalCount = this.conn.queryOne<{ count: number }>(
      `SELECT COUNT(*) as count FROM sync_history WHERE sync_type = 'journal'`
    );

    const documentCount = this.conn.queryOne<{ count: number }>(
      `SELECT COUNT(*) as count FROM document_attachments`
    );

    const lastSync = this.conn.queryOne<{ max_time: string }>(
      `SELECT MAX(synced_at) as max_time FROM sync_history`
    );

    return {
      total_deals: dealCount?.count ?? 0,
      total_journals: journalCount?.count ?? 0,
      total_documents: documentCount?.count ?? 0,
      last_sync: lastSync?.max_time ?? null,
    };
  }

  /**
   * Get metadata value
   */
  async getMetadata(key: string): Promise<string | null> {
    const result = this.conn.queryOne<{ value: string }>(
      `SELECT value FROM sync_metadata WHERE key = ?`,
      [key]
    );
    return result?.value ?? null;
  }

  /**
   * Set metadata value
   */
  async setMetadata(key: string, value: string): Promise<void> {
    this.conn.execute(
      `INSERT INTO sync_metadata (key, value, updated_at)
       VALUES (?, ?, CURRENT_TIMESTAMP)
       ON CONFLICT(key) DO UPDATE SET
         value = excluded.value,
         updated_at = CURRENT_TIMESTAMP`,
      [key, value]
    );
  }
}
