# Terraform State Bootstrap

このディレクトリは、Terraform Stateを保存するためのGCSバケットを作成します。

## なぜ必要か？

- **ローカルstateの問題点**
  - チーム開発で共有できない
  - バージョン管理できない（.gitignoreされる）
  - バックアップがない
  - ロック機能がない（同時実行で壊れる可能性）

- **GCS backendのメリット**
  - ✅ チーム全体で状態を共有
  - ✅ 自動バージョニング
  - ✅ 暗号化
  - ✅ ロック機能（同時実行を防止）
  - ✅ GCPとの親和性が高い

## セットアップ手順

### 1. 設定ファイルをコピー

```bash
cd terraform/bootstrap
cp terraform.tfvars.example terraform.tfvars
```

### 2. terraform.tfvarsを編集

```bash
vim terraform.tfvars
```

必要な値：
- `project_id` - GCPプロジェクトID
- `admin_email` - 自分のメールアドレス
  ```bash
  gcloud config get-value account
  ```

### 3. Terraformを実行

```bash
# 初期化
terraform init

# プラン確認
terraform plan

# バケット作成
terraform apply
```

### 4. 出力を確認

```bash
terraform output next_steps
```

次のステップが表示されます。

### 5. メインTerraformでBackendを設定

```bash
# メインディレクトリに移動
cd ..

# backend.tfのコメントを外す
vim backend.tf
```

以下のようにコメントを外します：

```hcl
terraform {
  backend "gcs" {
    bucket = "your-project-id-terraform-state"
    prefix = "freee-checker"
  }
}
```

### 6. Stateを移行

```bash
# 既存のローカルstateをGCSに移行
terraform init -migrate-state
```

プロンプトで `yes` を入力します。

### 7. 確認

```bash
# GCSバケットにstateが保存されているか確認
gsutil ls gs://your-project-id-terraform-state/freee-checker/
```

## トラブルシューティング

### エラー: Bucket name already exists

バケット名はグローバルでユニークである必要があります。
`main.tf`でバケット名を変更してください：

```hcl
resource "google_storage_bucket" "terraform_state" {
  name = "${var.project_id}-tf-state-${random_id.suffix.hex}"
  ...
}

resource "random_id" "suffix" {
  byte_length = 4
}
```

### 権限エラー

自分のアカウントに必要な権限を付与：

```bash
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="user:$(gcloud config get-value account)" \
  --role="roles/storage.admin"
```

## ベストプラクティス

### 1. バケットの削除保護

`lifecycle.prevent_destroy = true` で誤削除を防止しています。
削除する場合は、この設定を外してから `terraform destroy` を実行。

### 2. バージョニング

古いバージョンを5つまで保持し、それ以上は自動削除されます。

### 3. 暗号化

GCSのデフォルト暗号化を使用しています。
より高度なセキュリティが必要な場合は、Cloud KMSを設定できます。

### 4. CI/CD対応

GitHub ActionsやCloud Buildでterraformを実行する場合：

```bash
# サービスアカウント作成
gcloud iam service-accounts create terraform-ci \
  --display-name="Terraform CI/CD"

# 権限付与
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:terraform-ci@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/editor"

# terraform.tfvarsに追加
service_account_email = "terraform-ci@your-project-id.iam.gserviceaccount.com"

# 再度apply
terraform apply
```

## クリーンアップ

Stateバケットを削除する場合（**注意: 既存のStateも削除されます**）：

```bash
# 1. lifecycle.prevent_destroyをfalseに変更
vim main.tf

# 2. 削除
terraform destroy

# 3. バケットを強制削除（バージョンも含む）
gsutil -m rm -r gs://your-project-id-terraform-state
```

## 参考

- [Terraform GCS Backend](https://www.terraform.io/language/settings/backends/gcs)
- [Google Cloud Storage Best Practices](https://cloud.google.com/storage/docs/best-practices)
