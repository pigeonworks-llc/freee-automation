#!/usr/bin/env node

/**
 * Auto Download Amazon Receipts CLI
 * Fetches unregistered Amazon transactions from freee and downloads matching receipts
 */

import * as path from 'path';
import dayjs from 'dayjs';
import { loadConfig } from '@accounting-system/shared';
import { FreeeUploader, AmazonTransaction } from './freee-uploader';
import { AmazonScraper, OrderInfo } from './amazon-scraper';

// Load configuration from root .env file
const rootEnvPath = path.resolve(__dirname, '../../.env');
const config = loadConfig(rootEnvPath);

interface Args {
  dryRun: boolean;
  headless: boolean;
  cardFilter?: string;
  days: number;
}

function parseArgs(): Args {
  const args = process.argv.slice(2);
  let dryRun = false;
  let headless = false;
  let cardFilter: string | undefined;
  let days = 30;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '--dry-run') {
      dryRun = true;
    } else if (arg === '--headless') {
      headless = true;
    } else if ((arg === '--card' || arg === '-c') && args[i + 1]) {
      cardFilter = args[++i];
    } else if ((arg === '--days' || arg === '-d') && args[i + 1]) {
      days = parseInt(args[++i], 10) || 30;
    } else if (arg === '--help' || arg === '-h') {
      printHelp();
      process.exit(0);
    }
  }

  return { dryRun, headless, cardFilter, days };
}

function printHelp(): void {
  console.log(`
Auto Download Amazon Receipts

Fetches unregistered Amazon transactions from freee and downloads matching receipts.

Usage: npm run auto-download [options]

Options:
  --days, -d <N>         Look back N days for transactions (default: 30)
  --card, -c <PATTERN>   Filter by credit card (e.g., "Visa", "1234")
  --dry-run              Preview without downloading
  --headless             Run browser in headless mode
  -h, --help             Show this help

Examples:
  npm run auto-download
  npm run auto-download --days 60
  npm run auto-download --card "Mastercard" --dry-run
`);
}

/**
 * Find Amazon order matching a freee transaction by date and amount
 */
function findMatchingOrders(
  transaction: AmazonTransaction,
  amazonOrders: OrderInfo[],
  toleranceDays: number = 3
): OrderInfo[] {
  const txnDate = dayjs(transaction.date);
  const txnAmount = transaction.amount;

  return amazonOrders.filter(order => {
    const orderDate = dayjs(order.orderDate);
    const daysDiff = Math.abs(txnDate.diff(orderDate, 'day'));

    // Match if date is within tolerance and amount matches exactly
    if (daysDiff <= toleranceDays && order.amount === txnAmount) {
      return true;
    }

    return false;
  });
}

