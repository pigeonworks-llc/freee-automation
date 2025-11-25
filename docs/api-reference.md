# freee Accounting API Reference

This document provides a comprehensive reference for the freee Accounting API endpoints used in this project.

## Base URL

- **Production:** `https://api.freee.co.jp`
- **API Version:** v1

## Authentication

### OAuth 2.0

freee API uses OAuth 2.0 for authentication.

#### Authorization URL

```
https://accounts.secure.freee.co.jp/public_api/authorize?
  client_id={CLIENT_ID}&
  redirect_uri={REDIRECT_URI}&
  response_type=code&
  prompt=select_company
```

#### Token Endpoint

```
POST https://accounts.secure.freee.co.jp/public_api/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&
client_id={CLIENT_ID}&
client_secret={CLIENT_SECRET}&
code={AUTHORIZATION_CODE}&
redirect_uri={REDIRECT_URI}
```

#### Refresh Token

```
POST https://accounts.secure.freee.co.jp/public_api/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token&
client_id={CLIENT_ID}&
client_secret={CLIENT_SECRET}&
refresh_token={REFRESH_TOKEN}
```

#### Token Response

```json
{
  "access_token": "xxx",
  "token_type": "bearer",
  "expires_in": 21600,
  "refresh_token": "xxx",
  "scope": "...",
  "created_at": 1234567890,
  "company_id": 12345
}
```

**Note:** `expires_in` is 21600 seconds (6 hours). Always store and check `expires_at` to refresh before expiration.

### Request Headers

```
Authorization: Bearer {ACCESS_TOKEN}
Content-Type: application/json
```

---

## Wallet Transactions (明細)

freeeホーム画面の「未処理明細数」や「自動で経理」画面に表示される明細は、このAPIで取得できます。

### GET /api/1/wallet_txns

Get list of wallet transactions (口座明細一覧).

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `company_id` | integer | Yes | Company ID (事業所ID) |
| `walletable_type` | string | No | `bank_account`, `credit_card`, or `wallet` |
| `walletable_id` | integer | No | Account ID. **Required if walletable_type is set** |
| `start_date` | string | No | Start date (`yyyy-mm-dd`) |
| `end_date` | string | No | End date (`yyyy-mm-dd`) |
| `entry_side` | string | No | `income` or `expense` |
| `limit` | integer | No | Records per page (default: 50, max: 100) |
| `offset` | integer | No | Offset for pagination (default: 0) |

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | 明細ID |
| `company_id` | integer | 事業所ID |
| `date` | string | 取引日 (`yyyy-mm-dd`) |
| `amount` | integer | 明細金額 (支出はマイナス) |
| `due_amount` | integer | **取引登録待ち金額** (未処理判定に使用) |
| `balance` | integer | 残高 |
| `entry_side` | string | `income` (入金) or `expense` (出金) |
| `walletable_type` | string | `bank_account`, `credit_card`, or `wallet` |
| `walletable_id` | integer | 口座ID |
| `description` | string | 取引内容 |
| `rule_matched` | boolean | 自動登録ルールにマッチしたか |

#### Filtering Unprocessed Transactions (未処理明細の取得)

**重要:** 「未処理明細」の判定には `due_amount` フィールドを使用します。

| 状態 | 条件 | 説明 |
|------|------|------|
| 未処理 | `due_amount == amount` | 取り込み直後、まだ取引登録されていない |
| 処理済み | `due_amount == 0` | 完全に取引登録済み |
| 一部処理 | `0 < due_amount < amount` | 一部だけ取引登録済み |

**実務的なフィルタ条件:** `due_amount != 0` で未処理（＋一部未処理）を取得

```typescript
// 未処理明細のフィルタ例
const unprocessed = wallet_txns.filter(txn => txn.due_amount !== 0);
```

```go
// Go での例
var unprocessed []WalletTxn
for _, t := range txns {
    if t.DueAmount != 0 {
        unprocessed = append(unprocessed, t)
    }
}
```

#### Response Example

```json
{
  "wallet_txns": [
    {
      "id": 12345,
      "company_id": 999,
      "date": "2024-01-15",
      "amount": -1500,
      "due_amount": -1500,
      "balance": 50000,
      "entry_side": "expense",
      "walletable_type": "credit_card",
      "walletable_id": 456,
      "description": "AMAZON.CO.JP",
      "rule_matched": false
    }
  ]
}
```

---

## Deals (取引)

### GET /api/1/deals

