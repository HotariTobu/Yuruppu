# Expressions and References

Expressions are used to refer to or compute values within OpenTofu configurations. Understanding reference syntax and value access is essential for building dynamic infrastructure.

## Reference Syntax

### Resources

Reference resources using `<RESOURCE_TYPE>.<NAME>`:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = "us-central1"
}

output "service_url" {
  value = google_cloud_run_service.api.status[0].url
}
```

### Input Variables

Reference variables using `var.<NAME>`:

```hcl
variable "project_id" {
  type = string
}

resource "google_cloud_run_service" "api" {
  project = var.project_id
  name    = "api-service"
}
```

### Local Values

Reference locals using `local.<NAME>`:

```hcl
locals {
  common_labels = {
    environment = var.environment
    managed_by  = "opentofu"
  }

  service_name = "${var.environment}-api"
}

resource "google_cloud_run_service" "api" {
  name = local.service_name

  metadata {
    labels = local.common_labels
  }
}
```

### Child Module Outputs

Reference module outputs using `module.<MODULE_NAME>.<OUTPUT_NAME>`:

```hcl
module "network" {
  source = "./modules/network"

  project_id = var.project_id
  region     = var.region
}

resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  template {
    metadata {
      annotations = {
        "run.googleapis.com/vpc-access-connector" = module.network.vpc_connector_id
      }
    }
  }
}
```

### Data Sources

Reference data sources using `data.<DATA_TYPE>.<NAME>`:

```hcl
data "google_project" "current" {}

data "google_secret_manager_secret_version" "api_key" {
  secret = "api-key"
}

resource "google_cloud_run_service" "api" {
  project = data.google_project.current.project_id

  template {
    spec {
      containers {
        env {
          name  = "API_KEY"
          value = data.google_secret_manager_secret_version.api_key.secret_data
        }
      }
    }
  }
}
```

## Special Values

### Filesystem and Workspace

- `path.module` - Current module's directory
- `path.root` - Root module directory
- `path.cwd` - Original working directory
- `terraform.workspace` - Active workspace name

```hcl
resource "google_cloud_run_service" "api" {
  name = "api-${terraform.workspace}"

  template {
    spec {
      containers {
        image = var.container_image

        env {
          name  = "CONFIG_FILE"
          value = file("${path.module}/config.json")
        }
      }
    }
  }
}
```

### Block-Local Names

Within specific blocks:

- `count.index` - Current iteration in count loops
- `each.key` / `each.value` - for_each iteration identifiers
- `self` - Within provisioners and connections

```hcl
resource "google_secret_manager_secret" "secrets" {
  count     = length(var.secret_names)
  secret_id = var.secret_names[count.index]

  labels = {
    index = count.index
  }
}

resource "google_project_iam_member" "members" {
  for_each = toset(var.members)

  project = var.project_id
  role    = "roles/run.invoker"
  member  = each.value
}
```

## Attribute Access

Access nested attributes using dot notation:

```hcl
# Simple attribute
google_cloud_run_service.api.name

# Nested attribute
google_cloud_run_service.api.template[0].spec[0].service_account_name

# List index
google_cloud_run_service.api.status[0].url

# Map key
google_cloud_run_service.api.metadata[0].labels["environment"]
```

## String Interpolation

Embed expressions in strings using `${}`:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "${var.environment}-api-service"
  location = var.region

  template {
    spec {
      containers {
        image = "gcr.io/${var.project_id}/api:${var.image_tag}"

        env {
          name  = "ENVIRONMENT"
          value = "${upper(var.environment)}"
        }
      }
    }
  }
}
```

For simple variable references, interpolation is optional:

```hcl
# These are equivalent
name = var.service_name
name = "${var.service_name}"

# But interpolation is required for expressions
name = "${var.environment}-${var.service_name}"
```

## Operators

### Arithmetic

```hcl
count        = var.instance_count + 2
memory_limit = var.base_memory * 2
cpu_limit    = var.total_cpu / var.instance_count
```

### Comparison

```hcl
condition = var.instance_count > 0
condition = var.environment == "production"
condition = var.memory_limit >= 512
condition = var.region != "us-central1"
```

### Logical

```hcl
condition = var.enable_monitoring && var.environment == "production"
condition = var.region == "us-central1" || var.region == "us-east1"
condition = !var.disable_feature
```

## Conditional Expressions

Ternary operator syntax: `condition ? true_val : false_val`

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  template {
    spec {
      containers {
        image = var.container_image

        resources {
          limits = {
            cpu    = var.environment == "production" ? "2000m" : "1000m"
            memory = var.environment == "production" ? "1Gi" : "512Mi"
          }
        }
      }
    }
  }
}

