# Terraform State Backend Configuration
#
# このファイルは初回のみコメントアウトしておき、
# バケット作成後にコメントを外してください

# terraform {
#   backend "gcs" {
#     bucket = "your-project-id-terraform-state"
#     prefix = "freee-checker"
#   }
# }

# 初回セットアップ手順：
# 1. このファイルをコメントアウトしたまま terraform init
# 2. terraform apply で State バケットを作成
# 3. このファイルのコメントを外す
# 4. terraform init -migrate-state でStateをGCSに移行