Get list of deals (取引一覧).

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `company_id` | integer | Yes | Company ID |
| `partner_id` | integer | No | Partner ID |
| `account_item_id` | integer | No | Account item ID |
| `partner_code` | string | No | Partner code |
| `status` | string | No | `settled` or `unsettled` |
| `type` | string | No | `income` or `expense` |
| `start_issue_date` | string | No | Start date (`yyyy-mm-dd`) |
| `end_issue_date` | string | No | End date (`yyyy-mm-dd`) |
| `start_due_date` | string | No | Due date start |
| `end_due_date` | string | No | Due date end |
| `limit` | integer | No | Records per page (default: 100, max: 100) |
| `offset` | integer | No | Offset for pagination |

#### Response

```json
{
  "deals": [
    {
      "id": 12345,
      "company_id": 999,
      "issue_date": "2024-01-15",
      "due_date": "2024-02-15",
      "amount": 10000,
      "due_amount": 0,
      "type": "expense",
      "partner_id": 123,
      "ref_number": "INV-001",
      "status": "settled",
      "details": [...],
      "payments": [...],
      "receipts": [...]
    }
  ]
}
```

### POST /api/1/deals

Create a new deal (取引の作成).

#### Request Body

```json
{
  "company_id": 999,
  "issue_date": "2024-01-15",
  "type": "expense",
  "due_date": "2024-02-15",
  "partner_id": 123,
  "partner_code": "VENDOR001",
  "ref_number": "INV-001",
  "details": [
    {
      "account_item_id": 100,
      "tax_code": 21,
      "amount": 10000,
      "item_id": 1,
      "section_id": 1,
      "tag_ids": [1, 2],
      "segment_1_tag_id": 1,
      "segment_2_tag_id": 2,
      "segment_3_tag_id": 3,
      "description": "Description",
      "vat": 1000
    }
  ],
  "payments": [
    {
      "from_walletable_type": "credit_card",
      "from_walletable_id": 456,
      "date": "2024-01-15",
      "amount": 10000
    }
  ],
  "receipt_ids": [11111, 22222]
}
```

#### Response

```json
{
  "deal": {
    "id": 12345,
    ...
  }
}
```

### PUT /api/1/deals/{id}

Update an existing deal.

#### Request Body

Same structure as POST, with only fields to update.

---

## Receipts (ファイルボックス/証憑)

### GET /api/1/receipts

Get list of receipts.

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `company_id` | integer | Yes | Company ID |
| `start_date` | string | No | Issue date start (`yyyy-mm-dd`) |
| `end_date` | string | No | Issue date end (`yyyy-mm-dd`) |
| `user_name` | string | No | Filter by uploader name |
| `number` | integer | No | Receipt number |
| `comment_type` | string | No | `posted`, `raised`, `resolved` |
| `comment_important` | boolean | No | Important flag |
| `category` | string | No | Receipt category |
| `limit` | integer | No | Records per page (default: 50, max: 100) |
| `offset` | integer | No | Offset for pagination |

### POST /api/1/receipts

Upload a receipt file.

#### Request (multipart/form-data)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `company_id` | integer | Yes | Company ID |
| `receipt` | file | Yes | Receipt file (PDF, PNG, JPG, etc.) |
| `description` | string | No | Description/memo (max 255 chars) |
| `issue_date` | string | No | Issue date (`yyyy-mm-dd`) |

#### Example

```bash
curl -X POST "https://api.freee.co.jp/api/1/receipts" \
  -H "Authorization: Bearer {TOKEN}" \
  -F "company_id=999" \
  -F "receipt=@/path/to/receipt.pdf" \
  -F "description=Amazon Order: D01-1234567" \
  -F "issue_date=2024-01-15"
```

#### Response

```json
{
  "receipt": {
    "id": 12345,
    "status": "confirmed",
    "description": "Amazon Order: D01-1234567",
    "mime_type": "application/pdf",
    "origin": "public_api",
    "created_at": "2024-01-15T10:00:00+09:00",
    "user": {
      "id": 100,
      "email": "user@example.com",
      "display_name": "User"
    },
    "receipt_metadatum": {
      "partner_name": "",
      "issue_date": null,
      "amount": null
    }
  }
}
```

### GET /api/1/receipts/{id}

Get a specific receipt.

### PUT /api/1/receipts/{id}

Update receipt metadata.

### DELETE /api/1/receipts/{id}

Delete a receipt.

---

## Account Items (勘定科目)

### GET /api/1/account_items

