# freee API Emulator

freee会計APIのエミュレータです。ローカル開発やテスト環境でfreee APIを使用するアプリケーションの動作確認に利用できます。

## 特徴

- **軽量**: Pure Goで実装され、バイナリサイズが小さい
- **永続化**: bboltを使用したディスク永続化
- **コンテナ対応**: Dockerコンテナに最適化
- **OAuth2対応**: Bearer Token認証のエミュレーション
- **freee互換**: freee会計APIと互換性のあるエンドポイント

## サポートAPI

### OAuth2
- `POST /oauth/token` - アクセストークン発行

### 取引 (Deals)
- `GET /api/1/deals` - 取引一覧取得
- `GET /api/1/deals/{id}` - 取引詳細取得
- `POST /api/1/deals` - 取引作成
- `PUT /api/1/deals/{id}` - 取引更新
- `DELETE /api/1/deals/{id}` - 取引削除

### 仕訳 (Journals)
- `GET /api/1/journals` - 仕訳一覧取得
- `GET /api/1/journals/{id}` - 仕訳詳細取得
- `POST /api/1/journals` - 仕訳作成

### 明細 (Wallet Transactions)
- `GET /api/1/wallet_txns` - 明細一覧取得（未仕訳チェック対応）
- `GET /api/1/wallet_txns/{id}` - 明細詳細取得
- `POST /api/1/wallet_txns` - 明細作成
- `PUT /api/1/wallet_txns/{id}` - 明細更新
- `DELETE /api/1/wallet_txns/{id}` - 明細削除

## セットアップ

### 必要要件

- Go 1.23以上
- Task（タスクランナー）

### インストール

```bash
# リポジトリをクローン
git clone https://github.com/pigeonworks-llc/freee-emulator.git
cd freee-emulator

# 依存関係をインストール
task setup

# ビルド
task build
```

## 使い方

### サーバーの起動

```bash
# デフォルト設定で起動（ポート8080）
task run

# または開発モードで起動
task dev
```

### 環境変数

| 変数 | デフォルト | 説明 |
|------|-----------|------|
| `PORT` | `8080` | サーバーのポート番号 |
| `DB_PATH` | `./data/freee.db` | データベースファイルのパス |

### 使用例

#### 1. アクセストークンの取得

```bash
curl -X POST http://localhost:8080/oauth/token \
  -d "grant_type=client_credentials"
```

レスポンス:
```json
{
  "access_token": "ランダムなトークン文字列",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

#### 2. 取引の作成

```bash
curl -X POST http://localhost:8080/api/1/deals \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "issue_date": "2025-01-15",
    "type": "income",
    "details": [
      {
        "account_item_id": 1,
        "tax_code": 1,
        "amount": 10000
      }
    ]
  }'
```

#### 3. 取引一覧の取得

```bash
curl -X GET "http://localhost:8080/api/1/deals?company_id=1" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

#### 4. 仕訳の作成

```bash
curl -X POST http://localhost:8080/api/1/journals \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_id": 1,
    "issue_date": "2025-01-15",
    "details": [
      {
        "entry_type": "debit",
        "account_item_id": 1,
        "tax_code": 0,
        "amount": 10000,
        "vat": 0
      },
      {
        "entry_type": "credit",
        "account_item_id": 2,
        "tax_code": 1,
        "amount": 9090,
        "vat": 910
      }
    ]
  }'
```

### サンプルデータの投入

```bash
# サーバーを起動してサンプルデータを投入
task seed

# または別ターミナルでサーバーを起動してから
task dev  # ターミナル1
task seed # ターミナル2
```

投入されるデータ:
- 取引3件（収入2件、支出1件）
- 仕訳2件

### 明細（未仕訳チェック）の使用例

#### 1. 未仕訳明細のチェックプログラム

```bash
cd examples
FREEE_API_URL=http://localhost:8081 go run unbooked_checker.go
```

出力例:
```
Access Token: xxx...
Response Status: 200
未仕分け明細: 3件
```

#### 2. テストデータの投入

```bash
# エミュレータを起動
PORT=8081 task run

# 別ターミナルで
API_URL=http://localhost:8081 ./scripts/seed_wallet_txns.sh
```

投入されるデータ:
- 未仕訳明細: 3件
- 仕訳済み明細: 1件

#### 3. Google Chat通知（オプション）

環境変数でWebhook URLを設定すると、未仕訳がある場合に通知されます:

```bash
export GOOGLE_CHAT_WEBHOOK="https://chat.googleapis.com/v1/spaces/..."
FREEE_API_URL=http://localhost:8081 go run examples/unbooked_checker.go
```

## 開発

### タスク一覧

```bash
# すべてのタスクを表示
task

# テストを実行
task test

# 統合テストを実行
task test-integration

# 並列テストを実行（go-portallocを使用）
task test-parallel

# ストレステストを実行（並列20）
task test-parallel-stress

# シナリオテストを実行
task test-scenario

# すべてのテストを実行
task test-all

# リントを実行
task lint-all

# コードをフォーマット
task fmt

# サンプルデータを投入
task seed

# データベースをクリーン
task clean-data
```

### テスト

```bash
# すべてのテストを実行
task test-all

# カバレッジを表示
task test-coverage
```

テストの種類:
- **ユニットテスト**: 各コンポーネントの単体テスト
- **統合テスト**: API全体の動作テスト
- **並列テスト**: [go-portalloc](https://github.com/pigeonworks-llc/go-portalloc)を使用した並列実行テスト（ポート競合なし）
- **シナリオテスト**: 実際のビジネスフローをシミュレート

### プロジェクト構造

```
freee-emulator/
├── cmd/
│   └── server/          # メインエントリーポイント
├── internal/
│   ├── api/             # APIハンドラー
│   ├── models/          # データモデル
│   ├── oauth/           # OAuth2エミュレーション
│   └── store/           # bboltデータストア
├── test/
│   └── integration/     # 統合テスト・シナリオテスト
├── scripts/
│   └── seed.sh          # サンプルデータ投入スクリプト
├── testdata/            # テスト用サンプルデータ
├── data/                # データベースファイル
├── .golangci.yml        # golangci-lint設定
├── .go-arch-lint.yml    # アーキテクチャlint設定
└── Taskfile.yml         # タスク定義
```

## Docker

### Dockerイメージのビルド

```bash
task docker-build
```

### Dockerコンテナの実行

```bash
task docker-run
```

または直接：

```bash
docker run -p 8080:8080 -v $(pwd)/data:/app/data freee-emulator:latest
```

## ライセンス

MIT License

## 貢献

Issue、Pull Requestを歓迎します。
