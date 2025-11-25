/**
 * freee Receipt Uploader
 * Uploads receipt PDFs to freee API
 */

import * as fs from 'fs';
import FormData from 'form-data';
import { Deal } from './types';
import { findMatchingRule, AccountRule } from './account-rules';

export interface AccountItem {
  id: number;
  name: string;
  default_tax_code: number;
}

export interface CreateDealResult {
  success: boolean;
  dealId?: number;
  error?: string;
}

export interface UploadResult {
  success: boolean;
  receiptId?: number;
  error?: string;
}

export interface WalletTransaction {
  id: number;
  company_id: number;
  date: string;
  amount: number;
  due_amount: number;
  balance: number;
  entry_side: 'income' | 'expense';
  walletable_type: 'bank_account' | 'credit_card' | 'wallet';
  walletable_id: number;
  description: string;
  status: number; // 1: 未登録, 2: 登録済み
}

export interface AmazonTransaction {
  id: number;
  date: string;
  amount: number;
  description: string;
}

export class FreeeUploader {
  private apiUrl: string;
  private accessToken: string;
  private companyId: number;
  private accountItemCache: Map<string, AccountItem> = new Map();

  constructor(apiUrl: string, accessToken: string, companyId: number) {
    this.apiUrl = apiUrl;
    this.accessToken = accessToken;
    this.companyId = companyId;
  }

