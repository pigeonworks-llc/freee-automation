/**
 * Sync orchestration
 * Fetches data from freee and converts to Beancount format
 * With SQLite-based duplicate prevention
 */

import * as fs from 'fs';
import * as path from 'path';
import { FreeeClient } from './freee/client';
import { Deal, Journal } from './freee/types';
import { AccountMapper } from './converter/mapper';
import { BeancountConverter } from './converter/converter';
import { DatabaseConnection, createConnection } from './db/connection';
import { SyncHistory } from './db/sync-history';

export interface SyncOptions {
  dateFrom: string;
  dateTo: string;
  beancountDir: string;
  dryRun?: boolean;
}

export interface SyncResult {
  dealsCount: number;
  journalsCount: number;
  filesWritten: string[];
}

export class SyncOrchestrator {
  private client: FreeeClient;
  private mapper: AccountMapper;
  private converter: BeancountConverter;
  private dbConnection: DatabaseConnection;
  private syncHistory: SyncHistory;

  constructor(
    client: FreeeClient,
    mapper: AccountMapper,
    dbConnection?: DatabaseConnection
  ) {
    this.client = client;
    this.mapper = mapper;
    this.converter = new BeancountConverter(mapper);
    this.dbConnection = dbConnection || createConnection();
    this.syncHistory = new SyncHistory(this.dbConnection);
  }

  /**
   * Perform sync operation with duplicate prevention
   */
  async sync(options: SyncOptions): Promise<SyncResult> {
    console.log(`Syncing from ${options.dateFrom} to ${options.dateTo}`);

    // Fetch deals from freee
    console.log('Fetching deals from freee...');
    const allDeals = await this.client.fetchAllDeals(options.dateFrom, options.dateTo);
    console.log(`Fetched ${allDeals.length} deals`);

    // Fetch journals from freee
    console.log('Fetching journals from freee...');
    const allJournals = await this.client.fetchAllJournals(options.dateFrom, options.dateTo);
    console.log(`Fetched ${allJournals.length} journals`);

    // Filter out already synced items
    console.log('Checking for already synced items...');
    const syncedDealIds = await this.syncHistory.getSyncedIds('deal');
    const syncedJournalIds = await this.syncHistory.getSyncedIds('journal');

    const newDeals = allDeals.filter((d) => !syncedDealIds.includes(d.id));
    const newJournals = allJournals.filter((j) => !syncedJournalIds.includes(j.id));

    console.log(
      `New items: ${newDeals.length} deals, ${newJournals.length} journals ` +
      `(skipped: ${allDeals.length - newDeals.length} deals, ${allJournals.length - newJournals.length} journals)`
    );

    if (newDeals.length === 0 && newJournals.length === 0) {
      console.log('No new items to sync');
      return {
        dealsCount: 0,
        journalsCount: 0,
        filesWritten: [],
      };
    }

    // Group by month
    const dealsByMonth = this.groupByMonth(newDeals, (d) => d.issue_date);
    const journalsByMonth = this.groupByMonth(newJournals, (j) => j.issue_date);

    // Get all unique months
    const allMonths = new Set([
      ...Object.keys(dealsByMonth),
      ...Object.keys(journalsByMonth),
    ]);

    const filesWritten: string[] = [];

    // Process each month
    for (const monthKey of Array.from(allMonths).sort()) {
      const monthDeals = dealsByMonth[monthKey] || [];
      const monthJournals = journalsByMonth[monthKey] || [];

      const filePath = this.getMonthFilePath(monthKey, options.beancountDir);

      if (!options.dryRun) {
        // Append to existing file or create new
        this.appendBeancountTransactions(filePath, monthDeals, monthJournals);
        filesWritten.push(filePath);
        console.log(`Updated ${filePath} (${monthDeals.length} deals, ${monthJournals.length} journals)`);

        // Record sync history
        for (const deal of monthDeals) {
          await this.syncHistory.recordSync({
            sync_type: 'deal',
            freee_id: deal.id,
            issue_date: deal.issue_date,
            amount: deal.amount,
            beancount_file: filePath,
          });
        }

        for (const journal of monthJournals) {
          await this.syncHistory.recordSync({
            sync_type: 'journal',
            freee_id: journal.id,
            issue_date: journal.issue_date,
            amount: journal.details[0]?.amount || 0,
            beancount_file: filePath,
          });
        }
      } else {
        console.log(`[DRY RUN] Would append to ${filePath}`);
        for (const deal of monthDeals) {
          const txn = this.converter.convertDeal(deal);
          console.log(this.converter.formatTransaction(txn));
        }
        for (const journal of monthJournals) {
          const txn = this.converter.convertJournal(journal);
          console.log(this.converter.formatTransaction(txn));
        }
      }
    }

    // Display statistics
    const stats = await this.syncHistory.getStats();
    console.log('\n=== Sync Statistics ===');
    console.log(`Total synced deals: ${stats.total_deals}`);
    console.log(`Total synced journals: ${stats.total_journals}`);
    console.log(`Total documents: ${stats.total_documents}`);
    console.log(`Last sync: ${stats.last_sync}`);

    return {
      dealsCount: newDeals.length,
      journalsCount: newJournals.length,
      filesWritten,
    };
  }

