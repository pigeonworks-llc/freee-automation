const token = require('../freee-token.json');

async function main() {
  const res = await fetch('https://api.freee.co.jp/api/1/wallet_txns?company_id=' + token.company_id + '&limit=20', {
    headers: { 'Authorization': 'Bearer ' + token.access_token }
  });

  const data = await res.json();
  const txns = data.wallet_txns || [];

  // Find our target transactions
  const targets = txns.filter(t =>
    (t.amount === 184 || t.amount === -184) ||
    (t.amount === 39800 || t.amount === -39800)
  );

  console.log('=== Target Transactions ===');
  for (const t of targets) {
    console.log(JSON.stringify({
      id: t.id,
      date: t.date,
      amount: t.amount,
      due_amount: t.due_amount,
      walletable_type: t.walletable_type,
      walletable_id: t.walletable_id,
      description: t.description
    }, null, 2));
  }
}

main().catch(console.error);
