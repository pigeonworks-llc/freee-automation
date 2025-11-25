/**
 * CLI entry point
 */

import * as path from 'path';
import { loadConfig, validateConfig } from '@accounting-system/shared';
import { FreeeClient } from './freee/client';
import { AccountMapper } from './converter/mapper';
import { SyncOrchestrator } from './sync';

// Load configuration from root .env file
const rootEnvPath = path.resolve(__dirname, '../../.env');
const config = loadConfig(rootEnvPath);

async function handleAuth() {
  console.log('Starting OAuth2 authentication...');

  // Validate required config
  validateConfig(config, [
    ['freee', 'clientId'],
    ['freee', 'clientSecret'],
  ]);

  const client = new FreeeClient({
    apiUrl: config.freee.apiUrl,
    companyId: parseInt(config.freee.companyId || '1', 10),
  });

  try {
    const accessToken = await client.getAccessToken(
      config.freee.clientId!,
      config.freee.clientSecret!
    );
    console.log('Authentication successful!');
    console.log(`Access Token: ${accessToken}`);
    console.log('\nAdd this to your root .env file:');
    console.log(`FREEE_ACCESS_TOKEN=${accessToken}`);
  } catch (error) {
    console.error('Authentication failed:', error);
    process.exit(1);
  }
}

async function handleSync(args: string[]) {
  // Parse arguments
  let dateFrom: string | undefined;
  let dateTo: string | undefined;
  let dryRun = false;

  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--from' && args[i + 1]) {
      dateFrom = args[i + 1];
      i++;
    } else if (args[i] === '--to' && args[i + 1]) {
      dateTo = args[i + 1];
      i++;
    } else if (args[i] === '--dry-run') {
      dryRun = true;
    }
  }

  if (!dateFrom || !dateTo) {
    console.error('Error: --from and --to date arguments are required');
    console.error('Usage: npm run sync -- --from 2024-11-01 --to 2025-10-31 [--dry-run]');
    process.exit(1);
  }

  console.log(`Syncing from ${dateFrom} to ${dateTo}${dryRun ? ' (DRY RUN)' : ''}`);

  await performSync(dateFrom, dateTo, dryRun);
}

async function handleSyncMonthly() {
  // Calculate previous month
  const now = new Date();
  const lastMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1);
  const year = lastMonth.getFullYear();
  const month = (lastMonth.getMonth() + 1).toString().padStart(2, '0');

  const dateFrom = `${year}-${month}-01`;

  // Last day of the month
  const lastDay = new Date(year, lastMonth.getMonth() + 1, 0).getDate();
  const dateTo = `${year}-${month}-${lastDay.toString().padStart(2, '0')}`;

  console.log(`Monthly sync for ${year}-${month}`);
  console.log(`Date range: ${dateFrom} to ${dateTo}`);

  await performSync(dateFrom, dateTo, false);
}

async function performSync(dateFrom: string, dateTo: string, dryRun: boolean) {
  // Validate required config
  validateConfig(config, [
    ['freee', 'accessToken'],
    ['beancount', 'root'],
  ]);

  // Initialize client
  const client = new FreeeClient({
    apiUrl: config.freee.apiUrl,
    accessToken: config.freee.accessToken!,
    companyId: parseInt(config.freee.companyId || '1', 10),
  });

  // Initialize mapper
  const mapper = new AccountMapper();

  // Initialize sync orchestrator
  const orchestrator = new SyncOrchestrator(client, mapper);

  try {
    const result = await orchestrator.sync({
      dateFrom,
      dateTo,
      beancountDir: path.resolve(process.cwd(), config.beancount.root),
      dryRun,
    });

    console.log('\nSync completed successfully!');
    console.log(`Deals: ${result.dealsCount}`);
    console.log(`Journals: ${result.journalsCount}`);
    console.log(`Files written: ${result.filesWritten.length}`);

    if (!dryRun) {
      console.log('\nFiles written:');
      result.filesWritten.forEach((file) => console.log(`  - ${file}`));
    }
  } catch (error) {
    console.error('Sync failed:', error);
    process.exit(1);
  }
}

// Main CLI router
async function main() {
  const command = process.argv[2];

  if (!command) {
    console.error('Usage: ts-node src/cli.ts <command>');
    console.error('Commands:');
    console.error('  auth         - Authenticate with freee OAuth2');
    console.error('  sync         - Sync transactions (requires --from and --to)');
    console.error('  sync:monthly - Sync previous month');
    process.exit(1);
  }

  if (command === 'auth') {
    await handleAuth();
  } else if (command === 'sync') {
    await handleSync(process.argv.slice(3));
  } else if (command === 'sync:monthly') {
    await handleSyncMonthly();
  } else {
    console.error(`Unknown command: ${command}`);
    process.exit(1);
  }
}

// Run CLI
main().catch((error) => {
  console.error('Fatal error:', error);
  process.exit(1);
});
