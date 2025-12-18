# count Meta-Argument

The `count` meta-argument allows you to create multiple similar infrastructure objects without duplicating code blocks.

## Basic Usage

When `count` is set on a resource block, OpenTofu creates that many instances:

```hcl
resource "google_secret_manager_secret" "secrets" {
  count     = 3
  secret_id = "secret-${count.index}"

  replication {
    auto {}
  }
}
```

This creates three secrets:
- `secret-0`
- `secret-1`
- `secret-2`

## With Variables

```hcl
variable "secret_names" {
  type = list(string)
  default = [
    "api-key",
    "database-password",
    "jwt-secret"
  ]
}

resource "google_secret_manager_secret" "secrets" {
  count     = length(var.secret_names)
  secret_id = var.secret_names[count.index]

  replication {
    auto {}
  }
}
```

## Accessing count.index

Within the resource block, `count.index` provides each instance's index number (starting from 0):

```hcl
resource "google_cloud_run_service" "services" {
  count    = 4
  name     = "service-${count.index}"
  location = var.region

  template {
    spec {
      containers {
        image = var.container_image

        env {
          name  = "INSTANCE_ID"
          value = tostring(count.index)
        }

        env {
          name  = "INSTANCE_NAME"
          value = "service-${count.index}"
        }
      }
    }
  }
}
```

## Referencing Counted Resources

Reference individual instances using bracket notation:

```hcl
# Reference first instance
output "first_service_url" {
  value = google_cloud_run_service.services[0].status[0].url
}

# Reference specific instance
output "third_service_url" {
  value = google_cloud_run_service.services[2].status[0].url
}

# Reference all instances (splat expression)
output "all_service_urls" {
  value = google_cloud_run_service.services[*].status[0].url
}
```

## Conditional Creation

Use count with conditional expressions to create resources conditionally:

```hcl
resource "google_monitoring_dashboard" "main" {
  count          = var.enable_monitoring ? 1 : 0
  dashboard_json = file("${path.module}/dashboard.json")
}

# Access with index 0 when it exists
output "dashboard_id" {
  value = var.enable_monitoring ? google_monitoring_dashboard.main[0].id : null
}
```

## Common Patterns

### Create from List

```hcl
variable "members" {
  type = list(string)
  default = [
    "user:alice@example.com",
    "user:bob@example.com",
    "serviceAccount:app@project.iam.gserviceaccount.com"
  ]
}

resource "google_project_iam_member" "members" {
  count   = length(var.members)
  project = var.project_id
  role    = "roles/run.invoker"
  member  = var.members[count.index]
}
```

### Multi-Region Deployment

```hcl
variable "regions" {
  type = list(string)
  default = [
    "us-central1",
    "us-east1",
    "europe-west1"
  ]
}

resource "google_cloud_run_service" "regional" {
  count    = length(var.regions)
  name     = "api-service"
  location = var.regions[count.index]

  template {
    spec {
      containers {
        image = var.container_image

        env {
          name  = "REGION"
          value = var.regions[count.index]
        }
      }
    }
  }
}
```

## When to Use for_each Instead

Use `for_each` instead of `count` when:

1. **Resources need distinct values** that cannot be derived from simple integer indexing
2. **Order matters** - Removing an element from the middle of a list causes all subsequent resources to shift indices
3. **Semantic keys** - Using meaningful identifiers (map keys) instead of numeric indices

### Problem with count

```hcl
variable "services" {
  type = list(string)
  default = ["api", "web", "worker"]
}

resource "google_cloud_run_service" "services" {
  count    = length(var.services)
  name     = var.services[count.index]
  location = var.region
}
```

If you remove "web" from the list, "worker" shifts from index 2 to index 1, causing OpenTofu to:
- Destroy the service at index 1 (was "web", now "worker")
- Recreate it with the new configuration
- Destroy the service at index 2 (was "worker", no longer exists)

### Better with for_each

```hcl
variable "services" {
  type = set(string)
  default = ["api", "web", "worker"]
}

resource "google_cloud_run_service" "services" {
  for_each = var.services
  name     = each.value
  location = var.region
}
```

Removing "web" only destroys that specific service, leaving "api" and "worker" untouched.

## Limitations

1. **Computed counts** - The count value must be known before OpenTofu performs any actions
2. **Cannot reference resources** - Cannot use `count = length(google_cloud_run_service.other.*)` directly
3. **No semantic keys** - Only numeric indices available

## Best Practices

1. Use `count` for simple, ordered lists of similar resources
2. Use `for_each` when resources need meaningful identifiers
3. Avoid count when list order might change
4. Use `count = 0` or `count = 1` for conditional resource creation
5. Document the purpose of counted resources
6. Consider future modifications when choosing count vs for_each
7. Use `length()` function for dynamic counts based on variable lists
8. Always handle edge cases (empty lists, count = 0)

## Examples

### IAM Bindings

```hcl
variable "service_account_roles" {
  type = list(string)
  default = [
    "roles/secretmanager.secretAccessor",
    "roles/cloudsql.client",
    "roles/cloudtrace.agent"
  ]
}

resource "google_project_iam_member" "sa_roles" {
  count   = length(var.service_account_roles)
  project = var.project_id
  role    = var.service_account_roles[count.index]
  member  = "serviceAccount:${google_service_account.app.email}"
}
```

### Environment-Specific Resources

```hcl
variable "create_staging" {
  type    = bool
  default = true
}

resource "google_cloud_run_service" "staging" {
  count    = var.create_staging ? 1 : 0
  name     = "api-staging"
  location = var.region

  template {
    spec {
      containers {
        image = var.staging_image
      }
    }
  }
}
```

### Multiple Secrets from Names

```hcl
variable "secret_names" {
  type = list(string)
  default = [
    "api-key",
    "database-url",
    "jwt-secret",
    "encryption-key"
  ]
}

resource "google_secret_manager_secret" "secrets" {
  count     = length(var.secret_names)
  secret_id = var.secret_names[count.index]
  project   = var.project_id

  labels = {
    app   = var.app_name
    index = count.index
  }

  replication {
    auto {}
  }
}
```
