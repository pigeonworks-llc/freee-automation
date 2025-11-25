# Architecture

freee-automationはモノレポ構成で、freee会計APIを中心とした自動化ツール群を統合管理します。

## System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     freee-automation                         │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │    Gmail     │  │   Amazon     │  │   Manual     │     │
│  │   Receipt    │  │   Receipt    │  │    Input     │     │
│  │   Fetcher    │  │  Processor   │  │              │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                  │                  │             │
│         └──────────┬───────┴──────────────────┘             │
│                    ▼                                        │
│         ┌──────────────────────┐                           │
│         │   freee Emulator    │  (Development/Test)       │
│         │   or freee API      │                           │
│         └──────────┬───────────┘                           │
│                    │                                        │
│         ┌──────────▼───────────┐                           │
│         │   freee Accounting   │                           │
│         │      (Master)        │                           │
│         └──────────┬───────────┘                           │
│                    │                                        │
│                    ▼                                        │
│         ┌──────────────────────┐                           │
│         │    freee-sync        │                           │
│         └──────────┬───────────┘                           │
│                    │                                        │
│                    ▼                                        │
│         ┌──────────────────────┐                           │
│         │     Beancount        │  (Audit/Analysis)         │
│         │   (Plain-text)       │                           │
│         └──────────┬───────────┘                           │
│                    │                                        │
│                    ▼                                        │
│         ┌──────────────────────┐                           │
│         │     Fava Web UI      │                           │
│         └──────────────────────┘                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Input Layer: Receipt Collection

#### gmail-receipt-fetcher (Go)
- **Role**: Gmail APIからレシート自動取得
- **Input**: Gmail messages with receipts
- **Output**: wallet_txns to freee API
- **Key Features**:
  - OAuth2.0 authentication
  - PDF attachment download
  - Keyword-based expense categorization
  - Duplicate detection

#### amazon-receipt-processor (TypeScript + Playwright)
- **Role**: Amazon注文履歴スクレイピング
- **Input**: Amazon order history (web scraping)
- **Output**: Receipt PDFs + freee API upload
- **Key Features**:
  - Playwright browser automation
  - Headless scraping
  - PDF generation
  - freee API integration

### 2. API Layer: freee Interface

#### emulator (Go)
- **Role**: freee API emulator for local development
- **Storage**: bbolt (embedded key-value store)
- **Endpoints**:
  - OAuth2: `/oauth/token`
  - Companies: `/api/1/companies`
  - Account Items: `/api/1/account_items`
  - Walletables: `/api/1/walletables`
  - Wallet Transactions: `/api/1/wallet_txns`
  - Deals: `/api/1/deals` (with payments)
  - Journals: `/api/1/journals`
- **Key Features**:
  - Compatible with freee API v1
  - Support for wallet_txn settlement via deals
  - Lightweight Docker deployment
  - Cloud Run ready

#### freee Production API
- **Role**: Production freee accounting service
- **Authentication**: OAuth2.0 with refresh tokens
- **Rate Limits**: Follows freee API limits
- **Endpoint**: `https://api.freee.co.jp`

### 3. Sync Layer: Data Pipeline

#### cmd/freee-sync (Go)
- **Role**: freee → Beancount data synchronization
- **Input**: freee API (deals, transactions)
- **Output**: Beancount transaction files
- **Storage**: SQLite for duplicate tracking
- **Key Features**:
  - Date range filtering
  - Pagination support
  - Incremental sync
  - Cobra CLI framework
  - Conversion rules (freee account_items → Beancount accounts)

### 4. Analysis Layer: Accounting Engine

#### beancount/
- **Role**: Plain-text accounting system
- **Format**: `.beancount` files (human-readable)
- **Structure**:
  ```
  beancount/
  ├── main.beancount         # Entry point
  ├── accounts.beancount     # Account definitions
  ├── 2024/, 2025/           # Monthly transactions
  └── documents/             # Receipt PDFs
  ```
- **Key Features**:
  - Double-entry bookkeeping
  - Japanese GAAP compliance
  - Tax calculation (消費税)
  - Balance assertions
  - Monthly ledgers

#### Fava
- **Role**: Web UI for Beancount
- **Port**: 5000
- **Features**:
  - Dashboard
  - Reports (balance sheet, income statement)
  - Transaction search
  - Document viewer
  - Chart visualization

### 5. Shared Layer: Common Utilities

#### pkg/ (Go packages)
- **config**: Configuration management
- **freee**: freee API client
- **beancount**: Beancount file operations
- **converter**: freee → Beancount conversion
- **db**: SQLite database layer
- **pathutil**: Path resolution utilities

## Data Flow

### 1. Receipt Ingestion Flow

```
Gmail/Amazon
    ↓
Receipt Fetcher/Processor
    ↓
wallet_txn creation (freee API)
    ↓
Deal creation with payments linkage
    ↓
wallet_txn status → "settled"
```

### 2. Synchronization Flow

```
freee API
    ↓
freee-sync (with SQLite dedup)
    ↓
Beancount files (monthly)
    ↓
Fava Web UI
```

### 3. Accounting Workflow

```
Receipt → wallet_txn (未仕訳) → Deal (取引明細) → Settlement (消込)
                                    ↓
                            freee-sync
                                    ↓
                            Beancount transaction
```

## Technology Stack

