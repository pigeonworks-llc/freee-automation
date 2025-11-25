terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }

  # バックエンド設定（オプション - GCSを使う場合）
  # backend "gcs" {
  #   bucket = "your-terraform-state-bucket"
  #   prefix = "freee-checker"
  # }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# サービスアカウントの作成
resource "google_service_account" "freee_checker" {
  account_id   = "freee-checker-sa"
  display_name = "freee Checker Service Account"
  description  = "Service account for freee unbooked transaction checker"
}

# Secret Manager: freee Client ID
resource "google_secret_manager_secret" "freee_client_id" {
  secret_id = "freee-client-id"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "freee_client_id" {
  secret      = google_secret_manager_secret.freee_client_id.id
  secret_data = var.freee_client_id
}

# Secret Manager: freee Client Secret
resource "google_secret_manager_secret" "freee_client_secret" {
  secret_id = "freee-client-secret"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "freee_client_secret" {
  secret      = google_secret_manager_secret.freee_client_secret.id
  secret_data = var.freee_client_secret
}

# Secret Manager: Google Chat Webhook
resource "google_secret_manager_secret" "google_chat_webhook" {
  count     = var.google_chat_webhook != "" ? 1 : 0
  secret_id = "google-chat-webhook"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "google_chat_webhook" {
  count       = var.google_chat_webhook != "" ? 1 : 0
  secret      = google_secret_manager_secret.google_chat_webhook[0].id
  secret_data = var.google_chat_webhook
}

# Secret Manager: OAuth Token（初回は空、後で手動設定）
resource "google_secret_manager_secret" "oauth_token" {
  secret_id = "freee-oauth-token"

  replication {
    auto {}
  }
}

# IAM: Service AccountにSecret Managerへのアクセス権限を付与
resource "google_secret_manager_secret_iam_member" "freee_client_id_accessor" {
  secret_id = google_secret_manager_secret.freee_client_id.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.freee_checker.email}"
}

resource "google_secret_manager_secret_iam_member" "freee_client_secret_accessor" {
  secret_id = google_secret_manager_secret.freee_client_secret.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.freee_checker.email}"
}

resource "google_secret_manager_secret_iam_member" "google_chat_webhook_accessor" {
  count     = var.google_chat_webhook != "" ? 1 : 0
  secret_id = google_secret_manager_secret.google_chat_webhook[0].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.freee_checker.email}"
}

resource "google_secret_manager_secret_iam_member" "oauth_token_accessor" {
  secret_id = google_secret_manager_secret.oauth_token.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.freee_checker.email}"
}

# Cloud Runサービス
resource "google_cloud_run_v2_service" "freee_checker" {
  name     = var.service_name
  location = var.region

  template {
    service_account = google_service_account.freee_checker.email

    timeout = "300s"

    scaling {
      max_instance_count = 1
      min_instance_count = 0
    }

    containers {
      image = var.container_image

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }

      env {
        name  = "FREEE_COMPANY_ID"
        value = var.freee_company_id
      }

      env {
        name  = "FREEE_AUTO_REAUTH"
        value = "true"
      }

      env {
        name = "FREEE_CLIENT_ID"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.freee_client_id.secret_id
            version = "latest"
          }
        }
      }

      env {
        name = "FREEE_CLIENT_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.freee_client_secret.secret_id
            version = "latest"
          }
        }
      }

      dynamic "env" {
        for_each = var.google_chat_webhook != "" ? [1] : []
        content {
          name = "GOOGLE_CHAT_WEBHOOK"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.google_chat_webhook[0].secret_id
              version = "latest"
            }
          }
        }
      }
    }
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }

  depends_on = [
    google_secret_manager_secret_iam_member.freee_client_id_accessor,
    google_secret_manager_secret_iam_member.freee_client_secret_accessor,
  ]
}

# Cloud Run IAM: 認証なしアクセス（開発用 - 本番では削除推奨）
resource "google_cloud_run_v2_service_iam_member" "public_access" {
  count    = var.allow_unauthenticated ? 1 : 0
  name     = google_cloud_run_v2_service.freee_checker.name
  location = google_cloud_run_v2_service.freee_checker.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Cloud Run IAM: Cloud Schedulerからのアクセス
resource "google_cloud_run_v2_service_iam_member" "scheduler_invoker" {
  name     = google_cloud_run_v2_service.freee_checker.name
  location = google_cloud_run_v2_service.freee_checker.location
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.freee_checker.email}"
}

# Cloud Schedulerジョブ
resource "google_cloud_scheduler_job" "freee_checker" {
  name             = "${var.service_name}-job"
  description      = "Trigger freee checker twice a week"
  schedule         = var.schedule
  time_zone        = var.time_zone
  attempt_deadline = "320s"
  region           = var.region

  http_target {
    http_method = "POST"
    uri         = "${google_cloud_run_v2_service.freee_checker.uri}/check"

    oidc_token {
      service_account_email = google_service_account.freee_checker.email
    }
  }

  retry_config {
    retry_count = 1
  }

  depends_on = [google_cloud_run_v2_service.freee_checker]
}

# 必要なAPIの有効化
resource "google_project_service" "required_apis" {
  for_each = toset([
    "run.googleapis.com",
    "cloudscheduler.googleapis.com",
    "secretmanager.googleapis.com",
    "cloudbuild.googleapis.com",
    "monitoring.googleapis.com",
  ])

  service            = each.key
  disable_on_destroy = false
}

# メール通知チャネル（アラート用）
resource "google_monitoring_notification_channel" "email" {
  count        = var.alert_email != "" ? 1 : 0
  display_name = "freee Checker Error Alert Email"
  type         = "email"

  labels = {
    email_address = var.alert_email
  }

  enabled = true
}

# ログベースアラートポリシー（ERROR以上）
resource "google_monitoring_alert_policy" "error_logs" {
  count        = var.alert_email != "" ? 1 : 0
  display_name = "freee Checker - Error Logs Detected"
  combiner     = "OR"

  conditions {
    display_name = "ERROR or higher severity logs"

    condition_matched_log {
      filter = <<-EOT
        resource.type="cloud_run_revision"
        resource.labels.service_name="${var.service_name}"
        severity>=ERROR
      EOT
    }
  }

  alert_strategy {
    notification_rate_limit {
      period = "300s" # 5分に1回まで通知
    }

    auto_close = "1800s" # 30分後に自動クローズ
  }

  notification_channels = [
    google_monitoring_notification_channel.email[0].id
  ]

  documentation {
    content = <<-EOT
      freee Checkerでエラーが発生しました。

      以下のコマンドでログを確認してください:
      ```
      gcloud logging read \
        'resource.type="cloud_run_revision" AND resource.labels.service_name="${var.service_name}" AND severity>=ERROR' \
        --limit=50 \
        --project=${var.project_id}
      ```

      よくあるエラー:
      - OAuth認証エラー: トークンの更新が必要
      - API制限: freee APIのレート制限に達した
      - タイムアウト: 処理時間が長すぎる
    EOT
  }

  enabled = true
}
