#!/bin/bash

set -e

API_URL="${API_URL:-http://localhost:8080}"
TOKEN=""

echo "=== freee API Emulator - Data Seed Script ==="
echo ""

# Get access token
echo "1. Getting access token..."
TOKEN_RESPONSE=$(curl -s -X POST "$API_URL/oauth/token" \
  -d "grant_type=client_credentials")

TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Failed to get access token"
  echo "$TOKEN_RESPONSE"
  exit 1
fi

echo "   ✓ Access token obtained"
echo ""

# Create deals
echo "2. Creating sample deals..."

# Deal 1: Income
DEAL1=$(curl -s -X POST "$API_URL/api/1/deals" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "issue_date": "2025-01-15",
    "type": "income",
    "details": [
      {
        "account_item_id": 400,
        "tax_code": 1,
        "amount": 100000,
        "description": "売上"
      }
    ],
    "ref_number": "INV-2025-001"
  }')

DEAL1_ID=$(echo "$DEAL1" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo "   ✓ Created income deal (ID: $DEAL1_ID)"

# Deal 2: Expense
DEAL2=$(curl -s -X POST "$API_URL/api/1/deals" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "issue_date": "2025-01-20",
    "type": "expense",
    "details": [
      {
        "account_item_id": 801,
        "tax_code": 1,
        "amount": 50000,
        "description": "仕入"
      }
    ],
    "ref_number": "EXP-2025-001"
  }')

DEAL2_ID=$(echo "$DEAL2" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo "   ✓ Created expense deal (ID: $DEAL2_ID)"

# Deal 3: Multiple details
DEAL3=$(curl -s -X POST "$API_URL/api/1/deals" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "issue_date": "2025-01-25",
    "type": "income",
    "details": [
      {
        "account_item_id": 401,
        "tax_code": 1,
        "amount": 30000,
        "description": "コンサルティング売上"
      },
      {
        "account_item_id": 402,
        "tax_code": 1,
        "amount": 20000,
        "description": "システム開発売上"
      }
    ],
    "ref_number": "INV-2025-002"
  }')

DEAL3_ID=$(echo "$DEAL3" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo "   ✓ Created multi-detail deal (ID: $DEAL3_ID)"

echo ""

# Create journals
echo "3. Creating sample journals..."

# Journal 1: Simple entry
JOURNAL1=$(curl -s -X POST "$API_URL/api/1/journals" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "issue_date": "2025-02-01",
    "details": [
      {
        "entry_type": "debit",
        "account_item_id": 135,
        "tax_code": 0,
        "amount": 100000,
        "vat": 0,
        "description": "普通預金"
      },
      {
        "entry_type": "credit",
        "account_item_id": 400,
        "tax_code": 1,
        "amount": 90909,
        "vat": 9091,
        "description": "売上高"
      }
    ]
  }')

JOURNAL1_ID=$(echo "$JOURNAL1" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo "   ✓ Created journal entry (ID: $JOURNAL1_ID)"

# Journal 2: Complex entry
JOURNAL2=$(curl -s -X POST "$API_URL/api/1/journals" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "issue_date": "2025-02-05",
    "details": [
      {
        "entry_type": "debit",
        "account_item_id": 801,
        "tax_code": 1,
        "amount": 45454,
        "vat": 4546,
        "description": "仕入高"
      },
      {
        "entry_type": "debit",
        "account_item_id": 138,
        "tax_code": 0,
        "amount": 5000,
        "vat": 0,
        "description": "支払手数料"
      },
      {
        "entry_type": "credit",
        "account_item_id": 135,
        "tax_code": 0,
        "amount": 55000,
        "vat": 0,
        "description": "普通預金"
      }
    ]
  }')

JOURNAL2_ID=$(echo "$JOURNAL2" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
echo "   ✓ Created complex journal entry (ID: $JOURNAL2_ID)"

echo ""

# Verify data
echo "4. Verifying created data..."

DEALS_COUNT=$(curl -s -X GET "$API_URL/api/1/deals?company_id=1" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"id":[0-9]*' | wc -l)

JOURNALS_COUNT=$(curl -s -X GET "$API_URL/api/1/journals?company_id=1" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"id":[0-9]*' | wc -l)

echo "   ✓ Deals created: $DEALS_COUNT"
echo "   ✓ Journals created: $JOURNALS_COUNT"

echo ""
echo "=== Data seed completed successfully! ==="
echo ""
echo "Summary:"
echo "  - Deals: $DEALS_COUNT"
echo "  - Journals: $JOURNALS_COUNT"
echo "  - Access Token: $TOKEN"
echo ""
