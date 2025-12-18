# google_cloud_run_service_iam

Manages IAM policies for Cloud Run Services. Three resources provide different levels of control over IAM policies.

## Resource Types

### google_cloud_run_service_iam_policy

**Authoritative**: Sets the entire IAM policy and replaces any existing policy.

**Warning**: This resource will overwrite all existing IAM bindings. Use with caution.

### google_cloud_run_service_iam_binding

**Authoritative for a given role**: Manages all members for a specific role, replacing existing members for that role.

### google_cloud_run_service_iam_member

**Non-authoritative**: Adds a single member to a role without affecting other members.

## Usage Constraints

- **google_cloud_run_service_iam_policy** cannot be used with binding or member resources
- **google_cloud_run_service_iam_binding** and **google_cloud_run_service_iam_member** can coexist only when managing different roles

## Example Usage

### Make Service Publicly Accessible

```hcl
resource "google_cloud_run_service" "app" {
  name     = "my-app"
  location = "us-central1"

  template {
    spec {
      containers {
        image = "gcr.io/my-project/my-image"
      }
    }
  }
}

# Allow unauthenticated access
resource "google_cloud_run_service_iam_member" "public" {
  service  = google_cloud_run_service.app.name
  location = google_cloud_run_service.app.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
```

### Grant Access to Specific Users

```hcl
# Single user
resource "google_cloud_run_service_iam_member" "user" {
  service  = google_cloud_run_service.app.name
  location = google_cloud_run_service.app.location
  role     = "roles/run.invoker"
  member   = "user:alice@example.com"
}

# Service account
resource "google_cloud_run_service_iam_member" "service_account" {
  service  = google_cloud_run_service.app.name
  location = google_cloud_run_service.app.location
  role     = "roles/run.invoker"
  member   = "serviceAccount:my-sa@my-project.iam.gserviceaccount.com"
}
```

### Grant Role to Multiple Members

```hcl
resource "google_cloud_run_service_iam_binding" "invokers" {
  service  = google_cloud_run_service.app.name
  location = google_cloud_run_service.app.location
  role     = "roles/run.invoker"

  members = [
    "user:alice@example.com",
    "user:bob@example.com",
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
    "group:developers@example.com",
  ]
}
```

### Set Complete IAM Policy

```hcl
data "google_iam_policy" "invoker" {
  binding {
    role = "roles/run.invoker"
    members = [
      "user:alice@example.com",
      "serviceAccount:my-sa@my-project.iam.gserviceaccount.com",
    ]
  }

  binding {
    role = "roles/run.developer"
    members = [
      "group:developers@example.com",
    ]
  }
}

resource "google_cloud_run_service_iam_policy" "policy" {
  service     = google_cloud_run_service.app.name
  location    = google_cloud_run_service.app.location
  policy_data = data.google_iam_policy.invoker.policy_data
}
```

### Conditional Access with IAM Conditions

```hcl
resource "google_cloud_run_service_iam_member" "conditional" {
  service  = google_cloud_run_service.app.name
  location = google_cloud_run_service.app.location
  role     = "roles/run.invoker"
  member   = "user:alice@example.com"

  condition {
    title       = "Expires in 2024"
    description = "Access expires at end of 2024"
    expression  = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"
  }
}
```

### Grant Access to Cloud Scheduler

```hcl
resource "google_service_account" "scheduler" {
  account_id   = "cloud-scheduler-sa"
  display_name = "Cloud Scheduler Service Account"
}

resource "google_cloud_run_service_iam_member" "scheduler" {
  service  = google_cloud_run_service.app.name
  location = google_cloud_run_service.app.location
  role     = "roles/run.invoker"
  member   = google_service_account.scheduler.member
}

resource "google_cloud_scheduler_job" "job" {
  name     = "scheduled-job"
  schedule = "0 */6 * * *"

  http_target {
    http_method = "POST"
    uri         = google_cloud_run_service.app.status[0].url

    oidc_token {
      service_account_email = google_service_account.scheduler.email
    }
  }
}
```

## Argument Reference

### Common Arguments

- **service**: Cloud Run service name (required)
- **location**: Service region (defaults to provider region)
- **project**: GCP project ID (defaults to provider project)
- **role**: IAM role to grant (required for binding/member)

### google_cloud_run_service_iam_policy

- **policy_data**: Complete IAM policy JSON (required)
  - Generated using `google_iam_policy` data source

### google_cloud_run_service_iam_binding

- **members**: List of identity members (required)
- **condition**: Optional IAM condition

### google_cloud_run_service_iam_member

- **member**: Single identity member (required)
- **condition**: Optional IAM condition

## Member Formats

Identity members use these formats:

- **user:{emailid}**: Individual Google Account
  - Example: `user:alice@example.com`
- **serviceAccount:{emailid}**: Service account
  - Example: `serviceAccount:my-sa@my-project.iam.gserviceaccount.com`
