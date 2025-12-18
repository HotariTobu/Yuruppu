# Modules

Modules are the main way to package and reuse resource configurations with OpenTofu. A module is a container for multiple resources that are used together.

## Module Structure

### Root Module

Every OpenTofu configuration has at least one root module, consisting of `.tf`, `.tofu`, `.tf.json`, and/or `.tofu.json` files in the main working directory.

### Child Modules

A module that has been called by another module is referred to as a child module. Child modules can be called multiple times and reused across configurations.

## Basic Module Usage

### Calling a Module

```hcl
module "cloud_run_api" {
  source = "./modules/cloud-run"

  project_id      = var.project_id
  region          = var.region
  service_name    = "api-service"
  container_image = "gcr.io/my-project/api:latest"
  cpu             = "1000m"
  memory          = "512Mi"
}
```

### Module Block Arguments

- `source` (required) - Module location (local path, registry, Git, etc.)
- `version` - Module version constraint (for registry modules)
- Input variables - Pass values to the module

### Accessing Module Outputs

```hcl
output "api_url" {
  value = module.cloud_run_api.service_url
}

resource "google_cloud_run_service" "frontend" {
  name     = "frontend"
  location = var.region

  template {
    spec {
      containers {
        env {
          name  = "API_URL"
          value = module.cloud_run_api.service_url
        }
      }
    }
  }
}
```

## Module Structure Example

```
modules/
  cloud-run/
    main.tf           # Resource definitions
    variables.tf      # Input variable declarations
    outputs.tf        # Output value declarations
    README.md         # Module documentation
    versions.tf       # Provider requirements
```

### main.tf

```hcl
resource "google_cloud_run_service" "service" {
  name     = var.service_name
  location = var.region
  project  = var.project_id

  template {
    spec {
      containers {
        image = var.container_image

        resources {
          limits = {
            cpu    = var.cpu
            memory = var.memory
          }
        }
      }

      service_account_name = google_service_account.service.email
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}

resource "google_service_account" "service" {
  account_id   = "${var.service_name}-sa"
  display_name = "${var.service_name} Service Account"
  project      = var.project_id
}
```

### variables.tf

```hcl
variable "project_id" {
  type        = string
  description = "GCP project ID"
}

variable "region" {
  type        = string
  description = "GCP region"
}

variable "service_name" {
  type        = string
  description = "Name of the Cloud Run service"
}

variable "container_image" {
  type        = string
  description = "Container image to deploy"
}

variable "cpu" {
  type        = string
  description = "CPU limit"
  default     = "1000m"
}

variable "memory" {
  type        = string
  description = "Memory limit"
  default     = "512Mi"
}
```

### outputs.tf

```hcl
output "service_url" {
  value       = google_cloud_run_service.service.status[0].url
  description = "URL of the Cloud Run service"
}

output "service_name" {
  value       = google_cloud_run_service.service.name
  description = "Name of the Cloud Run service"
}

output "service_account_email" {
  value       = google_service_account.service.email
  description = "Email of the service account"
}
```

## Module Meta-Arguments

### count

Create multiple instances of a module:

```hcl
module "cloud_run_services" {
  count  = length(var.service_configs)
  source = "./modules/cloud-run"

  project_id      = var.project_id
  region          = var.region
  service_name    = var.service_configs[count.index].name
  container_image = var.service_configs[count.index].image
}

output "service_urls" {
  value = [for m in module.cloud_run_services : m.service_url]
}
```

### for_each

Create instances from maps or sets:

```hcl
module "cloud_run_services" {
  for_each = var.services
  source   = "./modules/cloud-run"

  project_id      = var.project_id
  region          = var.region
  service_name    = each.key
  container_image = each.value.image
  cpu             = each.value.cpu
  memory          = each.value.memory
}

output "service_urls" {
  value = {
    for k, m in module.cloud_run_services :
    k => m.service_url
  }
}
```

### providers

