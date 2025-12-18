# google_cloudbuild_trigger

Configures automated builds in response to source repository changes, webhooks, or Pub/Sub messages.

## Example Usage

### GitHub Push Trigger

```hcl
resource "google_cloudbuild_trigger" "github_push" {
  name        = "github-push-trigger"
  description = "Trigger on push to main branch"
  location    = "us-central1"

  github {
    owner = "my-org"
    name  = "my-repo"
    push {
      branch = "^main$"
    }
  }

  filename = "cloudbuild.yaml"

  substitutions = {
    _ENVIRONMENT = "production"
  }
}
```

### GitHub Pull Request Trigger

```hcl
resource "google_cloudbuild_trigger" "github_pr" {
  name     = "github-pr-trigger"
  location = "us-central1"

  github {
    owner = "my-org"
    name  = "my-repo"
    pull_request {
      branch          = "^main$"
      comment_control = "COMMENTS_ENABLED"
    }
  }

  filename = "cloudbuild-pr.yaml"
}
```

### Cloud Source Repositories Trigger

```hcl
resource "google_cloudbuild_trigger" "csr_trigger" {
  name     = "csr-trigger"
  location = "us-central1"

  trigger_template {
    project_id  = "my-project"
    repo_name   = "my-repo"
    branch_name = "^main$"
  }

  filename = "cloudbuild.yaml"
}
```

### Pub/Sub Trigger

```hcl
resource "google_pubsub_topic" "build_trigger" {
  name = "build-trigger-topic"
}

resource "google_cloudbuild_trigger" "pubsub_trigger" {
  name     = "pubsub-trigger"
  location = "us-central1"

  pubsub_config {
    topic = google_pubsub_topic.build_trigger.id
  }

  source_to_build {
    uri       = "https://github.com/my-org/my-repo"
    ref       = "refs/heads/main"
    repo_type = "GITHUB"
  }

  filename = "cloudbuild.yaml"

  # Optional: filter messages
  filter = "attr.status == \"ready\""
}
```

### Webhook Trigger

```hcl
resource "google_secret_manager_secret" "webhook_secret" {
  secret_id = "webhook-secret"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "webhook_secret" {
  secret      = google_secret_manager_secret.webhook_secret.id
  secret_data = "my-webhook-secret"
}

resource "google_cloudbuild_trigger" "webhook_trigger" {
  name     = "webhook-trigger"
  location = "us-central1"

  webhook_config {
    secret = google_secret_manager_secret_version.webhook_secret.id
  }

  source_to_build {
    uri       = "https://github.com/my-org/my-repo"
    ref       = "refs/heads/main"
    repo_type = "GITHUB"
  }

  filename = "cloudbuild.yaml"
}
```

### Inline Build Configuration

```hcl
resource "google_cloudbuild_trigger" "inline_build" {
  name     = "inline-build-trigger"
  location = "us-central1"

  trigger_template {
    repo_name   = "my-repo"
    branch_name = "^main$"
  }

  build {
    step {
      name = "gcr.io/cloud-builders/docker"
      args = ["build", "-t", "gcr.io/$PROJECT_ID/my-image:$COMMIT_SHA", "."]
    }

    step {
      name = "gcr.io/cloud-builders/docker"
      args = ["push", "gcr.io/$PROJECT_ID/my-image:$COMMIT_SHA"]
    }

    images = ["gcr.io/$PROJECT_ID/my-image:$COMMIT_SHA"]

    options {
      logging = "CLOUD_LOGGING_ONLY"
      machine_type = "E2_HIGHCPU_8"
    }
  }

  service_account = google_service_account.build.id
}
```

### Trigger with Approval

```hcl
resource "google_cloudbuild_trigger" "approval_required" {
  name     = "approval-required-trigger"
  location = "us-central1"

  trigger_template {
    repo_name   = "my-repo"
    branch_name = "^main$"
  }

  approval_config {
    approval_required = true
  }

  filename = "cloudbuild.yaml"
}
```

### Trigger with File Filters

```hcl
resource "google_cloudbuild_trigger" "filtered" {
  name     = "filtered-trigger"
  location = "us-central1"

  trigger_template {
    repo_name   = "my-repo"
    branch_name = "^main$"
  }

  # Only trigger on changes to specific files
  included_files = [
    "src/**",
    "Dockerfile",
    "cloudbuild.yaml"
  ]

  # Ignore changes to these files
  ignored_files = [
    "README.md",
    "docs/**",
    "*.md"
  ]

  filename = "cloudbuild.yaml"
}
```

## Argument Reference

### Basic Configuration

- **name**: Trigger name (optional, auto-generated if not specified)
- **description**: Human-readable trigger description
- **location**: Cloud Build region (default: "global")
- **project**: GCP project ID
- **disabled**: Disable trigger execution without deletion
- **tags**: Build annotation tags (list of strings)

### Trigger Source (Choose One)

