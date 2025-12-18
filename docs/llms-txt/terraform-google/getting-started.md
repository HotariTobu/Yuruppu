# Getting Started with Terraform Google Provider

## Overview

The Google Cloud provider enables infrastructure configuration on Google Cloud Platform through Terraform/OpenTofu. It's jointly maintained by Google's Terraform Team and HashiCorp.

## Installation

Add the provider to your Terraform configuration:

```hcl
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = "my-project-id"
  region  = "us-central1"
}
```

## Provider Configuration

### Required Arguments

- **project**: GCP project ID where resources will be created

### Optional Arguments

- **region**: Default region for regional resources (e.g., "us-central1")
- **zone**: Default zone for zonal resources (e.g., "us-central1-a")
- **credentials**: Path to service account key file or JSON credentials

### Authentication Methods

1. **Service Account Key**: Set `credentials` or use `GOOGLE_CREDENTIALS` environment variable
2. **Application Default Credentials**: For local development with `gcloud auth application-default login`
3. **Compute Engine Metadata**: Automatic when running on GCE, GKE, Cloud Functions, etc.

## Release Schedule

- **Minor releases**: Approximately weekly with bug fixes and new features
- **Major releases**: Roughly annually with breaking changes and upgrade guides
- **Versioning**: Follows semantic versioning standards

## Resource Naming Convention

All Google provider resources follow the pattern `google_{service}_{resource}`:

- `google_cloud_run_service` - Cloud Run service
- `google_artifact_registry_repository` - Artifact Registry repository
- `google_secret_manager_secret` - Secret Manager secret
- `google_service_account` - Service account

## Common Patterns

### Using Default Values

Many arguments are optional and will use provider defaults:

```hcl
resource "google_artifact_registry_repository" "example" {
  repository_id = "my-repo"
  format        = "DOCKER"
  # location defaults to provider region
  # project defaults to provider project
}
```

### Import Existing Resources

Most resources support multiple import formats:

```bash
# Full path
terraform import google_cloud_run_service.default locations/us-central1/namespaces/my-project/services/my-service

# Abbreviated path
terraform import google_cloud_run_service.default us-central1/my-project/my-service
```

### Timeouts

Configure custom operation timeouts:

```hcl
resource "google_cloud_run_service" "default" {
  name     = "my-service"
  location = "us-central1"

  timeouts {
    create = "20m"
    update = "20m"
    delete = "20m"
  }
}
```

## Support Resources

- **Community Forum**: [discuss.hashicorp.com](https://discuss.hashicorp.com) - Terraform Google category
- **Slack**: Google Cloud Community Slack #terraform channel
- **GitHub Issues**: [terraform-provider-google](https://github.com/hashicorp/terraform-provider-google/issues)
- **HashiCorp Support**: Available for Enterprise customers

## Best Practices

1. **Use version constraints** to prevent unexpected breaking changes
2. **Enable state locking** with remote backends (GCS recommended)
3. **Use service accounts** with minimal required permissions
4. **Tag resources** with labels for cost tracking and organization
5. **Review generated plans** before applying changes
