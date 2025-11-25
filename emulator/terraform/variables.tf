variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP region for Cloud Run and Cloud Scheduler"
  type        = string
  default     = "asia-northeast1"
}

variable "service_name" {
  description = "Cloud Run service name"
  type        = string
  default     = "freee-checker"
}

variable "container_image" {
  description = "Container image URL (e.g., gcr.io/PROJECT_ID/freee-checker:latest)"
  type        = string
}

variable "freee_client_id" {
  description = "freee OAuth2 Client ID"
  type        = string
  sensitive   = true
}

variable "freee_client_secret" {
  description = "freee OAuth2 Client Secret"
  type        = string
  sensitive   = true
}

variable "freee_company_id" {
  description = "freee Company ID (事業所ID)"
  type        = string
}

variable "google_chat_webhook" {
  description = "Google Chat Webhook URL (optional)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "schedule" {
  description = "Cloud Scheduler cron schedule (default: Mon and Thu at 9am JST)"
  type        = string
  default     = "0 9 * * 1,4"
}

variable "time_zone" {
  description = "Time zone for Cloud Scheduler"
  type        = string
  default     = "Asia/Tokyo"
}

variable "allow_unauthenticated" {
  description = "Allow unauthenticated access to Cloud Run (dev only, set false for production)"
  type        = bool
  default     = false
}

variable "alert_email" {
  description = "Email address for error alerts (optional)"
  type        = string
  default     = ""
}
