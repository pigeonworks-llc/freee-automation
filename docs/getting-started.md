# Getting Started

This guide will walk you through setting up the freee-automation monorepo from scratch.

## Prerequisites

### Required Software

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25+ | Build Go binaries |
| Node.js | 18+ | Amazon receipt processor |
| pnpm | Latest | TypeScript workspace management |
| Python | 3.10+ | Beancount |
| Beancount | 3.2.0+ | Accounting engine |
| Fava | 1.30.7+ | Web UI |
| Task | Latest | Task runner (emulator) |
| Git | Latest | Version control |
| curl | Latest | API testing |
| jq | Latest | JSON processing |

### Installation Commands

```bash
# macOS (Homebrew)
brew install go node pnpm python git curl jq
brew install go-task/tap/go-task

# Python packages
pip3 install beancount fava

# Verify installations
go version        # Should be 1.25+
node --version    # Should be v18+
pnpm --version
beancount --version
fava --version
task --version
```

## Step 1: Clone Repository

```bash
# Clone the monorepo
git clone https://github.com/pigeonworks-llc/freee-automation.git
cd freee-automation

# Verify Go workspace
cat go.work
```

Expected output:
```
go 1.25

use .
use ./emulator
use ./gmail-receipt-fetcher
```

## Step 2: Build All Components

### Option A: Build Everything at Once

```bash
# Build all Go binaries
make go-build-all

# Build TypeScript workspaces
make ts-build

# Or build everything
make build-all
```

### Option B: Build Individually

```bash
# Build freee-sync
make freee-sync-build
# Output: ./bin/freee-sync

# Build emulator
cd emulator
task build
# Output: ./bin/freee-emulator
cd ..

# Build Gmail receipt fetcher
cd gmail-receipt-fetcher
go build -o bin/auto-fetch ./cmd/auto-fetch
cd ..

# Build Amazon receipt processor
cd amazon-receipt-processor
pnpm install
pnpm build
cd ..
```

## Step 3: Set Up Emulator (Development)

The emulator allows you to develop and test without hitting the production freee API.

### Start Emulator

```bash
cd emulator

# Start on default port 8080
task run

# Or specify a custom port
PORT=8090 task run
```

### Verify Emulator

In another terminal:

```bash
# Health check
curl http://localhost:8080/health

# Get OAuth token
curl -X POST http://localhost:8080/oauth/token \
  -d "grant_type=authorization_code&code=test-auth-code&client_id=test&client_secret=test&redirect_uri=http://localhost" \
  | jq .

# Expected output:
# {
#   "access_token": "...",
#   "refresh_token": "...",
#   "company_id": 1,
#   "expires_in": 3600
# }
```

### Load Sample Data

```bash
cd emulator
task seed
```

This creates:
- 3 wallet transactions (unbooked)
- 2 deals
- 1 journal

## Step 4: Set Up Gmail Receipt Fetcher

### Create Gmail API Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or select existing)
3. Enable Gmail API
4. Create OAuth 2.0 credentials
5. Download `credentials.json`

### Configure Gmail Fetcher

```bash
cd gmail-receipt-fetcher

# Copy your downloaded credentials
cp ~/Downloads/credentials.json .

# Or create from template
cp credentials.json.example credentials.json
# Edit with your client_id and client_secret
```

### Run Gmail Fetcher (Dry Run)

```bash
cd gmail-receipt-fetcher

# Point to emulator (localhost)
./bin/auto-fetch \
  --credentials credentials.json \
  --freee-api http://localhost:8080 \
  --dry-run

# First run will open browser for OAuth consent
# Approve and complete authentication
```

### Run Against Production freee

```bash
# Set environment variables
export FREEE_CLIENT_ID="your_freee_client_id"
export FREEE_CLIENT_SECRET="your_freee_client_secret"

# Run (creates wallet_txns in production)
./bin/auto-fetch \
  --credentials credentials.json \
  --freee-api https://api.freee.co.jp
```

## Step 5: Set Up Amazon Receipt Processor

### Install Dependencies

```bash
cd amazon-receipt-processor
pnpm install
```

### Configure Amazon Credentials

```bash
# Set environment variables or create .env
export AMAZON_EMAIL="your_amazon_email"
export AMAZON_PASSWORD="your_amazon_password"
```

### Run Processor (Dry Run)

```bash
cd amazon-receipt-processor
pnpm start --dry-run
```

This will:
1. Launch Playwright browser
2. Log in to Amazon
3. Scrape order history
4. Download receipt PDFs
5. (In production) Upload to freee API

## Step 6: Set Up Beancount Integration

### Initialize Beancount Ledger

The `beancount/` directory already contains:
- `main.beancount`: Entry point
- `accounts.beancount`: Chart of accounts
- `2024/`, `2025/`: Monthly transaction files

### Configure freee-sync

```bash
# Check available commands
./bin/freee-sync --help

# Commands:
#   sync      Sync transactions from freee to Beancount
#   migrate   Run database migrations
```

### Sync from freee (via Emulator)

```bash
# Get OAuth token from emulator
TOKEN=$(curl -s -X POST "http://localhost:8080/oauth/token" \
  -d "grant_type=authorization_code&code=test&client_id=test&client_secret=test&redirect_uri=http://localhost" \
  | jq -r '.access_token')

# Sync transactions
./bin/freee-sync sync \
  --api-url http://localhost:8080 \
  --token "$TOKEN" \
  --from 2024-01-01 \
  --to 2024-12-31 \
  --output beancount/
```

### Sync from Production freee

