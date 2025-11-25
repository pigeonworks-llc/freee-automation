/**
 * BeancountRepository
 * Repository pattern for Beancount file operations
 */

import * as fs from 'fs';
import * as path from 'path';
import { PathResolver } from '../utils/path-resolver';

export interface BeancountTransaction {
  date: string; // YYYY-MM-DD
  narration: string;
  postings: BeancountPosting[];
  tags?: string[];
  links?: string[];
  metadata?: Record<string, string>;
}

export interface BeancountPosting {
  account: string;
  amount?: number;
  currency?: string;
  comment?: string;
}

export interface AppendTransactionOptions {
  /** Year-month in YYYY-MM format */
  yearMonth: string;
  /** Transaction to append */
  transaction: string;
  /** Additional header comment (optional) */
  comment?: string;
}

/**
 * Repository interface for Beancount operations
 */
export interface IBeancountRepository {
  /**
   * Append a transaction to a monthly file
   */
  appendTransaction(options: AppendTransactionOptions): void;

  /**
   * Read content of a monthly file
   */
  readMonthFile(yearMonth: string): string | null;

  /**
   * Check if a monthly file exists
   */
  monthFileExists(yearMonth: string): boolean;

  /**
   * Get all monthly files in a year
   */
  getMonthFilesInYear(year: string | number): string[];

  /**
   * Ensure monthly file exists with header
   */
  ensureMonthFile(yearMonth: string): void;
}

/**
 * File system implementation of BeancountRepository
 */
export class FileSystemBeancountRepository implements IBeancountRepository {
  constructor(private pathResolver: PathResolver) {}

  /**
   * Append a transaction to a monthly file
   */
  appendTransaction(options: AppendTransactionOptions): void {
    const filePath = this.pathResolver.getMonthFilePath(options.yearMonth);

    // Ensure file exists with header
    this.ensureMonthFile(options.yearMonth);

    // Append transaction
    let content = '';
    if (options.comment) {
      content += `; ${options.comment}\n`;
    }
    content += options.transaction;
    if (!options.transaction.endsWith('\n')) {
      content += '\n';
    }
    content += '\n'; // Add blank line after transaction

    fs.appendFileSync(filePath, content, 'utf-8');
  }

  /**
   * Read content of a monthly file
   */
  readMonthFile(yearMonth: string): string | null {
    const filePath = this.pathResolver.getMonthFilePath(yearMonth);
    if (!this.pathResolver.fileExists(filePath)) {
      return null;
    }
    return fs.readFileSync(filePath, 'utf-8');
  }

  /**
   * Check if a monthly file exists
   */
  monthFileExists(yearMonth: string): boolean {
    const filePath = this.pathResolver.getMonthFilePath(yearMonth);
    return this.pathResolver.fileExists(filePath);
  }

  /**
   * Get all monthly files in a year
   */
  getMonthFilesInYear(year: string | number): string[] {
    const yearDir = this.pathResolver.getYearDir(year);
    if (!this.pathResolver.fileExists(yearDir)) {
      return [];
    }

    const files = fs.readdirSync(yearDir);
    return files
      .filter((f) => f.endsWith('.beancount'))
      .map((f) => path.basename(f, '.beancount'))
      .sort();
  }

  /**
   * Ensure monthly file exists with header
   */
  ensureMonthFile(yearMonth: string): void {
    const filePath = this.pathResolver.getMonthFilePath(yearMonth);

    if (this.pathResolver.fileExists(filePath)) {
      return;
    }

    // Ensure parent directory exists
    this.pathResolver.ensureParentDir(filePath);

    // Create file with header
    const header = this.generateFileHeader(yearMonth);
    fs.writeFileSync(filePath, header, 'utf-8');
  }

  /**
   * Generate file header for a monthly file
   */
  private generateFileHeader(yearMonth: string): string {
    const now = new Date().toISOString();
    return `; Beancount file for ${yearMonth}\n; Generated at ${now}\n\n`;
  }
}

/**
 * Factory function to create BeancountRepository from environment
 */
export function createBeancountRepository(): IBeancountRepository {
  const pathResolver = PathResolver.fromEnv();
  return new FileSystemBeancountRepository(pathResolver);
}
