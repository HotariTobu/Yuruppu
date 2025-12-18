# google_project_iam

Manages IAM policies at the Google Cloud project level. Four resources provide different management approaches.

## Resource Types

### google_project_iam_policy

**Authoritative**: Sets the entire IAM policy and replaces any existing policy.

**Warning**: This will overwrite all existing IAM bindings. Use with extreme caution.

### google_project_iam_binding

**Authoritative for a given role**: Manages all members for a specific role.

### google_project_iam_member

**Non-authoritative**: Adds a single member to a role without affecting other members.

### google_project_iam_audit_config

**Authoritative for a given service**: Configures audit logging for specific services.

## Usage Constraints

- **google_project_iam_policy** cannot be used with binding, member, or audit config resources
- **google_project_iam_binding** can only be used once per role
- Binding and member resources can coexist when managing different roles

## Example Usage

### Grant Project-Level Role to Service Account

```hcl
resource "google_service_account" "app" {
  account_id   = "my-app"
  display_name = "Application Service Account"
}

# Grant Cloud Run developer role
resource "google_project_iam_member" "app_run_developer" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = google_service_account.app.member
}

# Grant storage access
resource "google_project_iam_member" "app_storage" {
  project = var.project_id
  role    = "roles/storage.objectViewer"
  member  = google_service_account.app.member
}
```

### Multiple Members for a Role

```hcl
resource "google_project_iam_binding" "editors" {
  project = var.project_id
  role    = "roles/editor"

  members = [
    "user:alice@example.com",
    "user:bob@example.com",
    "group:developers@example.com",
  ]
}
```

### Complete Project IAM Policy

```hcl
data "google_iam_policy" "project" {
  binding {
    role = "roles/viewer"
    members = [
      "user:viewer@example.com",
      "group:viewers@example.com",
    ]
  }

  binding {
    role = "roles/editor"
    members = [
      "user:editor@example.com",
    ]
  }

  binding {
    role = "roles/owner"
    members = [
      "user:admin@example.com",
    ]
  }
}

resource "google_project_iam_policy" "project" {
  project     = var.project_id
  policy_data = data.google_iam_policy.project.policy_data
}
```

### Audit Logging Configuration

```hcl
# Enable audit logging for Cloud Storage
resource "google_project_iam_audit_config" "storage" {
  project = var.project_id
  service = "storage.googleapis.com"

  audit_log_config {
    log_type = "DATA_READ"
  }

  audit_log_config {
    log_type = "DATA_WRITE"
  }
}

# Enable audit logging for Secret Manager
resource "google_project_iam_audit_config" "secret_manager" {
  project = var.project_id
  service = "secretmanager.googleapis.com"

  audit_log_config {
    log_type = "DATA_READ"
  }

  audit_log_config {
    log_type = "DATA_WRITE"
  }
}

# Enable audit logging for all services
resource "google_project_iam_audit_config" "all_services" {
  project = var.project_id
  service = "allServices"

  audit_log_config {
    log_type = "ADMIN_READ"
  }
}
```

### Audit Logging with Exemptions

```hcl
resource "google_project_iam_audit_config" "compute_audit" {
  project = var.project_id
  service = "compute.googleapis.com"

  audit_log_config {
    log_type = "DATA_READ"
    exempted_members = [
      "serviceAccount:monitoring@my-project.iam.gserviceaccount.com",
    ]
  }

  audit_log_config {
    log_type = "DATA_WRITE"
  }
}
```

### Conditional IAM Binding

```hcl
resource "google_project_iam_member" "conditional_editor" {
  project = var.project_id
  role    = "roles/editor"
  member  = "user:temp-admin@example.com"

  condition {
    title       = "Temporary editor access"
    description = "Access expires on 2024-12-31"
    expression  = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"
  }
}
```

### Service Account Setup Example

