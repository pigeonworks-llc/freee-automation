/**
 * Beancount Updater
 * Adds document metadata to Beancount transactions
 */

import * as fs from 'fs';
import * as path from 'path';
import dayjs from 'dayjs';
import { Deal } from './types';

export interface UpdateResult {
  success: boolean;
  filePath?: string;
  error?: string;
}

export class BeancountUpdater {
  private beancountDir: string;

  constructor(beancountDir: string) {
    this.beancountDir = beancountDir;
  }

  /**
   * Add document metadata to a transaction
   */
  async addDocument(
    deal: Deal,
    documentPath: string
  ): Promise<UpdateResult> {
    try {
      // Determine which Beancount file to update (based on deal date)
      const year = dayjs(deal.issue_date).format('YYYY');
      const month = dayjs(deal.issue_date).format('MM');
      const beancountFile = path.join(
        this.beancountDir,
        year,
        `${year}-${month}.beancount`
      );

      // Check if file exists
      if (!fs.existsSync(beancountFile)) {
        return {
          success: false,
          error: `Beancount file not found: ${beancountFile}`,
        };
      }

      // Read file content
      const content = fs.readFileSync(beancountFile, 'utf-8');

      // Find transaction and add document metadata
      const updatedContent = this.addDocumentMetadata(
        content,
        deal,
        documentPath
      );

      if (updatedContent === content) {
        return {
          success: false,
          error: `Transaction not found in ${beancountFile}`,
        };
      }

      // Write updated content
      fs.writeFileSync(beancountFile, updatedContent, 'utf-8');

      return {
        success: true,
        filePath: beancountFile,
      };
    } catch (error: any) {
      return {
        success: false,
        error: error.message || String(error),
      };
    }
  }

  /**
   * Add document metadata to transaction
   */
  private addDocumentMetadata(
    content: string,
    deal: Deal,
    documentPath: string
  ): string {
    const lines = content.split('\n');
    const result: string[] = [];
    let inTransaction = false;
    let transactionFound = false;
    let transactionDate = '';
    let hasRefNumber = false;
    let refNumber = '';

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Check if this is a transaction line
      if (line.match(/^\d{4}-\d{2}-\d{2}\s+\*/)) {
        const dateMatch = line.match(/^(\d{4}-\d{2}-\d{2})/);
        if (dateMatch) {
          transactionDate = dateMatch[1];
        }

        // Check for ref_number tag
        const refMatch = line.match(/#([^\s]+)/);
        if (refMatch) {
          hasRefNumber = true;
          refNumber = refMatch[1];
        }

        // Check if this is our transaction
        if (
          transactionDate === deal.issue_date &&
          (!deal.ref_number || refNumber === deal.ref_number)
        ) {
          inTransaction = true;
          transactionFound = true;
        }
      }

      // Add the line
      result.push(line);

      // If we found our transaction, add document metadata after the transaction header
      if (inTransaction && line.match(/^\d{4}-\d{2}-\d{2}\s+\*/)) {
        // Check if document metadata already exists
        const nextLine = lines[i + 1];
        if (!nextLine || !nextLine.trim().startsWith('document:')) {
          result.push(`  document: "${documentPath}"`);
        }
        inTransaction = false;
      }
    }

    return transactionFound ? result.join('\n') : content;
  }

  /**
   * Move PDF file to documents directory
   */
  async moveToDocuments(
    sourcePath: string,
    orderNumber: string,
    orderDate: string
  ): Promise<string> {
    const year = dayjs(orderDate).format('YYYY');
    const targetDir = path.join(this.beancountDir, '..', 'documents', year);

    // Create directory if it doesn't exist
    if (!fs.existsSync(targetDir)) {
      fs.mkdirSync(targetDir, { recursive: true });
    }

    // Generate target filename
    const targetFileName = `amazon-${orderNumber}.pdf`;
    const targetPath = path.join(targetDir, targetFileName);

    // Move file
    fs.renameSync(sourcePath, targetPath);

    // Return relative path for Beancount
    return `documents/${year}/${targetFileName}`;
  }
}
