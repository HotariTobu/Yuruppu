# google_secret_manager_secret_version

Manages secret versions in Google Cloud Secret Manager. Stores the actual secret data for a secret.

## Example Usage

### Basic Secret Version

```hcl
resource "google_secret_manager_secret" "secret" {
  secret_id = "my-secret"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "basic" {
  secret      = google_secret_manager_secret.secret.id
  secret_data = "my-secret-value"
}
```

### Secret Version with Base64 Encoding

```hcl
resource "google_secret_manager_secret_version" "binary" {
  secret                 = google_secret_manager_secret.secret.id
  secret_data            = base64encode(file("${path.module}/secret-file.bin"))
  is_secret_data_base64  = true
}
```

### Write-Only Secret Version

```hcl
resource "google_secret_manager_secret_version" "write_only" {
  secret         = google_secret_manager_secret.secret.id
  secret_data_wo = var.api_key  # Value not stored in state file
}
```

### Disabled Secret Version

```hcl
resource "google_secret_manager_secret_version" "disabled" {
  secret      = google_secret_manager_secret.secret.id
  secret_data = "my-secret-value"
  enabled     = false
}
```

### Secret Version with Deletion Policy

```hcl
resource "google_secret_manager_secret_version" "abandon" {
  secret          = google_secret_manager_secret.secret.id
  secret_data     = "my-secret-value"
  deletion_policy = "ABANDON"  # Leave version on destroy
}
```

### Secret Version with Lifecycle Management

```hcl
resource "google_secret_manager_secret_version" "rolling" {
  secret      = google_secret_manager_secret.secret.id
  secret_data = var.database_password

  lifecycle {
    create_before_destroy = true
  }
}
```

## Argument Reference

### Required Arguments

- **secret**: Reference to the parent `google_secret_manager_secret` resource

### Secret Data (Choose One)

- **secret_data**: The secret data (max 64KiB)
  - Stored in Terraform state file
  - Readable via API
  - Use for most cases

- **secret_data_wo**: Write-only secret data (max 64KiB)
  - NOT stored in Terraform state file
  - NOT returned by API reads
  - Use for sensitive data
  - Requires `secret_data_wo_version` to trigger updates

### Optional Arguments

- **enabled**: Controls whether the version is enabled (default: true)
  - `true`: Version can be accessed
  - `false`: Version is disabled but not deleted
- **deletion_policy**: Behavior on destroy (default: "DELETE")
  - `DELETE`: Permanently delete the version
  - `DISABLE`: Disable the version but keep it
  - `ABANDON`: Leave the version unchanged in GCP
- **is_secret_data_base64**: Set to true if `secret_data` is base64-encoded (default: false)
- **secret_data_wo_version**: Counter to trigger updates for write-only secrets
  - Increment this value to force secret data update
- **project**: GCP project ID

## Attributes Reference

- **id**: Resource identifier (format: `projects/{project}/secrets/{secret_id}/versions/{version}`)
- **name**: Full resource name (same as `id`)
- **version**: Secret version number (e.g., "1", "2", "3")
- **create_time**: Version creation timestamp (RFC3339)
- **destroy_time**: Version destruction timestamp (RFC3339, only after destruction)

## Import

Import formats:

```bash
# Full path
terraform import google_secret_manager_secret_version.default projects/my-project/secrets/my-secret/versions/1

# Abbreviated
terraform import google_secret_manager_secret_version.default my-project/my-secret/1
terraform import google_secret_manager_secret_version.default my-secret/1
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

1. **Resource replacement**: Modifying `secret_data` triggers resource replacement. Use `create_before_destroy` to prevent service disruptions:

```hcl
resource "google_secret_manager_secret_version" "api_key" {
  secret      = google_secret_manager_secret.secret.id
  secret_data = var.api_key

  lifecycle {
    create_before_destroy = true
  }
}
```

2. **State file security**: All arguments including `secret_data` are stored in the Terraform state file as plain text. Secure your state file by:
   - Using remote state with encryption (e.g., GCS with encryption)
   - Restricting state file access with IAM
   - Using `secret_data_wo` for highly sensitive data

3. **Write-only secrets**: Use `secret_data_wo` for data that should never be stored in state:

```hcl
resource "google_secret_manager_secret_version" "password" {
  secret                 = google_secret_manager_secret.secret.id
  secret_data_wo         = var.database_password
  secret_data_wo_version = 1  # Increment to trigger update
}
```

To update write-only secrets, increment `secret_data_wo_version`:

```hcl
secret_data_wo_version = 2  # Increment when secret_data_wo changes
```

4. **Version lifecycle**: Versions follow this lifecycle:
   - **Enabled**: Active and accessible
   - **Disabled**: Inactive but recoverable (enable it again)
   - **Destroyed**: Scheduled for deletion (destroy_time set)
   - **Permanently deleted**: Removed after TTL expires

5. **Deletion policies**:
   - **DELETE**: Standard behavior, permanently removes version
   - **DISABLE**: Useful for compliance/audit requirements
   - **ABANDON**: For pre-existing secrets imported into Terraform

6. **Binary data**: For binary secrets, use base64 encoding:

```hcl
resource "google_secret_manager_secret_version" "certificate" {
  secret                = google_secret_manager_secret.secret.id
  secret_data           = base64encode(file("${path.module}/cert.pem"))
  is_secret_data_base64 = true
}
```

7. **Secret size limit**: Maximum 64KiB (65536 bytes). For larger secrets:
   - Split into multiple secrets
   - Store in Cloud Storage and reference the path
   - Use Secret Manager for the encryption key only

8. **Version references**: Access specific versions or use "latest":

```hcl
# Data source for version access
data "google_secret_manager_secret_version" "latest" {
  secret = google_secret_manager_secret.secret.id
  # Defaults to latest version
}

data "google_secret_manager_secret_version" "specific" {
  secret  = google_secret_manager_secret.secret.id
  version = "2"
}

# Use in other resources
resource "google_cloud_run_service" "app" {
  # ...
  template {
    spec {
      containers {
        env {
          name = "API_KEY"
          value_from {
            secret_key_ref {
              name = google_secret_manager_secret.secret.secret_id
              key  = "latest"  # or specific version
            }
          }
        }
      }
    }
  }
}
```

## Related Resources

- [google_secret_manager_secret](secret-manager-secret.md): Create secret metadata
- [google_secret_manager_secret_iam](secret-manager-secret-iam.md): Manage access control