```hcl
resource "google_service_account" "cloudbuild" {
  account_id   = "cloudbuild-sa"
  display_name = "Cloud Build Service Account"
}

# Grant necessary permissions for Cloud Build
resource "google_project_iam_member" "cloudbuild_builder" {
  project = var.project_id
  role    = "roles/cloudbuild.builds.builder"
  member  = google_service_account.cloudbuild.member
}

resource "google_project_iam_member" "cloudbuild_storage" {
  project = var.project_id
  role    = "roles/storage.admin"
  member  = google_service_account.cloudbuild.member
}

resource "google_project_iam_member" "cloudbuild_artifact_registry" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = google_service_account.cloudbuild.member
}

resource "google_project_iam_member" "cloudbuild_run_developer" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = google_service_account.cloudbuild.member
}

resource "google_project_iam_member" "cloudbuild_sa_user" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = google_service_account.cloudbuild.member
}
```

## Argument Reference

### Common Arguments

- **project**: GCP project ID (required)
- **role**: IAM role to grant (required for binding/member)

### google_project_iam_policy

- **policy_data**: Complete IAM policy JSON (required)

### google_project_iam_binding

- **members**: List of identity members (required)
- **condition**: Optional IAM condition

### google_project_iam_member

- **member**: Single identity member (required)
- **condition**: Optional IAM condition

### google_project_iam_audit_config

- **service**: Service to configure audit logging (required)
  - Use `"allServices"` for all services
  - Or specific service: `"storage.googleapis.com"`, `"secretmanager.googleapis.com"`, etc.
- **audit_log_config**: One or more audit log configurations (required)
  - **log_type**: `ADMIN_READ`, `DATA_READ`, or `DATA_WRITE`
  - **exempted_members**: Optional list of members exempt from logging

## Member Formats

- **user:{emailid}**: Individual Google Account
- **serviceAccount:{emailid}**: Service account
- **group:{emailid}**: Google Group
- **domain:{domain}**: G Suite/Workspace domain
- **allUsers**: Anyone on the internet
- **allAuthenticatedUsers**: Any authenticated Google Account
- **principal://iam.googleapis.com/...**: Workload Identity
- **principalSet://iam.googleapis.com/...**: Workload Identity pool

## Common Project IAM Roles

### Basic Roles (Legacy - Use Predefined Roles Instead)

- **roles/viewer**: Read-only access to all resources
- **roles/editor**: Read-write access to all resources
- **roles/owner**: Full control including IAM management

### Predefined Roles (Recommended)

**Cloud Run:**
- **roles/run.invoker**: Invoke services
- **roles/run.developer**: Deploy and manage services
- **roles/run.admin**: Full Cloud Run administration

**Cloud Build:**
- **roles/cloudbuild.builds.builder**: Run builds
- **roles/cloudbuild.builds.editor**: Manage builds

**Artifact Registry:**
- **roles/artifactregistry.reader**: Pull artifacts
- **roles/artifactregistry.writer**: Push and pull artifacts
- **roles/artifactregistry.repoAdmin**: Manage repositories

**Secret Manager:**
- **roles/secretmanager.secretAccessor**: Read secrets
- **roles/secretmanager.secretVersionAdder**: Create secret versions
- **roles/secretmanager.admin**: Full secret management

**Storage:**
- **roles/storage.objectViewer**: Read objects
- **roles/storage.objectCreator**: Write objects
- **roles/storage.objectAdmin**: Full object management

**Service Accounts:**
- **roles/iam.serviceAccountUser**: Use service accounts
- **roles/iam.serviceAccountTokenCreator**: Create tokens

**Logging and Monitoring:**
- **roles/logging.logWriter**: Write logs
- **roles/monitoring.metricWriter**: Write metrics

## Audit Log Types

- **ADMIN_READ**: Administrative read operations (e.g., list resources)
- **DATA_READ**: Data read operations (e.g., read object content)
- **DATA_WRITE**: Data write operations (e.g., create, update, delete)

## IAM Conditions

Enable context-aware access:

```hcl
condition {
  title       = "Limited time access"
  description = "Access expires on 2024-12-31"
  expression  = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"
}
```

### Common Condition Examples

```hcl
# Time-based expiration
expression = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"

# Business hours (UTC)
expression = "request.time.getHours(\"UTC\") >= 9 && request.time.getHours(\"UTC\") < 17"

# Weekdays only
expression = "request.time.getDayOfWeek(\"UTC\") >= 1 && request.time.getDayOfWeek(\"UTC\") <= 5"

# Resource name pattern
expression = "resource.name.startsWith(\"projects/my-project/locations/us-central1/\")"

# Resource type and labels
expression = "resource.type == \"storage.googleapis.com/Bucket\" && resource.labels.environment == \"production\""
```

## Import

### Import Binding

```bash
terraform import google_project_iam_binding.default "my-project roles/editor"
```

### Import Member

```bash
terraform import google_project_iam_member.default "my-project roles/editor user:alice@example.com"
```

### Import Policy

```bash
terraform import google_project_iam_policy.default "my-project"
```

### Import Audit Config

```bash
terraform import google_project_iam_audit_config.default "my-project storage.googleapis.com"
```

## Important Considerations

1. **Use IAM member for incremental changes**: For most use cases, use `google_project_iam_member` to add specific permissions without affecting other bindings.

2. **Avoid IAM policy resource**: The `google_project_iam_policy` resource is extremely dangerous as it replaces all IAM bindings. Only use when you need complete control and understand the implications.

3. **Project IAM vs Resource IAM**: Prefer resource-specific IAM when possible:

```hcl
# Less privilege: Resource-specific
resource "google_cloud_run_service_iam_member" "invoker" {
  service  = google_cloud_run_service.app.name
  location = google_cloud_run_service.app.location
  role     = "roles/run.invoker"
  member   = "user:alice@example.com"
}

# More privilege: Project-level
resource "google_project_iam_member" "run_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "user:alice@example.com"
}
```

4. **Basic roles**: Avoid using basic roles (Viewer, Editor, Owner) in production. Use predefined roles for better security:

```hcl
# Bad: Too permissive
role = "roles/editor"

# Good: Specific permissions
role = "roles/run.developer"
role = "roles/artifactregistry.writer"
```

5. **Service accounts need serviceAccountUser**: When a service account needs to perform actions, it often needs `roles/iam.serviceAccountUser`:

```hcl
# Cloud Build needs to deploy Cloud Run with a service account
resource "google_project_iam_member" "cloudbuild_sa_user" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = google_service_account.cloudbuild.member
}
```

6. **Audit logging costs**: Audit logs (especially DATA_READ) can generate significant log volume and costs. Enable selectively:

```hcl
# Good: Enable for sensitive services only
resource "google_project_iam_audit_config" "secret_manager" {
  service = "secretmanager.googleapis.com"
  # ...
}

# Expensive: Enable for all services
resource "google_project_iam_audit_config" "all" {
  service = "allServices"
  # ...
}
```

7. **Eventually consistent**: IAM changes can take up to 80 seconds to propagate. Plan accordingly.

8. **Testing permissions**: Verify service account permissions:

```bash
# Test as service account
gcloud auth activate-service-account --key-file=key.json

# Or use impersonation
gcloud run services list \
  --impersonate-service-account=my-sa@my-project.iam.gserviceaccount.com
```

9. **Organization policies**: Some organizations have policies restricting IAM changes. Check with your security team.

10. **Terraform state security**: IAM configurations in state files reveal your security model. Protect state files appropriately.

## Related Resources

- [google_service_account](service-account.md): Create service accounts
- [google_service_account_iam](service-account-iam.md): Service account IAM
- [google_cloud_run_service_iam](cloud-run-service-iam.md): Cloud Run IAM
- [google_secret_manager_secret_iam](secret-manager-secret-iam.md): Secret Manager IAM