- **group:{emailid}**: Google Group
  - Example: `group:developers@example.com`
- **domain:{domain}**: G Suite domain
  - Example: `domain:example.com`
- **allUsers**: Anyone on the internet (use with caution)
- **allAuthenticatedUsers**: Any authenticated Google Account
- **principal://iam.googleapis.com/projects/{project-number}/locations/global/workloadIdentityPools/{pool-id}/subject/{subject}**: Workload Identity
- **principalSet://iam.googleapis.com/projects/{project-number}/locations/global/workloadIdentityPools/{pool-id}/***: Workload Identity pool
- **principal://iam.googleapis.com/locations/global/workforcePools/{pool-id}/subject/{subject}**: Workforce Identity
- **principalSet://iam.googleapis.com/locations/global/workforcePools/{pool-id}/***: Workforce Identity pool

## IAM Roles for Cloud Run

Common Cloud Run IAM roles:

- **roles/run.invoker**: Invoke Cloud Run service
  - Most common for application access
  - Required for HTTP requests to authenticated services
- **roles/run.developer**: Full service management
  - Deploy, update, and delete services
  - View service details and logs
- **roles/run.admin**: Complete Cloud Run administration
  - All developer permissions
  - Manage IAM policies
- **roles/run.viewer**: Read-only access
  - View service configurations
  - Cannot invoke services

## IAM Conditions

Conditions enable context-aware access control using Common Expression Language (CEL):

```hcl
condition {
  title       = "Expires in 2024"
  description = "Optional description"
  expression  = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"
}
```

### Common Condition Examples

Time-based access:

```hcl
# Expires at specific time
expression = "request.time < timestamp(\"2024-12-31T23:59:59Z\")"

# Access during business hours (UTC)
expression = "request.time.getHours(\"UTC\") >= 9 && request.time.getHours(\"UTC\") < 17"

# Access on weekdays only
expression = "request.time.getDayOfWeek(\"UTC\") >= 1 && request.time.getDayOfWeek(\"UTC\") <= 5"
```

Resource-based access:

```hcl
# Specific resource only
expression = "resource.name == \"projects/my-project/locations/us-central1/services/my-service\""

# Resources with specific tag
expression = "resource.labels[\"environment\"] == \"production\""
```

**Note**: Cloud Run has some limitations with IAM Conditions. Refer to the [Cloud IAM Conditions documentation](https://cloud.google.com/iam/docs/conditions-overview) for details.

## Import

### Import Binding

```bash
# Format: "{project}/{location}/{service} {role}"
terraform import google_cloud_run_service_iam_binding.default "my-project/us-central1/my-service roles/run.invoker"
```

### Import Member

```bash
# Format: "{project}/{location}/{service} {role} {member}"
terraform import google_cloud_run_service_iam_member.default "my-project/us-central1/my-service roles/run.invoker user:alice@example.com"
```

### Import Policy

```bash
# Format: "{project}/{location}/{service}"
terraform import google_cloud_run_service_iam_policy.default "my-project/us-central1/my-service"
```

## Important Considerations

1. **Public access**: Using `allUsers` makes your service publicly accessible. Ensure your application handles authentication/authorization:

```hcl
# Public service
member = "allUsers"

# Authenticated users only
member = "allAuthenticatedUsers"
```

2. **Service-to-service authentication**: Use service accounts for Cloud Run to Cloud Run communication:

```hcl
# Caller service account
resource "google_service_account" "caller" {
  account_id = "caller-sa"
}

# Grant invoker role
resource "google_cloud_run_service_iam_member" "caller" {
  service  = google_cloud_run_service.target.name
  location = google_cloud_run_service.target.location
  role     = "roles/run.invoker"
  member   = google_service_account.caller.member
}

# Caller service uses this SA
resource "google_cloud_run_service" "caller_service" {
  name     = "caller"
  location = "us-central1"

  template {
    spec {
      service_account_name = google_service_account.caller.email
      # ...
    }
  }
}
```

3. **IAM policy conflicts**: Do not mix policy resource with binding/member resources. Choose one approach:
   - Use **policy** for complete control
   - Use **binding/member** for incremental management

4. **Eventually consistent**: IAM changes may take up to 7 minutes to propagate. Add retry logic in applications.

5. **Least privilege**: Grant minimum required permissions. Use `roles/run.invoker` instead of `roles/run.admin` when only invocation is needed.

6. **Audit logging**: Enable Cloud Audit Logs to track IAM changes and service invocations.

7. **Testing authenticated services**: Use `gcloud` to test authenticated services:

```bash
# Get auth token
gcloud auth print-identity-token

# Call service
curl -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
  https://my-service-abc123-uc.a.run.app
```

## Related Resources

- [google_cloud_run_service](cloud-run-service.md): Create Cloud Run services
- [google_service_account](service-account.md): Service accounts for authentication
- [google_project_iam](project-iam.md): Project-level IAM