  /**
   * Upload receipt to freee API
   * POST /api/1/receipts
   */
  async uploadReceipt(
    deal: Deal,
    pdfPath: string,
    orderNumber: string
  ): Promise<UploadResult> {
    const maxRetries = 3;
    let lastError: string = '';

    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        const result = await this.attemptUpload(deal, pdfPath, orderNumber);
        return result;
      } catch (error: any) {
        lastError = error.message || String(error);
        console.warn(
          `Upload attempt ${attempt}/${maxRetries} failed: ${lastError}`
        );

        // Check if we should retry
        if (this.shouldRetry(error, attempt, maxRetries)) {
          const delay = this.getRetryDelay(error);
          console.log(`Waiting ${delay}ms before retry...`);
          await this.sleep(delay);
          continue;
        } else {
          // Don't retry
          break;
        }
      }
    }

    return {
      success: false,
      error: `Failed after ${maxRetries} attempts: ${lastError}`,
    };
  }

  /**
   * Attempt to upload receipt
   */
  private async attemptUpload(
    deal: Deal,
    pdfPath: string,
    orderNumber: string
  ): Promise<UploadResult> {
    // Create form data
    const form = new FormData();
    form.append('company_id', String(this.companyId));
    form.append('receipt', fs.createReadStream(pdfPath));
    form.append('description', `Amazon Order: ${orderNumber}`);
    form.append('issue_date', deal.issue_date);

    // Make request
    const url = `${this.apiUrl}/api/1/receipts`;
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${this.accessToken}`,
        ...form.getHeaders(),
      },
      body: form as any,
    });

    // Handle response
    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(
        `HTTP ${response.status}: ${response.statusText} - ${errorText}`
      );
    }

    const data = await response.json() as any;
    return {
      success: true,
      receiptId: data.receipt?.id,
    };
  }

  /**
   * Fetch all deals from freee API
   */
  async fetchDeals(): Promise<Deal[]> {
    const url = `${this.apiUrl}/api/1/deals?company_id=${this.companyId}`;
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${this.accessToken}`,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(
        `Failed to fetch deals: HTTP ${response.status} ${response.statusText}`
      );
    }

    const data = await response.json() as any;
    return data.deals || [];
  }

  /**
   * Fetch unregistered wallet transactions from freee API
   * GET /api/1/wallet_txns
   *
   * 未処理明細の判定には due_amount フィールドを使用:
   * - 未処理: due_amount == amount (取り込み直後)
   * - 処理済み: due_amount == 0 (完全に取引登録済み)
   * - 一部処理: 0 < due_amount < amount
   *
   * See: docs/freee-api-reference.md for full specification
   */
  async fetchUnregisteredTransactions(): Promise<WalletTransaction[]> {
    const url = `${this.apiUrl}/api/1/wallet_txns?company_id=${this.companyId}`;
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${this.accessToken}`,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(
        `Failed to fetch wallet transactions: HTTP ${response.status} ${response.statusText}`
      );
    }

    const data = await response.json() as any;
    const allTxns: WalletTransaction[] = data.wallet_txns || [];

    // Filter by due_amount != 0 to get unprocessed transactions
    return allTxns.filter(txn => txn.due_amount !== 0);
  }

  /**
   * Filter transactions for Amazon.co.jp purchases
   */
  filterAmazonTransactions(transactions: WalletTransaction[]): AmazonTransaction[] {
    const amazonPatterns = [
      /amazon/i,
      /アマゾン/,
      /AMAZON/,
      /amzn/i,
    ];

    return transactions
      .filter(txn => {
        // Only expense transactions
        if (txn.entry_side !== 'expense') return false;
        // Check if description contains Amazon
        return amazonPatterns.some(pattern => pattern.test(txn.description));
      })
      .map(txn => ({
        id: txn.id,
        date: txn.date,
        amount: Math.abs(txn.amount),
        description: txn.description,
      }));
  }

  /**
   * Check if error should be retried
   */
  private shouldRetry(error: any, attempt: number, maxRetries: number): boolean {
    if (attempt >= maxRetries) {
      return false;
    }

    const message = error.message || String(error);

    // Retry on rate limit (429)
    if (message.includes('429')) {
      return true;
    }

    // Retry on server errors (5xx)
    if (message.includes('500') || message.includes('502') || message.includes('503')) {
      return true;
    }

    // Don't retry on client errors (4xx except 429)
    if (message.includes('HTTP 4')) {
      return false;
    }

    // Retry on network errors
    if (
      message.includes('ECONNREFUSED') ||
      message.includes('ETIMEDOUT') ||
      message.includes('ENOTFOUND')
    ) {
      return true;
    }

    return false;
  }

  /**
   * Get retry delay based on error type
   */
  private getRetryDelay(error: any): number {
    const message = error.message || String(error);

    // 10 seconds for rate limit
    if (message.includes('429')) {
      return 10000;
    }

    // 5 seconds for server errors
    if (message.includes('500') || message.includes('502') || message.includes('503')) {
      return 5000;
    }

    // 2 seconds default
    return 2000;
  }

  /**
   * Sleep helper
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * Fetch account items from freee API
   * GET /api/1/account_items
   */
  async fetchAccountItems(): Promise<AccountItem[]> {
    const url = `${this.apiUrl}/api/1/account_items?company_id=${this.companyId}`;
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${this.accessToken}`,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(
        `Failed to fetch account items: HTTP ${response.status} - ${errorText}`
      );
    }

    const data = (await response.json()) as any;
    const items: AccountItem[] = (data.account_items || []).map((item: any) => ({
      id: item.id,
      name: item.name,
      default_tax_code: item.default_tax_code,
    }));

    // Cache items by name
    for (const item of items) {
      this.accountItemCache.set(item.name, item);
    }

    return items;
  }

  /**
   * Get account item ID by name
   */
  async getAccountItemId(name: string): Promise<number | null> {
    // Check cache first
    if (this.accountItemCache.has(name)) {
      return this.accountItemCache.get(name)!.id;
    }

    // Fetch and cache
    await this.fetchAccountItems();

    if (this.accountItemCache.has(name)) {
      return this.accountItemCache.get(name)!.id;
    }

    return null;
  }

  /**
   * Create a deal with receipt attached
   * POST /api/1/deals
   */
  async createDeal(params: {
    issueDate: string;
    amount: number;
    description: string;
    accountItemId: number;
    taxCode: number;
    walletableType: 'credit_card' | 'bank_account' | 'wallet';
    walletableId: number;
    receiptIds?: number[];
  }): Promise<CreateDealResult> {
    const url = `${this.apiUrl}/api/1/deals`;

    const body = {
      company_id: this.companyId,
      issue_date: params.issueDate,
      type: 'expense',
      details: [
        {
          account_item_id: params.accountItemId,
          tax_code: params.taxCode,
          amount: params.amount,
          description: params.description,
        },
      ],
      payments: [
        {
          from_walletable_type: params.walletableType,
          from_walletable_id: params.walletableId,
          date: params.issueDate,
          amount: params.amount,
        },
      ],
      receipt_ids: params.receiptIds || [],
    };

    const response = await fetch(url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${this.accessToken}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const errorText = await response.text();
      return {
        success: false,
        error: `HTTP ${response.status}: ${errorText}`,
      };
    }

    const data = (await response.json()) as any;
    return {
      success: true,
      dealId: data.deal?.id,
    };
  }

  /**
   * Create deal from wallet transaction using rule-based account matching
   */
  async createDealFromTransaction(
    txn: WalletTransaction,
    receiptId?: number
  ): Promise<CreateDealResult> {
    // Find matching rule
    const rule = findMatchingRule(txn.description);
    if (!rule) {
      return {
        success: false,
        error: `No matching rule for: ${txn.description}`,
      };
    }

    // Get account item ID
    const accountItemId = await this.getAccountItemId(rule.accountName);
    if (!accountItemId) {
      return {
        success: false,
        error: `Account item not found: ${rule.accountName}`,
      };
    }

    // Create deal
    return this.createDeal({
      issueDate: txn.date,
      amount: Math.abs(txn.amount),
      description: txn.description,
      accountItemId,
      taxCode: rule.taxCode,
      walletableType: txn.walletable_type,
      walletableId: txn.walletable_id,
      receiptIds: receiptId ? [receiptId] : [],
    });
  }
}
