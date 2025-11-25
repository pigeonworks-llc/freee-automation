# freee Automation

Monorepo for freee accounting automation tools and integrations.

freee会計の自動化ツールと連携システムのモノレポ。Gmail/Amazonからのレシート取得、freee APIエミュレーター、Beancount連携を統合管理します。

## Repository

https://github.com/pigeonworks-llc/freee-automation

## Components

### 1. emulator

freee会計APIのエミュレーター。ローカル開発やテスト環境で動作確認が可能。

- OAuth2.0対応（refresh_token, company_id）
- Walletables API (GET /api/1/walletables)
- Deals API with Payments支援
- wallet_txns自動リンク（自動で経理）
- 詳細: [emulator/README.md](emulator/README.md)

### 2. gmail-receipt-fetcher

Gmailからレシートを自動取得し、freeeへアップロードするツール（Go）。

- Gmail API連携（OAuth2.0）
- PDF領収書の自動ダウンロード
- freee API統合（wallet_txn作成、Deal作成）
- 勘定科目の自動判定（keyword-based mapping）
- 詳細: [gmail-receipt-fetcher/README.md](gmail-receipt-fetcher/README.md)

### 3. amazon-receipt-processor

Amazonの領収書を自動取得・処理するツール（TypeScript + Playwright）。

- Amazon注文履歴のスクレイピング
- 領収書PDFの自動ダウンロード
- freee APIへのアップロード
- 詳細: [amazon-receipt-processor/README.md](amazon-receipt-processor/README.md)

### 4. beancount

freeeとBeancountを組み合わせたハイブリッド会計システム。

- freeeをマスターデータとして使用
- Beancountで監査・分析レイヤーを構築
- 月次監査、証憑突合、定期取引チェック
- Fava Web UI でダッシュボード表示

### 5. cmd/freee-sync

freeeからBeancountへのデータ同期ツール（Go CLI）。

- freee API → Beancount変換
- SQLite based duplicate prevention
- Pagination対応
- Cobra CLI framework

### 6. pkg

共有Goパッケージ群。

- `config`: 設定管理
- `freee`: freee APIクライアント
- `beancount`: Beancountファイル操作
- `converter`: freee → Beancount変換
- `db`: SQLite database layer
- `pathutil`: Path resolution

## Project Structure

```
freee-automation/
├── emulator/                   # freee API emulator (Go)
├── gmail-receipt-fetcher/      # Gmail receipt automation (Go)
├── amazon-receipt-processor/   # Amazon receipt automation (TypeScript)
├── beancount/                  # Beancount ledger files
│   ├── main.beancount
│   ├── accounts.beancount
│   ├── 2024/, 2025/           # Monthly transactions
│   └── documents/             # Receipt PDFs
├── cmd/freee-sync/             # freee → Beancount sync tool
├── pkg/                        # Shared Go packages
├── config/                     # Configuration files
├── docs/                       # Documentation
├── bin/                        # Executables
└── go.work                     # Go workspace
```

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 18+ (for amazon-receipt-processor)
- Beancount 3.2.0+ (for beancount integration)
- Fava 1.30.7+ (for web UI)

### 1. Clone Repository

```bash
git clone https://github.com/pigeonworks-llc/freee-automation.git
cd freee-automation
```

### 2. Build

```bash
# Build all Go binaries
make go-build-all

# Or build individually
make freee-sync-build
make emulator-build
```

### 3. Start Emulator (for development)

```bash
cd emulator
task run

# Or using Make
make emulator-dev
```

### 4. Gmail Receipt Fetcher

```bash
cd gmail-receipt-fetcher

# Setup credentials
cp credentials.json.example credentials.json
# Edit credentials.json with your Gmail API credentials

# Run
./bin/auto-fetch --credentials credentials.json --freee-api http://localhost:8080
```

### 5. Beancount Integration

```bash
# Sync from freee
./bin/freee-sync sync --from 2024-01-01 --to 2024-12-31

# Launch Fava dashboard
./bin/fava-start
```

Open http://localhost:5000 in your browser.

## Development

### Go Workspace

This monorepo uses Go workspace (go.work):

```bash
# All Go modules are managed together
go work use . ./emulator ./gmail-receipt-fetcher

# Build from anywhere
go build ./emulator/cmd/server
go build ./gmail-receipt-fetcher/cmd/auto-fetch
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

## Documentation

- [API Reference](docs/api-reference.md) - freee API reference
- [Architecture](docs/architecture.md) - System architecture
- [Getting Started](docs/getting-started.md) - Detailed setup guide
- [Emulator Deployment](emulator/docs/deployment.md) - Cloud Run deployment
- [Automation Guide](emulator/docs/automation-guide.md) - freee API automation

## Japanese Accounting Standards

This system follows Japanese GAAP (Generally Accepted Accounting Principles):

- **Fiscal Year**: April 1 - March 31
- **Consumption Tax**: 10% (standard), 8% (reduced), 0% (exempt), non-taxable
- **Account Structure**: Assets, Liabilities, Equity, Income, Expenses

## Tech Stack

- **Language**: Go 1.25, TypeScript
- **Beancount**: 3.2.0 (double-entry accounting)
- **Fava**: 1.30.7 (web UI)
- **freee API**: OAuth2 + REST
- **Database**: SQLite, BoltDB
- **CLI Framework**: Cobra
- **Web Automation**: Playwright

## License

Private project

## Author

Shunichi Ikebuchi