Get list of account items.

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `company_id` | integer | Yes | Company ID |
| `account_category_id` | integer | No | Filter by category |

#### Response

```json
{
  "account_items": [
    {
      "id": 100,
      "name": "消耗品費",
      "shortcut": "sho",
      "shortcut_num": "100",
      "default_tax_code": 21,
      "account_category": "expenses",
      "account_category_id": 10,
      "available": true,
      "walletable_id": null
    }
  ]
}
```

---

## Partners (取引先)

### GET /api/1/partners

Get list of partners.

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `company_id` | integer | Yes | Company ID |
| `keyword` | string | No | Search keyword |
| `limit` | integer | No | Records per page (default: 100) |
| `offset` | integer | No | Offset for pagination |

### POST /api/1/partners

Create a new partner.

---

## Walletables (口座)

### GET /api/1/walletables

Get list of walletables (bank accounts, credit cards, etc.).

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `company_id` | integer | Yes | Company ID |
| `with_balance` | boolean | No | Include balance info |
| `type` | string | No | `bank_account`, `credit_card`, or `wallet` |

#### Response

```json
{
  "walletables": [
    {
      "id": 456,
      "name": "アメリカン・エキスプレス・ビジネス・カード",
      "type": "credit_card",
      "bank_id": null,
      "last_balance": -50000,
      "walletable_balance": -50000
    }
  ]
}
```

---

## Tax Codes (税区分)

### Common Tax Codes

| Code | Description |
|------|-------------|
| 21 | 課税売上10% |
| 22 | 課税売上8%(軽減) |
| 23 | 非課税売上 |
| 24 | 不課税売上 |
| 121 | 課税仕入10% |
| 122 | 課税仕入8%(軽減) |
| 136 | 課税仕入10%(税込) |
| 137 | 課税仕入8%(軽減・税込) |

---

## Error Handling

### Error Response Format

```json
{
  "status_code": 400,
  "errors": [
    {
      "type": "status",
      "messages": ["不正なリクエストです。"]
    },
    {
      "type": "validation",
      "messages": ["Details 存在しない account_item_id が含まれています。"]
    }
  ]
}
```

### Common HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 401 | Unauthorized (invalid/expired token) |
| 403 | Forbidden (no permission / rate limited) |
| 404 | Not Found |
| 429 | Too Many Requests |
| 500 | Internal Server Error |

### Token Expiration

```json
{
  "message": "アクセスする権限がありません",
  "code": "expired_access_token"
}
```

**Solution:** Use refresh token to get a new access token.

---

## Rate Limiting

- **General rate limit:** HTTP 403 (cooldown ~10 minutes)
- **Per-endpoint rate limit:** HTTP 429

**Best Practices:**
- Add delays between requests (1-2 seconds)
- Implement exponential backoff on errors
- Cache responses when possible

---

## Pagination

### Standard Pagination

Most endpoints use `limit` and `offset`:

```
GET /api/1/wallet_txns?company_id=999&limit=100&offset=0
GET /api/1/wallet_txns?company_id=999&limit=100&offset=100
```

### Retrieving All Records

```typescript
async function fetchAll<T>(
  endpoint: string,
  params: Record<string, string>
): Promise<T[]> {
  const results: T[] = [];
  let offset = 0;
  const limit = 100;

  while (true) {
    const url = `${endpoint}?${new URLSearchParams({
      ...params,
      limit: String(limit),
      offset: String(offset),
    })}`;

    const response = await fetch(url, { headers });
    const data = await response.json();
    const items = data[Object.keys(data)[0]] || [];

    results.push(...items);

    if (items.length < limit) break;
    offset += limit;
  }

  return results;
}
```

---

## OAuth Scopes

Required scopes for this project:

| Scope | Description |
|-------|-------------|
| `accounting:wallet_txns:read` | Read wallet transactions |
| `accounting:wallet_txns:write` | Write wallet transactions |
| `accounting:deals:read` | Read deals |
| `accounting:deals:write` | Write deals |
| `accounting:receipts:read` | Read receipts |
| `accounting:receipts:write` | Upload/manage receipts |
| `accounting:account_items:read` | Read account items |
| `accounting:partners:read` | Read partners |
| `accounting:companies:read` | Read company info |

---

## Official Documentation

- [freee API Reference](https://developer.freee.co.jp/reference/accounting)
- [freee Developer Portal](https://developer.freee.co.jp/)
- [freee API Schema (GitHub)](https://github.com/freee/freee-api-schema)
- [freee SDKs](https://github.com/freee)
