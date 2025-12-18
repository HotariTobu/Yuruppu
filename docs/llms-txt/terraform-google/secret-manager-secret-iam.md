# google_secret_manager_secret_iam

Manages IAM access policies for Secret Manager secrets. Three resources provide different levels of control.

## Resource Types

### google_secret_manager_secret_iam_policy

**Authoritative**: Sets the entire IAM policy and replaces any existing policy.

### google_secret_manager_secret_iam_binding

**Authoritative for a given role**: Manages all members for a specific role.

### google_secret_manager_secret_iam_member

**Non-authoritative**: Adds a single member to a role without affecting other members.

## Usage Constraints

- **google_secret_manager_secret_iam_policy** cannot be used with binding or member resources
- **google_secret_manager_secret_iam_binding** can only be used once per role
- **google_cloud_run_service_iam_binding** and **google_cloud_run_service_iam_member** can coexist when managing different roles

## Example Usage

### Grant Secret Access to Service Account

```hcl
resource "google_secret_manager_secret" "api_key" {
  secret_id = "api-key"

  replication {
    auto {}
  }
}

resource "google_service_account" "app" {
  account_id   = "my-app"
  display_name = "Application Service Account"
}

# Grant read access to secret
resource "google_secret_manager_secret_iam_member" "app" {
  secret_id = google_secret_manager_secret.api_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.app.member
}
```

### Grant Access to Multiple Members

```hcl
resource "google_secret_manager_secret_iam_binding" "accessors" {
  secret_id = google_secret_manager_secret.api_key.id
  role      = "roles/secretmanager.secretAccessor"

  members = [
    "serviceAccount:app1@my-project.iam.gserviceaccount.com",
    "serviceAccount:app2@my-project.iam.gserviceaccount.com",
    "user:admin@example.com",
  ]
}
```

### Set Complete IAM Policy

```hcl
data "google_iam_policy" "secret" {
  binding {
    role = "roles/secretmanager.secretAccessor"
    members = [
      "serviceAccount:app@my-project.iam.gserviceaccount.com",
    ]
  }

  binding {
    role = "roles/secretmanager.secretVersionAdder"
    members = [
      "serviceAccount:ci-cd@my-project.iam.gserviceaccount.com",
    ]
  }

  binding {
    role = "roles/secretmanager.admin"
    members = [
      "group:security-team@example.com",
    ]
  }
}

resource "google_secret_manager_secret_iam_policy" "policy" {
  secret_id   = google_secret_manager_secret.api_key.id
  policy_data = data.google_iam_policy.secret.policy_data
}
```

### Conditional Access

```hcl
resource "google_secret_manager_secret_iam_member" "conditional" {
  secret_id = google_secret_manager_secret.api_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:temp-app@my-project.iam.gserviceaccount.com"

  condition {
    title       = "Temporary access"
    description = "Access expires on 2024-12-31"
    expression  = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"
  }
}
```

### Cloud Run Service with Secret Access

```hcl
resource "google_secret_manager_secret" "db_password" {
  secret_id = "db-password"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "db_password" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = var.database_password
}

resource "google_service_account" "app" {
  account_id = "my-app"
}

resource "google_secret_manager_secret_iam_member" "app_db_password" {
  secret_id = google_secret_manager_secret.db_password.id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.app.member
}

resource "google_cloud_run_service" "app" {
  name     = "my-app"
  location = "us-central1"

  template {
    spec {
      service_account_name = google_service_account.app.email

      containers {
        image = "gcr.io/my-project/my-app"

        env {
          name = "DB_PASSWORD"
          value_from {
            secret_key_ref {
              name = google_secret_manager_secret.db_password.secret_id
              key  = "latest"
            }
          }
        }
      }
    }
  }

  depends_on = [google_secret_manager_secret_iam_member.app_db_password]
}
```

### Cloud Build with Secret Access

```hcl
data "google_project_service_identity" "cloudbuild" {
  provider = google-beta
  project  = var.project_id
  service  = "cloudbuild.googleapis.com"
}

resource "google_secret_manager_secret_iam_member" "cloudbuild" {
  secret_id = google_secret_manager_secret.api_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${data.google_project_service_identity.cloudbuild.email}"
}
```

## Argument Reference

### Common Arguments

- **secret_id**: Secret resource ID or full name (required)
  - Can be just the secret_id: `"my-secret"`
  - Or full resource name: `"projects/my-project/secrets/my-secret"`
- **project**: GCP project ID (defaults to provider project)
- **role**: IAM role to grant (required for binding/member)

### google_secret_manager_secret_iam_policy

- **policy_data**: Complete IAM policy JSON (required)
  - Generated using `google_iam_policy` data source

### google_secret_manager_secret_iam_binding

- **members**: List of identity members (required)
- **condition**: Optional IAM condition

### google_secret_manager_secret_iam_member

- **member**: Single identity member (required)
- **condition**: Optional IAM condition

## Member Formats

- **user:{emailid}**: Individual Google Account
- **serviceAccount:{emailid}**: Service account
- **group:{emailid}**: Google Group
- **domain:{domain}**: G Suite/Workspace domain
- **principal://iam.googleapis.com/...**: Workload Identity
- **principal://iam.googleapis.com/locations/global/workforcePools/...**: Workforce Identity

