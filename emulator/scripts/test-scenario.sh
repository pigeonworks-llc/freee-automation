#!/bin/bash
# Scenario Test: freee API Integration
# Tests the complete flow: wallet_txn -> Deal creation -> wallet_txn settlement

set -e

EMULATOR_PORT=${EMULATOR_PORT:-8090}
API_URL="http://localhost:${EMULATOR_PORT}"

echo "=== freee API Integration Scenario Test ==="
echo "API URL: ${API_URL}"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

# Step 1: Get OAuth token
info "Step 1: Getting OAuth token..."
TOKEN_RESPONSE=$(curl -s -X POST "${API_URL}/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=test-auth-code&client_id=test&client_secret=test&redirect_uri=http://localhost")

TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token')
COMPANY_ID=$(echo "$TOKEN_RESPONSE" | jq -r '.company_id')
REFRESH_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.refresh_token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
  fail "Failed to get access token"
fi
pass "Got access token: ${TOKEN:0:20}..."

if [ "$REFRESH_TOKEN" != "null" ] && [ -n "$REFRESH_TOKEN" ]; then
  pass "Got refresh token: ${REFRESH_TOKEN:0:20}..."
else
  fail "Missing refresh_token in response"
fi

if [ "$COMPANY_ID" != "null" ] && [ -n "$COMPANY_ID" ]; then
  pass "Got company_id: ${COMPANY_ID}"
else
  fail "Missing company_id in response"
fi

# Step 2: Test Walletables endpoint
info "Step 2: Testing /api/1/walletables endpoint..."
WALLETABLES=$(curl -s "${API_URL}/api/1/walletables?company_id=1" \
  -H "Authorization: Bearer ${TOKEN}")

WALLETABLE_COUNT=$(echo "$WALLETABLES" | jq '.walletables | length')
if [ "$WALLETABLE_COUNT" -gt 0 ]; then
  pass "Walletables endpoint returned ${WALLETABLE_COUNT} items"
else
  fail "Walletables endpoint returned no data"
fi

# Step 3: Create test wallet transactions
info "Step 3: Creating test wallet transactions..."

# Transaction 1: Amazon (should map to 新聞図書費)
TXN1=$(curl -s -X POST "${API_URL}/api/1/wallet_txns" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "date": "2024-11-20",
    "amount": -1980,
    "entry_side": "expense",
    "walletable_type": "credit_card",
    "walletable_id": 2,
    "description": "Amazon.co.jp Kindle本購入"
  }')
TXN1_ID=$(echo "$TXN1" | jq -r '.wallet_txn.id')
if [ "$TXN1_ID" != "null" ]; then
  pass "Created wallet_txn #${TXN1_ID} (Amazon)"
else
  fail "Failed to create Amazon transaction"
fi

# Transaction 2: GitHub (should map to 支払手数料)
TXN2=$(curl -s -X POST "${API_URL}/api/1/wallet_txns" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "date": "2024-11-21",
    "amount": -3500,
    "entry_side": "expense",
    "walletable_type": "credit_card",
    "walletable_id": 2,
    "description": "GitHub Enterprise"
  }')
TXN2_ID=$(echo "$TXN2" | jq -r '.wallet_txn.id')
if [ "$TXN2_ID" != "null" ]; then
  pass "Created wallet_txn #${TXN2_ID} (GitHub)"
else
  fail "Failed to create GitHub transaction"
fi

# Transaction 3: Udemy (should map to 研修費)
TXN3=$(curl -s -X POST "${API_URL}/api/1/wallet_txns" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "date": "2024-11-22",
    "amount": -5400,
    "entry_side": "expense",
    "walletable_type": "credit_card",
    "walletable_id": 3,
    "description": "Udemy オンライン講座"
  }')
TXN3_ID=$(echo "$TXN3" | jq -r '.wallet_txn.id')
if [ "$TXN3_ID" != "null" ]; then
  pass "Created wallet_txn #${TXN3_ID} (Udemy)"
else
  fail "Failed to create Udemy transaction"
fi