#### trigger_template

Cloud Source Repositories template:

```hcl
trigger_template {
  project_id  = "my-project"
  repo_name   = "my-repo"
  branch_name = "^main$"     # Regex pattern
  tag_name    = "^v.*$"      # Regex pattern (mutually exclusive with branch_name)
  commit_sha  = "abc123..."  # Specific commit
  dir         = "subdir"     # Build directory
  invert_regex = false       # Negate regex matching
}
```

#### github

GitHub repository events:

```hcl
github {
  owner = "my-org"
  name  = "my-repo"

  # Push events
  push {
    branch = "^main$"  # Regex pattern
    tag    = "^v.*$"   # Regex pattern
    invert_regex = false
  }

  # OR Pull request events
  pull_request {
    branch          = "^main$"
    comment_control = "COMMENTS_ENABLED"  # COMMENTS_ENABLED, COMMENTS_DISABLED, COMMENTS_ENABLED_FOR_EXTERNAL_CONTRIBUTORS_ONLY
    invert_regex    = false
  }

  enterprise_config_resource_name = "projects/my-project/locations/us-central1/githubEnterpriseConfigs/my-config"
}
```

#### pubsub_config

Pub/Sub message-triggered builds:

```hcl
pubsub_config {
  topic                 = "projects/my-project/topics/build-trigger"
  service_account_email = "my-sa@my-project.iam.gserviceaccount.com"
}
```

#### webhook_config

Webhook-triggered builds:

```hcl
webhook_config {
  secret = "projects/my-project/secrets/webhook-secret/versions/latest"
}
```

#### source_to_build

Repository source for manual/webhook/Pub/Sub triggers:

```hcl
source_to_build {
  uri       = "https://github.com/my-org/my-repo"
  ref       = "refs/heads/main"
  repo_type = "GITHUB"  # GITHUB, BITBUCKET_SERVER, CLOUD_SOURCE_REPOSITORIES

  # Optional: for GitHub Enterprise or Bitbucket Server
  github_enterprise_config = "projects/my-project/locations/us-central1/githubEnterpriseConfigs/my-config"
  bitbucket_server_config  = "projects/my-project/locations/us-central1/bitbucketServerConfigs/my-config"
}
```

### Build Configuration (Choose One)

#### filename

Path to build configuration file (cloudbuild.yaml):

```hcl
filename = "cloudbuild.yaml"  # Relative to repository root
```

#### git_file_source

Build file from another repository:

```hcl
git_file_source {
  path      = "cloudbuild.yaml"
  uri       = "https://github.com/my-org/build-configs"
  repo_type = "GITHUB"
  revision  = "refs/heads/main"
}
```

#### build

Inline build specification (see Build Configuration section below)

### Additional Options

- **service_account**: Service account email for build execution
- **substitutions**: Build variable substitutions (key-value map)
- **filter**: CEL expression for Pub/Sub/Webhook message filtering
- **ignored_files**: File glob patterns to exclude from triggering (list)
- **included_files**: File glob patterns that must match for triggering (list)
- **approval_config**: Require manual approval before build execution

```hcl
approval_config {
  approval_required = true
}
```

- **include_build_logs**: Include build logs in GitHub check runs (default: true)

## Build Configuration

Inline build specification:

```hcl
build {
  # Build steps (required)
  step {
    name = "gcr.io/cloud-builders/docker"
    args = ["build", "-t", "gcr.io/$PROJECT_ID/image", "."]
    env  = ["ENV_VAR=value"]
    dir  = "subdir"
    id   = "build-step"
    wait_for = ["-"]  # Run immediately (don't wait for previous steps)
  }

  step {
    name       = "gcr.io/cloud-builders/docker"
    args       = ["push", "gcr.io/$PROJECT_ID/image"]
    wait_for   = ["build-step"]  # Wait for specific step
  }

  # Images to push after build
  images = ["gcr.io/$PROJECT_ID/image"]

  # Build tags
  tags = ["production", "v1.0"]

  # Substitutions
  substitutions = {
    _CUSTOM_VAR = "value"
  }

  # Timeout
  timeout = "1800s"  # 30 minutes

  # Cloud Storage log location
  logs_bucket = "gs://my-build-logs"

  # Queue TTL
  queue_ttl = "3600s"

  # Secrets
  secret {
    kms_key_name = "projects/my-project/locations/global/keyRings/my-ring/cryptoKeys/my-key"
    secret_env = {
      MY_SECRET = "encrypted-value"
    }
  }

  # Secret Manager secrets
  available_secrets {
    secret_manager {
      env          = "DATABASE_PASSWORD"
      version_name = "projects/my-project/secrets/db-password/versions/latest"
    }
  }

  # Artifacts
  artifacts {
    images = ["gcr.io/$PROJECT_ID/image"]

    objects {
      location = "gs://my-artifacts"
      paths    = ["output/*.jar"]
    }
  }

  # Build options
  options {
    machine_type            = "E2_HIGHCPU_8"
    disk_size_gb            = 100
    logging                 = "CLOUD_LOGGING_ONLY"
    log_streaming_option    = "STREAM_ON"
    worker_pool             = "projects/my-project/locations/us-central1/workerPools/my-pool"
    substitution_option     = "ALLOW_LOOSE"
    dynamic_substitutions   = true
    requested_verify_option = "VERIFIED"

    env = ["GLOBAL_ENV=value"]

    volumes {
      name = "vol1"
      path = "/workspace/cache"
    }
  }
}
```

