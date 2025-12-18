# for_each Meta-Argument

The `for_each` meta-argument enables you to create multiple resource instances from a single resource block using maps or sets of strings.

## Basic Usage

If a resource or module block includes a `for_each` argument whose value is a map or a set of strings, OpenTofu creates one instance for each member:

```hcl
resource "google_project_iam_member" "members" {
  for_each = toset([
    "user:alice@example.com",
    "user:bob@example.com",
    "serviceAccount:app@project.iam.gserviceaccount.com"
  ])

  project = var.project_id
  role    = "roles/run.invoker"
  member  = each.value
}
```

## The each Object

Within blocks using `for_each`, you have access to an `each` object with two properties:

- `each.key` - The identifier for the current instance
- `each.value` - The corresponding value (for sets, this equals `each.key`)

## With Sets

```hcl
variable "service_accounts" {
  type = set(string)
  default = [
    "api-service",
    "worker-service",
    "scheduler-service"
  ]
}

resource "google_service_account" "accounts" {
  for_each = var.service_accounts

  account_id   = each.value
  display_name = "${each.value} Service Account"
  project      = var.project_id
}
```

## With Maps

Maps provide both keys and values:

```hcl
variable "cloud_run_services" {
  type = map(object({
    image  = string
    cpu    = string
    memory = string
  }))
  default = {
    api = {
      image  = "gcr.io/project/api:latest"
      cpu    = "2000m"
      memory = "1Gi"
    }
    worker = {
      image  = "gcr.io/project/worker:latest"
      cpu    = "1000m"
      memory = "512Mi"
    }
  }
}

resource "google_cloud_run_service" "services" {
  for_each = var.cloud_run_services

  name     = each.key
  location = var.region

  template {
    spec {
      containers {
        image = each.value.image

        resources {
          limits = {
            cpu    = each.value.cpu
            memory = each.value.memory
          }
        }
      }
    }
  }
}
```

## Referencing for_each Resources

Reference instances using map or set keys:

```hcl
# Reference specific instance
output "api_service_url" {
  value = google_cloud_run_service.services["api"].status[0].url
}

# Reference all instances
output "service_urls" {
  value = {
    for k, v in google_cloud_run_service.services :
    k => v.status[0].url
  }
}

# Using values() for splat expressions
output "all_urls" {
  value = values(google_cloud_run_service.services)[*].status[0].url
}
```

## Converting Lists to Sets

Use `toset()` to convert lists:

```hcl
variable "member_list" {
  type = list(string)
  default = [
    "user:alice@example.com",
    "user:bob@example.com"
  ]
}

resource "google_project_iam_member" "members" {
  for_each = toset(var.member_list)

  project = var.project_id
  role    = "roles/viewer"
  member  = each.value
}
```

## With Filtering

Use for expressions to filter:

```hcl
variable "services" {
  type = map(object({
    enabled = bool
    image   = string
  }))
}

resource "google_cloud_run_service" "enabled_services" {
  for_each = {
    for k, v in var.services :
    k => v if v.enabled
  }

  name     = each.key
  location = var.region

  template {
    spec {
      containers {
        image = each.value.image
      }
    }
  }
}
```

## Common Patterns

### IAM Bindings with Multiple Roles

```hcl
variable "service_account_roles" {
  type = map(string)
  default = {
    secrets  = "roles/secretmanager.secretAccessor"
    sql      = "roles/cloudsql.client"
    tracing  = "roles/cloudtrace.agent"
    logging  = "roles/logging.logWriter"
  }
}

resource "google_project_iam_member" "sa_roles" {
  for_each = var.service_account_roles

  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.app.email}"
}
```

### Secrets with Configurations

```hcl
variable "secrets" {
  type = map(object({
    replication_locations = list(string)
  }))
  default = {
    api-key = {
      replication_locations = ["us-central1", "us-east1"]
    }
    database-password = {
      replication_locations = ["us-central1"]
    }
  }
}

resource "google_secret_manager_secret" "secrets" {
  for_each  = var.secrets
  secret_id = each.key
  project   = var.project_id

  replication {
    user_managed {
      dynamic "replicas" {
        for_each = each.value.replication_locations

        content {
          location = replicas.value
        }
      }
    }
  }
}
```

### Regional Deployments

```hcl
variable "regional_configs" {
  type = map(object({
    min_instances = number
    max_instances = number
  }))
  default = {
    us-central1 = {
      min_instances = 1
      max_instances = 10
    }
    us-east1 = {
      min_instances = 0
      max_instances = 5
    }
  }
}

resource "google_cloud_run_service" "regional" {
  for_each = var.regional_configs

  name     = "api-service"
  location = each.key

  template {
    metadata {
      annotations = {
        "autoscaling.knative.dev/minScale" = tostring(each.value.min_instances)
        "autoscaling.knative.dev/maxScale" = tostring(each.value.max_instances)
      }
    }

    spec {
      containers {
        image = var.container_image

        env {
          name  = "REGION"
          value = each.key
        }
      }
    }
  }
}
```

