# Accounting System (freee + Beancount)

Hybrid accounting system combining freee (cloud accounting) with Beancount (plain-text accounting) for audit and analysis.

freee（クラウド会計）とBeancount（プレーンテキスト会計）を組み合わせたハイブリッド会計システム。freeeを日常業務に使用し、Beancountで監査・分析を行います。

## System Architecture

```
freee (Master Data)
  ↓ freee-sync (Go CLI)
Beancount (Audit & Analysis Layer)
  ↓
Fava Web UI (Dashboard)
```

## Features

- **Monthly Audit**: Balance sheet and income statement validation
- **Evidence Reconciliation**: Bank statement and credit card statement matching
- **Recurring Transaction Check**: Detect missing rent, subscriptions, etc.
- **Amazon Receipt Integration**: Semi-automated receipt download and attachment
- **Detailed Analysis**: Complex reports not available in freee
- **Duplicate Prevention**: SQLite-based sync history tracking

## Project Structure

```
accounting-system/
├── beancount/                # Beancount ledger files
│   ├── main.beancount       # Main ledger (includes all)
│   ├── accounts.beancount   # Account definitions
│   ├── opening-balances.beancount
│   ├── 2024/                # Monthly transaction files
│   ├── 2025/
│   ├── documents/           # Receipt PDFs
│   ├── attachments/         # Document attachments
│   └── .sync/               # SQLite sync database
├── cmd/                      # Go CLI applications
│   └── freee-sync/          # freee → Beancount sync tool
├── pkg/                      # Go shared packages
│   ├── config/              # Configuration management
│   ├── pathutil/            # Path resolution utilities
│   ├── db/                  # SQLite database layer
│   ├── beancount/           # Beancount file operations
│   ├── freee/               # freee API client
│   └── converter/           # freee → Beancount converter
├── emulator/                 # freee API emulator (for development)
├── amazon-receipt-processor/ # Amazon receipt download tool (TypeScript)
├── config/                   # Configuration files
│   └── account-mapping.yaml # freee ↔ Beancount mapping
├── bin/                      # Executable scripts & binaries
│   ├── fava-start           # Fava startup script
│   └── freee-sync           # Sync CLI binary
├── _archive/                 # Archived TypeScript implementations
│   ├── freee-sync-ts/       # Original freee-sync (TypeScript)
│   └── shared-ts/           # Original shared library (TypeScript)
└── docs/                     # Documentation
```

## Quick Start

### 1. Build

```bash
# Build all Go binaries
make go-build-all

# Or build individually
make freee-sync-build
make emulator-build
```

### 2. Configure

```bash
# Copy and edit configuration
cp .env.example .env
# Edit .env with your freee API credentials
```

### 3. Sync from freee

```bash
# Sync transactions
./bin/freee-sync sync --from 2024-01-01 --to 2024-12-31

# Dry run (preview without writing)
./bin/freee-sync sync --from 2024-01-01 --to 2024-12-31 --dry-run

# View sync statistics
./bin/freee-sync stats
```

### 4. Launch Fava Dashboard

```bash
./bin/fava-start
```

Then open http://localhost:5000 in your browser.

## Development

### Prerequisites

- Go 1.25+ (for freee-sync and emulator)
- Beancount 3.2.0+ (installed via Homebrew)
- Fava 1.30.7+ (installed via Homebrew)
- Node.js 18+ (for amazon-receipt-processor)

### Using the Emulator

For development without freee API access:

```bash
# Start emulator
make emulator-dev

# Seed test data
cd emulator && ./scripts/seed.sh

# Get access token and configure .env
curl -X POST http://localhost:8080/oauth/token -d "grant_type=client_credentials"
```

### Make Targets

```bash
make help              # Show all targets

# Go
make freee-sync-build  # Build freee-sync binary
make emulator-build    # Build freee-emulator binary
make emulator-dev      # Run emulator in development mode
make go-build-all      # Build all Go binaries
make go-clean          # Clean all Go build artifacts

# TypeScript (amazon-receipt-processor)
make ts-install        # Install npm dependencies
make ts-build          # Build TypeScript workspaces

# All
make build-all         # Build everything
make clean-all         # Clean everything
```

### Running Tests

```bash
# Validate Beancount syntax
bean-check beancount/main.beancount

# Run balance check
bean-query beancount/main.beancount "SELECT account, sum(position) WHERE account ~ '^Assets'"
```

## Current Status

### Phase 1: Environment Setup - COMPLETED

- [x] Project structure created
- [x] Beancount 3.2.0 installed
- [x] Fava 1.30.7 installed
- [x] Account chart designed (Japanese GAAP compliant)
- [x] freee ↔ Beancount mapping defined
- [x] Basic .beancount files created
- [x] Fava startup script created

### Phase 2: Go Rewrite - COMPLETED

- [x] freee-sync rewritten in Go
- [x] Shared packages (pkg/*) implemented
- [x] freee API client with pagination
- [x] SQLite sync history for duplicate prevention
- [x] freee-emulator integrated into monorepo
- [x] CLI with Cobra (sync, stats commands)
- [x] TypeScript version archived to _archive/

### Phase 3: Audit Features - PLANNED

- [ ] Balance sheet checker
- [ ] Income statement checker
- [ ] Recurring transaction checker

### Phase 4: Amazon Integration - PLANNED

- [ ] Semi-automated receipt download
- [ ] Receipt upload to freee
- [ ] Transaction matching

## Account Mapping

See `config/account-mapping.yaml` for the complete mapping between freee accounts and Beancount accounts.

Example:
- freee: "普通預金" → Beancount: `Assets:Current:Bank:Ordinary`
- freee: "地代家賃" → Beancount: `Expenses:SGA:Rent`
- freee: "売上高" → Beancount: `Income:Sales:Products`

## Configuration

Environment variables (`.env`):

```bash
# freee API
FREEE_API_URL=http://localhost:8080  # Use emulator for development
FREEE_ACCESS_TOKEN=your_token_here
FREEE_COMPANY_ID=1

# Beancount
BEANCOUNT_ROOT=./beancount
BEANCOUNT_DB_PATH=./beancount/.sync/sync.db
BEANCOUNT_ATTACHMENTS_DIR=./beancount/attachments
```

## Japanese Accounting Standards

This system follows Japanese GAAP (Generally Accepted Accounting Principles):

- **Fiscal Year**: April 1 - March 31
- **Consumption Tax**: 10% (standard), 8% (reduced), 0% (exempt), non-taxable
- **Account Structure**: Assets, Liabilities, Equity, Income, Expenses

## Tech Stack

- **Language**: Go 1.25 (sync tool, emulator), TypeScript (amazon-receipt-processor)
- **Beancount**: 3.2.0 (double-entry accounting)
- **Fava**: 1.30.7 (web UI)
- **freee API**: OAuth2 + REST
- **Database**: SQLite (sync history)
- **CLI Framework**: Cobra

## License

Private project

## Author

Shunichi Ikebuchi
