# Terraform Configuration for freee Checker

このディレクトリには、freee未仕訳チェッカーをCloud Run + Cloud Schedulerにデプロイするためのterraform設定が含まれています。

## ディレクトリ構成

```
terraform/
├── main.tf                      # メインのリソース定義
├── variables.tf                 # 変数定義
├── outputs.tf                   # 出力定義
├── backend.tf                   # Stateバックエンド設定
├── terraform.tfvars.example     # 設定例
├── .gitignore                   # Git除外設定
├── Makefile                     # 便利コマンド集
├── README.md                    # このファイル
└── bootstrap/                   # State用GCSバケット作成
    ├── main.tf
    ├── variables.tf
    ├── outputs.tf
    └── README.md
```

## 構成リソース

以下のGCPリソースが作成されます：

- **Cloud Run Service** - HTTPサーバー版チェッカー
- **Cloud Scheduler Job** - 週2回の定期実行
- **Secret Manager Secrets** - OAuth認証情報の安全な保管
- **Service Account** - 最小権限でのリソースアクセス
- **IAM Bindings** - 適切な権限設定
- **Monitoring Alert Policy** - ERRORログのメール通知（オプション）
- **Notification Channel** - アラート通知先メールアドレス（オプション）

## 前提条件

### 1. ツールのインストール

```bash
# Terraformのインストール（Homebrewを使用）
brew install terraform

# gcloud CLIのインストール
brew install google-cloud-sdk

# gcloud認証
gcloud auth login
gcloud auth application-default login
```

### 2. GCPプロジェクトの準備

```bash
# プロジェクトIDを設定
export PROJECT_ID="your-gcp-project-id"

# プロジェクトを設定
gcloud config set project $PROJECT_ID

# 課金アカウントの有効化（必要に応じて）
gcloud alpha billing accounts list
gcloud alpha billing projects link $PROJECT_ID --billing-account=BILLING_ACCOUNT_ID
```

### 3. Dockerイメージのビルド＆プッシュ

Terraformを実行する前に、Dockerイメージを作成してGCRにプッシュしておく必要があります。

```bash
# プロジェクトルートから実行
cd ..

# Dockerイメージをビルド
docker build -f Dockerfile.checker -t gcr.io/$PROJECT_ID/freee-checker:latest .

# GCRにプッシュ
docker push gcr.io/$PROJECT_ID/freee-checker:latest

# terraformディレクトリに戻る
cd terraform
```

## セットアップ手順

### 0. （推奨）Terraform State用GCSバケットの作成

本番環境では、Terraform StateをGCS（Google Cloud Storage）に保存することを強く推奨します。

```bash
# bootstrapディレクトリに移動
cd bootstrap

# 設定ファイルをコピー
cp terraform.tfvars.example terraform.tfvars

# terraform.tfvarsを編集
vim terraform.tfvars

# State用バケットを作成
terraform init
terraform apply

# 次のステップを確認
terraform output next_steps

# メインディレクトリに戻る
cd ..

# backend.tfのコメントを外して、terraform init -migrate-state を実行
```

詳細は `bootstrap/README.md` を参照してください。

### 1. 設定ファイルのコピー

```bash
cp terraform.tfvars.example terraform.tfvars
```

### 2. terraform.tfvarsの編集

```bash
# エディタで開いて値を設定
vim terraform.tfvars
```

必要な値：
- `project_id` - GCPプロジェクトID
- `container_image` - Dockerイメージのパス
- `freee_client_id` - freee OAuth2 Client ID
- `freee_client_secret` - freee OAuth2 Client Secret
- `freee_company_id` - freee事業所ID
- `google_chat_webhook` - Google Chat Webhook URL（オプション）
- `alert_email` - エラーアラート通知先メールアドレス（オプション）

### 3. Terraformの初期化

```bash
terraform init
```

### 4. プランの確認

```bash
terraform plan
```

作成されるリソースを確認します。

### 5. リソースの作成

```bash
terraform apply
```

`yes` を入力して実行します。

### 6. 初回OAuth認証

Cloud Runにデプロイされましたが、まだOAuthトークンがありません。
ローカルで初回認証を実行してトークンを取得します。

```bash
# プロジェクトルートで実行
cd ..

# 環境変数設定
export FREEE_CLIENT_ID="your_client_id"
export FREEE_CLIENT_SECRET="your_client_secret"
export FREEE_LOGIN_EMAIL="your@email.com"
export FREEE_LOGIN_PASSWORD="your_password"
export FREEE_TOTP_SECRET="BASE32_SECRET"

# 自動認証実行
go run examples/freee_oauth_auto.go

# トークンをSecret Managerにアップロード
gcloud secrets versions add freee-oauth-token \
  --data-file=$HOME/.freee/token.json \
  --project=$PROJECT_ID
```

### 7. 動作確認

```bash
# 出力された情報を確認
terraform output

# ヘルスチェック
curl $(terraform output -raw health_check_url)

# 手動でチェックを実行
curl -X POST $(terraform output -raw check_endpoint_url)

# Cloud Schedulerを手動実行
eval $(terraform output -raw manual_trigger_command)
```

## リソースの更新

設定を変更してリソースを更新する場合：

```bash
# terraform.tfvarsを編集
vim terraform.tfvars

# プランを確認
terraform plan

# 適用
terraform apply
```

