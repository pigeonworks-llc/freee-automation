/**
 * freee Accounting API Client
 */

import axios, { AxiosInstance, AxiosError } from 'axios';
import {
  Deal,
  DealsResponse,
  Journal,
  JournalsResponse,
  WalletTxn,
  WalletTxnsResponse,
  TokenResponse,
  ErrorResponse,
} from './types';

export interface FreeeClientConfig {
  apiUrl: string;
  clientId?: string;
  clientSecret?: string;
  accessToken?: string;
  companyId: number;
}

export interface ListDealsParams {
  company_id: number;
  issue_date_from?: string;
  issue_date_to?: string;
  limit?: number;
  offset?: number;
}

export interface ListJournalsParams {
  company_id: number;
  issue_date_from?: string;
  issue_date_to?: string;
  limit?: number;
  offset?: number;
}

export interface ListWalletTxnsParams {
  company_id: number;
  date_from?: string;
  date_to?: string;
  walletable_type?: 'bank_account' | 'credit_card';
  walletable_id?: number;
  limit?: number;
  offset?: number;
}

export class FreeeClient {
  private client: AxiosInstance;
  private accessToken: string;
  private companyId: number;

  constructor(config: FreeeClientConfig) {
    this.companyId = config.companyId;
    this.accessToken = config.accessToken || '';

    this.client = axios.create({
      baseURL: config.apiUrl,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Request interceptor to add authorization header
    this.client.interceptors.request.use((config) => {
      if (this.accessToken) {
        config.headers.Authorization = `Bearer ${this.accessToken}`;
      }
      return config;
    });

    // Response interceptor for error handling
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError<ErrorResponse>) => {
        if (error.response) {
          const { error: errCode, error_description } = error.response.data;
          throw new Error(
            `freee API Error: ${errCode}${error_description ? ` - ${error_description}` : ''}`
          );
        }
        throw error;
      }
    );
  }

  /**
   * Obtain OAuth2 access token
   */
  async getAccessToken(clientId: string, clientSecret: string): Promise<string> {
    const response = await axios.post<TokenResponse>(
      `${this.client.defaults.baseURL}/oauth/token`,
      new URLSearchParams({
        grant_type: 'client_credentials',
        client_id: clientId,
        client_secret: clientSecret,
      }),
      {
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
        },
      }
    );

    this.accessToken = response.data.access_token;
    return this.accessToken;
  }

  /**
   * Set access token manually
   */
  setAccessToken(token: string): void {
    this.accessToken = token;
  }

  /**
   * List deals (取引一覧)
   */
  async listDeals(params?: Partial<ListDealsParams>): Promise<Deal[]> {
    const queryParams: ListDealsParams = {
      company_id: this.companyId,
      ...params,
    };

    const response = await this.client.get<DealsResponse>('/api/1/deals', {
      params: queryParams,
    });

    return response.data.deals;
  }

  /**
   * Get a single deal by ID
   */
  async getDeal(dealId: number): Promise<Deal> {
    const response = await this.client.get<{ deal: Deal }>(`/api/1/deals/${dealId}`, {
      params: { company_id: this.companyId },
    });

    return response.data.deal;
  }

  /**
   * List journals (仕訳一覧)
   */
  async listJournals(params?: Partial<ListJournalsParams>): Promise<Journal[]> {
    const queryParams: ListJournalsParams = {
      company_id: this.companyId,
      ...params,
    };

    const response = await this.client.get<JournalsResponse>('/api/1/journals', {
      params: queryParams,
    });

    return response.data.journals;
  }

  /**
   * Get a single journal by ID
   */
  async getJournal(journalId: number): Promise<Journal> {
    const response = await this.client.get<{ journal: Journal }>(
      `/api/1/journals/${journalId}`,
      {
        params: { company_id: this.companyId },
      }
    );

    return response.data.journal;
  }

  /**
   * List wallet transactions (口座明細一覧)
   */
  async listWalletTxns(params?: Partial<ListWalletTxnsParams>): Promise<WalletTxn[]> {
    const queryParams: ListWalletTxnsParams = {
      company_id: this.companyId,
      ...params,
    };

    const response = await this.client.get<WalletTxnsResponse>('/api/1/wallet_txns', {
      params: queryParams,
    });

    return response.data.wallet_txns;
  }

  /**
   * Get a single wallet transaction by ID
   */
  async getWalletTxn(txnId: number): Promise<WalletTxn> {
    const response = await this.client.get<{ wallet_txn: WalletTxn }>(
      `/api/1/wallet_txns/${txnId}`,
      {
        params: { company_id: this.companyId },
      }
    );

    return response.data.wallet_txn;
  }

  /**
   * Fetch all deals in a date range (with pagination)
   */
  async fetchAllDeals(dateFrom: string, dateTo: string): Promise<Deal[]> {
    const allDeals: Deal[] = [];
    let offset = 0;
    const limit = 100;

    while (true) {
      const deals = await this.listDeals({
        issue_date_from: dateFrom,
        issue_date_to: dateTo,
        limit,
        offset,
      });

      if (deals.length === 0) {
        break;
      }

      allDeals.push(...deals);

      if (deals.length < limit) {
        break;
      }

      offset += limit;
    }

    return allDeals;
  }

  /**
   * Fetch all journals in a date range (with pagination)
   */
  async fetchAllJournals(dateFrom: string, dateTo: string): Promise<Journal[]> {
    const allJournals: Journal[] = [];
    let offset = 0;
    const limit = 100;

    while (true) {
      const journals = await this.listJournals({
        issue_date_from: dateFrom,
        issue_date_to: dateTo,
        limit,
        offset,
      });

      if (journals.length === 0) {
        break;
      }

      allJournals.push(...journals);

      if (journals.length < limit) {
        break;
      }

      offset += limit;
    }

    return allJournals;
  }
}