async function main(): Promise<void> {
  const { dryRun, headless, cardFilter, days } = parseArgs();

  console.log('=== Auto Download Amazon Receipts ===');
  console.log('');

  // Validate configuration
  if (!config.freee.apiUrl || !config.freee.accessToken || !config.freee.companyId) {
    console.error('Error: freee API configuration missing in .env');
    console.error('Required: FREEE_API_URL, FREEE_ACCESS_TOKEN, FREEE_COMPANY_ID');
    process.exit(1);
  }

  if (!config.amazon?.downloadDir) {
    console.error('Error: AMAZON_DOWNLOAD_DIR not configured in .env');
    process.exit(1);
  }

  console.log(`Looking back ${days} days for unregistered transactions`);
  if (cardFilter) {
    console.log(`Card filter: "${cardFilter}"`);
  }
  if (dryRun) {
    console.log('Mode: DRY RUN (no downloads)');
  }
  console.log('');

  // Initialize freee client
  const freeeClient = new FreeeUploader(
    config.freee.apiUrl,
    config.freee.accessToken,
    parseInt(config.freee.companyId, 10)
  );

  // Fetch unregistered transactions
  console.log('Fetching unregistered transactions from freee...');
  let transactions;
  try {
    transactions = await freeeClient.fetchUnregisteredTransactions();
  } catch (error: any) {
    console.error(`Failed to fetch transactions: ${error.message}`);
    process.exit(1);
  }

  console.log(`Found ${transactions.length} unregistered transactions`);

  // Filter for Amazon transactions
  const amazonTxns = freeeClient.filterAmazonTransactions(transactions);
  console.log(`Found ${amazonTxns.length} Amazon transactions`);

  if (amazonTxns.length === 0) {
    console.log('No Amazon transactions to process');
    return;
  }

  // Display Amazon transactions
  console.log('');
  console.log('Amazon transactions:');
  for (const txn of amazonTxns) {
    console.log(`  ${txn.date}  ¥${txn.amount.toLocaleString()}  ${txn.description}`);
  }
  console.log('');

  // Calculate date range from transactions
  const dates = amazonTxns.map(t => t.date).sort();
  const minDate = dayjs(dates[0]).subtract(3, 'day').format('YYYY-MM-DD');
  const maxDate = dayjs(dates[dates.length - 1]).add(3, 'day').format('YYYY-MM-DD');

  console.log(`Searching Amazon orders from ${minDate} to ${maxDate}...`);
  console.log('');

  // Initialize Amazon scraper
  const userDataDir = config.amazon.userDataDir || path.resolve(__dirname, '../../playwright-data');
  const scraper = new AmazonScraper({
    userDataDir,
    downloadDir: config.amazon.downloadDir,
    headless,
  });

  try {
    await scraper.initialize();

    // Get Amazon orders in the date range
    const page = await (scraper as any).context.newPage();
    const loggedIn = await scraper.ensureLoggedIn(page);
    if (!loggedIn) {
      throw new Error('Failed to log in to Amazon');
    }

    const orders = await scraper.getOrdersInRange(page, minDate, maxDate);
    console.log(`Found ${orders.length} Amazon orders`);

    // Display Amazon orders for debugging
    if (orders.length > 0) {
      console.log('');
      console.log('Amazon orders:');
      for (const order of orders) {
        console.log(`  ${order.orderDate}  ¥${order.amount.toLocaleString()}  ${order.orderId}`);
      }
    }

    // If card filter is specified, fetch payment info
    if (cardFilter && orders.length > 0) {
      console.log(`Fetching payment info to filter by card "${cardFilter}"...`);
      for (let i = 0; i < orders.length; i++) {
        const order = orders[i];
        console.log(`  [${i + 1}/${orders.length}] Checking ${order.orderId}...`);
        const paymentMethod = await scraper.getPaymentMethodFromDetailPage(page, order.orderDetailUrl);
        order.paymentMethod = paymentMethod;
        if (paymentMethod) {
          console.log(`    Payment: ${paymentMethod}`);
        }
        await new Promise(r => setTimeout(r, 1500));
      }
    }

    // Match transactions to orders
    console.log('');
    console.log('Matching transactions to orders...');
    const toDownload: OrderInfo[] = [];
    const matched: Map<number, OrderInfo[]> = new Map();

    for (const txn of amazonTxns) {
      const matches = findMatchingOrders(txn, orders);

      // Apply card filter if specified
      let filteredMatches = matches;
      if (cardFilter) {
        filteredMatches = matches.filter(o =>
          o.paymentMethod?.toLowerCase().includes(cardFilter.toLowerCase())
        );
      }

      if (filteredMatches.length > 0) {
        console.log(`  ${txn.date} ¥${txn.amount.toLocaleString()} -> ${filteredMatches.length} match(es)`);
        for (const m of filteredMatches) {
          console.log(`    - ${m.orderId} (${m.orderDate})`);
          if (!toDownload.some(o => o.orderId === m.orderId)) {
            toDownload.push(m);
          }
        }
        matched.set(txn.id, filteredMatches);
      } else {
        console.log(`  ${txn.date} ¥${txn.amount.toLocaleString()} -> No match found`);
      }
    }

    console.log('');
    console.log(`${toDownload.length} unique receipts to download`);

    if (toDownload.length === 0) {
      console.log('No receipts to download');
      await page.close();
      return;
    }

    // Download receipts
    if (dryRun) {
      console.log('');
      console.log('[DRY RUN] Would download:');
      for (const order of toDownload) {
        console.log(`  - ${order.orderId} (${order.orderDate})`);
      }
    } else {
      console.log('');
      console.log('Downloading receipts...');
      for (let i = 0; i < toDownload.length; i++) {
        const order = toDownload[i];
        console.log(`[${i + 1}/${toDownload.length}] ${order.orderId} (${order.orderDate})`);
        const result = await scraper.downloadReceipt(page, order);
        if (result.status === 'downloaded') {
          console.log(`    Downloaded: ${result.filePath}`);
        } else if (result.status === 'skipped') {
          console.log(`    Skipped (already exists)`);
        } else {
          console.log(`    Error: ${result.error}`);
        }
        await new Promise(r => setTimeout(r, 2000));
      }
    }

    await page.close();

    // Summary
    console.log('');
    console.log('=== Summary ===');
    console.log(`Unregistered Amazon transactions: ${amazonTxns.length}`);
    console.log(`Matched to orders: ${matched.size}`);
    console.log(`Receipts downloaded: ${toDownload.length}`);
    console.log('');
    console.log(`Receipt PDFs are saved in: ${config.amazon.downloadDir}`);

  } finally {
    await scraper.close();
  }
}

main().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
});
