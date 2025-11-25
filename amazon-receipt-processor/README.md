# Amazon Receipt Processor

Amazon領収書PDFを自動的に処理し、freee取引と照合して、レシートをアップロードするツールです。

## 機能

- PDFから注文番号・日付・金額を抽出
- freee取引と自動照合（日付±3日、金額完全一致）
- freee APIへのレシート自動アップロード（必須）
- Beancountファイルへのdocumentメタデータ追加

## セットアップ

### 1. 依存関係のインストール

```bash
npm install
```

### 2. 環境変数の設定

`.env.local`ファイルを作成（`.env.example`を参考に）:

```bash
cp .env.example .env.local
```

設定項目:
- `FREEE_API_URL`: freee APIエンドポイント
- `FREEE_ACCESS_TOKEN`: アクセストークン
- `FREEE_COMPANY_ID`: 会社ID
- `AMAZON_DOWNLOAD_DIR`: PDFダウンロード先ディレクトリ
- `BEANCOUNT_DIR`: Beancountファイルのディレクトリ

### 3. ビルド

```bash
npm run build
```

## 使い方

### 1. Chrome拡張機能でAmazon領収書をダウンロード

「アマゾン注文履歴フィルタ」などのChrome拡張機能を使用して、
領収書PDFを`~/Downloads/`にダウンロードします。

### 2. レシート処理の実行

#### ドライランモード（確認のみ）

```bash
npm run process -- --dry-run
```

#### 実際の処理

```bash
npm run process
```

## 処理フロー

1. **PDF検出**: `~/Downloads/領収書*.pdf`を検索
2. **PDF解析**: 注文番号・日付・金額を抽出
3. **取引検索**: freee APIから全取引を取得
4. **マッチング**: 日付±3日 AND 金額完全一致で照合
5. **結果**:
   - **1件一致**: 自動処理
     - PDF移動: `documents/2025/amazon-{order-id}.pdf`
     - freee APIへアップロード
     - Beancount更新
   - **複数一致**: エラー（手動確認が必要）
   - **0件**: スキップ

## 出力例

```
=== Amazon Receipt Processor ===

📁 Searching for PDFs in: /Users/shunichi/Downloads
   Found 1 PDF file(s)

📡 Fetching deals from freee...
   Retrieved 3 deal(s)

📄 Processing: 領収書-123-4567890-1234567.pdf
   Order: 123-4567890-1234567
   Date: 2025-02-15
   Amount: ¥12,980
   ✓ Matched with Deal ID 5
   📦 Moving PDF to documents directory...
   ✓ Moved to: documents/2025/amazon-123-4567890-1234567.pdf
   ☁️  Uploading to freee...
   ✓ Uploaded (Receipt ID: 42)
   📝 Updating Beancount...
   ✓ Updated: ../beancount/2025/2025-02.beancount

=== Summary ===
Total PDFs: 1
✓ Processed: 1
⚠️  Skipped: 0
❌ Errors: 0
```

## トラブルシューティング

### PDFが検出されない

- ファイル名パターンが `領収書*.pdf` であることを確認
- `AMAZON_DOWNLOAD_DIR` の設定を確認

### 取引が見つからない

- 日付が±3日以内か確認
- 金額が完全一致しているか確認（税込金額）
- freee側で取引が登録されているか確認

### freee APIエラー

- `FREEE_ACCESS_TOKEN` が有効か確認
- トークンが期限切れの場合は再取得

## 開発

### ディレクトリ構造

```
src/
├── cli.ts                  # CLIエントリーポイント
├── pdf-parser.ts           # PDF解析
├── matcher.ts              # 取引マッチング
├── freee-uploader.ts       # freee API連携
├── beancount-updater.ts    # Beancount更新
└── types.ts                # 型定義
```

### テスト実行

```bash
npm run dev -- --dry-run
```