### Languages
- **Go 1.25**: emulator, gmail-receipt-fetcher, cmd/freee-sync, pkg
- **TypeScript**: amazon-receipt-processor
- **Beancount DSL**: Accounting ledgers

### Frameworks & Libraries
- **Go**:
  - `chi/v5`: HTTP router (emulator)
  - `cobra`: CLI framework (freee-sync)
  - `bbolt`: Embedded database (emulator)
  - `sqlite3`: Duplicate tracking (freee-sync)
  - `oauth2`: Google/freee authentication
- **TypeScript**:
  - `playwright`: Browser automation
  - `pnpm`: Package manager (workspaces)
- **Beancount**:
  - `beancount 3.2.0`: Accounting engine
  - `fava 1.30.7`: Web interface

### Storage
- **bbolt**: Key-value store for emulator (disk persistence)
- **SQLite**: Duplicate prevention for freee-sync
- **Plain-text**: Beancount ledger files (`.beancount`)
- **File system**: Receipt PDFs in `beancount/documents/`

### Deployment
- **Docker**: Containerization for emulator
- **Cloud Run**: Serverless deployment option
- **Local**: Direct binary execution for CLI tools

## Design Decisions

### Why Monorepo?
- **Shared dependencies**: Go workspace (`go.work`) allows unified dependency management
- **Atomic changes**: Changes across components can be committed together
- **Easier testing**: Integration tests can span multiple components
- **Single source of truth**: All automation tools in one place

### Why Emulator?
- **Local development**: No need for freee production credentials during development
- **Testing**: Automated tests without API rate limits
- **CI/CD**: Integration tests in GitHub Actions
- **Data isolation**: Test data doesn't pollute production

### Why Beancount?
- **Plain-text**: Version control friendly, human-readable
- **Audit trail**: Git history provides complete audit trail
- **Flexibility**: Custom queries and reports via Fava
- **Backup**: Easy to backup (just text files)
- **Hybrid approach**: freee as master, Beancount as analysis layer

### Why Go + TypeScript?
- **Go**: Performance, concurrency, single binary distribution
- **TypeScript**: Browser automation (Playwright), rich npm ecosystem
- **Separation of concerns**: Each tool uses the best language for its purpose

## Component Interactions

### Go Workspace
```
go.work
  use .
  use ./emulator
  use ./gmail-receipt-fetcher
```

All Go modules share:
- Common dependencies
- Version consistency
- Cross-component imports (via `pkg/`)

### Configuration
- **Environment variables**: `PORT`, `DB_PATH`, `FREEE_API_URL`
- **Credential files**: `credentials.json` (Gmail), `freee-token.json` (freee)
- **Config files**: `config/` directory for shared settings

### API Compatibility
- gmail-receipt-fetcher and amazon-receipt-processor can point to:
  - Emulator: `http://localhost:8080` (development)
  - Production: `https://api.freee.co.jp` (production)
- Same API contract ensures drop-in compatibility

## Security Considerations

### Credentials Management
- **Never committed**: `.gitignore` excludes all `credentials*.json` and `*-token.json`
- **Local only**: Credentials stored locally on developer machine
- **OAuth refresh tokens**: Long-lived tokens with rotation support

### API Access
- **Bearer tokens**: All freee API calls use Bearer authentication
- **Rate limiting**: Respect freee API rate limits
- **Retry logic**: Exponential backoff for transient failures

### Data Privacy
- **Receipt PDFs**: Stored locally in `beancount/documents/`
- **Transaction data**: Plain-text in `.beancount` files (assume private repo)
- **Database encryption**: Not implemented (local development only)

## Scalability

### Current Limitations
- **Single user**: Designed for personal/small business use
- **Sequential processing**: Receipt fetchers run sequentially
- **Local storage**: All data stored locally (no cloud storage)

### Future Improvements
- **Parallel processing**: Concurrent receipt fetching
- **Cloud storage**: Optional S3/GCS backend for receipts
- **Multi-company**: Support multiple freee companies
- **Webhook support**: Real-time freee updates

## Testing Strategy

### Emulator
- **Unit tests**: Handler-level tests
- **Integration tests**: Full API flow tests
- **Scenario tests**: Business workflow simulations
- **Parallel tests**: Using go-portalloc for port allocation

### CLI Tools
- **Unit tests**: Package-level tests in `pkg/`
- **Integration tests**: End-to-end with emulator
- **Manual testing**: Scripts in `scripts/` directory

### Beancount
- **Balance assertions**: Built into `.beancount` files
- **bean-check**: Validates ledger consistency
- **Manual review**: Monthly reconciliation via Fava

## Documentation Structure

```
docs/
├── api-reference.md       # freee API reference
├── architecture.md        # This file
└── getting-started.md     # Setup guide

emulator/docs/
├── deployment.md          # Cloud Run deployment
├── production.md          # Production readiness
├── automation-guide.md    # freee API automation
└── openapi/              # OpenAPI specifications

Component READMEs:
├── emulator/README.md
├── gmail-receipt-fetcher/README.md
├── amazon-receipt-processor/README.md
└── cmd/freee-sync/README.md
```

## Related Documentation

- [API Reference](api-reference.md) - freee API reference
- [Getting Started](getting-started.md) - Setup guide
- [Emulator Deployment](../emulator/docs/deployment.md) - Cloud Run deployment
- [Automation Guide](../emulator/docs/automation-guide.md) - freee API automation
