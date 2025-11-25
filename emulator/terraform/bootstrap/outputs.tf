output "state_bucket_name" {
  description = "Name of the Terraform state bucket"
  value       = google_storage_bucket.terraform_state.name
}

output "state_bucket_url" {
  description = "URL of the Terraform state bucket"
  value       = google_storage_bucket.terraform_state.url
}

output "backend_config" {
  description = "Backend configuration to use in main Terraform"
  value = <<-EOT
    terraform {
      backend "gcs" {
        bucket = "${google_storage_bucket.terraform_state.name}"
        prefix = "freee-checker"
      }
    }
  EOT
}

output "next_steps" {
  description = "Next steps to configure backend"
  value = <<-EOT
    ✅ Terraform State bucket created!

    Next steps:
    1. Go to the main terraform directory:
       cd ..

    2. Edit backend.tf and uncomment the backend configuration:
       terraform {
         backend "gcs" {
           bucket = "${google_storage_bucket.terraform_state.name}"
           prefix = "freee-checker"
         }
       }

    3. Initialize Terraform with the new backend:
       terraform init -migrate-state

    4. Confirm the migration when prompted.

    Your Terraform state will now be stored in GCS!
  EOT
}
