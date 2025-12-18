# google_service_account

Creates and manages Google Cloud Platform service accounts for application authentication and authorization.

## Example Usage

### Basic Service Account

```hcl
resource "google_service_account" "default" {
  account_id   = "my-service-account"
  display_name = "My Service Account"
  description  = "Service account for application authentication"
}
```

### Service Account with Project Override

```hcl
resource "google_service_account" "custom_project" {
  account_id   = "cross-project-sa"
  display_name = "Cross-Project Service Account"
  project      = "other-project-id"
}
```

### Disabled Service Account

```hcl
resource "google_service_account" "disabled" {
  account_id   = "disabled-sa"
  display_name = "Disabled Service Account"
  disabled     = true
}
```

### Service Account with Ignore Existing

```hcl
resource "google_service_account" "safe_create" {
  account_id                   = "existing-sa"
  display_name                 = "Existing Service Account"
  create_ignore_already_exists = true
}
```

### Complete Example with IAM Roles

```hcl
resource "google_service_account" "app" {
  account_id   = "app-service-account"
  display_name = "Application Service Account"
  description  = "Service account for Cloud Run application"
}

# Grant roles to service account
resource "google_project_iam_member" "app_storage" {
  project = var.project_id
  role    = "roles/storage.objectViewer"
  member  = google_service_account.app.member
}

resource "google_project_iam_member" "app_secretmanager" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = google_service_account.app.member
}

# Use service account in Cloud Run
resource "google_cloud_run_service" "app" {
  name     = "my-app"
  location = "us-central1"

  template {
    spec {
      service_account_name = google_service_account.app.email
      # ...
    }
  }
}
```

## Argument Reference

### Required Arguments

- **account_id**: Service account identifier (email prefix)
  - Must be 6-30 characters long
  - Must start with lowercase letter
  - Can contain lowercase letters, numbers, and hyphens
  - Forms email: `{account_id}@{project}.iam.gserviceaccount.com`

### Optional Arguments

- **display_name**: Human-readable name displayed in console
  - Max 100 characters
  - Can be updated without recreation
- **description**: Text description of the service account
  - Max 256 UTF-8 bytes
- **disabled**: Disable service account after creation (default: false)
  - Disabled accounts cannot authenticate
  - Can be re-enabled by setting to false
- **project**: GCP project ID (defaults to provider project)
- **create_ignore_already_exists**: Prevent errors if account exists (default: false)
  - Useful for importing pre-existing accounts
  - Does not import the account into state

## Attributes Reference

- **id**: Resource identifier (format: `projects/{project}/serviceAccounts/{email}`)
- **email**: Service account email address
  - Format: `{account_id}@{project}.iam.gserviceaccount.com`
- **name**: Fully-qualified service account name (same as `id`)
- **unique_id**: Numeric unique identifier (21 digits)
- **member**: IAM member format for role bindings
  - Format: `serviceAccount:{email}`
  - Use in `google_project_iam_member` and similar resources

## Import

Import formats:

```bash
# Using email (recommended)
terraform import google_service_account.default my-sa@my-project.iam.gserviceaccount.com

# Using full path
terraform import google_service_account.default projects/my-project/serviceAccounts/my-sa@my-project.iam.gserviceaccount.com
```

## Timeouts

Default creation timeout: 5 minutes.

```hcl
timeouts {
  create = "5m"
}
```

## Important Considerations

1. **Email format**: Service account emails follow the format `{account_id}@{project}.iam.gserviceaccount.com`. The account_id becomes the email prefix.

2. **Deletion and recreation**: Deleting and recreating a service account requires reapplying all IAM roles and permissions. Consider:
   - Documenting all roles in Terraform
   - Using `disabled = true` instead of deletion for temporary deactivation
   - Planning for key rotation and permission reapplication

3. **Creation timing**: Service account creation is eventually consistent. Immediate use may fail:

```hcl
# Add delay for immediate IAM usage
resource "time_sleep" "wait_for_sa" {
  depends_on      = [google_service_account.app]
  create_duration = "30s"
}

resource "google_project_iam_member" "app_role" {
  depends_on = [time_sleep.wait_for_sa]
  project    = var.project_id
  role       = "roles/storage.objectViewer"
  member     = google_service_account.app.member
}
```

4. **Unique ID vs Email**: The `unique_id` is a numeric identifier that never changes, while the email can be reused after deletion (after a grace period). Use `unique_id` for audit logging.

5. **Member attribute**: Use the `member` attribute for IAM bindings instead of manually constructing the format:

```hcl
# Good
member = google_service_account.app.member

# Avoid
member = "serviceAccount:${google_service_account.app.email}"
```

6. **Service account limits**: Each project has limits:
   - Max 100 service accounts per project (default)
   - Request quota increases if needed
   - Use fewer service accounts with specific roles instead of many accounts

7. **Naming conventions**: Use descriptive account_ids:
   - `app-name-env` - e.g., "my-app-prod"
   - `function-purpose` - e.g., "data-pipeline-reader"
   - `service-role` - e.g., "cloudbuild-deployer"

8. **Disabled accounts**: Disabling a service account:
   - Prevents new authentication
   - Invalidates existing tokens within 1 hour
   - Preserves all IAM bindings
   - Can be re-enabled without reconfiguration

9. **Security best practices**:
   - Use separate service accounts per application/service
   - Follow principle of least privilege
   - Regularly audit service account permissions
   - Rotate service account keys regularly (if using keys)
   - Prefer Workload Identity over service account keys

10. **Cross-project usage**: Service accounts can be granted roles in other projects:

```hcl
resource "google_service_account" "app" {
  project    = "project-a"
  account_id = "cross-project-sa"
}

resource "google_project_iam_member" "cross_project" {
  project = "project-b"
  role    = "roles/storage.objectViewer"
  member  = google_service_account.app.member
}
```

## Common IAM Roles for Service Accounts

- **roles/cloudbuild.builds.builder**: Run Cloud Build builds
- **roles/run.developer**: Deploy and manage Cloud Run services
- **roles/artifactregistry.writer**: Push images to Artifact Registry
- **roles/secretmanager.secretAccessor**: Read Secret Manager secrets
- **roles/storage.objectViewer**: Read Cloud Storage objects
- **roles/storage.objectCreator**: Write Cloud Storage objects
- **roles/logging.logWriter**: Write logs to Cloud Logging
- **roles/monitoring.metricWriter**: Write metrics to Cloud Monitoring
- **roles/cloudtrace.agent**: Write traces to Cloud Trace

## Related Resources

- [google_service_account_iam](service-account-iam.md): Grant permissions to service accounts
- [google_service_account_key](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account_key): Create service account keys
- [google_project_iam_member](project-iam.md): Grant project-level permissions
