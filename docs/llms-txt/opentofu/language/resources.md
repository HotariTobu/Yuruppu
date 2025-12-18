# Resources

Resources are the most important element in the OpenTofu language. Each resource block describes one or more infrastructure objects, such as virtual networks, compute instances, or higher-level components such as DNS records.

## Basic Syntax

```hcl
resource "provider_resource" "local_name" {
  argument = value
}
```

A `resource` block declares a resource of a given type (e.g., `google_cloud_run_service`) with a given local name (e.g., `api`). The name is used to refer to this resource from elsewhere in the same module.

## Example: Cloud Run Service

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = "us-central1"

  template {
    spec {
      containers {
        image = "gcr.io/my-project/api:latest"

        env {
          name  = "DATABASE_URL"
          value = var.database_url
        }

        resources {
          limits = {
            cpu    = "1000m"
            memory = "512Mi"
          }
        }
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}
```

## Naming Requirements

Resource names must start with a letter or underscore, and may contain only letters, digits, underscores, and dashes.

## Resource Behavior

When you apply a configuration, OpenTofu:

1. **Create** - Creates resources that don't exist
2. **Update** - Updates resources whose arguments have changed
3. **Destroy** - Removes resources no longer in the configuration
4. **Replace** - Destroys and recreates resources when updates aren't possible

OpenTofu tracks each resource in the state file and compares the desired state (configuration) with the actual state (infrastructure).

## Meta-Arguments

Special arguments available across all resource types:

### depends_on

Specifies explicit dependencies when implicit dependencies cannot be recognized:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = "us-central1"

  depends_on = [
    google_project_service.run_api
  ]
}
```

### count

Creates multiple instances based on a count value:

```hcl
resource "google_secret_manager_secret" "secrets" {
  count     = length(var.secret_names)
  secret_id = var.secret_names[count.index]

  replication {
    auto {}
  }
}
```

### for_each

Creates instances from maps or sets:

```hcl
resource "google_project_iam_member" "members" {
  for_each = toset(var.members)

  project = var.project_id
  role    = "roles/run.invoker"
  member  = each.value
}
```

### provider

Selects a non-default provider configuration:

```hcl
resource "google_cloud_run_service" "api" {
  provider = google.us-west

  name     = "api-service"
  location = "us-west1"
}
```

### lifecycle

Customizes resource lifecycle behavior:

```hcl
resource "google_sql_database_instance" "main" {
  name             = "main-instance"
  database_version = "POSTGRES_15"

  lifecycle {
    prevent_destroy = true
    ignore_changes  = [settings[0].disk_size]
  }
}
```

## Resource References

Reference other resources using the syntax:

```hcl
resource_type.name.attribute
```

Example:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = "us-central1"

  template {
    spec {
      containers {
        image = "gcr.io/${var.project_id}/api:latest"
      }

      service_account_name = google_service_account.api.email
    }
  }
}

resource "google_service_account" "api" {
  account_id   = "api-service"
  display_name = "API Service Account"
}
```

OpenTofu automatically understands that the Cloud Run service depends on the service account and will create them in the correct order.

## Provisioners

Provisioners execute actions after resource creation. They should be considered a last resort due to their non-declarative nature.

```hcl
resource "google_compute_instance" "vm" {
  name         = "example-vm"
  machine_type = "e2-medium"

  provisioner "local-exec" {
    command = "echo ${self.network_interface[0].network_ip} >> private_ips.txt"
  }
}
```

## Custom Conditions

Add validation through `precondition` and `postcondition` blocks within lifecycle:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.location

  lifecycle {
    precondition {
      condition     = contains(["us-central1", "us-east1"], var.location)
      error_message = "Location must be us-central1 or us-east1."
    }
  }
}
```

## Timeouts

Some resource types support custom timeout configurations:

```hcl
resource "google_sql_database_instance" "main" {
  name             = "main-instance"
  database_version = "POSTGRES_15"

  timeouts {
    create = "30m"
    update = "30m"
    delete = "30m"
  }
}
```

## Resource Addressing

Resources are addressed using:

- Simple: `google_cloud_run_service.api`
- With count: `google_secret_manager_secret.secrets[0]`
- With for_each: `google_project_iam_member.members["user:alice@example.com"]`
- In modules: `module.network.google_compute_network.vpc`
