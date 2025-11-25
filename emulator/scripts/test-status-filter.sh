#!/bin/bash
set -e

pkill -f "PORT=8096" 2>/dev/null || true
sleep 1

rm -rf ./data/freee.db
go build -o bin/freee-emulator ./cmd/server/
PORT=8096 ./bin/freee-emulator &
sleep 3

TOKEN=$(curl -s -X POST "http://localhost:8096/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=test-auth-code&client_id=test&client_secret=test&redirect_uri=http://localhost" | jq -r '.access_token')

echo "Creating wallet transaction..."
curl -s -X POST "http://localhost:8096/api/1/wallet_txns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"company_id":1,"date":"2024-11-20","amount":-1980,"entry_side":"expense","walletable_type":"credit_card","walletable_id":2,"description":"Test"}' | jq .

echo ""
echo "=== Query with status=1 (should find unbooked) ==="
curl -s "http://localhost:8096/api/1/wallet_txns?company_id=1&status=1" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo ""
echo "=== Query with status=unbooked ==="
curl -s "http://localhost:8096/api/1/wallet_txns?company_id=1&status=unbooked" \
  -H "Authorization: Bearer $TOKEN" | jq .

pkill -f "PORT=8096" 2>/dev/null || true
