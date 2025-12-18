# Input Variables

Input variables let you customize aspects of OpenTofu configurations without altering the source code. They enable you to share modules across projects, each with different values.

## Declaration

Variables are declared using a `variable` block:

```hcl
variable "project_id" {
  type        = string
  description = "The GCP project ID"
}

variable "region" {
  type        = string
  description = "The GCP region for resources"
  default     = "us-central1"
}

variable "environment" {
  type        = string
  description = "Deployment environment (dev, staging, prod)"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}
```

The label after `variable` serves as the variable name for external assignment and internal reference.

## Type Constraints

While type constraints are optional, we recommend specifying them as they serve as helpful reminders and enable OpenTofu to return helpful error messages if the wrong type is used.

### Primitive Types

- `string` - Text values
- `number` - Numeric values (integers and floats)
- `bool` - Boolean values (true/false)

```hcl
variable "instance_count" {
  type    = number
  default = 3
}

variable "enable_monitoring" {
  type    = bool
  default = true
}
```

### Complex Types

#### Lists

Ordered collections of values of the same type:

```hcl
variable "availability_zones" {
  type    = list(string)
  default = ["us-central1-a", "us-central1-b", "us-central1-c"]
}
```

#### Sets

Unordered collections of unique values:

```hcl
variable "allowed_ips" {
  type = set(string)
  default = [
    "10.0.1.0/24",
    "10.0.2.0/24"
  ]
}
```

#### Maps

Key-value pairs:

```hcl
variable "service_config" {
  type = map(string)
  default = {
    cpu    = "1000m"
    memory = "512Mi"
  }
}
```

#### Objects

Structured data with named attributes:

```hcl
variable "cloud_run_service" {
  type = object({
    name     = string
    location = string
    image    = string
    cpu      = string
    memory   = string
  })
}
```

#### Tuples

Fixed-length heterogeneous sequences:

```hcl
variable "service_config" {
  type = tuple([string, number, bool])
}
```

#### Any

Accepts any type (use sparingly):

```hcl
variable "metadata" {
  type    = any
  default = {}
}
```

## Optional Arguments

### default

Provides a default value when no value is supplied:

```hcl
variable "region" {
  type    = string
  default = "us-central1"
}
```

Variables without defaults are required.

### description

Documents the variable's purpose:

```hcl
variable "project_id" {
  type        = string
  description = "The GCP project ID where resources will be created"
}
```

### validation

Custom validation rules:

```hcl
variable "image_id" {
  type        = string
  description = "Container image ID"

  validation {
    condition     = length(var.image_id) > 4 && substr(var.image_id, 0, 4) == "gcr."
    error_message = "The image_id must start with 'gcr.' for Google Container Registry."
  }
}
```

Multiple validation blocks can be defined:

```hcl
variable "cpu" {
  type = string

  validation {
    condition     = contains(["1000m", "2000m", "4000m"], var.cpu)
    error_message = "CPU must be 1000m, 2000m, or 4000m."
  }

  validation {
    condition     = can(regex("^[0-9]+m$", var.cpu))
    error_message = "CPU must be specified in millicores (e.g., 1000m)."
  }
}
```

### sensitive

Marks the variable as containing sensitive data:

```hcl
variable "database_password" {
  type      = string
  sensitive = true
}
```

OpenTofu will redact sensitive values in output, though they remain in state files.

### nullable

Controls whether the variable can be null:

```hcl
variable "optional_tag" {
  type     = string
  nullable = true
  default  = null
}
```

## Using Variables

Reference variables using `var.<NAME>` syntax:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-${var.environment}"
  location = var.region
  project  = var.project_id

  template {
    spec {
      containers {
        image = var.container_image

        resources {
          limits = {
            cpu    = var.cpu_limit
            memory = var.memory_limit
          }
        }
      }
    }
  }
}
```

## Assigning Values

### Command Line

```bash
tofu apply -var="project_id=my-project" -var="region=us-east1"
```

### Variable Files

Create a `.tfvars` file:

```hcl
# terraform.tfvars or variables.tfvars
project_id  = "my-gcp-project"
region      = "us-central1"
environment = "production"

service_config = {
  cpu    = "2000m"
  memory = "1Gi"
}
```

Apply with:

```bash
tofu apply -var-file="production.tfvars"
```

OpenTofu automatically loads:
- `terraform.tfvars`
- `terraform.tfvars.json`
- `*.auto.tfvars`
- `*.auto.tfvars.json`

### Environment Variables

Prefix with `TF_VAR_`:

```bash
export TF_VAR_project_id="my-project"
export TF_VAR_region="us-central1"
export TF_VAR_enable_monitoring="true"

tofu apply
```

### Precedence Order

When multiple sources provide values, OpenTofu uses this precedence (highest to lowest):

1. Command-line `-var` flags
2. `-var-file` flags (in order specified)
3. `*.auto.tfvars` files (alphabetical order)
4. `terraform.tfvars`
5. Environment variables

## Best Practices

1. Always include descriptions for documentation
2. Use type constraints to catch errors early
3. Provide sensible defaults for optional variables
4. Use validation rules for business logic constraints
5. Mark sensitive variables appropriately
6. Group related variables in objects when appropriate
7. Keep variable definitions in a separate `variables.tf` file
8. Document required vs. optional variables
9. Use consistent naming conventions
10. Don't hardcode values that might change per environment

## Example: Complete Variable Configuration

```hcl
# variables.tf

variable "project_id" {
  type        = string
  description = "GCP project ID"
}

variable "region" {
  type        = string
  description = "GCP region"
  default     = "us-central1"
}

variable "environment" {
  type        = string
  description = "Deployment environment"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Must be dev, staging, or prod."
  }
}

variable "cloud_run_config" {
  type = object({
    name            = string
    cpu             = string
    memory          = string
    min_instances   = number
    max_instances   = number
    container_image = string
  })

  description = "Cloud Run service configuration"
}

variable "labels" {
  type        = map(string)
  description = "Labels to apply to all resources"
  default     = {}
}

variable "enable_monitoring" {
  type        = bool
  description = "Enable Cloud Monitoring"
  default     = true
}

variable "secrets" {
  type        = map(string)
  description = "Secret Manager secret names and their environment variable names"
  sensitive   = true
  default     = {}
}
```
