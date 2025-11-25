output "service_url" {
  description = "Cloud Run service URL"
  value       = google_cloud_run_v2_service.freee_checker.uri
}

output "service_name" {
  description = "Cloud Run service name"
  value       = google_cloud_run_v2_service.freee_checker.name
}

output "service_account_email" {
  description = "Service account email"
  value       = google_service_account.freee_checker.email
}

output "scheduler_job_name" {
  description = "Cloud Scheduler job name"
  value       = google_cloud_scheduler_job.freee_checker.name
}

output "oauth_token_secret_id" {
  description = "Secret Manager ID for OAuth token (manually set token here)"
  value       = google_secret_manager_secret.oauth_token.secret_id
}

output "health_check_url" {
  description = "Health check endpoint"
  value       = "${google_cloud_run_v2_service.freee_checker.uri}/health"
}

output "check_endpoint_url" {
  description = "Check endpoint URL (POST to trigger check)"
  value       = "${google_cloud_run_v2_service.freee_checker.uri}/check"
}

output "manual_trigger_command" {
  description = "Command to manually trigger Cloud Scheduler job"
  value       = "gcloud scheduler jobs run ${google_cloud_scheduler_job.freee_checker.name} --location=${var.region}"
}

output "upload_token_command" {
  description = "Command to upload OAuth token to Secret Manager"
  value       = "gcloud secrets versions add ${google_secret_manager_secret.oauth_token.secret_id} --data-file=~/.freee/token.json"
}
