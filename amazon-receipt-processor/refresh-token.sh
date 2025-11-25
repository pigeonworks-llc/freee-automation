#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TOKEN_FILE="$SCRIPT_DIR/../freee-token.json"
CLIENT_ID="634466446256351"
CLIENT_SECRET="Fhsy-u0VR2aqZJcTfynFjM7fqjlB2Z6BPE02oeoulC3Zg2WS617Uo9vqLnEtQKTbSjMQgcLRcwz-23KwBACmaw"

REFRESH_TOKEN=$(cat "$TOKEN_FILE" | jq -r '.refresh_token')

echo "Refreshing token..."

RESPONSE=$(curl -s -X POST "https://accounts.secure.freee.co.jp/public_api/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token" \
  -d "client_id=$CLIENT_ID" \
  -d "client_secret=$CLIENT_SECRET" \
  -d "refresh_token=$REFRESH_TOKEN")

echo "$RESPONSE" | jq '.'

# Check for error
if echo "$RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
  echo "Error refreshing token"
  exit 1
fi

# Extract new values
ACCESS_TOKEN=$(echo "$RESPONSE" | jq -r '.access_token')
NEW_REFRESH_TOKEN=$(echo "$RESPONSE" | jq -r '.refresh_token')
EXPIRES_IN=$(echo "$RESPONSE" | jq -r '.expires_in')
COMPANY_ID=$(echo "$RESPONSE" | jq -r '.company_id')

# Calculate expires_at
EXPIRES_AT=$(($(date +%s) + EXPIRES_IN))

# Save to file
cat > "$TOKEN_FILE" << EOF
{
  "access_token": "$ACCESS_TOKEN",
  "refresh_token": "$NEW_REFRESH_TOKEN",
  "expires_at": $EXPIRES_AT,
  "company_id": $COMPANY_ID
}
EOF

echo ""
echo "Token refreshed. Expires at: $(date -r $EXPIRES_AT)"
