# lifecycle Meta-Argument

The `lifecycle` block is a meta-argument available in all resource declarations that customizes how OpenTofu manages infrastructure objects throughout their lifecycle.

## Basic Structure

```hcl
resource "google_sql_database_instance" "main" {
  name             = "main-instance"
  database_version = "POSTGRES_15"

  lifecycle {
    create_before_destroy = true
    prevent_destroy       = true
    ignore_changes        = [settings[0].disk_size]
  }
}
```

## Arguments

### create_before_destroy

When enabled, OpenTofu creates the replacement resource before destroying the old one, rather than the default destroy-then-create sequence.

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  template {
    spec {
      containers {
        image = var.container_image
      }
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}
```

**Use cases:**
- Resources that must remain available during updates
- Resources where downtime is unacceptable
- Resources with name constraints that allow temporary duplicates

**Considerations:**
- Requires that concurrent objects don't violate naming requirements
- May require additional resources temporarily
- Dependencies must also support this pattern

### prevent_destroy

Setting this to `true` causes OpenTofu to reject any plan that would destroy the associated infrastructure.

```hcl
resource "google_sql_database_instance" "main" {
  name             = "production-db"
  database_version = "POSTGRES_15"

  lifecycle {
    prevent_destroy = true
  }
}
```

**Use cases:**
- Protecting production databases
- Safeguarding stateful resources
- Preventing accidental deletion of critical infrastructure

**Limitations:**
- Does not prevent destruction via `tofu destroy`
- Does not prevent destruction if the resource block is removed from configuration
- Can be overridden with `-refresh=false` or by removing the lifecycle block

**Warning:** This is not a comprehensive solution for protecting resources. Use additional safeguards like IAM policies and backup strategies.

### ignore_changes

Specifies resource attributes that OpenTofu should disregard during update planning.

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  template {
    spec {
      containers {
        image = var.container_image
      }
    }

    metadata {
      labels = {
        managed-by = "opentofu"
        version    = "1.0.0"
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].metadata[0].labels["version"],
      template[0].spec[0].containers[0].image
    ]
  }
}
```

**Ignore all attributes:**

```hcl
resource "google_compute_instance" "vm" {
  name         = "example-vm"
  machine_type = "e2-medium"

  lifecycle {
    ignore_changes = all
  }
}
```

This prevents all updates while permitting creation and destruction.

**Use cases:**
- Resources modified by external processes (auto-scalers, controllers)
- Attributes managed by other systems
- Temporary workarounds for provider issues
- Resources where certain fields change frequently outside OpenTofu

**Common scenarios:**

Ignore auto-generated labels:
```hcl
lifecycle {
  ignore_changes = [
    metadata[0].labels,
    template[0].metadata[0].annotations
  ]
}
```

Ignore scaling configuration managed elsewhere:
```hcl
lifecycle {
  ignore_changes = [
    template[0].metadata[0].annotations["autoscaling.knative.dev/minScale"],
    template[0].metadata[0].annotations["autoscaling.knative.dev/maxScale"]
  ]
}
```

### replace_triggered_by

Automatically replaces a resource when referenced managed resources or their attributes change.

```hcl
resource "google_secret_manager_secret_version" "api_key" {
  secret      = google_secret_manager_secret.api_key.id
  secret_data = var.api_key_value
}

resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  template {
    spec {
      containers {
        image = var.container_image
      }
    }
  }

  lifecycle {
    replace_triggered_by = [
      google_secret_manager_secret_version.api_key
    ]
  }
}
```

This replaces the Cloud Run service whenever the secret version changes.

**Use cases:**
- Coordinating updates across dependent resources
- Triggering replacements based on external changes
- Ensuring fresh deployments when dependencies change

## Preconditions and Postconditions

Custom validation checks that document assumptions and surface configuration errors early.

### Precondition

Validates conditions before resource operations:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  lifecycle {
    precondition {
      condition     = contains(["us-central1", "us-east1", "europe-west1"], var.region)
      error_message = "Region must be one of: us-central1, us-east1, europe-west1."
    }

    precondition {
      condition     = var.min_instances <= var.max_instances
      error_message = "min_instances must be less than or equal to max_instances."
    }
  }

  template {
    spec {
      containers {
        image = var.container_image
      }
    }
  }
}
```

### Postcondition

Validates resource state after operations:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  template {
    spec {
      containers {
        image = var.container_image
      }
    }
  }

  lifecycle {
    postcondition {
      condition     = self.status[0].url != ""
      error_message = "Service URL must be available after deployment."
    }

    postcondition {
      condition     = self.status[0].conditions[0].status == "True"
      error_message = "Service must be in ready state after deployment."
    }
  }
}
```

