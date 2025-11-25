variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP region for the state bucket"
  type        = string
  default     = "asia-northeast1"
}

variable "admin_email" {
  description = "Email address of the admin user who will manage Terraform state"
  type        = string
}

variable "service_account_email" {
  description = "Service account email for CI/CD (optional)"
  type        = string
  default     = ""
}
