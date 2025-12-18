# google_service_account_iam

Manages IAM policies for service accounts, controlling who can impersonate or manage service accounts.

## Resource Types

### google_service_account_iam_policy

**Authoritative**: Sets the entire IAM policy and replaces any existing policy.

### google_service_account_iam_binding

**Authoritative for a given role**: Manages all members for a specific role.

### google_service_account_iam_member

**Non-authoritative**: Adds a single member to a role without affecting other members.

## Usage Constraints

- **google_service_account_iam_policy** cannot be used with binding or member resources
- **google_service_account_iam_binding** can only be used once per role
- Binding and member resources can coexist when managing different roles

## Example Usage

### Allow Service Account Impersonation

```hcl
resource "google_service_account" "app" {
  account_id   = "my-app"
  display_name = "Application Service Account"
}

# Allow user to impersonate service account
resource "google_service_account_iam_member" "impersonator" {
  service_account_id = google_service_account.app.name
  role               = "roles/iam.serviceAccountUser"
  member             = "user:admin@example.com"
}
```

### Grant Token Creator Permission

```hcl
resource "google_service_account" "app" {
  account_id = "my-app"
}

resource "google_service_account" "ci_cd" {
  account_id = "ci-cd-pipeline"
}

# Allow CI/CD service account to create tokens for app service account
resource "google_service_account_iam_member" "token_creator" {
  service_account_id = google_service_account.app.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = google_service_account.ci_cd.member
}
```

### Multiple Members for Service Account Administration

```hcl
resource "google_service_account_iam_binding" "admins" {
  service_account_id = google_service_account.app.name
  role               = "roles/iam.serviceAccountAdmin"

  members = [
    "user:admin1@example.com",
    "user:admin2@example.com",
    "group:sa-admins@example.com",
  ]
}
```

### Complete IAM Policy

```hcl
data "google_iam_policy" "sa_policy" {
  binding {
    role = "roles/iam.serviceAccountUser"
    members = [
      "user:admin@example.com",
    ]
  }

  binding {
    role = "roles/iam.serviceAccountTokenCreator"
    members = [
      "serviceAccount:ci-cd@my-project.iam.gserviceaccount.com",
    ]
  }

  binding {
    role = "roles/iam.serviceAccountAdmin"
    members = [
      "group:security-team@example.com",
    ]
  }
}

resource "google_service_account_iam_policy" "policy" {
  service_account_id = google_service_account.app.name
  policy_data        = data.google_iam_policy.sa_policy.policy_data
}
```

### Workload Identity Binding

```hcl
resource "google_service_account" "gke_app" {
  account_id   = "gke-app"
  display_name = "GKE Application Service Account"
}

# Allow Kubernetes service account to impersonate GCP service account
resource "google_service_account_iam_member" "workload_identity" {
  service_account_id = google_service_account.gke_app.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:my-project.svc.id.goog[my-namespace/my-ksa]"
}
```

### Conditional Access

```hcl
resource "google_service_account_iam_member" "conditional_impersonation" {
  service_account_id = google_service_account.app.name
  role               = "roles/iam.serviceAccountUser"
  member             = "user:temp-admin@example.com"

  condition {
    title       = "Temporary access"
    description = "Expires at end of 2024"
    expression  = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"
  }
}
```

### Cloud Build Impersonation

```hcl
data "google_project_service_identity" "cloudbuild" {
  provider = google-beta
  project  = var.project_id
  service  = "cloudbuild.googleapis.com"
}

resource "google_service_account" "deploy" {
  account_id = "deployer"
}

# Allow Cloud Build to impersonate deployment service account
resource "google_service_account_iam_member" "cloudbuild_impersonation" {
  service_account_id = google_service_account.deploy.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${data.google_project_service_identity.cloudbuild.email}"
}

resource "google_cloudbuild_trigger" "deploy" {
  name = "deploy-trigger"

  trigger_template {
    repo_name   = "my-repo"
    branch_name = "^main$"
  }

  build {
    step {
      name = "gcr.io/cloud-builders/gcloud"
      args = [
        "run", "deploy", "my-service",
        "--image", "gcr.io/my-project/my-image",
        "--impersonate-service-account", google_service_account.deploy.email
      ]
    }
  }
}
```

## Argument Reference

### Common Arguments

- **service_account_id**: Service account resource name (required)
  - Full format: `projects/{project}/serviceAccounts/{email}`
  - Can use output from `google_service_account.name`
- **role**: IAM role to grant (required for binding/member)

### google_service_account_iam_policy

- **policy_data**: Complete IAM policy JSON (required)

### google_service_account_iam_binding

- **members**: List of identity members (required)
- **condition**: Optional IAM condition

### google_service_account_iam_member

- **member**: Single identity member (required)
- **condition**: Optional IAM condition

## Member Formats

- **user:{emailid}**: Individual Google Account
- **serviceAccount:{emailid}**: Service account
- **group:{emailid}**: Google Group
- **domain:{domain}**: G Suite/Workspace domain
- **serviceAccount:{project}.svc.id.goog[{namespace}/{ksa}]**: Workload Identity (GKE)

## IAM Roles for Service Accounts

Common service account IAM roles:

