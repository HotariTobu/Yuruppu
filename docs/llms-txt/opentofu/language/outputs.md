# Output Values

Output values make information about your infrastructure available on the command line and expose data for other OpenTofu configurations to use.

## Declaration

Output values are declared using an `output` block:

```hcl
output "service_url" {
  value       = google_cloud_run_service.api.status[0].url
  description = "URL of the deployed Cloud Run service"
}

output "service_account_email" {
  value = google_service_account.api.email
}

output "secret_ids" {
  value     = { for k, v in google_secret_manager_secret.secrets : k => v.secret_id }
  sensitive = true
}
```

The label after `output` serves as the output name, which must be a valid identifier.

## Required Arguments

### value

The `value` argument accepts any valid expression. The result is exposed to users and other configurations.

```hcl
output "project_id" {
  value = var.project_id
}

output "service_urls" {
  value = [
    for service in google_cloud_run_service.services :
    service.status[0].url
  ]
}

output "resource_names" {
  value = {
    cloud_run      = google_cloud_run_service.api.name
    service_account = google_service_account.api.email
    secret         = google_secret_manager_secret.api_key.secret_id
  }
}
```

## Optional Arguments

### description

Documents the output's purpose for module users:

```hcl
output "service_url" {
  value       = google_cloud_run_service.api.status[0].url
  description = "The URL where the Cloud Run service is accessible"
}
```

### sensitive

Marks values as sensitive, hiding them in CLI output:

```hcl
output "database_password" {
  value     = random_password.db.result
  sensitive = true
}
```

Note: Sensitive outputs are still stored in the state file in plain text. Use proper state encryption and access controls.

### depends_on

Creates explicit dependencies when implicit ones cannot be recognized:

```hcl
output "service_ready" {
  value = "Service is ready"

  depends_on = [
    google_cloud_run_service.api,
    google_project_iam_member.invoker
  ]
}
```

### precondition

Specifies guarantees about output data through custom condition checks:

```hcl
output "service_url" {
  value = google_cloud_run_service.api.status[0].url

  precondition {
    condition     = google_cloud_run_service.api.status[0].url != ""
    error_message = "Service URL must be available."
  }
}
```

## Usage

### In Root Modules

After running `tofu apply`, OpenTofu displays output values:

```bash
$ tofu apply
...
Apply complete! Resources: 3 added, 0 changed, 0 destroyed.

Outputs:

project_id = "my-gcp-project"
service_url = "https://api-service-abc123-uc.a.run.app"
```

Query specific outputs:

```bash
$ tofu output service_url
"https://api-service-abc123-uc.a.run.app"

$ tofu output -raw service_url
https://api-service-abc123-uc.a.run.app

$ tofu output -json
{
  "project_id": {
    "sensitive": false,
    "type": "string",
    "value": "my-gcp-project"
  },
  "service_url": {
    "sensitive": false,
    "type": "string",
    "value": "https://api-service-abc123-uc.a.run.app"
  }
}
```

### In Child Modules

Parent modules access child module outputs using:

```hcl
module.<MODULE_NAME>.<OUTPUT_NAME>
```

Example:

```hcl
# modules/cloud-run/outputs.tf
output "service_url" {
  value = google_cloud_run_service.api.status[0].url
}

# main.tf
module "api_service" {
  source = "./modules/cloud-run"

  project_id = var.project_id
  region     = var.region
}

output "api_url" {
  value = module.api_service.service_url
}

resource "google_cloud_run_service" "frontend" {
  name     = "frontend"
  location = var.region

  template {
    spec {
      containers {
        image = var.frontend_image

        env {
          name  = "API_URL"
          value = module.api_service.service_url
        }
      }
    }
  }
}
```

### With Remote State

Root module outputs can be accessed by other configurations via a `terraform_remote_state` data source:

```hcl
# Configuration A - outputs.tf
output "vpc_id" {
  value = google_compute_network.vpc.id
}

# Configuration B - data.tf
data "terraform_remote_state" "network" {
  backend = "gcs"

  config = {
    bucket = "my-terraform-state"
    prefix = "network"
  }
}

resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  # Use output from Configuration A
  template {
    metadata {
      annotations = {
        "run.googleapis.com/vpc-access-connector" = data.terraform_remote_state.network.outputs.vpc_connector_id
      }
    }
  }
}
```

## Examples

### Basic GCP Outputs

```hcl
output "project_id" {
  value       = var.project_id
  description = "GCP project ID"
}

output "region" {
  value       = var.region
  description = "GCP region"
}
```

### Cloud Run Service

```hcl
output "service_name" {
  value       = google_cloud_run_service.api.name
  description = "Name of the Cloud Run service"
}

output "service_url" {
  value       = google_cloud_run_service.api.status[0].url
  description = "URL of the Cloud Run service"
}

output "service_location" {
  value       = google_cloud_run_service.api.location
  description = "Location where the service is deployed"
}
```

### Multiple Services

```hcl
output "service_urls" {
  value = {
    for k, v in google_cloud_run_service.services :
    k => v.status[0].url
  }
  description = "Map of service names to their URLs"
}
```

### IAM and Service Accounts

```hcl
output "service_account_email" {
  value       = google_service_account.api.email
  description = "Email of the service account"
}

output "service_account_id" {
  value       = google_service_account.api.id
  description = "Full identifier of the service account"
}
```

### Secrets

```hcl
output "secret_ids" {
  value = {
    for k, v in google_secret_manager_secret.secrets :
    k => v.secret_id
  }
  description = "Map of secret names to their IDs"
  sensitive   = true
}
```

### Artifact Registry

```hcl
output "repository_url" {
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.name}"
  description = "Docker repository URL for pushing images"
}
```

### Cloud Build

```hcl
output "trigger_id" {
  value       = google_cloudbuild_trigger.deploy.id
  description = "Cloud Build trigger ID"
}

output "trigger_name" {
  value       = google_cloudbuild_trigger.deploy.name
  description = "Cloud Build trigger name"
}
```

## Best Practices

1. **Include descriptions** - Always document what each output represents
2. **Mark sensitive data** - Use `sensitive = true` for credentials and secrets
3. **Use meaningful names** - Choose clear, descriptive output names
4. **Output useful information** - Include data needed by users or other configurations
5. **Structure complex outputs** - Use objects or maps for related values
6. **Document in README** - List all outputs in module documentation
7. **Consider consumers** - Think about who will use the outputs and how
8. **Avoid excessive outputs** - Only output what's needed
9. **Use consistent naming** - Follow conventions like `service_url`, `resource_id`
10. **Group related outputs** - Use maps or objects for logically related values

## Common Patterns

### Resource Identifiers

```hcl
output "resource_ids" {
  value = {
    project        = var.project_id
    service        = google_cloud_run_service.api.id
    service_account = google_service_account.api.id
    secret         = google_secret_manager_secret.api_key.id
  }
}
```

### Connection Information

```hcl
output "connection_info" {
  value = {
    url             = google_cloud_run_service.api.status[0].url
    service_account = google_service_account.api.email
  }
}
```

### Conditional Outputs

```hcl
output "monitoring_dashboard" {
  value       = var.enable_monitoring ? google_monitoring_dashboard.main[0].id : null
  description = "Monitoring dashboard ID (if monitoring is enabled)"
}
```
