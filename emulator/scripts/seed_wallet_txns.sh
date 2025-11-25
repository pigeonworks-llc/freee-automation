#!/bin/bash

set -e

API_URL="${API_URL:-http://localhost:8080}"
TOKEN=""

echo "=== Wallet Transactions Seed Script ==="
echo ""

# Get access token
echo "1. Getting access token..."
TOKEN_RESPONSE=$(curl -s -X POST "$API_URL/oauth/token" \
  -d "grant_type=client_credentials")

TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Failed to get access token"
  exit 1
fi

echo "   ✓ Access token obtained"
echo ""

# Create wallet transactions
echo "2. Creating sample wallet transactions..."

# Unbooked transaction 1
curl -s -X POST "$API_URL/api/1/wallet_txns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "date": "2025-01-10",
    "amount": 50000,
    "entry_side": "income",
    "walletable_type": "bank_account",
    "walletable_id": 1,
    "description": "売上入金（未仕訳）"
  }' > /dev/null

echo "   ✓ Created unbooked transaction 1"

# Unbooked transaction 2
curl -s -X POST "$API_URL/api/1/wallet_txns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "date": "2025-01-15",
    "amount": 30000,
    "entry_side": "expense",
    "walletable_type": "bank_account",
    "walletable_id": 1,
    "description": "仕入代金支払（未仕訳）"
  }' > /dev/null

echo "   ✓ Created unbooked transaction 2"

# Unbooked transaction 3
curl -s -X POST "$API_URL/api/1/wallet_txns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "date": "2025-01-20",
    "amount": 10000,
    "entry_side": "expense",
    "walletable_type": "credit_card",
    "walletable_id": 2,
    "description": "経費支払（未仕訳）"
  }' > /dev/null

echo "   ✓ Created unbooked transaction 3"

# Settled transaction (for comparison)
SETTLED_RESP=$(curl -s -X POST "$API_URL/api/1/wallet_txns" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "date": "2025-01-05",
    "amount": 100000,
    "entry_side": "income",
    "walletable_type": "bank_account",
    "walletable_id": 1,
    "description": "売上入金（仕訳済み）"
  }')

SETTLED_ID=$(echo "$SETTLED_RESP" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)

# Mark as settled
curl -s -X PUT "$API_URL/api/1/wallet_txns/$SETTLED_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "settled"
  }' > /dev/null

echo "   ✓ Created settled transaction"

echo ""

# Verify
echo "3. Verifying created data..."

UNBOOKED_COUNT=$(curl -s -X GET "$API_URL/api/1/wallet_txns?company_id=1&status=unbooked" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"id":[0-9]*' | wc -l)

TOTAL_COUNT=$(curl -s -X GET "$API_URL/api/1/wallet_txns?company_id=1" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"id":[0-9]*' | wc -l)

echo "   ✓ Total wallet transactions: $TOTAL_COUNT"
echo "   ✓ Unbooked (未仕訳): $UNBOOKED_COUNT"

echo ""
echo "=== Seed completed! ==="
echo ""
echo "Summary:"
echo "  - Total: $TOTAL_COUNT transactions"
echo "  - Unbooked: $UNBOOKED_COUNT transactions"
echo "  - Access Token: $TOKEN"
echo ""
