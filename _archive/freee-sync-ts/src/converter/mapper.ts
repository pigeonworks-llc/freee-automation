/**
 * Account Mapper
 * Maps freee account names to Beancount account names
 */

import * as fs from 'fs';
import * as path from 'path';
import * as yaml from 'yaml';

export interface AccountMapping {
  freee: string;
  beancount: string;
  type: string;
}

export interface TaxCodeMapping {
  code: string;
  rate: number;
  description: string;
  beancount_account: string | null;
}

export interface AccountMappingConfig {
  assets: {
    current: AccountMapping[];
    fixed: AccountMapping[];
  };
  liabilities: {
    current: AccountMapping[];
    longterm: AccountMapping[];
  };
  equity: AccountMapping[];
  income: AccountMapping[];
  expenses: {
    cogs: AccountMapping[];
    sga: AccountMapping[];
    nonoperating: AccountMapping[];
  };
  tax_codes: TaxCodeMapping[];
}

export class AccountMapper {
  private mappingConfig: AccountMappingConfig;
  private freeeToBeancountMap: Map<string, string>;
  private taxCodeMap: Map<string, TaxCodeMapping>;

  constructor(configPath?: string) {
    const defaultConfigPath = path.join(__dirname, '../../config/account-mapping.yaml');
    const mappingFile = configPath || defaultConfigPath;

    const yamlContent = fs.readFileSync(mappingFile, 'utf-8');
    this.mappingConfig = yaml.parse(yamlContent) as AccountMappingConfig;

    this.freeeToBeancountMap = new Map();
    this.taxCodeMap = new Map();

    this.buildMappingMaps();
  }

  /**
   * Build internal mapping maps from configuration
   */
  private buildMappingMaps(): void {
    // Assets
    this.mappingConfig.assets.current.forEach((mapping) => {
      this.freeeToBeancountMap.set(mapping.freee, mapping.beancount);
    });
    this.mappingConfig.assets.fixed.forEach((mapping) => {
      this.freeeToBeancountMap.set(mapping.freee, mapping.beancount);
    });

    // Liabilities
    this.mappingConfig.liabilities.current.forEach((mapping) => {
      this.freeeToBeancountMap.set(mapping.freee, mapping.beancount);
    });
    this.mappingConfig.liabilities.longterm.forEach((mapping) => {
      this.freeeToBeancountMap.set(mapping.freee, mapping.beancount);
    });

    // Equity
    this.mappingConfig.equity.forEach((mapping) => {
      this.freeeToBeancountMap.set(mapping.freee, mapping.beancount);
    });

    // Income
    this.mappingConfig.income.forEach((mapping) => {
      this.freeeToBeancountMap.set(mapping.freee, mapping.beancount);
    });

    // Expenses
    this.mappingConfig.expenses.cogs.forEach((mapping) => {
      this.freeeToBeancountMap.set(mapping.freee, mapping.beancount);
    });
    this.mappingConfig.expenses.sga.forEach((mapping) => {
      this.freeeToBeancountMap.set(mapping.freee, mapping.beancount);
    });
    this.mappingConfig.expenses.nonoperating.forEach((mapping) => {
      this.freeeToBeancountMap.set(mapping.freee, mapping.beancount);
    });

    // Tax codes
    this.mappingConfig.tax_codes.forEach((taxCode) => {
      this.taxCodeMap.set(taxCode.code, taxCode);
    });
  }

  /**
   * Get Beancount account name from freee account name
   * @param freeeName - freee account name (Japanese)
   * @returns Beancount account name (English) or undefined if not found
   */
  getBeancountAccount(freeeName: string): string | undefined {
    return this.freeeToBeancountMap.get(freeeName);
  }

  /**
   * Get Beancount account name from freee account name (with fallback)
   * @param freeeName - freee account name (Japanese)
   * @param fallback - fallback account name if not found
   * @returns Beancount account name
   */
  getBeancountAccountWithFallback(freeeName: string, fallback: string): string {
    return this.freeeToBeancountMap.get(freeeName) || fallback;
  }

  /**
   * Get tax code mapping information
   * @param taxCode - Tax code string
   * @returns Tax code mapping information or undefined
   */
  getTaxCode(taxCode: string): TaxCodeMapping | undefined {
    return this.taxCodeMap.get(taxCode);
  }

  /**
   * Get tax rate from tax code
   * @param taxCode - Tax code string
   * @returns Tax rate (e.g., 0.10 for 10%) or 0 if not found
   */
  getTaxRate(taxCode: string): number {
    const mapping = this.taxCodeMap.get(taxCode);
    return mapping ? mapping.rate : 0;
  }

  /**
   * Get consumption tax account for a given tax code
   * @param taxCode - Tax code string
   * @returns Beancount tax account or null if exempt/not applicable
   */
  getTaxAccount(taxCode: string): string | null {
    const mapping = this.taxCodeMap.get(taxCode);
    return mapping?.beancount_account || null;
  }

  /**
   * Check if a mapping exists for a freee account
   * @param freeeName - freee account name
   * @returns true if mapping exists
   */
  hasMapping(freeeName: string): boolean {
    return this.freeeToBeancountMap.has(freeeName);
  }

  /**
   * Get all mapped account names
   * @returns Array of [freee, beancount] pairs
   */
  getAllMappings(): Array<[string, string]> {
    return Array.from(this.freeeToBeancountMap.entries());
  }
}