Pass specific provider configurations to modules:

```hcl
module "cloud_run_us_central" {
  source = "./modules/cloud-run"

  providers = {
    google = google.us-central
  }

  project_id   = var.project_id
  region       = "us-central1"
  service_name = "api-service"
}
```

### depends_on

Specify explicit module dependencies:

```hcl
module "cloud_run" {
  source = "./modules/cloud-run"

  depends_on = [
    google_project_service.run_api
  ]

  project_id   = var.project_id
  service_name = "api-service"
}
```

## Module Sources

Modules can be loaded from various sources:

### Local Paths

```hcl
module "cloud_run" {
  source = "./modules/cloud-run"
}

module "network" {
  source = "../shared-modules/network"
}
```

### Registry

```hcl
module "gce_container" {
  source  = "terraform-google-modules/container-vm/google"
  version = "~> 3.0"
}
```

### GitHub

```hcl
module "cloud_run" {
  source = "github.com/myorg/terraform-modules//cloud-run?ref=v1.0.0"
}
```

### Git

```hcl
module "cloud_run" {
  source = "git::https://example.com/terraform-modules.git//cloud-run?ref=v1.0.0"
}
```

### GCS Bucket

```hcl
module "cloud_run" {
  source = "gcs::https://www.googleapis.com/storage/v1/my-bucket/modules/cloud-run.zip"
}
```

## Published Modules

### Public OpenTofu Registry

Browse and use community modules from https://registry.opentofu.org/

```hcl
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

module "gke" {
  source  = "terraform-google-modules/kubernetes-engine/google"
  version = "~> 30.0"

  project_id = var.project_id
  name       = "my-gke-cluster"
  region     = var.region
}
```

### Private Registries

For organizational use within TACOS (TF Automation and Collaboration Software).

## Module Development Best Practices

1. **Single Responsibility** - Each module should have a clear, focused purpose
2. **Documented Variables** - Include descriptions for all variables
3. **Sensible Defaults** - Provide defaults for optional variables
4. **Useful Outputs** - Export values that consumers might need
5. **README Documentation** - Document usage, variables, and outputs
6. **Version Constraints** - Specify provider version requirements
7. **Examples** - Include example usage in an `examples/` directory
8. **Testing** - Test modules in isolation
9. **Semantic Versioning** - Use semver for module versions
10. **Minimal Dependencies** - Keep modules loosely coupled

## Example: Complete Module

```
modules/cloud-run-api/
├── main.tf
├── variables.tf
├── outputs.tf
├── versions.tf
├── README.md
└── examples/
    └── basic/
        ├── main.tf
        └── variables.tf
```

### versions.tf

```hcl
terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0"
    }
  }
}
```

### README.md

```markdown
# Cloud Run API Module

Deploys a Cloud Run service with service account and IAM bindings.

## Usage

\`\`\`hcl
module "api" {
  source = "./modules/cloud-run-api"

  project_id      = "my-project"
  region          = "us-central1"
  service_name    = "api-service"
  container_image = "gcr.io/my-project/api:latest"
}
\`\`\`

## Variables

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| project_id | GCP project ID | string | n/a | yes |
| service_name | Name of the service | string | n/a | yes |
| container_image | Container image | string | n/a | yes |
| cpu | CPU limit | string | "1000m" | no |

## Outputs

| Name | Description |
|------|-------------|
| service_url | URL of the deployed service |
| service_account_email | Service account email |
```

## Module Composition

Modules can call other modules to build complex infrastructure:

```hcl
# Root module
module "network" {
  source = "./modules/network"

  project_id = var.project_id
  region     = var.region
}

module "cloud_run" {
  source = "./modules/cloud-run"

  project_id     = var.project_id
  region         = var.region
  vpc_connector  = module.network.vpc_connector_id

  depends_on = [module.network]
}
```

This creates modular, maintainable infrastructure code with clear boundaries and reusable components.
