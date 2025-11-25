.PHONY: help init plan apply destroy output

help:
	@echo "Terraform State Bootstrap Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make init     - Initialize Terraform"
	@echo "  make plan     - Show execution plan"
	@echo "  make apply    - Create state bucket"
	@echo "  make output   - Show next steps"
	@echo "  make destroy  - Delete state bucket (DANGEROUS!)"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. cp terraform.tfvars.example terraform.tfvars"
	@echo "  2. Edit terraform.tfvars"
	@echo "  3. make init"
	@echo "  4. make apply"
	@echo "  5. make output"

init:
	terraform init

plan:
	terraform plan

apply:
	terraform apply

output:
	@echo ""
	@terraform output -raw next_steps
	@echo ""

destroy:
	@echo "⚠️  WARNING: This will delete the Terraform state bucket!"
	@echo "⚠️  All stored Terraform states will be LOST!"
	@echo ""
	@read -p "Are you sure? Type 'yes' to continue: " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		terraform destroy; \
	else \
		echo "Aborted."; \
	fi

setup: init apply output
	@echo ""
	@echo "✅ Bootstrap complete!"
	@echo ""
	@echo "Next: Go to parent directory and configure backend"
	@echo "  cd .."
	@echo "  vim backend.tf"
	@echo "  terraform init -migrate-state"
