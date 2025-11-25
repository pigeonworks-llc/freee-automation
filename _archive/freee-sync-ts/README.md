# freee-sync

freee会計APIからデータを取得し、Beancount形式に変換する同期ツール。

## Features

- freee OAuth2 認証
- 取引データの自動取得
- Beancount形式への変換
- 月次同期の自動化
- 同期履歴の管理（JSON）

## Requirements

- Node.js 18+ (25.1.0 confirmed working)
- freee会計アカウント
- freee API credentials (Client ID & Secret)

## Setup

### 1. Install Dependencies

```bash
npm install
```

### 2. Configure Environment Variables

```bash
cp .env.example .env.local
```

Edit `.env.local` and add your freee API credentials:
- Get credentials from: https://developer.freee.co.jp/

### 3. First-time OAuth Authentication

```bash
npm run auth
```

This will:
1. Open your browser for freee authorization
2. Save access token to `.env.local`
3. Fetch company ID

## Usage

### Sync Transactions

```bash
# Sync specific date range
npm run sync -- --from 2024-11-01 --to 2025-10-31

# Dry run (don't write files)
npm run sync -- --from 2024-11-01 --to 2025-10-31 --dry-run

# Monthly sync (previous month)
npm run sync:monthly
```

### Development

```bash
# Watch mode with auto-reload
npm run dev

# Build TypeScript
npm run build

# Run compiled code
npm start
```

## Project Structure

```
freee-sync/
├── src/
│   ├── index.ts           # Entry point
│   ├── cli.ts             # CLI commands
│   ├── sync.ts            # Sync orchestration
│   ├── freee/
│   │   ├── auth.ts        # OAuth2 authentication
│   │   ├── client.ts      # API client
│   │   └── types.ts       # Type definitions
│   ├── converter/
│   │   ├── converter.ts   # freee → Beancount converter
│   │   └── mapper.ts      # Account mapping
│   └── database/
│       └── sync-history.ts # Sync history management
├── config/
│   └── account-mapping.yaml # freee ↔ Beancount mapping
├── data/
│   └── sync-history.json    # Sync history (auto-generated)
├── dist/                     # Compiled JavaScript (auto-generated)
└── package.json
```

## Output

Synced transactions are written to:
- `../beancount/2024/2024-11.beancount`
- `../beancount/2024/2024-12.beancount`
- `../beancount/2025/2025-01.beancount`
- ... (monthly files)

## Implementation Status

- [x] TypeScript project setup
- [ ] freee OAuth2 authentication
- [ ] freee API client
- [ ] freee → Beancount converter
- [ ] Sync history management
- [ ] CLI implementation
- [ ] Initial data import

## API Rate Limits

- freee API: 600 requests/hour
- Automatic retry with exponential backoff

## License

Private project

## Author

Shunichi Ikebuchi