## IAM Roles for Secret Manager

Common Secret Manager IAM roles:

- **roles/secretmanager.secretAccessor**: Read secret values
  - Most common for application access
  - Required to access secret versions
  - Does NOT allow listing secrets or viewing metadata
- **roles/secretmanager.viewer**: View secret metadata
  - View secret configuration
  - Cannot access secret values
  - Good for auditing/monitoring
- **roles/secretmanager.secretVersionAdder**: Add new secret versions
  - Create new versions
  - Cannot read existing versions
  - Useful for CI/CD pipelines
- **roles/secretmanager.secretVersionManager**: Manage versions
  - Add, disable, destroy versions
  - Cannot read secret values
- **roles/secretmanager.admin**: Complete secret administration
  - All permissions including IAM management
  - Use sparingly

## IAM Conditions

Enable time-based or context-aware access:

```hcl
condition {
  title       = "Limited time access"
  description = "Access expires on 2024-12-31"
  expression  = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"
}
```

### Common Condition Examples

Time-based:

```hcl
# Expires at specific time
expression = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"

# Business hours only (UTC)
expression = "request.time.getHours(\"UTC\") >= 9 && request.time.getHours(\"UTC\") < 17"
```

Resource-based:

```hcl
# Specific secret only
expression = "resource.name.endsWith(\"/secrets/production-api-key\")"

# Secrets with specific label
expression = "resource.labels[\"environment\"] == \"production\""
```

**Note**: Secret Manager has some known limitations with IAM Conditions. Refer to [Cloud IAM Conditions documentation](https://cloud.google.com/iam/docs/conditions-overview) for details.

## Import

### Import Binding

```bash
# Format: "projects/{project}/secrets/{secret_id} {role}"
terraform import google_secret_manager_secret_iam_binding.default "projects/my-project/secrets/my-secret roles/secretmanager.secretAccessor"

# Abbreviated
terraform import google_secret_manager_secret_iam_binding.default "my-secret roles/secretmanager.secretAccessor"
```

### Import Member

```bash
# Format: "projects/{project}/secrets/{secret_id} {role} {member}"
terraform import google_secret_manager_secret_iam_member.default "projects/my-project/secrets/my-secret roles/secretmanager.secretAccessor serviceAccount:my-sa@my-project.iam.gserviceaccount.com"
```

### Import Policy

```bash
# Format: "projects/{project}/secrets/{secret_id}"
terraform import google_secret_manager_secret_iam_policy.default "projects/my-project/secrets/my-secret"
```

## Important Considerations

1. **Principle of least privilege**: Grant only the minimum required role:
   - Applications reading secrets: `roles/secretmanager.secretAccessor`
   - CI/CD adding versions: `roles/secretmanager.secretVersionAdder`
   - Administrators: `roles/secretmanager.admin`

2. **Secret accessor vs viewer**:
   - `secretAccessor` can read secret values but NOT list secrets
   - `viewer` can see metadata but NOT read secret values
   - Applications need `secretAccessor`, monitoring needs `viewer`

3. **Service identity**: Cloud services have service identities that need secret access:

```hcl
# Cloud Build service account
data "google_project_service_identity" "cloudbuild" {
  provider = google-beta
  service  = "cloudbuild.googleapis.com"
}

# Cloud Run uses your service account
resource "google_service_account" "app" {
  account_id = "my-app"
}
```

4. **Eventually consistent**: IAM changes can take up to 80 seconds to propagate. Add `depends_on` for proper ordering:

```hcl
resource "google_cloud_run_service" "app" {
  # ...
  depends_on = [google_secret_manager_secret_iam_member.app]
}
```

5. **Secret versions**: IAM permissions apply to the secret, not individual versions. A member with `secretAccessor` can read all versions.

6. **Cross-project access**: Service accounts can access secrets in other projects:

```hcl
resource "google_service_account" "app" {
  project    = "app-project"
  account_id = "my-app"
}

resource "google_secret_manager_secret_iam_member" "cross_project" {
  project   = "secrets-project"
  secret_id = google_secret_manager_secret.shared.id
  role      = "roles/secretmanager.secretAccessor"
  member    = google_service_account.app.member
}
```

7. **Workload Identity**: Use Workload Identity for GKE workloads:

```hcl
resource "google_secret_manager_secret_iam_member" "workload_identity" {
  secret_id = google_secret_manager_secret.api_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "principal://iam.googleapis.com/projects/${var.project_number}/locations/global/workloadIdentityPools/${var.project_id}.svc.id.goog/subject/ns/${var.namespace}/sa/${var.ksa_name}"
}
```

8. **Audit logging**: Enable Data Access audit logs for Secret Manager to track secret access:

```hcl
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
```

9. **Testing access**: Verify service account can access secrets:

```bash
# Impersonate service account
gcloud secrets versions access latest \
  --secret="my-secret" \
  --impersonate-service-account="my-sa@my-project.iam.gserviceaccount.com"
```

## Related Resources

- [google_secret_manager_secret](secret-manager-secret.md): Create secrets
- [google_secret_manager_secret_version](secret-manager-secret-version.md): Store secret data
- [google_service_account](service-account.md): Service accounts
- [google_cloud_run_service](cloud-run-service.md): Use secrets in Cloud Run
