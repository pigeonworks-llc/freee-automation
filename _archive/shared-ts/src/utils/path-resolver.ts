/**
 * PathResolver
 * Centralized path management for Beancount files and directories
 */

import * as path from 'path';
import * as fs from 'fs';

export interface PathResolverConfig {
  /** Root directory for all Beancount files (e.g., ~/accounting/beancount) */
  beancountRoot: string;
  /** Database file path for sync history */
  databasePath?: string;
  /** Attachments directory for receipts, invoices, etc. */
  attachmentsDir?: string;
}

export class PathResolver {
  private config: Required<PathResolverConfig>;

  constructor(config: PathResolverConfig) {
    this.config = {
      beancountRoot: config.beancountRoot,
      databasePath: config.databasePath || path.join(config.beancountRoot, '.sync', 'sync.db'),
      attachmentsDir: config.attachmentsDir || path.join(config.beancountRoot, 'attachments'),
    };
  }

  /**
   * Get Beancount root directory
   */
  getBeancountRoot(): string {
    return this.config.beancountRoot;
  }

  /**
   * Get database file path
   */
  getDatabasePath(): string {
    return this.config.databasePath;
  }

  /**
   * Get attachments directory
   */
  getAttachmentsDir(): string {
    return this.config.attachmentsDir;
  }

  /**
   * Get directory path for a year (e.g., ~/accounting/beancount/2024)
   */
  getYearDir(year: string | number): string {
    return path.join(this.config.beancountRoot, year.toString());
  }

  /**
   * Get file path for a month (e.g., ~/accounting/beancount/2024/2024-01.beancount)
   */
  getMonthFilePath(yearMonth: string): string {
    const [year, month] = yearMonth.split('-');
    if (!year || !month || month.length !== 2) {
      throw new Error(`Invalid year-month format: ${yearMonth}. Expected YYYY-MM`);
    }
    return path.join(this.getYearDir(year), `${yearMonth}.beancount`);
  }

  /**
   * Get attachment file path for a given date and filename
   * Creates subdirectory by year/month (e.g., attachments/2024/01/receipt.pdf)
   */
  getAttachmentPath(date: string, filename: string): string {
    const [year, month] = date.split('-');
    if (!year || !month) {
      throw new Error(`Invalid date format: ${date}. Expected YYYY-MM-DD`);
    }
    return path.join(this.config.attachmentsDir, year, month, filename);
  }

  /**
   * Ensure directory exists (create if not exists)
   */
  ensureDir(dirPath: string): void {
    if (!fs.existsSync(dirPath)) {
      fs.mkdirSync(dirPath, { recursive: true });
    }
  }

  /**
   * Ensure parent directory of a file exists
   */
  ensureParentDir(filePath: string): void {
    const dir = path.dirname(filePath);
    this.ensureDir(dir);
  }

  /**
   * Check if file exists
   */
  fileExists(filePath: string): boolean {
    return fs.existsSync(filePath);
  }

  /**
   * Create from environment variables
   * Expected env vars:
   * - BEANCOUNT_ROOT: Root directory for Beancount files
   * - BEANCOUNT_DB_PATH: (optional) Database file path
   * - BEANCOUNT_ATTACHMENTS_DIR: (optional) Attachments directory
   */
  static fromEnv(): PathResolver {
    const beancountRoot = process.env.BEANCOUNT_ROOT;
    if (!beancountRoot) {
      throw new Error('BEANCOUNT_ROOT environment variable is required');
    }

    return new PathResolver({
      beancountRoot,
      databasePath: process.env.BEANCOUNT_DB_PATH,
      attachmentsDir: process.env.BEANCOUNT_ATTACHMENTS_DIR,
    });
  }
}
