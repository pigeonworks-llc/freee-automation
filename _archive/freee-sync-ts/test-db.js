// Quick test script for SQLite integration
const { createConnection } = require('./dist/db/connection');
const { SyncHistory } = require('./dist/db/sync-history');

async function test() {
  console.log('Testing SQLite integration...\n');

  try {
    // Create connection
    const conn = createConnection();
    console.log('✓ Database connection created');
    console.log(`  Path: ${conn.getPath()}\n`);

    // Create sync history
    const syncHistory = new SyncHistory(conn);

    // Get stats
    const stats = await syncHistory.getStats();
    console.log('✓ Retrieved sync statistics:');
    console.log(`  Total deals: ${stats.total_deals}`);
    console.log(`  Total journals: ${stats.total_journals}`);
    console.log(`  Total documents: ${stats.total_documents}`);
    console.log(`  Last sync: ${stats.last_sync || 'Never'}\n`);

    // Get metadata
    const version = await syncHistory.getMetadata('schema_version');
    const createdAt = await syncHistory.getMetadata('created_at');
    console.log('✓ Database metadata:');
    console.log(`  Schema version: ${version}`);
    console.log(`  Created at: ${createdAt}\n`);

    // Close connection
    conn.close();
    console.log('✓ Database connection closed');

    console.log('\n✅ All tests passed!');
  } catch (error) {
    console.error('❌ Test failed:', error.message);
    process.exit(1);
  }
}

test();