locals {
  instance_count = var.high_availability ? 3 : 1
  service_tier   = var.environment == "production" ? "premium" : "standard"
}
```

## For Expressions

Transform collections:

### List Comprehension

```hcl
locals {
  service_urls = [
    for service in google_cloud_run_service.services :
    service.status[0].url
  ]

  uppercase_names = [
    for name in var.service_names :
    upper(name)
  ]

  # With filtering
  production_services = [
    for k, v in var.services :
    v.name if v.environment == "production"
  ]
}
```

### Map Comprehension

```hcl
locals {
  service_url_map = {
    for k, v in google_cloud_run_service.services :
    k => v.status[0].url
  }

  # Transform keys and values
  env_vars = {
    for key, value in var.raw_env_vars :
    upper(key) => lower(value)
  }

  # With filtering
  required_secrets = {
    for k, v in var.secrets :
    k => v if v.required
  }
}
```

## Splat Expressions

Access attributes from lists of resources:

```hcl
# Get all service URLs
output "service_urls" {
  value = google_cloud_run_service.services[*].status[0].url
}

# Get all service names
output "service_names" {
  value = google_cloud_run_service.services[*].name
}

# With for_each (use values())
output "service_urls_for_each" {
  value = values(google_cloud_run_service.services)[*].status[0].url
}
```

## Function Calls

OpenTofu provides built-in functions:

### String Functions

```hcl
locals {
  upper_env       = upper(var.environment)
  lower_name      = lower(var.service_name)
  trimmed         = trimspace(var.description)
  formatted       = format("service-%s-%s", var.environment, var.region)
  replaced        = replace(var.image_url, "gcr.io", "docker.pkg.dev")
  joined          = join("-", [var.prefix, var.name, var.suffix])
  split_list      = split(",", var.comma_separated_list)
}
```

### Collection Functions

```hcl
locals {
  unique_regions  = distinct(var.regions)
  sorted_names    = sort(var.service_names)
  list_length     = length(var.items)
  merged_map      = merge(var.default_labels, var.custom_labels)
  flattened       = flatten(var.nested_lists)
  contains_prod   = contains(var.environments, "production")
}
```

### Encoding Functions

```hcl
locals {
  json_config     = jsonencode(var.config_object)
  decoded_json    = jsondecode(var.json_string)
  base64_encoded  = base64encode(var.secret_value)
  base64_decoded  = base64decode(var.encoded_value)
}
```

### Type Conversion

```hcl
locals {
  string_to_list  = tolist(var.string_set)
  list_to_set     = toset(var.string_list)
  map_to_list     = [for k, v in var.map : "${k}=${v}"]
  to_number       = tonumber(var.string_number)
  to_string       = tostring(var.number_value)
}
```

### Filesystem Functions

```hcl
locals {
  config_content  = file("${path.module}/config.json")
  config_parsed   = jsondecode(file("${path.module}/config.json"))
  file_exists     = fileexists("${path.module}/optional.txt")
  file_base64     = filebase64("${path.module}/binary-file")
}
```

## Dynamic Blocks

Generate nested blocks dynamically:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  template {
    spec {
      containers {
        image = var.container_image

        dynamic "env" {
          for_each = var.environment_variables

          content {
            name  = env.key
            value = env.value
          }
        }

        dynamic "env" {
          for_each = var.secret_env_variables

          content {
            name = env.key

            value_from {
              secret_key_ref {
                name = env.value.secret_name
                key  = env.value.secret_key
              }
            }
          }
        }
      }
    }
  }
}
```

## Type Constraints

Specify expected types for variables:

- `string`, `number`, `bool` - Primitives
- `list(<TYPE>)` - Ordered collections
- `set(<TYPE>)` - Unordered unique collections
- `map(<TYPE>)` - Key-value pairs
- `object({...})` - Structured data
- `tuple([...])` - Fixed-length sequences
- `any` - Any type

## Null and Undefined

Handle optional values:

```hcl
variable "optional_label" {
  type     = string
  nullable = true
  default  = null
}

locals {
  # Use coalesce to provide fallback
  label = coalesce(var.optional_label, "default-label")

  # Conditional based on null
  labels = var.optional_label != null ? {
    custom = var.optional_label
  } : {}
}
```

## Best Practices

1. Use locals for repeated expressions
2. Prefer explicit references over interpolation for simple variables
3. Use for expressions instead of complex nested loops
4. Comment complex expressions
5. Use functions to keep expressions readable
6. Validate inputs with variable validation blocks
7. Use type constraints for all variables
8. Leverage dynamic blocks for repeated nested blocks
9. Use splat expressions for list transformations
10. Keep expressions simple and maintainable
