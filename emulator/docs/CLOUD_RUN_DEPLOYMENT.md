# Cloud Run + Cloud Scheduler デプロイガイド

このガイドでは、freee未仕訳チェッカーをCloud Run + Cloud Schedulerで運用する方法を説明します。

## アーキテクチャ

```
Cloud Scheduler (週2回実行)
    ↓ HTTP POST /check
Cloud Run (freee-checker)
    ↓ OAuth2 API呼び出し
freee API
    ↓ 通知
Google Chat Webhook
```

## 前提条件

- Google Cloud Project
- gcloud CLI インストール済み
- Docker インストール済み
- freee OAuth2アプリケーション作成済み

## セットアップ手順

### 1. GCP プロジェクトの設定

```bash
# プロジェクトIDを設定
export PROJECT_ID="your-project-id"
gcloud config set project $PROJECT_ID

# 必要なAPIを有効化
gcloud services enable \
  run.googleapis.com \
  cloudscheduler.googleapis.com \
  secretmanager.googleapis.com \
  cloudbuild.googleapis.com
```

### 2. 環境変数の設定

```bash
# freee OAuth2設定
export FREEE_CLIENT_ID="your_client_id"
export FREEE_CLIENT_SECRET="your_client_secret"
export FREEE_COMPANY_ID="123456"

# Google Chat Webhook（オプション）
export GOOGLE_CHAT_WEBHOOK="https://chat.googleapis.com/v1/spaces/..."

# Cloud Run設定
export REGION="asia-northeast1"  # 東京リージョン
export SERVICE_NAME="freee-checker"
```

### 3. 初回OAuth認証（ローカルで実行）

Cloud Runにデプロイする前に、ローカルでトークンを取得します。

```bash
# 環境変数設定（2FAも含む）
export FREEE_LOGIN_EMAIL="your@email.com"
export FREEE_LOGIN_PASSWORD="your_password"
export FREEE_TOTP_SECRET="BASE32_SECRET"

# Playwrightインストール
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install

# 自動認証実行
go run examples/freee_oauth_auto.go
```

これで `~/.freee/token.json` にトークンが保存されます。

### 4. Secret Managerにトークンを保存（推奨）

```bash
# トークンファイルをSecret Managerに保存
gcloud secrets create freee-oauth-token \
  --data-file=$HOME/.freee/token.json \
  --project=$PROJECT_ID

# Secret Managerへのアクセス権限を付与
gcloud secrets add-iam-policy-binding freee-oauth-token \
  --member="serviceAccount:${PROJECT_ID}@appspot.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor" \
  --project=$PROJECT_ID
```

### 5. Cloud Runへデプロイ

```bash
# デプロイスクリプトを実行
chmod +x scripts/deploy-cloud-run.sh
PROJECT_ID=$PROJECT_ID ./scripts/deploy-cloud-run.sh
```

デプロイスクリプトは以下を自動実行します：
1. Dockerイメージのビルド
2. GCRへのプッシュ
3. Cloud Runへのデプロイ
4. Cloud Schedulerジョブの作成

### 6. 動作確認

```bash
# サービスURLを取得
SERVICE_URL=$(gcloud run services describe freee-checker \
  --region asia-northeast1 \
  --format 'value(status.url)')

# ヘルスチェック
curl $SERVICE_URL/health

# 手動でチェック実行
curl -X POST $SERVICE_URL/check
```

## Cloud Schedulerの設定

デフォルトでは週2回（月曜と木曜の9時JST）実行されます。

### スケジュールの変更

```bash
# 毎日9時に変更
gcloud scheduler jobs update http freee-checker-job \
  --schedule "0 9 * * *" \
  --location asia-northeast1
```

### 手動実行

```bash
gcloud scheduler jobs run freee-checker-job \
  --location asia-northeast1
```

## トラブルシューティング

### ログの確認

```bash
# Cloud Runのログを確認
gcloud logs read \
  --project=$PROJECT_ID \
  --limit=50 \
  --format=json \
  "resource.type=cloud_run_revision AND resource.labels.service_name=freee-checker"

# Cloud Schedulerのログを確認
gcloud logs read \
  --project=$PROJECT_ID \
  --limit=20 \
  "resource.type=cloud_scheduler_job"
```

### トークンのリフレッシュ失敗

トークンが期限切れの場合：

1. ローカルで再認証を実行
   ```bash
   go run examples/freee_oauth_auto.go
   ```

2. 新しいトークンをSecret Managerに保存
   ```bash
   gcloud secrets versions add freee-oauth-token \
     --data-file=$HOME/.freee/token.json
   ```

3. Cloud Runを再デプロイ
   ```bash
   ./scripts/deploy-cloud-run.sh
   ```

### メモリ不足エラー

Playwrightを使う場合、メモリを増やす必要があります：

```bash
gcloud run services update freee-checker \
  --memory 1Gi \
  --region asia-northeast1
```

## コスト見積もり

### Cloud Run
- リクエスト数: 週2回 × 4週 = 月8回
- 実行時間: 1回あたり約10秒
- メモリ: 512Mi
- **月額: 無料枠内（$0）**

### Cloud Scheduler
- ジョブ数: 1個
- 実行回数: 月8回
- **月額: 無料枠内（$0）**

### ストレージ（Secret Manager）
- シークレット数: 1個
- アクセス回数: 月8回
- **月額: $0.06（約10円）**

**合計: 約10円/月**

## セキュリティのベストプラクティス

### 1. IAM権限の最小化

```bash
# 専用サービスアカウントを作成
gcloud iam service-accounts create freee-checker-sa \
  --display-name="freee Checker Service Account"

# 必要な権限のみ付与
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:freee-checker-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# Cloud Runに適用
gcloud run services update freee-checker \
  --service-account=freee-checker-sa@${PROJECT_ID}.iam.gserviceaccount.com \
  --region=asia-northeast1
```

### 2. 認証の強化

Cloud Schedulerからのリクエストのみ許可：

```bash
# Cloud Runを認証必須に変更
gcloud run services update freee-checker \
  --no-allow-unauthenticated \
  --region=asia-northeast1

# Cloud Schedulerに権限付与
gcloud run services add-iam-policy-binding freee-checker \
  --member="serviceAccount:${PROJECT_ID}@appspot.gserviceaccount.com" \
  --role="roles/run.invoker" \
  --region=asia-northeast1
```

### 3. シークレットの管理

環境変数ではなくSecret Managerを使用：

```bash
# シークレット作成
echo -n "$FREEE_CLIENT_SECRET" | gcloud secrets create freee-client-secret --data-file=-

# Cloud Runから参照
gcloud run services update freee-checker \
  --update-secrets=FREEE_CLIENT_SECRET=freee-client-secret:latest \
  --region=asia-northeast1
```

## まとめ

Cloud Run + Cloud Schedulerの運用なら：
- ✅ サーバーレスで運用不要
- ✅ 月額10円程度の低コスト
- ✅ 自動スケーリング
- ✅ 構造化ログ（Cloud Logging連携）
- ✅ トークン自動リフレッシュ

週2回の実行で、トークンは90日間有効なので、手動再認証は年4回のみです。
