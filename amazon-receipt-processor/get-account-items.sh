#!/bin/bash
TOKEN=$(cat ../freee-token.json | jq -r '.access_token')
COMPANY_ID=$(cat ../freee-token.json | jq -r '.company_id')

echo "=== Account Items ==="
curl -s "https://api.freee.co.jp/api/1/account_items?company_id=$COMPANY_ID" \
  -H "Authorization: Bearer $TOKEN"
