# Bootstrap Configuration for Terraform State Backend
# このファイルで Terraform State 用の GCS バケットを作成します

terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }

  # 初回はローカルstateを使用
  # backend "local" {}
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Terraform State用GCSバケット
resource "google_storage_bucket" "terraform_state" {
  name          = "${var.project_id}-terraform-state"
  location      = var.region
  force_destroy = false

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  # ライフサイクル管理（古いバージョンを削除）
  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      num_newer_versions = 5
      with_state         = "ARCHIVED"
    }
  }

  # 暗号化
  encryption {
    default_kms_key_name = null
  }

  # 削除保護
  lifecycle {
    prevent_destroy = true
  }
}

# State Lock用のバケット（オプション）
# Terraform は GCS の場合、自動的にロック機能を使用します

# IAM: 現在のユーザーにバケットへのアクセス権限を付与
resource "google_storage_bucket_iam_member" "terraform_state_admin" {
  bucket = google_storage_bucket.terraform_state.name
  role   = "roles/storage.admin"
  member = "user:${var.admin_email}"
}

# サービスアカウント用の権限（CI/CDで使用する場合）
resource "google_storage_bucket_iam_member" "terraform_state_service_account" {
  count  = var.service_account_email != "" ? 1 : 0
  bucket = google_storage_bucket.terraform_state.name
  role   = "roles/storage.admin"
  member = "serviceAccount:${var.service_account_email}"
}
