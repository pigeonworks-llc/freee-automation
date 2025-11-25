# Gmail Receipt Fetcher

A Go tool to automatically fetch receipt PDFs from Gmail based on freee unregistered transactions.

## Features

- Fetches unregistered transactions from freee API
- Searches Gmail for matching receipt emails
- Downloads PDF attachments from matched emails
- Supports Gmail label filtering
- Vendor-based filtering (Amazon, Rakuten, Yodobashi, etc.)
- Date tolerance matching (configurable)
- Environment variable configuration
- Gmail emulator support for testing

## Installation

```bash
go build -o bin/auto-fetch ./cmd/auto-fetch/
go build -o bin/gmail-fetcher ./cmd/gmail-fetcher/
```

Or using task:

```bash
task build
```

## Usage

### auto-fetch (Recommended)

Automatically matches freee unregistered transactions to Gmail receipts.

```bash
# Using flags
./bin/auto-fetch \
  --freee-api http://localhost:8080 \
  --freee-token YOUR_TOKEN \
  --label "領収書" \
  --vendors "amazon,楽天"

# Using environment variables
export FREEE_API_URL=http://localhost:8080
export FREEE_ACCESS_TOKEN=your-token
export GMAIL_LABEL=領収書
./bin/auto-fetch

# Dry run (preview without downloading)
./bin/auto-fetch --dry-run
```

### gmail-fetcher

Direct Gmail search for receipts.

```bash
./bin/gmail-fetcher \
  --from 2024-01-01 \
  --to 2024-01-31 \
  --vendors "amazon"
```

## Configuration

### Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--freee-api` | `FREEE_API_URL` | (required) | freee API URL |
| `--freee-token` | `FREEE_ACCESS_TOKEN` | (required) | freee access token |
| `--freee-company` | `FREEE_COMPANY_ID` | `1` | freee company ID |
| `--gmail-api` | `GMAIL_API_URL` | | Gmail API URL (for emulator) |
| `--label` | `GMAIL_LABEL` | | Gmail label to search |
| `--vendors` | `RECEIPT_VENDORS` | | Comma-separated vendor names |
| `--output` | `RECEIPT_OUTPUT_DIR` | `./receipts` | Output directory |
| `--credentials` | `GMAIL_CREDENTIALS_PATH` | `credentials.json` | Gmail OAuth credentials |
| `--token` | `GMAIL_TOKEN_PATH` | `~/.config/gmail-fetcher/token.json` | Gmail OAuth token |
| `--tolerance` | `RECEIPT_TOLERANCE_DAYS` | `3` | Date matching tolerance (days) |
| `--dry-run` | | `false` | Preview without downloading |

### Gmail OAuth Setup

1. Create a project in [Google Cloud Console](https://console.cloud.google.com/)
2. Enable Gmail API
3. Create OAuth 2.0 credentials (Desktop app)
4. Download `credentials.json`
5. Run the tool - it will open a browser for authentication

### With Gmail Emulator

For testing, use the [Gmail Emulator](https://github.com/pigeonworks-llc/gmail-emulator):

```bash
# Start Gmail emulator
cd ~/src/github.com/pigeonworks-llc/gmail-emulator
PORT=8081 ./bin/gmail-emulator

# Start freee emulator
cd ~/src/github.com/pigeonworks-llc/freee-emulator
PORT=8080 ./bin/freee-emulator

# Run auto-fetch with emulators
./bin/auto-fetch \
  --freee-api http://localhost:8080 \
  --freee-token dummy \
  --gmail-api http://localhost:8081 \
  --label "領収書"
```

## Architecture

```
gmail-receipt-fetcher/
├── cmd/
│   ├── auto-fetch/     # Main CLI for freee-integrated workflow
│   └── gmail-fetcher/  # Standalone Gmail search CLI
├── internal/
│   ├── freee/          # freee API client
│   ├── gmail/          # Gmail API client with OAuth2
│   └── receipt/        # Receipt search and download logic
└── Taskfile.yml        # Task runner configuration
```

## Workflow

1. **Fetch**: Get unregistered transactions from freee API
2. **Filter**: Filter by vendor names (if specified)
3. **Search**: Search Gmail for receipts in the transaction date range
4. **Match**: Match transactions to receipts by date and amount
5. **Download**: Download PDF attachments from matched emails

## Supported Vendors

Auto-detected vendors:
- Amazon
- Rakuten (楽天)
- Yodobashi (ヨドバシ)
- Apple
- Google

Custom vendors can be specified with `--vendors` flag.

## Testing

```bash
go test ./...
```

## License

MIT
