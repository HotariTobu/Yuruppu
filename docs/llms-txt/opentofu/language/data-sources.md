# Data Sources

Data sources enable OpenTofu to use information defined outside of OpenTofu, defined by another separate OpenTofu configuration, or modified by functions.

## Purpose

Unlike managed resources that create and modify infrastructure, data resources cause OpenTofu only to read objects. Data sources are read-only views into existing infrastructure or computed values.

## Basic Syntax

```hcl
data "provider_data_source" "name" {
  # Query parameters
}
```

## Common Use Cases

Data sources are used to:

- Read information about existing infrastructure
- Query cloud provider APIs for current information
- Render templates
- Read local files
- Compute values using functions

## Examples

### Current GCP Project

```hcl
data "google_project" "current" {}

output "project_id" {
  value = data.google_project.current.project_id
}
```

### Secret Manager Secret

```hcl
data "google_secret_manager_secret_version" "api_key" {
  secret  = "api-key"
  project = var.project_id
}

resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = "us-central1"

  template {
    spec {
      containers {
        image = "gcr.io/my-project/api:latest"

        env {
          name  = "API_KEY"
          value = data.google_secret_manager_secret_version.api_key.secret_data
        }
      }
    }
  }
}
```

### Service Account

```hcl
data "google_service_account" "existing" {
  account_id = "existing-sa"
  project    = var.project_id
}

resource "google_project_iam_member" "sa_role" {
  project = var.project_id
  role    = "roles/cloudrun.invoker"
  member  = "serviceAccount:${data.google_service_account.existing.email}"
}
```

### Artifact Registry Repository

```hcl
data "google_artifact_registry_repository" "images" {
  location      = "us-central1"
  repository_id = "docker-images"
  project       = var.project_id
}

output "repository_url" {
  value = data.google_artifact_registry_repository.images.name
}
```

### Local File

```hcl
data "local_file" "config" {
  filename = "${path.module}/config.json"
}

locals {
  config = jsondecode(data.local_file.config.content)
}
```

### Template File

```hcl
data "template_file" "cloudinit" {
  template = file("${path.module}/cloud-init.yaml")

  vars = {
    hostname = var.hostname
    domain   = var.domain
  }
}

resource "google_compute_instance" "vm" {
  name         = "example-vm"
  machine_type = "e2-medium"

  metadata = {
    user-data = data.template_file.cloudinit.rendered
  }
}
```

## Referencing Data Sources

Reference data source attributes using:

```hcl
data.<TYPE>.<NAME>.<ATTRIBUTE>
```

Example:

```hcl
data.google_project.current.project_id
data.google_secret_manager_secret_version.api_key.secret_data
```

## Processing Timing

Data resources are typically read during the planning phase when possible. However, OpenTofu defers reading when arguments reference computed values from unbuilt resources, ensuring proper dependency ordering.

```hcl
# This data source is read during plan
data "google_project" "current" {}

# This data source is read during apply (depends on computed value)
data "google_cloud_run_service" "api" {
  name     = google_cloud_run_service.api.name
  location = google_cloud_run_service.api.location
}
```

## Meta-Arguments

Data sources support several meta-arguments:

### depends_on

Specify explicit dependencies:

```hcl
data "google_secret_manager_secret_version" "api_key" {
  secret = "api-key"

  depends_on = [
    google_secret_manager_secret.api_key
  ]
}
```

### count

Create multiple data source instances:

```hcl
data "google_secret_manager_secret_version" "secrets" {
  count  = length(var.secret_names)
  secret = var.secret_names[count.index]
}
```

### for_each

Query multiple items from maps or sets:

```hcl
data "google_service_account" "accounts" {
  for_each = toset(var.service_account_ids)

  account_id = each.value
  project    = var.project_id
}
```

### provider

Use a specific provider configuration:

```hcl
data "google_project" "other" {
  provider   = google.other-project
  project_id = "other-project-id"
}
```

### lifecycle

Add preconditions and postconditions:

```hcl
data "google_project" "current" {
  lifecycle {
    postcondition {
      condition     = self.number != ""
      error_message = "Project number must be set."
    }
  }
}
```

## Local-Only Data Sources

Some data sources operate entirely within OpenTofu without querying external systems:

- `local_file` - Read local files
- `template_file` - Render templates
- `external` - Execute external programs
- `null_data_source` - Compute values (deprecated, use locals instead)

## Provider-Specific Data Sources

Each provider offers its own set of data sources. For GCP:

- `google_project` - Current or specific project information
- `google_client_config` - Current client configuration
- `google_service_account` - Service account details
- `google_secret_manager_secret_version` - Secret values
- `google_cloud_run_service` - Cloud Run service details
- `google_artifact_registry_repository` - Repository information
- `google_compute_zones` - Available zones in a region
- And many more

Consult the provider documentation for a complete list of available data sources and their arguments.
