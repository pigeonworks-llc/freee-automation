/**
 * Test script for rule-based deal creation
 */
import { FreeeUploader, WalletTransaction } from './src/freee-uploader';
import { findMatchingRule } from './src/account-rules';
import * as path from 'path';

// Load token
const tokenPath = path.resolve(__dirname, '../freee-token.json');
const token = require(tokenPath);

const FREEE_API_URL = 'https://api.freee.co.jp';

// Receipt IDs from earlier uploads
const RECEIPT_IDS: { [key: string]: number } = {
  // Key format: date_amount
  '2025-11-21_6000': 381862867,  // NOMAD 11/21
  '2025-11-22_6000': 381862933,  // NOMAD 11/22
};

async function main() {
  console.log('=== Test Rule-Based Deal Creation ===\n');

  const uploader = new FreeeUploader(
    FREEE_API_URL,
    token.access_token,
    token.company_id
  );

  // Step 1: Fetch account items to verify access
  console.log('1. Fetching account items...');
  try {
    const items = await uploader.fetchAccountItems();
    console.log(`   Found ${items.length} account items`);

    // Show relevant ones
    const relevantNames = ['新聞図書費', '研修費', '消耗品費', '通信費'];
    for (const name of relevantNames) {
      const item = items.find(i => i.name === name);
      if (item) {
        console.log(`   - ${item.name}: ID=${item.id}`);
      }
    }
  } catch (error: any) {
    console.error(`   Error: ${error.message}`);
    return;
  }

  // Step 2: Fetch unregistered transactions
  console.log('\n2. Fetching unregistered transactions...');
  let transactions: WalletTransaction[];
  try {
    transactions = await uploader.fetchUnregisteredTransactions();
    console.log(`   Found ${transactions.length} unregistered transactions`);
  } catch (error: any) {
    console.error(`   Error: ${error.message}`);
    return;
  }

  // Step 3: Find matching rules
  console.log('\n3. Matching rules for transactions:');
  const targets: { txn: WalletTransaction; rule: any; receiptId?: number }[] = [];

  for (const txn of transactions) {
    const rule = findMatchingRule(txn.description);
    if (rule) {
      // Match receipt by date_amount key
      const receiptKey = `${txn.date}_${Math.abs(txn.amount)}`;
      const receiptId = RECEIPT_IDS[receiptKey];

      console.log(`   ✓ ${txn.date} ¥${txn.amount} "${txn.description.substring(0, 30)}"`);
      console.log(`     → ${rule.accountName} (tax: ${rule.taxCode})${receiptId ? ` [Receipt: ${receiptId}]` : ''}`);

      targets.push({ txn, rule, receiptId });
    } else {
      console.log(`   ✗ ${txn.date} ¥${txn.amount} "${txn.description.substring(0, 30)}" - No rule`);
    }
  }

  if (targets.length === 0) {
    console.log('\nNo transactions matched rules.');
    return;
  }

  // Step 4: Ask for confirmation
  console.log(`\n4. Ready to create ${targets.length} deals.`);
  console.log('   Run with --execute to create deals.');

  const shouldExecute = process.argv.includes('--execute');
  if (!shouldExecute) {
    console.log('\n[DRY RUN] No deals created.');
    return;
  }

  // Step 5: Create deals
  console.log('\n5. Creating deals...');
  for (const { txn, receiptId } of targets) {
    console.log(`   Creating deal for ¥${txn.amount}...`);
    const result = await uploader.createDealFromTransaction(txn, receiptId);

    if (result.success) {
      console.log(`   ✓ Deal created: ID=${result.dealId}`);
    } else {
      console.log(`   ✗ Error: ${result.error}`);
    }

    // Wait between requests
    await new Promise(r => setTimeout(r, 2000));
  }

  console.log('\n=== Done ===');
}

main().catch(console.error);
