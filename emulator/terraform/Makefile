.PHONY: help init plan apply destroy fmt validate output clean

# デフォルトターゲット
help:
	@echo "freee Checker Terraform Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make init       - Initialize Terraform"
	@echo "  make plan       - Show execution plan"
	@echo "  make apply      - Apply changes"
	@echo "  make destroy    - Destroy all resources"
	@echo "  make fmt        - Format Terraform files"
	@echo "  make validate   - Validate configuration"
	@echo "  make output     - Show outputs"
	@echo "  make clean      - Clean Terraform cache"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. cp terraform.tfvars.example terraform.tfvars"
	@echo "  2. Edit terraform.tfvars with your values"
	@echo "  3. make init"
	@echo "  4. make plan"
	@echo "  5. make apply"

# Terraform初期化
init:
	terraform init

# 実行プランの確認
plan:
	terraform plan

# リソースの作成・更新
apply:
	terraform apply

# リソースの削除
destroy:
	terraform destroy

# コードフォーマット
fmt:
	terraform fmt -recursive

# 設定の検証
validate:
	terraform validate

# 出力の表示
output:
	@echo "=== Service URL ==="
	@terraform output -raw service_url
	@echo ""
	@echo ""
	@echo "=== Health Check URL ==="
	@terraform output -raw health_check_url
	@echo ""
	@echo ""
	@echo "=== Check Endpoint URL ==="
	@terraform output -raw check_endpoint_url
	@echo ""
	@echo ""
	@echo "=== OAuth Token Upload Command ==="
	@terraform output -raw upload_token_command
	@echo ""
	@echo ""
	@echo "=== Manual Trigger Command ==="
	@terraform output -raw manual_trigger_command
	@echo ""

# キャッシュのクリーンアップ
clean:
	rm -rf .terraform
	rm -f .terraform.lock.hcl
	rm -f terraform.tfstate.backup

# 完全なセットアップフロー
setup: init plan apply output
	@echo ""
	@echo "✅ Setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Run initial OAuth authentication locally"
	@echo "2. Upload token to Secret Manager:"
	@echo "   $$(terraform output -raw upload_token_command)"
	@echo "3. Test the service:"
	@echo "   curl -X POST $$(terraform output -raw check_endpoint_url)"
