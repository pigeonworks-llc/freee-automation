const fs = require('fs');
const path = require('path');
const FormData = require('form-data');
const token = require('../freee-token.json');

const receipts = [
  {
    name: 'Amazon ¥184',
    path: '/Users/shunichi/Downloads/領収書-D01-0143457-6508244.pdf',
    description: 'Amazon Order: D01-0143457-6508244 (Wild Signs and Star Paths)',
    issueDate: '2025-11-20'
  },
  {
    name: 'Teachable ¥39,800',
    path: '/Users/shunichi/src/localhost/shunichi-ikebuchi/accounting-system/gmail-receipt-fetcher/receipts/2025-11-18_teachable_receipt.pdf',
    description: 'Teachable: CYOHN SCHOOL',
    issueDate: '2025-11-18'
  }
];

async function uploadReceipt(receipt) {
  console.log(`Uploading: ${receipt.name}...`);

  const form = new FormData();
  form.append('company_id', String(token.company_id));
  form.append('receipt', fs.createReadStream(receipt.path));
  form.append('description', receipt.description);
  form.append('issue_date', receipt.issueDate);

  const response = await fetch('https://api.freee.co.jp/api/1/receipts', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token.access_token}`,
      ...form.getHeaders()
    },
    body: form
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`HTTP ${response.status}: ${errorText}`);
  }

  const data = await response.json();
  console.log(`  Receipt ID: ${data.receipt.id}`);
  return data.receipt;
}

async function main() {
  console.log('=== Uploading Receipts to freee ===\n');

  const results = [];
  for (const receipt of receipts) {
    try {
      const result = await uploadReceipt(receipt);
      results.push({ ...receipt, receiptId: result.id, success: true });
    } catch (error) {
      console.error(`  Error: ${error.message}`);
      results.push({ ...receipt, success: false, error: error.message });
    }
    // Wait 2 seconds between uploads
    await new Promise(r => setTimeout(r, 2000));
  }

  console.log('\n=== Results ===');
  for (const r of results) {
    if (r.success) {
      console.log(`✓ ${r.name}: Receipt ID ${r.receiptId}`);
    } else {
      console.log(`✗ ${r.name}: ${r.error}`);
    }
  }
}

main().catch(console.error);