### Build Step Options

- **name**: Container image for step execution (required)
- **args**: Step arguments (list of strings)
- **script**: Shell script (alternative to args)
- **env**: Environment variables (list of "KEY=value")
- **secret_env**: Environment variables from secrets (list of keys)
- **dir**: Working directory
- **id**: Step identifier for dependencies
- **entrypoint**: Override container entrypoint
- **wait_for**: Wait for specific step IDs (use ["-"] to skip waiting)
- **timeout**: Step execution timeout
- **allow_failure**: Continue build even if step fails
- **allow_exit_codes**: List of acceptable exit codes
- **volumes**: Volume mounts

### Build Options

- **machine_type**: VM size - N1_HIGHCPU_8, N1_HIGHCPU_32, E2_HIGHCPU_8, E2_HIGHCPU_32, E2_MEDIUM, etc.
- **disk_size_gb**: Disk allocation (max 1000GB)
- **logging**: Log storage mode - LOGGING_UNSPECIFIED, LEGACY, GCS_ONLY, CLOUD_LOGGING_ONLY
- **log_streaming_option**: STREAM_DEFAULT, STREAM_ON, STREAM_OFF
- **worker_pool**: Custom worker pool resource name
- **substitution_option**: MUST_MATCH (strict) or ALLOW_LOOSE (permissive)
- **dynamic_substitutions**: Enable bash-style string operations in substitutions
- **requested_verify_option**: NOT_VERIFIED or VERIFIED
- **source_provenance_hash**: SHA256, MD5, NONE

## Attributes Reference

- **id**: Resource identifier (format: `projects/{project}/locations/{location}/triggers/{trigger_id}`)
- **trigger_id**: Unique trigger identifier (auto-generated)
- **create_time**: Trigger creation timestamp
- **pubsub_config.subscription**: Generated Pub/Sub subscription name (for pubsub_config)
- **webhook_config.state**: Webhook configuration status

## Import

Import formats:

```bash
# Full path
terraform import google_cloudbuild_trigger.default projects/my-project/locations/us-central1/triggers/abc123

# Project and trigger ID
terraform import google_cloudbuild_trigger.default my-project/abc123

# Trigger ID only
terraform import google_cloudbuild_trigger.default abc123
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

1. **Service account**: By default, builds run with the Cloud Build service account. Use a dedicated service account with minimal permissions:

```hcl
# Get Cloud Build service account
data "google_project_service_identity" "cloudbuild" {
  provider = google-beta
  project  = var.project_id
  service  = "cloudbuild.googleapis.com"
}
```

2. **Trigger location**: Use regional locations ("us-central1") instead of "global" for better reliability and lower latency.

3. **Substitutions**: Built-in substitutions available in builds:
   - `$PROJECT_ID` - GCP project ID
   - `$BUILD_ID` - Build ID
   - `$COMMIT_SHA` - Git commit SHA
   - `$BRANCH_NAME` - Git branch name
   - `$TAG_NAME` - Git tag name
   - `$REPO_NAME` - Repository name
   - Custom: `$_CUSTOM_VAR` (prefix with underscore)

4. **File filtering**: Use `included_files` and `ignored_files` to prevent unnecessary builds:
   - Patterns use glob syntax
   - `ignored_files` takes precedence over `included_files`
   - Changes to both included and ignored files = no trigger

5. **Manual approval**: Use `approval_config` for production deployments requiring human verification before execution.

6. **Webhook secrets**: Store webhook secrets in Secret Manager and rotate regularly.

7. **GitHub integration**: Requires GitHub App installation and repository connection through Cloud Build.

8. **Build configuration priority**: If both `filename` and `build` are specified, `filename` takes precedence.

9. **Trigger disabling**: Use `disabled = true` to temporarily disable triggers without deleting them.

10. **Regex patterns**: Branch and tag filters support regex. Use `^` and `$` for exact matches:
    - `^main$` - Matches "main" only
    - `^v.*$` - Matches any tag starting with "v"
    - `^feature/.*$` - Matches any branch starting with "feature/"

## Related Resources

- [google_service_account](service-account.md): Service account for builds
- [google_secret_manager_secret](secret-manager-secret.md): Store webhook secrets
- [google_pubsub_topic](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic): Pub/Sub triggers
