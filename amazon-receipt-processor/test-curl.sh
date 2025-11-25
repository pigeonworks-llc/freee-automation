#!/bin/bash
set -e

TOKEN=$(cat ../freee-token.json | jq -r '.access_token')
COMPANY_ID=$(cat ../freee-token.json | jq -r '.company_id')

echo "Uploading Teachable receipt..."
curl -X POST "https://api.freee.co.jp/api/1/receipts" \
  -H "Authorization: Bearer $TOKEN" \
  -F "company_id=$COMPANY_ID" \
  -F "receipt=@../gmail-receipt-fetcher/receipts/2025-11-18_teachable_receipt.pdf" \
  -F "description=Teachable: CYOHN SCHOOL" \
  -F "issue_date=2025-11-18" \
  2>&1

echo ""
echo "Done"