  /**
   * Group items by month (YYYY-MM)
   */
  private groupByMonth<T>(
    items: T[],
    getDate: (item: T) => string
  ): Record<string, T[]> {
    const groups: Record<string, T[]> = {};

    for (const item of items) {
      const date = getDate(item);
      const monthKey = date.substring(0, 7); // YYYY-MM

      if (!groups[monthKey]) {
        groups[monthKey] = [];
      }

      groups[monthKey].push(item);
    }

    return groups;
  }

  /**
   * Generate Beancount file content for a month
   */
  private generateBeancountFile(deals: Deal[], journals: Journal[]): string {
    let content = '';

    // Header comment
    if (deals.length > 0 || journals.length > 0) {
      const month = deals.length > 0 ? deals[0].issue_date.substring(0, 7) : journals[0].issue_date.substring(0, 7);
      content += `; freee sync - ${month}\n`;
      content += `; Generated at ${new Date().toISOString()}\n`;
      content += `; Deals: ${deals.length}, Journals: ${journals.length}\n`;
      content += '\n';
    }

    // Convert deals
    for (const deal of deals) {
      const txn = this.converter.convertDeal(deal);
      content += this.converter.formatTransaction(txn);
      content += '\n';
    }

    // Convert journals
    for (const journal of journals) {
      const txn = this.converter.convertJournal(journal);
      content += this.converter.formatTransaction(txn);
      content += '\n';
    }

    return content;
  }

  /**
   * Get file path for a month
   */
  private getMonthFilePath(monthKey: string, beancountDir: string): string {
    const [year, month] = monthKey.split('-');
    const yearDir = path.join(beancountDir, year);
    const fileName = `${year}-${month}.beancount`;

    return path.join(yearDir, fileName);
  }

  /**
   * Write Beancount file
   */
  private writeBeancountFile(filePath: string, content: string): void {
    const dir = path.dirname(filePath);

    // Ensure directory exists
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    fs.writeFileSync(filePath, content, 'utf-8');
  }

  /**
   * Append transactions to Beancount file
   * Creates new file if it doesn't exist
   */
  private appendBeancountTransactions(
    filePath: string,
    deals: Deal[],
    journals: Journal[]
  ): void {
    const dir = path.dirname(filePath);

    // Ensure directory exists
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    // Check if file exists
    const fileExists = fs.existsSync(filePath);

    if (!fileExists) {
      // Create new file with header
      const monthKey = path.basename(filePath, '.beancount');
      let content = `; freee sync - ${monthKey}\n`;
      content += `; Generated at ${new Date().toISOString()}\n\n`;
      fs.writeFileSync(filePath, content, 'utf-8');
    }

    // Append transactions
    let content = '';

    for (const deal of deals) {
      const txn = this.converter.convertDeal(deal);
      content += this.converter.formatTransaction(txn);
      content += '\n';
    }

    for (const journal of journals) {
      const txn = this.converter.convertJournal(journal);
      content += this.converter.formatTransaction(txn);
      content += '\n';
    }

    // Append to file
    fs.appendFileSync(filePath, content, 'utf-8');
  }

  /**
   * Close database connection
   * Should be called when done with sync operations
   */
  close(): void {
    this.dbConnection.close();
  }
}
