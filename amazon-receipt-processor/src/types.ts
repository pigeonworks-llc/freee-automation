/**
 * freee API Types
 * Simplified types for receipt processing
 */

export interface Deal {
  id: number;
  company_id: number;
  issue_date: string;
  type: 'income' | 'expense';
  amount: number;
  due_amount?: number;
  partner_id?: number;
  partner_code?: string;
  ref_number?: string;
  details: DealDetail[];
  payments?: Payment[];
}

export interface DealDetail {
  id: number;
  account_item_id: number;
  account_item_name: string;
  tax_code: number;
  amount: number;
  vat: number;
  description?: string;
}

export interface Payment {
  id: number;
  date: string;
  from_walletable_type: string;
  from_walletable_id: number;
  amount: number;
}

export interface DealsResponse {
  deals: Deal[];
}
