# Terraform Google Provider

> Official Terraform provider for Google Cloud Platform (GCP), enabling infrastructure management through Terraform/OpenTofu. Jointly maintained by Google's Terraform Team and HashiCorp.

This documentation focuses on core GCP resources for serverless deployments: Cloud Run services, Artifact Registry, Secret Manager, Cloud Build triggers, Service Accounts, and IAM management.

## Provider Configuration

```hcl
provider "google" {
  project = "my-project-id"
  region  = "us-central1"
}
```

## Getting Started

- [Provider Overview](getting-started.md): Installation, configuration, and basic usage
- [Authentication Guide](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started): Credential setup and best practices

## Cloud Run

- [google_cloud_run_service](cloud-run-service.md): Deploy and manage Cloud Run services with container autoscaling
- [google_cloud_run_service_iam](cloud-run-service-iam.md): IAM access control for Cloud Run services

## Artifact Registry

- [google_artifact_registry_repository](artifact-registry-repository.md): Container and artifact repository management
- [google_artifact_registry_repository_iam](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/artifact_registry_repository_iam): IAM policies for repositories

## Secret Manager

- [google_secret_manager_secret](secret-manager-secret.md): Secret metadata and replication configuration
- [google_secret_manager_secret_version](secret-manager-secret-version.md): Secret data and versioning
- [google_secret_manager_secret_iam](secret-manager-secret-iam.md): IAM access control for secrets

## Cloud Build

- [google_cloudbuild_trigger](cloudbuild-trigger.md): Automated build triggers for CI/CD pipelines

## Service Accounts & IAM

- [google_service_account](service-account.md): Create and manage service accounts
- [google_service_account_iam](service-account-iam.md): Grant IAM permissions to service accounts
- [google_project_iam](project-iam.md): Project-level IAM policy management

## Optional

- [All Resources](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources): Complete resource reference
- [Data Sources](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources): Query existing GCP resources
- [Guides](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides): Advanced topics and best practices