- **roles/iam.serviceAccountUser**: Use service account
  - Required to attach service account to resources (VMs, Cloud Run, etc.)
  - Required for service account impersonation
  - Most common permission needed
- **roles/iam.serviceAccountTokenCreator**: Create tokens
  - Generate OAuth tokens, ID tokens, signed JWTs
  - Required for service account key-less authentication
  - Used by CI/CD pipelines
- **roles/iam.serviceAccountKeyAdmin**: Manage service account keys
  - Create, delete, list keys
  - Highly privileged - use sparingly
- **roles/iam.serviceAccountAdmin**: Complete service account administration
  - Create, update, delete service accounts
  - Manage IAM policies
  - Highest privilege level
- **roles/iam.workloadIdentityUser**: Workload Identity
  - Allow Kubernetes service accounts to impersonate GCP service accounts
  - Used in GKE with Workload Identity

## IAM Conditions

Time-based or context-aware access control:

```hcl
condition {
  title       = "Limited time impersonation"
  description = "Temporary access for migration"
  expression  = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"
}
```

## Import

### Import Binding

```bash
# Format: "projects/{project}/serviceAccounts/{email} {role}"
terraform import google_service_account_iam_binding.default "projects/my-project/serviceAccounts/my-sa@my-project.iam.gserviceaccount.com roles/iam.serviceAccountUser"
```

### Import Member

```bash
# Format: "projects/{project}/serviceAccounts/{email} {role} {member}"
terraform import google_service_account_iam_member.default "projects/my-project/serviceAccounts/my-sa@my-project.iam.gserviceaccount.com roles/iam.serviceAccountUser user:admin@example.com"
```

### Import Policy

```bash
# Format: "projects/{project}/serviceAccounts/{email}"
terraform import google_service_account_iam_policy.default "projects/my-project/serviceAccounts/my-sa@my-project.iam.gserviceaccount.com"
```

## Important Considerations

1. **Service account user role**: The most common use case is granting `roles/iam.serviceAccountUser` to allow:
   - Attaching service account to Cloud Run services
   - Creating Compute Engine instances with service account
   - Cloud Build impersonation
   - Manual impersonation with `gcloud --impersonate-service-account`

2. **Impersonation security**: Service account impersonation is powerful. Grant sparingly:

```hcl
# Bad: Too permissive
member = "domain:example.com"

# Good: Specific users/services
member = "user:admin@example.com"
member = "serviceAccount:ci-cd@my-project.iam.gserviceaccount.com"
```

3. **Token creator vs user**:
   - `serviceAccountUser`: Attach SA to resources, full impersonation
   - `serviceAccountTokenCreator`: Generate tokens only (more limited)
   - Use `tokenCreator` when possible for least privilege

4. **Workload Identity setup**: Complete setup requires two bindings:

```hcl
# 1. Allow K8s SA to impersonate GCP SA
resource "google_service_account_iam_member" "workload_identity" {
  service_account_id = google_service_account.app.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${var.namespace}/${var.ksa_name}]"
}

# 2. Annotate K8s service account (outside Terraform)
# kubectl annotate serviceaccount ${KSA_NAME} \
#   iam.gke.io/gcp-service-account=${GSA_EMAIL}
```

5. **Service account chains**: Be cautious with service account chains (SA1 can impersonate SA2, SA2 can impersonate SA3). Document the chain clearly.

6. **Audit logging**: Enable audit logs to track service account impersonation:

```hcl
resource "google_project_iam_audit_config" "iam" {
  project = var.project_id
  service = "iam.googleapis.com"

  audit_log_config {
    log_type = "ADMIN_READ"
  }
  audit_log_config {
    log_type = "DATA_READ"
  }
  audit_log_config {
    log_type = "DATA_WRITE"
  }
}
```

7. **Testing impersonation**: Verify impersonation works:

```bash
# Impersonate service account
gcloud auth print-access-token \
  --impersonate-service-account=my-sa@my-project.iam.gserviceaccount.com

# List resources as service account
gcloud storage buckets list \
  --impersonate-service-account=my-sa@my-project.iam.gserviceaccount.com
```

8. **Key management**: Avoid using `serviceAccountKeyAdmin` when possible. Prefer:
   - Workload Identity (for GKE)
   - Service account impersonation
   - Short-lived tokens from `serviceAccountTokenCreator`

9. **Organizational policies**: Organizations may have policies restricting service account usage:
   - `iam.disableServiceAccountKeyCreation`
   - `iam.disableServiceAccountCreation`
   - Check with your security team

10. **Cross-project impersonation**: Service accounts in one project can impersonate service accounts in another:

```hcl
resource "google_service_account" "project_a_sa" {
  project    = "project-a"
  account_id = "app-sa"
}

resource "google_service_account" "project_b_sa" {
  project    = "project-b"
  account_id = "privileged-sa"
}

# Allow project A's SA to impersonate project B's SA
resource "google_service_account_iam_member" "cross_project" {
  service_account_id = google_service_account.project_b_sa.name
  role               = "roles/iam.serviceAccountUser"
  member             = google_service_account.project_a_sa.member
}
```

## Related Resources

- [google_service_account](service-account.md): Create service accounts
- [google_project_iam](project-iam.md): Project-level IAM
- [google_cloud_run_service](cloud-run-service.md): Use service accounts in Cloud Run