Access the resource's attributes using `self`:

```hcl
lifecycle {
  postcondition {
    condition     = self.status[0].latest_ready_revision_name != ""
    error_message = "Service must have a ready revision."
  }
}
```

## Combining Lifecycle Arguments

Multiple arguments can be used together:

```hcl
resource "google_sql_database_instance" "main" {
  name             = "production-db"
  database_version = "POSTGRES_15"

  settings {
    tier            = "db-custom-2-7680"
    disk_size       = 100
    disk_autoresize = true
  }

  lifecycle {
    # Prevent accidental deletion
    prevent_destroy = true

    # Create new instance before destroying old one
    create_before_destroy = true

    # Ignore disk size changes (managed by auto-resize)
    ignore_changes = [
      settings[0].disk_size
    ]

    # Validate configuration
    precondition {
      condition     = var.environment == "production"
      error_message = "This instance should only be created in production."
    }
  }
}
```

## Important Constraint

Only literal values are permitted in lifecycle settings because dependency graph processing occurs before arbitrary expression evaluation.

**Valid:**
```hcl
lifecycle {
  prevent_destroy = true
  ignore_changes  = [settings[0].disk_size]
}
```

**Invalid:**
```hcl
lifecycle {
  prevent_destroy = var.protect_resources  # Error: variable not allowed
  ignore_changes  = var.ignored_attributes  # Error: variable not allowed
}
```

## Common Patterns

### Database Protection

```hcl
resource "google_sql_database_instance" "main" {
  name             = "production-db"
  database_version = "POSTGRES_15"

  lifecycle {
    prevent_destroy       = true
    create_before_destroy = false  # Databases can't be renamed

    ignore_changes = [
      settings[0].disk_size,  # Auto-resize manages this
      settings[0].maintenance_window  # Managed elsewhere
    ]
  }
}
```

### Zero-Downtime Deployments

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  lifecycle {
    create_before_destroy = true

    postcondition {
      condition     = self.status[0].url != ""
      error_message = "New service must be available before destroying old."
    }
  }
}
```

### Externally Managed Attributes

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  lifecycle {
    # Ignore attributes managed by Kubernetes HPA or other controllers
    ignore_changes = [
      template[0].metadata[0].annotations["autoscaling.knative.dev/minScale"],
      template[0].metadata[0].annotations["autoscaling.knative.dev/maxScale"],
      template[0].spec[0].containers[0].resources[0].limits
    ]
  }
}
```

### Environment-Specific Protection

```hcl
resource "google_cloud_run_service" "api" {
  name     = "${var.environment}-api-service"
  location = var.region

  lifecycle {
    precondition {
      condition     = var.environment != "production" || var.approval_required
      error_message = "Production deployments require approval."
    }

    postcondition {
      condition     = var.environment != "production" || self.status[0].conditions[0].status == "True"
      error_message = "Production service must be healthy after deployment."
    }
  }
}
```

## Best Practices

1. **Document why** - Comment why lifecycle arguments are needed
2. **Use preconditions** - Validate inputs early to fail fast
3. **Use postconditions** - Verify critical resource state after operations
4. **Protect stateful resources** - Use `prevent_destroy` for databases and other stateful resources
5. **Zero-downtime updates** - Use `create_before_destroy` for services requiring high availability
6. **Ignore external changes** - Use `ignore_changes` for attributes managed by other systems
7. **Test lifecycle behavior** - Verify lifecycle settings work as expected
8. **Combine judiciously** - Use multiple arguments together when needed
9. **Understand limitations** - Know what lifecycle arguments can and cannot prevent
10. **Layer protections** - Combine lifecycle settings with IAM policies and other safeguards
