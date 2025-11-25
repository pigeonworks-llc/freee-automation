/**
 * Beancount Converter
 * Converts freee transactions to Beancount format
 */

import { Deal, Journal, JournalDetail } from '../freee/types';
import { AccountMapper } from './mapper';

export interface BeancountTransaction {
  date: string;
  narration: string;
  payee?: string;
  tags?: string[];
  postings: BeancountPosting[];
}

export interface BeancountPosting {
  account: string;
  amount: number;
  currency: string;
  comment?: string;
}

export class BeancountConverter {
  private mapper: AccountMapper;
  private currency: string;

  constructor(mapper: AccountMapper, currency: string = 'JPY') {
    this.mapper = mapper;
    this.currency = currency;
  }

  /**
   * Sanitize account name for Beancount (remove spaces)
   */
  private sanitizeAccountName(name: string): string {
    return name.replace(/\s+/g, '');
  }

  /**
   * Convert a Deal to Beancount transaction
   */
  convertDeal(deal: Deal): BeancountTransaction {
    const postings: BeancountPosting[] = [];

    // For income transactions, amounts should be negative (credit side)
    // For expense transactions, amounts should be positive (debit side)
    const amountMultiplier = deal.type === 'income' ? -1 : 1;

    // Process each detail line in the deal
    for (const detail of deal.details) {
      const beancountAccount = this.mapper.getBeancountAccount(detail.account_item_name);

      if (!beancountAccount) {
        console.warn(
          `No mapping found for account: ${detail.account_item_name}, using original name`
        );
      }

      const account = beancountAccount || `Expenses:Unmapped:${this.sanitizeAccountName(detail.account_item_name)}`;

      // Add posting for the main account (excluding VAT)
      postings.push({
        account,
        amount: detail.amount * amountMultiplier,
        currency: this.currency,
        comment: detail.description || undefined,
      });

      // Add VAT posting if applicable
      if (detail.vat > 0) {
        const taxAccount = this.mapper.getTaxAccount('tax_10'); // Default to 10% tax
        if (taxAccount) {
          postings.push({
            account: taxAccount,
            amount: detail.vat * amountMultiplier,
            currency: this.currency,
            comment: '消費税',
          });
        }
      }
    }

    // Add payment postings
    if (deal.payments && deal.payments.length > 0) {
      for (const payment of deal.payments) {
        const walletAccount = this.getWalletAccount(
          payment.from_walletable_type,
          payment.from_walletable_id
        );

        postings.push({
          account: walletAccount,
          amount: -payment.amount, // Negative for outflow
          currency: this.currency,
          comment: `Payment from ${payment.from_walletable_type}`,
        });
      }
    } else {
      // If no payment specified, add a balancing entry to a default account
      const totalAmount = deal.amount;
      const defaultAccount = 'Assets:Current:Bank:Ordinary';

      // Opposite sign from the detail amounts to balance the transaction
      postings.push({
        account: defaultAccount,
        amount: totalAmount * -amountMultiplier,
        currency: this.currency,
      });
    }

    return {
      date: deal.issue_date,
      narration: this.buildDealNarration(deal),
      payee: deal.partner_code,
      tags: deal.ref_number ? [deal.ref_number] : undefined,
      postings,
    };
  }

  /**
   * Convert a Journal to Beancount transaction
   */
  convertJournal(journal: Journal): BeancountTransaction {
    const postings: BeancountPosting[] = [];

    for (const detail of journal.details) {
      const beancountAccount = this.mapper.getBeancountAccount(detail.account_item_name);

      if (!beancountAccount) {
        console.warn(
          `No mapping found for account: ${detail.account_item_name}, using original name`
        );
      }

      const account = beancountAccount || `Expenses:Unmapped:${this.sanitizeAccountName(detail.account_item_name)}`;

      // Debit = positive, Credit = negative
      const amount = detail.entry_type === 'debit' ? detail.amount : -detail.amount;

      postings.push({
        account,
        amount,
        currency: this.currency,
        comment: detail.description || undefined,
      });

      // Add VAT posting if applicable
      if (detail.vat > 0) {
        const taxAccount = this.mapper.getTaxAccount('tax_10');
        if (taxAccount) {
          const vatAmount = detail.entry_type === 'debit' ? detail.vat : -detail.vat;
          postings.push({
            account: taxAccount,
            amount: vatAmount,
            currency: this.currency,
            comment: '消費税',
          });
        }
      }
    }

    return {
      date: journal.issue_date,
      narration: this.buildJournalNarration(journal),
      postings,
    };
  }

  /**
   * Format a Beancount transaction as a string
   */
  formatTransaction(txn: BeancountTransaction): string {
    let output = '';

    // Transaction header
    output += `${txn.date} *`;
    if (txn.payee) {
      output += ` "${txn.payee}"`;
    }
    output += ` "${txn.narration}"`;
    if (txn.tags && txn.tags.length > 0) {
      output += ` #${txn.tags.join(' #')}`;
    }
    output += '\n';

    // Postings
    for (const posting of txn.postings) {
      output += `  ${posting.account}`;

      // Right-align amount (typical Beancount style)
      const spaces = Math.max(1, 60 - posting.account.length);
      output += ' '.repeat(spaces);

      // Format amount
      const sign = posting.amount >= 0 ? '' : '-';
      const absAmount = Math.abs(posting.amount).toFixed(0);
      output += `${sign}${absAmount} ${posting.currency}`;

      if (posting.comment) {
        output += ` ; ${posting.comment}`;
      }

      output += '\n';
    }

    return output;
  }

  /**
   * Build narration from Deal
   */
  private buildDealNarration(deal: Deal): string {
    if (deal.details.length === 1 && deal.details[0].description) {
      return deal.details[0].description;
    }

    const accountNames = deal.details.map((d) => d.account_item_name).join(', ');
    return `${deal.type === 'income' ? '収入' : '支出'}: ${accountNames}`;
  }

  /**
   * Build narration from Journal
   */
  private buildJournalNarration(journal: Journal): string {
    const descriptions = journal.details
      .filter((d) => d.description)
      .map((d) => d.description);

    if (descriptions.length > 0) {
      return descriptions[0]!;
    }

    return '仕訳';
  }

  /**
   * Get wallet account from walletable_type and ID
   */
  private getWalletAccount(type: string, id: number): string {
    if (type === 'bank_account') {
      return 'Assets:Current:Bank:Ordinary';
    } else if (type === 'credit_card') {
      return 'Liabilities:Current:CreditCard';
    }
    return 'Assets:Current:Bank:Ordinary';
  }
}