## リソースの削除

全てのリソースを削除する場合：

```bash
terraform destroy
```

**注意**: Secret Managerのシークレットは即座に削除されず、30日間の削除予定期間があります。

## トラブルシューティング

### エラー: API not enabled

以下のコマンドで必要なAPIを手動で有効化してください：

```bash
gcloud services enable run.googleapis.com
gcloud services enable cloudscheduler.googleapis.com
gcloud services enable secretmanager.googleapis.com
gcloud services enable cloudbuild.googleapis.com
```

### エラー: Permission denied

サービスアカウントに必要な権限がない場合：

```bash
# 自分のアカウントに権限付与
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="user:$(gcloud config get-value account)" \
  --role="roles/editor"
```

### Cloud Schedulerが実行されない

Cloud Schedulerのログを確認：

```bash
gcloud logging read \
  "resource.type=cloud_scheduler_job" \
  --limit=20 \
  --project=$PROJECT_ID
```

### Cloud Runのログ確認

```bash
gcloud logging read \
  "resource.type=cloud_run_revision AND resource.labels.service_name=freee-checker" \
  --limit=50 \
  --format=json \
  --project=$PROJECT_ID
```

## エラー監視とアラート

### ログベースアラート（ERROR以上）

`alert_email`を設定すると、ERROR以上のログが出力された際に自動でメール通知されます。

#### 設定方法

```hcl
# terraform.tfvars
alert_email = "your-email@example.com"
```

#### アラートの特徴

- **対象**: Cloud Run（freee-checker）のERROR、CRITICAL、ALERTレベルのログ
- **通知頻度**: 5分に1回まで（スパム防止）
- **自動クローズ**: 30分後に自動的にクローズ
- **通知内容**: エラーログの内容とトラブルシューティング手順

#### アラート通知例

エラーが発生すると、以下のような内容のメールが届きます：

```
freee Checkerでエラーが発生しました。

以下のコマンドでログを確認してください:
gcloud logging read \
  'resource.type="cloud_run_revision" AND resource.labels.service_name="freee-checker" AND severity>=ERROR' \
  --limit=50 \
  --project=your-project-id

よくあるエラー:
- OAuth認証エラー: トークンの更新が必要
- API制限: freee APIのレート制限に達した
- タイムアウト: 処理時間が長すぎる
```

#### アラートポリシーの確認

```bash
# アラートポリシーの一覧表示
gcloud alpha monitoring policies list --project=$PROJECT_ID

# 通知チャネルの確認
gcloud alpha monitoring channels list --project=$PROJECT_ID
```

#### アラート無効化

一時的にアラートを無効化したい場合：

```bash
# GCPコンソールでアラートポリシーを無効化
# または terraform.tfvars で alert_email を空文字に設定
alert_email = ""

# 再度apply
terraform apply
```

### Cloud Loggingでのエラー確認

ERRORログを直接確認する場合：

```bash
# 最新50件のERRORログを表示
gcloud logging read \
  'resource.type="cloud_run_revision"
   AND resource.labels.service_name="freee-checker"
   AND severity>=ERROR' \
  --limit=50 \
  --format=json \
  --project=$PROJECT_ID

# 過去24時間のERRORログを表示
gcloud logging read \
  'resource.type="cloud_run_revision"
   AND resource.labels.service_name="freee-checker"
   AND severity>=ERROR
   AND timestamp>="2024-01-01T00:00:00Z"' \
  --limit=100 \
  --project=$PROJECT_ID
```

## カスタマイズ

### スケジュールの変更

`terraform.tfvars`の`schedule`を変更：

```hcl
# 毎日9時に変更
schedule = "0 9 * * *"

# 月・水・金の10時
schedule = "0 10 * * 1,3,5"
```

### リージョンの変更

```hcl
# シンガポールリージョン
region = "asia-southeast1"

# アメリカ（オレゴン）
region = "us-west1"
```

### メモリ・CPUの変更

`main.tf`の`google_cloud_run_v2_service`リソースを編集：

```hcl
resources {
  limits = {
    cpu    = "2"
    memory = "1Gi"
  }
}
```

## セキュリティのベストプラクティス

### 1. 本番環境では認証必須

```hcl
allow_unauthenticated = false
```

### 2. 最小権限の原則

サービスアカウントには必要最小限の権限のみ付与されています。

### 3. シークレットの管理

- terraform.tfvarsは`.gitignore`に含まれています
- 機密情報は全てSecret Managerで管理
- シークレットへのアクセスはサービスアカウント経由のみ

### 4. Terraformステートの管理

本番環境ではGCSバックエンドを使用：

```hcl
terraform {
  backend "gcs" {
    bucket = "your-terraform-state-bucket"
    prefix = "freee-checker"
  }
}
```

## コスト見積もり

月間コスト（週2回実行の場合）：

- Cloud Run: 無料枠内（$0）
- Cloud Scheduler: 無料枠内（$0）
- Secret Manager: $0.06（約10円）
- **合計: 約10円/月**

## 参考リンク

- [Terraform Google Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs)
- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud Scheduler Documentation](https://cloud.google.com/scheduler/docs)
- [Secret Manager Documentation](https://cloud.google.com/secret-manager/docs)