```bash
# Export freee credentials
export FREEE_CLIENT_ID="your_client_id"
export FREEE_CLIENT_SECRET="your_client_secret"

# Get token (implement OAuth flow)
# ... obtain access_token ...

# Sync
./bin/freee-sync sync \
  --api-url https://api.freee.co.jp \
  --token "$ACCESS_TOKEN" \
  --from 2024-04-01 \
  --to 2025-03-31 \
  --output beancount/
```

### Validate Beancount Files

```bash
# Check for errors
bean-check beancount/main.beancount

# Expected: No errors
```

### Launch Fava Web UI

```bash
# Start Fava
fava beancount/main.beancount

# Or use the convenience script
./bin/fava-start
```

Open http://localhost:5000 in your browser.

## Step 7: Complete Workflow Example

Here's a complete workflow from receipt to Beancount:

### 1. Start Emulator

```bash
cd emulator
task run
# Keep this running in terminal 1
```

### 2. Fetch Gmail Receipts

```bash
# Terminal 2
cd gmail-receipt-fetcher
./bin/auto-fetch \
  --credentials credentials.json \
  --freee-api http://localhost:8080
```

This creates wallet_txns in the emulator.

### 3. Create Deals (Manual or via API)

```bash
# Get token
TOKEN=$(curl -s -X POST "http://localhost:8080/oauth/token" \
  -d "grant_type=authorization_code&code=test&client_id=test&client_secret=test&redirect_uri=http://localhost" \
  | jq -r '.access_token')

# Create deal with payment linkage
curl -X POST "http://localhost:8080/api/1/deals" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "issue_date": "2024-11-20",
    "type": "expense",
    "details": [{
      "account_item_id": 502,
      "tax_code": 136,
      "amount": 1980,
      "description": "Amazon購入"
    }],
    "payments": [{
      "date": "2024-11-20",
      "from_walletable_type": "credit_card",
      "from_walletable_id": 2,
      "amount": 1980
    }]
  }'
```

### 4. Sync to Beancount

```bash
# Terminal 3
./bin/freee-sync sync \
  --api-url http://localhost:8080 \
  --token "$TOKEN" \
  --from 2024-11-01 \
  --to 2024-11-30 \
  --output beancount/
```

### 5. View in Fava

```bash
# Terminal 4
fava beancount/main.beancount
```

Navigate to http://localhost:5000 and verify the transaction appears.

## Configuration

### Environment Variables

Create `.env` files for each component:

**emulator/.env**
```bash
PORT=8080
DB_PATH=./data/freee.db
```

**gmail-receipt-fetcher/.env**
```bash
FREEE_CLIENT_ID=your_freee_client_id
FREEE_CLIENT_SECRET=your_freee_client_secret
FREEE_TOKEN_PATH=./freee-token.json
```

**amazon-receipt-processor/.env**
```bash
AMAZON_EMAIL=your_email
AMAZON_PASSWORD=your_password
FREEE_API_URL=http://localhost:8080
```

### Configuration Files

**config/accounts.yaml** (for freee-sync)
```yaml
mappings:
  # freee account_item_id → Beancount account
  502: "Expenses:Books"        # 新聞図書費
  505: "Expenses:Fees"         # 支払手数料
  510: "Expenses:Training"     # 研修費
  599: "Expenses:Misc"         # 雑費
```

## Troubleshooting

### Emulator Won't Start

**Error**: `panic: failed to open database`
**Solution**:
```bash
cd emulator
rm -rf data/freee.db
task run
```

### Gmail OAuth Fails

**Error**: `redirect_uri_mismatch`
**Solution**:
1. Check `credentials.json` redirect URIs
2. Add `http://localhost` to authorized redirect URIs in Google Cloud Console
3. Try again: `./bin/auto-fetch --credentials credentials.json --freee-api http://localhost:8080`

### Beancount Parse Error

**Error**: `Syntax error, unexpected TOKEN`
**Solution**:
```bash
# Validate specific file
bean-check beancount/2024/11.beancount

# Check line numbers in error message
# Fix syntax (common issues: missing quotes, wrong date format)
```

### Port Already in Use

**Error**: `bind: address already in use`
**Solution**:
```bash
# Find process using port 8080
lsof -i :8080

# Kill it
kill -9 <PID>

# Or use different port
PORT=8090 task run
```

### Go Build Fails

**Error**: `cannot find package`
**Solution**:
```bash
# Sync Go workspace
go work sync

# Update dependencies
go mod download

# Try again
make go-build-all
```

## Next Steps

Now that you have everything set up:

1. **Read the Architecture**: [architecture.md](architecture.md)
2. **Review API Reference**: [api-reference.md](api-reference.md)
3. **Deploy Emulator**: [emulator/docs/deployment.md](../emulator/docs/deployment.md)
4. **Automate with Cron**: Set up scheduled receipt fetching
5. **Customize Account Mappings**: Edit `config/accounts.yaml`
6. **Add Your Receipts**: Start collecting real receipts

## Useful Commands

```bash
# Development
make freee-sync-build       # Build freee-sync
make emulator-dev           # Run emulator in dev mode
make ts-build              # Build TypeScript projects

# Testing
cd emulator && task test-all       # Run all emulator tests
cd emulator && task test-scenario  # Run scenario tests

# Cleanup
make clean-all             # Clean all build artifacts
rm -rf emulator/data/      # Reset emulator database
rm -rf beancount/.sync/    # Reset sync database

# Beancount
bean-check beancount/main.beancount          # Validate
bean-report beancount/main.beancount balances # Print balances
fava beancount/main.beancount                # Launch web UI
```

## Support

For issues or questions:
- Check [Troubleshooting](#troubleshooting) section above
- Review component READMEs for detailed documentation
- Open an issue on GitHub: https://github.com/pigeonworks-llc/freee-automation/issues
