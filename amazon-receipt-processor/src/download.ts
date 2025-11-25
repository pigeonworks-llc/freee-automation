#!/usr/bin/env node

/**
 * Amazon Receipt Download CLI
 * Downloads receipt PDFs from Amazon order history using Playwright
 */

import * as path from 'path';
import { loadConfig } from '@accounting-system/shared';
import { AmazonScraper } from './amazon-scraper';
import dayjs from 'dayjs';

// Load configuration from root .env file
const rootEnvPath = path.resolve(__dirname, '../../.env');
const config = loadConfig(rootEnvPath);

interface Args {
  fromDate: string;
  toDate: string;
  dryRun: boolean;
  headless: boolean;
  cardFilter?: string;
}

function parseArgs(): Args {
  const args = process.argv.slice(2);
  let fromDate = '';
  let toDate = '';
  let dryRun = false;
  let headless = false;
  let cardFilter: string | undefined;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '--from' && args[i + 1]) {
      fromDate = args[++i];
    } else if (arg === '--to' && args[i + 1]) {
      toDate = args[++i];
    } else if (arg === '--dry-run') {
      dryRun = true;
    } else if (arg === '--headless') {
      headless = true;
    } else if ((arg === '--card' || arg === '-c') && args[i + 1]) {
      cardFilter = args[++i];
    } else if (arg === '--help' || arg === '-h') {
      printHelp();
      process.exit(0);
    }
  }

  // Default to last 3 months if not specified
  if (!fromDate) {
    fromDate = dayjs().subtract(3, 'month').format('YYYY-MM-DD');
  }
  if (!toDate) {
    toDate = dayjs().format('YYYY-MM-DD');
  }

  return { fromDate, toDate, dryRun, headless, cardFilter };
}

function printHelp(): void {
  console.log(`
Amazon Receipt Download

Usage: npm run download [options]

Options:
  --from YYYY-MM-DD      Start date (default: 3 months ago)
  --to YYYY-MM-DD        End date (default: today)
  --card, -c <PATTERN>   Filter by credit card (e.g., "Visa", "1234", "JCB")
  --dry-run              Preview without downloading
  --headless             Run browser in headless mode (no window)
  -h, --help             Show this help

Examples:
  npm run download --from 2024-01-01 --to 2024-12-31
  npm run download --dry-run
  npm run download --headless --from 2024-06-01
  npm run download --card "Visa" --from 2024-01-01
  npm run download --card "1234" --dry-run
`);
}

async function main(): Promise<void> {
  const { fromDate, toDate, dryRun, headless, cardFilter } = parseArgs();

  console.log('=== Amazon Receipt Download ===');
  console.log('');
  console.log(`Date range: ${fromDate} to ${toDate}`);
  if (cardFilter) {
    console.log(`Card filter: "${cardFilter}"`);
  }
  if (dryRun) {
    console.log('Mode: DRY RUN (no downloads)');
  }
  if (headless) {
    console.log('Mode: Headless (no browser window)');
  }
  console.log('');

  // Validate configuration
  if (!config.amazon?.downloadDir) {
    console.error('Error: AMAZON_DOWNLOAD_DIR not configured in .env');
    process.exit(1);
  }

  // Initialize scraper
  const userDataDir = config.amazon.userDataDir || path.resolve(__dirname, '../../playwright-data');
  const scraper = new AmazonScraper({
    userDataDir,
    downloadDir: config.amazon.downloadDir,
    headless,
  });

  try {
    await scraper.initialize();

    console.log('Downloading receipts...');
    console.log('');

    const results = await scraper.downloadReceipts(fromDate, toDate, dryRun, cardFilter);

    // Summary
    console.log('');
    console.log('=== Summary ===');

    const downloaded = results.filter(r => r.status === 'downloaded').length;
    const skipped = results.filter(r => r.status === 'skipped').length;
    const errors = results.filter(r => r.status === 'error').length;

    console.log(`Total orders: ${results.length}`);
    console.log(`Downloaded: ${downloaded}`);
    console.log(`Skipped (already exists): ${skipped}`);
    console.log(`Errors: ${errors}`);
    console.log('');

    if (errors > 0) {
      console.log('Errors:');
      for (const result of results.filter(r => r.status === 'error')) {
        console.log(`  - Order ${result.orderId}: ${result.error}`);
      }
      console.log('');
    }

    if (downloaded > 0 || skipped > 0) {
      console.log(`Receipt PDFs are saved in: ${config.amazon.downloadDir}`);
      console.log('');
      console.log('Next step: Run "npm run process" to process the receipts');
    }
  } finally {
    await scraper.close();
  }
}

main().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
});
