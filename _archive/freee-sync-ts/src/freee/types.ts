/**
 * freee Accounting API Type Definitions
 * Based on freee-emulator models
 */

/**
 * Deal represents a transaction in freee accounting API
 */
export interface Deal {
  id: number;
  company_id: number;
  issue_date: string; // YYYY-MM-DD
  due_date?: string;
  type: 'income' | 'expense';
  details: Detail[];
  payments?: Payment[];
  amount: number;
  due_amount?: number;
  ref_number?: string;
  partner_id?: number;
  partner_code?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Detail represents a line item in a deal
 */
export interface Detail {
  id: number;
  account_item_id: number;
  account_item_name: string;
  tax_code: number;
  amount: number;
  vat: number;
  description?: string;
  item_id?: number;
  item_name?: string;
  section_id?: number;
  section_name?: string;
  tag_ids?: number[];
  tag_names?: string[];
  segment_1_tag_id?: number;
  segment_1_tag_name?: string;
  segment_2_tag_id?: number;
  segment_2_tag_name?: string;
  segment_3_tag_id?: number;
  segment_3_tag_name?: string;
}

/**
 * Payment represents payment information for a deal
 */
export interface Payment {
  id: number;
  date: string; // YYYY-MM-DD
  amount: number;
  from_walletable_type: 'bank_account' | 'credit_card';
  from_walletable_id: number;
}

/**
 * Journal represents a journal entry (仕訳) in freee accounting API
 */
export interface Journal {
  id: number;
  company_id: number;
  issue_date: string; // YYYY-MM-DD
  details: JournalDetail[];
  created_at: string;
  updated_at: string;
}

/**
 * JournalDetail represents a single line in a journal entry
 */
export interface JournalDetail {
  id: number;
  entry_type: 'debit' | 'credit';
  account_item_id: number;
  account_item_name: string;
  tax_code: number;
  partner_id?: number;
  partner_code?: string;
  amount: number;
  vat: number;
  description?: string;
  item_id?: number;
  item_name?: string;
  section_id?: number;
  section_name?: string;
  tag_ids?: number[];
  tag_names?: string[];
  segment_1_tag_id?: number;
  segment_1_tag_name?: string;
  segment_2_tag_id?: number;
  segment_2_tag_name?: string;
  segment_3_tag_id?: number;
  segment_3_tag_name?: string;
}

/**
 * WalletTxn represents a wallet transaction (明細) in freee accounting API
 */
export interface WalletTxn {
  id: number;
  company_id: number;
  date: string; // YYYY-MM-DD
  amount: number;
  balance?: number;
  entry_side: 'income' | 'expense';
  walletable_type: 'bank_account' | 'credit_card';
  walletable_id: number;
  description: string;
  status: 'settled' | 'unbooked' | 'passed';
  deal_id?: number;
  deal_balance?: number;
  created_at: string;
  updated_at: string;
}

/**
 * API List Response wrapper
 */
export interface DealsResponse {
  deals: Deal[];
}

export interface JournalsResponse {
  journals: Journal[];
}

export interface WalletTxnsResponse {
  wallet_txns: WalletTxn[];
}

/**
 * OAuth2 Token Response
 */
export interface TokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
}

/**
 * API Error Response
 */
export interface ErrorResponse {
  error: string;
  error_description?: string;
}
