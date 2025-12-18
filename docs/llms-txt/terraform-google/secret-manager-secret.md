# google_secret_manager_secret

Creates a logical secret in Google Cloud Secret Manager. Use `google_secret_manager_secret_version` to store actual secret data.

## Example Usage

### Basic Secret with Auto Replication

```hcl
resource "google_secret_manager_secret" "basic" {
  secret_id = "my-secret"

  replication {
    auto {}
  }
}
```

### Secret with User-Managed Replication

```hcl
resource "google_secret_manager_secret" "regional" {
  secret_id = "my-regional-secret"

  replication {
    user_managed {
      replicas {
        location = "us-central1"
      }
      replicas {
        location = "us-east1"
      }
    }
  }
}
```

### Secret with KMS Encryption

```hcl
resource "google_secret_manager_secret" "encrypted" {
  secret_id = "my-encrypted-secret"

  replication {
    user_managed {
      replicas {
        location = "us-central1"
        customer_managed_encryption {
          kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
        }
      }
    }
  }
}
```

### Secret with Expiration and Rotation

```hcl
resource "google_pubsub_topic" "rotation" {
  name = "secret-rotation-topic"
}

resource "google_secret_manager_secret" "expiring" {
  secret_id = "my-expiring-secret"

  ttl = "3600s"  # Expires in 1 hour

  rotation {
    next_rotation_time = "2024-12-31T00:00:00Z"
    rotation_period    = "2592000s"  # 30 days
  }

  topics {
    name = google_pubsub_topic.rotation.id
  }

  replication {
    auto {}
  }
}
```

### Secret with Version Aliases and Labels

```hcl
resource "google_secret_manager_secret" "labeled" {
  secret_id = "my-labeled-secret"

  labels = {
    environment = "production"
    service     = "api"
  }

  annotations = {
    "owner"       = "team-backend"
    "description" = "API credentials for external service"
  }

  version_aliases = {
    "stable"  = "1"
    "canary"  = "2"
    "current" = "2"
  }

  replication {
    auto {}
  }
}
```

### Secret with Deletion Protection

```hcl
resource "google_secret_manager_secret" "protected" {
  secret_id = "critical-secret"

  deletion_protection = true

  replication {
    auto {}
  }
}
```

## Argument Reference

### Required Arguments

- **secret_id**: Unique identifier within the project (must match `[a-zA-Z0-9_-]{1,255}`)
- **replication**: Replication policy (cannot be changed after creation)

### Optional Arguments

- **labels**: Key-value pairs for organization (max 64 entries)
  - Keys: 1-63 characters, lowercase letters, numbers, underscores, dashes
  - Values: 0-63 characters, same character set as keys
- **annotations**: Custom metadata (total size limited to 16KiB)
- **version_aliases**: Map version aliases to version names (max 50 aliases, 63 characters each)
- **version_destroy_ttl**: Time-to-live for deleted versions before permanent destruction (e.g., "86400s" for 24 hours)
- **topics**: Pub/Sub topics for notifications (max 10 topics)
- **expire_time**: Scheduled expiration timestamp (RFC3339 format, e.g., "2024-12-31T23:59:59Z")
- **ttl**: Duration until expiration (e.g., "3600s" for 1 hour) - cannot combine with `expire_time`
- **rotation**: Rotation schedule configuration (requires `topics`)
- **tags**: Resource Manager tags (format: `tagKeys/{id}`/`tagValues/{id}`)
- **deletion_protection**: Prevents accidental deletion (default: false)
- **project**: GCP project ID

### Replication Configuration

Choose one replication strategy:

#### Auto Replication

Automatic distribution without restrictions:

```hcl
replication {
  auto {
    customer_managed_encryption {
      kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
    }
  }
}
```

#### User-Managed Replication

Specify target regions explicitly:

```hcl
replication {
  user_managed {
    replicas {
      location = "us-central1"
      customer_managed_encryption {
        kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
      }
    }
    replicas {
      location = "us-east1"
    }
  }
}
```

### Rotation Configuration

```hcl
rotation {
  next_rotation_time = "2024-12-31T00:00:00Z"  # RFC3339 timestamp
  rotation_period    = "2592000s"               # Duration in seconds (30 days)
}
```

**Note**: Rotation requires at least one Pub/Sub topic in the `topics` list.

### Topics Configuration

```hcl
topics {
  name = "projects/my-project/topics/secret-rotation"
}
```

Up to 10 Pub/Sub topics for notifications about:
- Rotation events
- Version creation/destruction
- Secret expiration

## Attributes Reference

- **id**: Resource identifier (format: `projects/{project}/secrets/{secret_id}`)
- **name**: Full resource name (same as `id`)
- **create_time**: Creation timestamp (RFC3339)
- **terraform_labels**: Labels applied by Terraform (merges provider and resource labels)
- **effective_labels**: All labels present in GCP (including system labels)
- **effective_annotations**: All annotations present in GCP

## Import

Import formats:

```bash
# Full path
terraform import google_secret_manager_secret.default projects/my-project/secrets/my-secret

# Abbreviated
terraform import google_secret_manager_secret.default my-project/my-secret
terraform import google_secret_manager_secret.default my-secret
```

## Timeouts

Default timeouts: 20 minutes for create, update, and delete operations.

```hcl
timeouts {
  create = "20m"
  update = "20m"
  delete = "20m"
}
```

## Important Considerations

1. **Replication is immutable**: The replication policy cannot be changed after secret creation. Plan your replication strategy carefully based on:
   - Geographic requirements
   - Latency considerations
   - Compliance and data residency requirements

2. **Secret vs Secret Version**: This resource creates the secret metadata only. Use `google_secret_manager_secret_version` to store actual secret data:

```hcl
resource "google_secret_manager_secret_version" "basic" {
  secret      = google_secret_manager_secret.basic.id
  secret_data = "my-secret-value"
}
```

3. **Deletion protection**: Enable `deletion_protection = true` for critical secrets. You must explicitly disable it before deletion:

```bash
# Disable protection first
terraform apply -var="deletion_protection=false"
# Then destroy
terraform destroy
```

4. **Labels vs Annotations**:
   - **Labels**: Used for filtering, grouping, and cost allocation
   - **Annotations**: Free-form metadata for documentation purposes

5. **Version aliases**: Useful for tracking specific versions (e.g., "stable", "canary", "latest") without hardcoding version numbers:

```hcl
# Reference by alias
data "google_secret_manager_secret_version" "stable" {
  secret  = google_secret_manager_secret.labeled.id
  version = "stable"  # Resolves to the version specified in version_aliases
}
```

6. **Rotation automation**: Pub/Sub notifications enable automated rotation workflows. The rotation schedule is advisory - you must implement the actual rotation logic.

7. **KMS encryption**: Customer-managed encryption keys (CMEK) provide additional security. Ensure the Secret Manager service account has `cloudkms.cryptoKeyEncrypterDecrypter` role on the KMS key.

8. **Version TTL**: `version_destroy_ttl` provides a grace period before permanent deletion, allowing recovery of accidentally deleted versions.

## Related Resources

- [google_secret_manager_secret_version](secret-manager-secret-version.md): Store secret data
- [google_secret_manager_secret_iam](secret-manager-secret-iam.md): Manage access control
