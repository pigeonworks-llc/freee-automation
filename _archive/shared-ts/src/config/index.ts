/**
 * Configuration management
 * Loads configuration from environment variables
 */

import * as path from 'path';
import * as dotenv from 'dotenv';

/**
 * Application configuration interface
 */
export interface AppConfig {
  // freee API
  freee: {
    clientId?: string;
    clientSecret?: string;
    redirectUri?: string;
    accessToken?: string;
    companyId?: string;
    apiUrl: string;
  };

  // Beancount paths
  beancount: {
    root: string;
    dbPath?: string;
    attachmentsDir?: string;
  };

  // Amazon receipt processor
  amazon?: {
    downloadDir: string;
    pdfPattern: string;
    userDataDir?: string;
  };

  // Development
  debug: boolean;
  nodeEnv: string;
}

/**
 * Load configuration from environment variables
 * Automatically loads .env file from project root if available
 */
export function loadConfig(envPath?: string): AppConfig {
  // Load .env file from project root by default
  const defaultEnvPath = path.resolve(process.cwd(), '.env');
  dotenv.config({ path: envPath || defaultEnvPath });

  return {
    freee: {
      clientId: process.env.FREEE_CLIENT_ID,
      clientSecret: process.env.FREEE_CLIENT_SECRET,
      redirectUri: process.env.FREEE_REDIRECT_URI,
      accessToken: process.env.FREEE_ACCESS_TOKEN,
      companyId: process.env.FREEE_COMPANY_ID,
      apiUrl: process.env.FREEE_API_URL || 'http://localhost:8080',
    },
    beancount: {
      root: process.env.BEANCOUNT_ROOT || './beancount',
      dbPath: process.env.BEANCOUNT_DB_PATH,
      attachmentsDir: process.env.BEANCOUNT_ATTACHMENTS_DIR,
    },
    amazon: process.env.AMAZON_DOWNLOAD_DIR ? {
      downloadDir: process.env.AMAZON_DOWNLOAD_DIR,
      pdfPattern: process.env.AMAZON_PDF_PATTERN || '領収書*.pdf',
      userDataDir: process.env.AMAZON_USER_DATA_DIR,
    } : undefined,
    debug: process.env.DEBUG === 'true',
    nodeEnv: process.env.NODE_ENV || 'development',
  };
}

/**
 * Validate required configuration fields
 */
export function validateConfig(config: AppConfig, required: string[][]): void {
  const missing: string[] = [];

  for (const path of required) {
    let value: any = config;
    for (const key of path) {
      value = value?.[key];
    }
    if (value === undefined || value === '') {
      missing.push(path.join('.'));
    }
  }

  if (missing.length > 0) {
    throw new Error(
      `Missing required configuration: ${missing.join(', ')}\n` +
      'Please check your .env file or environment variables.'
    );
  }
}
