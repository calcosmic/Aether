---
name: terraform
description: Use when the project uses Terraform or OpenTofu for infrastructure-as-code provisioning
type: domain
domains: [infrastructure, devops, cloud]
agent_roles: [builder]
detect_files: ["*.tf", "*.tfvars", "terraform.tfstate", ".terraform.lock.hcl"]
detect_packages: ["terraform", "opentofu", "terragrunt"]
priority: normal
version: "1.0"
---

# Terraform/OpenTofu Best Practices

## Module Composition

- Organize code into reusable modules: `modules/{resource-type}/` with `main.tf`, `variables.tf`, `outputs.tf`
- Use the `module` block to compose infrastructure: pass inputs via variables, expose outputs for downstream use
- Follow the principle of one module per logical resource group (e.g., `vpc`, `database`, `eks-cluster`)
- Version modules with git tags or registry versions: `source = "git::https://...?ref=v1.2.0"`
- Use `locals` for computed values and transformations; avoid complex logic inside resource blocks

## State Management

- Store state remotely: use S3+DynamoDB (AWS), Azure Blob (Azure), or GCS (GCP) with locking enabled
- Never commit `.tfstate` to version control -- it contains sensitive outputs
- Use `state encryption` for sensitive deployments; configure in the backend block
- Run `terraform plan` before every apply; review the diff carefully for destructive changes
- Use `terraform import` to bring existing resources under Terraform management without recreation

## Workspace Strategies

- Use Terraform workspaces for environment separation (dev/staging/prod) with identical configurations
- Map workspace names to environment-specific variables via `terraform.workspace` in locals
- For large-scale separation, prefer separate state files per environment over workspaces
- Label resources with workspace tags: `Environment = terraform.workspace` for cost tracking

## Provider Configuration

- Pin provider versions with `required_providers` block: `version = "~> 4.0"`
- Use `provider` aliases for multi-region or multi-account deployments
- Pass provider credentials via environment variables or IAM roles -- never hardcode in `.tf` files
- Use `dynamic` blocks for repeatable nested configurations: security group rules, IAM policies

## Testing and Validation

- Run `terraform validate` and `terraform fmt` in CI before every merge
- Use `tflint` for linting beyond built-in validation: detect unused variables, deprecated syntax
- Write Terratest integration tests for critical infrastructure: `terraform.InitAndApply` + assertions
- Use `check` blocks (Terraform 1.3+) for custom data assertions without external tools
- Implement `terraform plan` as a PR check; post the plan diff as a comment for review