## Advantages over count

### 1. Semantic Keys

```hcl
# With for_each - clear, semantic keys
google_cloud_run_service.services["api"]
google_cloud_run_service.services["worker"]

# With count - just numeric indices
google_cloud_run_service.services[0]
google_cloud_run_service.services[1]
```

### 2. Stability When Removing Items

```hcl
# Initial configuration
variable "services" {
  default = toset(["api", "web", "worker"])
}

resource "google_cloud_run_service" "services" {
  for_each = var.services
  name     = each.value
  location = var.region
}

# Remove "web" - only "web" service is destroyed
variable "services" {
  default = toset(["api", "worker"])
}
# "api" and "worker" remain untouched
```

With `count`, removing the middle element would shift all subsequent indices, causing unnecessary resource recreation.

### 3. Explicit Dependencies

Dependencies are clearer with semantic keys:

```hcl
resource "google_cloud_run_service" "frontend" {
  name     = "frontend"
  location = var.region

  template {
    spec {
      containers {
        env {
          name  = "API_URL"
          value = google_cloud_run_service.services["api"].status[0].url
        }
      }
    }
  }
}
```

## Key Constraints

1. **Keys must be strings** - Map keys and set values must be strings
2. **No computed values from impure functions** - Cannot use `uuid()`, `bcrypt()`, or `timestamp()`
3. **No sensitive values** - Sensitive values cannot be used as `for_each` arguments
4. **Known at plan time** - The collection must be known before apply

## Conditional Creation

Use empty maps or sets to conditionally create resources:

```hcl
variable "enable_monitoring" {
  type    = bool
  default = false
}

resource "google_monitoring_dashboard" "dashboards" {
  for_each = var.enable_monitoring ? {
    main = "dashboard-config-1.json"
    alt  = "dashboard-config-2.json"
  } : {}

  dashboard_json = file("${path.module}/${each.value}")
}
```

## With Modules

```hcl
variable "environments" {
  type = map(object({
    project_id = string
    region     = string
  }))
  default = {
    dev = {
      project_id = "myapp-dev"
      region     = "us-central1"
    }
    staging = {
      project_id = "myapp-staging"
      region     = "us-central1"
    }
    prod = {
      project_id = "myapp-prod"
      region     = "us-east1"
    }
  }
}

module "cloud_run" {
  for_each = var.environments
  source   = "./modules/cloud-run"

  environment = each.key
  project_id  = each.value.project_id
  region      = each.value.region
}

output "environment_urls" {
  value = {
    for k, m in module.cloud_run :
    k => m.service_url
  }
}
```

## Best Practices

1. **Use maps for complex objects** - When each instance needs multiple attributes
2. **Use sets for simple lists** - When instances are identical except for one identifier
3. **Prefer for_each over count** - When resources might be reordered or removed
4. **Use meaningful keys** - Choose keys that clearly identify each instance
5. **Document the structure** - Explain the expected map/set structure in variable descriptions
6. **Handle empty collections** - Test behavior with empty maps/sets
7. **Use toset() for lists** - Convert lists to sets to use with for_each
8. **Filter in for expressions** - Use filtering to control which items are created
9. **Consistent naming** - Use each.key consistently for resource naming
10. **Avoid deep nesting** - Keep map structures simple and flat when possible

## Examples

### Multiple Service Accounts with IAM

```hcl
variable "service_accounts" {
  type = map(object({
    display_name = string
    roles        = list(string)
  }))
  default = {
    api = {
      display_name = "API Service"
      roles = [
        "roles/secretmanager.secretAccessor",
        "roles/cloudsql.client"
      ]
    }
    worker = {
      display_name = "Worker Service"
      roles = [
        "roles/pubsub.subscriber",
        "roles/storage.objectViewer"
      ]
    }
  }
}

resource "google_service_account" "accounts" {
  for_each = var.service_accounts

  account_id   = each.key
  display_name = each.value.display_name
  project      = var.project_id
}

resource "google_project_iam_member" "account_roles" {
  for_each = {
    for pair in flatten([
      for account, config in var.service_accounts : [
        for role in config.roles : {
          key     = "${account}-${replace(role, "/", "-")}"
          account = account
          role    = role
        }
      ]
    ]) : pair.key => pair
  }

  project = var.project_id
  role    = each.value.role
  member  = "serviceAccount:${google_service_account.accounts[each.value.account].email}"
}
```
