# Providers

Providers are plugins that enable OpenTofu to interact with cloud providers, SaaS providers, and other APIs. Each provider adds resource types and data sources that OpenTofu can manage.

## Purpose

OpenTofu relies on providers to:
- Understand API interactions for specific platforms
- Define resource types and data sources
- Authenticate with cloud providers
- Manage the lifecycle of infrastructure resources

## Provider Configuration

### Declaring Provider Requirements

Configurations must declare which providers they require so OpenTofu can install and use them.

```hcl
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}
```

### Configuring Providers

After declaring requirements, configure the provider:

```hcl
provider "google" {
  project = "my-gcp-project"
  region  = "us-central1"
}
```

## Google Cloud Provider

### Basic Configuration

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
  project = var.project_id
  region  = var.region
}
```

### Authentication

The Google provider supports multiple authentication methods:

#### 1. Application Default Credentials (Recommended)

```bash
gcloud auth application-default login
```

No explicit credentials in configuration needed.

#### 2. Service Account Key

```hcl
provider "google" {
  credentials = file("path/to/service-account-key.json")
  project     = var.project_id
  region      = var.region
}
```

Or via environment variable:

```bash
export GOOGLE_CREDENTIALS=$(cat path/to/key.json)
```

#### 3. Environment Variables

```bash
export GOOGLE_PROJECT="my-project-id"
export GOOGLE_REGION="us-central1"
```

### Available Configuration Arguments

- `project` - Default GCP project for resources
- `region` - Default region for resources
- `zone` - Default zone for zonal resources
- `credentials` - Service account key file path or content
- `access_token` - Temporary OAuth 2.0 access token
- `impersonate_service_account` - Service account to impersonate
- `user_project_override` - Override project for API calls
- `billing_project` - Project for billing and quota

## Provider Installation

Providers are installed during initialization:

```bash
tofu init
```

OpenTofu downloads providers from the OpenTofu Registry (or configured registries) and stores them in the `.terraform` directory.

### Plugin Cache

To save bandwidth when working with multiple configurations, enable a plugin cache:

```hcl
# In CLI configuration file (~/.tofurc)
plugin_cache_dir = "$HOME/.terraform.d/plugin-cache"
```

## Version Constraints

Use version constraints to ensure consistency:

```hcl
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0.0, < 6.0.0"
    }
  }
}
```

Version constraint operators:
- `=` - Exact version
- `!=` - Exclude version
- `>`, `>=`, `<`, `<=` - Comparisons
- `~>` - Pessimistic constraint (e.g., `~> 5.0` allows `5.x` but not `6.0`)

## Dependency Lock File

Create a dependency lock file and commit it to version control to ensure OpenTofu always installs the same provider versions:

```bash
tofu init
```

This creates `.terraform.lock.hcl` containing exact versions and checksums.

## Multiple Provider Configurations

Use aliases for multiple configurations of the same provider:

```hcl
provider "google" {
  alias   = "us-central"
  project = var.project_id
  region  = "us-central1"
}

provider "google" {
  alias   = "us-east"
  project = var.project_id
  region  = "us-east1"
}

resource "google_cloud_run_service" "api_central" {
  provider = google.us-central

  name     = "api-service"
  location = "us-central1"
}

resource "google_cloud_run_service" "api_east" {
  provider = google.us-east

  name     = "api-service"
  location = "us-east1"
}
```

## Provider Meta-Arguments

Resources can specify which provider configuration to use:

```hcl
resource "google_cloud_run_service" "api" {
  provider = google.us-central

  name     = "api-service"
  location = "us-central1"
}
```

## Third-Party Providers

Providers can come from various sources:

- **Official** - Published by HashiCorp
- **Partner** - Published by partner organizations
- **Community** - Published by individual contributors

All are available in the Public OpenTofu Registry: https://registry.opentofu.org/

## Development Requirements

Providers are written in Go using the Terraform Plugin SDK. Developer-specific environment variables required for testing include:

- `TF_ACC_TERRAFORM_PATH`
- `TF_ACC_PROVIDER_NAMESPACE`
- `TF_ACC_PROVIDER_HOST`

## Provider Resources

For GCP infrastructure management, the Google provider offers resources for:

- **Compute**: Cloud Run, Compute Engine, GKE
- **Storage**: Cloud Storage, Filestore, Persistent Disk
- **Networking**: VPC, Load Balancers, Cloud DNS
- **Security**: IAM, Secret Manager, KMS
- **CI/CD**: Cloud Build, Artifact Registry
- **Databases**: Cloud SQL, Firestore, Bigtable
- **Monitoring**: Cloud Monitoring, Cloud Logging
- And many more

Consult the provider documentation for complete resource and data source listings: https://registry.opentofu.org/providers/hashicorp/google/latest/docs