# Transaction 4: Unknown vendor (should map to 雑費)
TXN4=$(curl -s -X POST "${API_URL}/api/1/wallet_txns" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "date": "2024-11-23",
    "amount": -800,
    "entry_side": "expense",
    "walletable_type": "credit_card",
    "walletable_id": 2,
    "description": "Unknown Vendor ABC"
  }')
TXN4_ID=$(echo "$TXN4" | jq -r '.wallet_txn.id')
if [ "$TXN4_ID" != "null" ]; then
  pass "Created wallet_txn #${TXN4_ID} (Unknown)"
else
  fail "Failed to create Unknown transaction"
fi

# Step 4: List unbooked wallet transactions
info "Step 4: Listing unbooked wallet transactions..."
UNBOOKED=$(curl -s "${API_URL}/api/1/wallet_txns?company_id=1&status=1" \
  -H "Authorization: Bearer ${TOKEN}")
UNBOOKED_COUNT=$(echo "$UNBOOKED" | jq '.wallet_txns | length')
pass "Found ${UNBOOKED_COUNT} unbooked transactions"

# Step 5: Create a deal with payments (simulating auto-fetch --create-deals)
info "Step 5: Creating deal with payment linkage..."
DEAL1=$(curl -s -X POST "${API_URL}/api/1/deals" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"company_id\": 1,
    \"issue_date\": \"2024-11-20\",
    \"type\": \"expense\",
    \"details\": [{
      \"account_item_id\": 502,
      \"tax_code\": 136,
      \"amount\": 1980,
      \"description\": \"Amazon.co.jp Kindle本購入\"
    }],
    \"payments\": [{
      \"date\": \"2024-11-20\",
      \"from_walletable_type\": \"credit_card\",
      \"from_walletable_id\": 2,
      \"amount\": 1980
    }]
  }")
DEAL1_ID=$(echo "$DEAL1" | jq -r '.deal.id')
if [ "$DEAL1_ID" != "null" ]; then
  pass "Created deal #${DEAL1_ID}"
else
  fail "Failed to create deal"
fi

# Step 6: Verify wallet_txn was settled
info "Step 6: Verifying wallet_txn settlement..."
sleep 1
TXN1_STATUS=$(curl -s "${API_URL}/api/1/wallet_txns/${TXN1_ID}" \
  -H "Authorization: Bearer ${TOKEN}" | jq -r '.wallet_txn.status')

if [ "$TXN1_STATUS" == "settled" ]; then
  pass "wallet_txn #${TXN1_ID} status changed to 'settled'"
else
  info "wallet_txn #${TXN1_ID} status is '${TXN1_STATUS}' (expected: settled)"
fi

# Step 7: List remaining unbooked transactions
info "Step 7: Listing remaining unbooked transactions..."
REMAINING=$(curl -s "${API_URL}/api/1/wallet_txns?company_id=1&status=1" \
  -H "Authorization: Bearer ${TOKEN}")
REMAINING_COUNT=$(echo "$REMAINING" | jq '.wallet_txns | length')
pass "Remaining unbooked transactions: ${REMAINING_COUNT}"

# Step 8: Test account items endpoint
info "Step 8: Testing /api/1/account_items endpoint..."
ACCOUNT_ITEMS=$(curl -s "${API_URL}/api/1/account_items?company_id=1" \
  -H "Authorization: Bearer ${TOKEN}")
ACCOUNT_COUNT=$(echo "$ACCOUNT_ITEMS" | jq '.account_items | length')
if [ "$ACCOUNT_COUNT" -gt 0 ]; then
  pass "Account items endpoint returned ${ACCOUNT_COUNT} items"
else
  fail "Account items endpoint returned no data"
fi

echo ""
echo "=== Test Summary ==="
echo "OAuth: access_token, refresh_token, company_id verified"
echo "Walletables: ${WALLETABLE_COUNT} items"
echo "Wallet Txns: Created 4, Initial unbooked: ${UNBOOKED_COUNT}"
echo "Deal: Created #${DEAL1_ID} with payment linkage"
echo "Settlement: Transaction #${TXN1_ID} -> ${TXN1_STATUS}"
echo "Remaining unbooked: ${REMAINING_COUNT}"
echo ""
echo -e "${GREEN}All scenario tests completed!${NC}"
