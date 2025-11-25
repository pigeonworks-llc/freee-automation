#!/usr/bin/env node

/**
 * Amazon Receipt Processor CLI
 * Process Amazon receipt PDFs and match with freee deals
 */

import * as path from 'path';
import { glob } from 'glob';
import { loadConfig, validateConfig } from '@accounting-system/shared';
import { parseAmazonReceipt } from './pdf-parser';
import { findMatchingDeals, filterDealsByDateRange } from './matcher';
import { FreeeUploader } from './freee-uploader';
import { BeancountUpdater } from './beancount-updater';

// Load configuration from root .env file
const rootEnvPath = path.resolve(__dirname, '../../.env');
const config = loadConfig(rootEnvPath);

interface ProcessResult {
  total: number;
  processed: number;
  skipped: number;
  errors: number;
}

async function main() {
  const isDryRun = process.argv.includes('--dry-run');

  console.log('=== Amazon Receipt Processor ===');
  console.log('');

  // Validate required configuration
  validateConfig(config, [
    ['freee', 'apiUrl'],
    ['freee', 'accessToken'],
    ['freee', 'companyId'],
    ['beancount', 'root'],
  ]);

  if (!config.amazon) {
    console.error('❌ Amazon configuration is required');
    console.error('   Please set AMAZON_DOWNLOAD_DIR in .env file');
    process.exit(1);
  }

  if (isDryRun) {
    console.log('🔍 DRY RUN MODE - No changes will be made');
    console.log('');
  }

  // Initialize services
  const uploader = new FreeeUploader(
    config.freee.apiUrl,
    config.freee.accessToken!,
    parseInt(config.freee.companyId!, 10)
  );
  const updater = new BeancountUpdater(
    path.resolve(process.cwd(), config.beancount.root)
  );

  // Find PDF files
  console.log(`📁 Searching for PDFs in: ${config.amazon.downloadDir}`);
  const pdfFiles = await glob(path.join(config.amazon.downloadDir, config.amazon.pdfPattern));
  console.log(`   Found ${pdfFiles.length} PDF file(s)`);
  console.log('');

  if (pdfFiles.length === 0) {
    console.log('✅ No PDFs to process');
    return;
  }

  // Fetch deals from freee
  console.log('📡 Fetching deals from freee...');
  const allDeals = await uploader.fetchDeals();
  console.log(`   Retrieved ${allDeals.length} deal(s)`);
  console.log('');

  // Process each PDF
  const result: ProcessResult = {
    total: pdfFiles.length,
    processed: 0,
    skipped: 0,
    errors: 0,
  };

  for (const pdfFile of pdfFiles) {
    console.log(`📄 Processing: ${path.basename(pdfFile)}`);

    try {
      // Parse PDF
      const receiptData = await parseAmazonReceipt(pdfFile);
      if (!receiptData) {
        console.log('   ⚠️  Could not parse PDF, skipping');
        result.skipped++;
        console.log('');
        continue;
      }

      console.log(`   Order: ${receiptData.orderNumber}`);
      console.log(`   Date: ${receiptData.orderDate}`);
      console.log(`   Amount: ¥${receiptData.totalAmount.toLocaleString()}`);

      // Filter deals by date range
      const relevantDeals = filterDealsByDateRange(
        allDeals,
        receiptData.orderDate
      );

      // Find matching deal
      const matchResult = findMatchingDeals(receiptData, relevantDeals);

      if (matchResult.status === 'none') {
        console.log('   ⚠️  No matching deal found, skipping');
        result.skipped++;
        console.log('');
        continue;
      }

      if (matchResult.status === 'multiple') {
        console.log('   ❌ Multiple matching deals found:');
        for (const deal of matchResult.matches) {
          console.log(`      - Deal ID ${deal.id}: ${deal.issue_date} ¥${deal.amount}`);
        }
        console.log('   Manual review required, skipping');
        result.errors++;
        console.log('');
        continue;
      }

      // Unique match found
      const deal = matchResult.matches[0];
      console.log(`   ✓ Matched with Deal ID ${deal.id}`);

      if (isDryRun) {
        console.log('   [DRY RUN] Would process this receipt');
        result.processed++;
        console.log('');
        continue;
      }

      // Move PDF to documents directory
      console.log('   📦 Moving PDF to documents directory...');
      const documentPath = await updater.moveToDocuments(
        pdfFile,
        receiptData.orderNumber,
        receiptData.orderDate
      );
      console.log(`   ✓ Moved to: ${documentPath}`);

      // Upload to freee
      console.log('   ☁️  Uploading to freee...');
      const uploadResult = await uploader.uploadReceipt(
        deal,
        pdfFile,
        receiptData.orderNumber
      );

      if (!uploadResult.success) {
        console.log(`   ❌ Upload failed: ${uploadResult.error}`);
        result.errors++;
        console.log('');
        continue;
      }

      console.log(`   ✓ Uploaded (Receipt ID: ${uploadResult.receiptId})`);

      // Update Beancount
      console.log('   📝 Updating Beancount...');
      const updateResult = await updater.addDocument(deal, documentPath);

      if (!updateResult.success) {
        console.log(`   ❌ Beancount update failed: ${updateResult.error}`);
        result.errors++;
        console.log('');
        continue;
      }

      console.log(`   ✓ Updated: ${updateResult.filePath}`);
      result.processed++;
      console.log('');
    } catch (error: any) {
      console.log(`   ❌ Error: ${error.message || String(error)}`);
      result.errors++;
      console.log('');
    }
  }

  // Summary
  console.log('=== Summary ===');
  console.log(`Total PDFs: ${result.total}`);
  console.log(`✓ Processed: ${result.processed}`);
  console.log(`⚠️  Skipped: ${result.skipped}`);
  console.log(`❌ Errors: ${result.errors}`);
  console.log('');

  if (result.errors > 0) {
    process.exit(1);
  }
}

main().catch((error) => {
  console.error('Fatal error:', error);
  process.exit(1);
});
