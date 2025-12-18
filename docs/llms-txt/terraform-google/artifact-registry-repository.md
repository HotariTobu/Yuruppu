# google_artifact_registry_repository

Manages artifact repositories in Google Cloud's Artifact Registry service for storing containers, packages, and other artifacts.

## Example Usage

### Docker Repository

```hcl
resource "google_artifact_registry_repository" "docker" {
  repository_id = "my-docker-repo"
  format        = "DOCKER"
  location      = "us-central1"
  description   = "Docker container repository"

  docker_config {
    immutable_tags = true
  }

  labels = {
    environment = "production"
    managed-by  = "terraform"
  }
}
```

### Maven Repository with Cleanup Policy

```hcl
resource "google_artifact_registry_repository" "maven" {
  repository_id = "my-maven-repo"
  format        = "MAVEN"
  location      = "us-central1"

  maven_config {
    allow_snapshot_overwrites = true
    version_policy            = "SNAPSHOT"
  }

  cleanup_policies {
    id     = "delete-old-snapshots"
    action = "DELETE"

    condition {
      tag_state  = "UNTAGGED"
      older_than = "2592000s"  # 30 days
    }
  }
}
```

### Virtual Repository

```hcl
resource "google_artifact_registry_repository" "virtual" {
  repository_id = "my-virtual-repo"
  format        = "DOCKER"
  location      = "us-central1"
  mode          = "VIRTUAL_REPOSITORY"

  virtual_repository_config {
    upstream_policies {
      id         = "upstream-1"
      repository = google_artifact_registry_repository.docker.id
      priority   = 100
    }
  }
}
```

### Remote Repository with Authentication

```hcl
resource "google_secret_manager_secret" "registry_password" {
  secret_id = "registry-password"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "registry_password" {
  secret      = google_secret_manager_secret.registry_password.id
  secret_data = "my-password"
}

resource "google_artifact_registry_repository" "remote" {
  repository_id = "my-remote-repo"
  format        = "DOCKER"
  location      = "us-central1"
  mode          = "REMOTE_REPOSITORY"

  remote_repository_config {
    description = "Remote Docker Hub proxy"

    docker_repository {
      public_repository = "DOCKER_HUB"
    }

    upstream_credentials {
      username_password_credentials {
        username                = "my-username"
        password_secret_version = google_secret_manager_secret_version.registry_password.name
      }
    }
  }
}
```

## Argument Reference

### Required Arguments

- **format**: Package format - one of:
  - `DOCKER` - Docker containers
  - `MAVEN` - Maven packages
  - `NPM` - NPM packages
  - `PYTHON` - Python packages
  - `APT` - Debian packages
  - `YUM` - RPM packages
  - `GO` - Go modules
  - `KFP` - Kubeflow Pipelines
  - `GENERIC` - Generic artifacts

- **repository_id**: Repository name (last segment of the name)

### Optional Arguments

- **location**: Region or multi-region (e.g., "us-central1", "us", "europe", "asia")
- **description**: User-provided repository description
- **labels**: Key-value pairs for organization (max 64, 63 characters each)
- **kms_key_name**: Cloud KMS encryption key (immutable after creation)
- **mode**: Repository type (default: `STANDARD_REPOSITORY`)
  - `STANDARD_REPOSITORY` - Regular artifact storage
  - `VIRTUAL_REPOSITORY` - Aggregates multiple repositories
  - `REMOTE_REPOSITORY` - Proxy to external registries
- **project**: GCP project ID

### Docker Configuration

```hcl
docker_config {
  immutable_tags = true  # Prevent tag modification/deletion
}
```

### Maven Configuration

```hcl
maven_config {
  allow_snapshot_overwrites = true  # Allow duplicate snapshot versions
  version_policy            = "SNAPSHOT"  # RELEASE, SNAPSHOT, or VERSION_POLICY_UNSPECIFIED
}
```

### Virtual Repository Configuration

```hcl
virtual_repository_config {
  upstream_policies {
    id         = "policy-1"
    repository = "projects/my-project/locations/us-central1/repositories/upstream-repo"
    priority   = 100  # Lower number = higher priority
  }
}
```

### Remote Repository Configuration

```hcl
remote_repository_config {
  description = "Proxy to external registry"

  # Choose one repository type
  docker_repository {
    public_repository = "DOCKER_HUB"  # or custom_repository
  }

  # Or for custom registries
  docker_repository {
    custom_repository {
      uri = "https://registry.example.com"
    }
  }

  # NPM, Maven, Python, APT, YUM repositories also supported
  npm_repository {
    public_repository = "NPMJS"
  }

  # Optional authentication
  upstream_credentials {
    username_password_credentials {
      username                = "username"
      password_secret_version = "projects/my-project/secrets/password/versions/latest"
    }
  }
}
```

### Cleanup Policies

Automatically delete old versions:

```hcl
cleanup_policies {
  id     = "policy-name"
  action = "DELETE"  # or "KEEP"

  condition {
    tag_state             = "TAGGED"  # TAGGED, UNTAGGED, or ANY
    tag_prefixes          = ["v", "release-"]
    version_name_prefixes = ["test-"]
    package_name_prefixes = ["com/example/"]
    older_than            = "2592000s"  # Duration in seconds (e.g., 30 days)
    newer_than            = "86400s"    # Keep versions newer than this
  }

  most_recent_versions {
    package_name_prefixes = ["com/example/"]
    keep_count            = 10  # Keep N most recent versions
  }
}

cleanup_policy_dry_run = true  # Preview deletions without executing
```

### Vulnerability Scanning

```hcl
vulnerability_scanning_config {
  enablement_config = "INHERITED"  # or "DISABLED"
  enablement_state  = "ENABLED"    # Output: actual state
  last_enable_time  = "..."        # Output: timestamp
}
```

## Attributes Reference

- **id**: Repository identifier
- **name**: Full repository name (format: `projects/{project}/locations/{location}/repositories/{repository_id}`)
- **create_time**: Repository creation timestamp (RFC3339)
- **update_time**: Last update timestamp (RFC3339)
- **registry_uri**: Repository access endpoint
  - Format: `{location}-{format}.pkg.dev/{project}/{repository_id}`
  - Example: `us-central1-docker.pkg.dev/my-project/my-repo`
- **terraform_labels**: Labels applied by Terraform
- **effective_labels**: All labels present in GCP (including system labels)

## Import

Import formats:

```bash
# Full path
terraform import google_artifact_registry_repository.default projects/my-project/locations/us-central1/repositories/my-repo

# Abbreviated
terraform import google_artifact_registry_repository.default my-project/us-central1/my-repo
terraform import google_artifact_registry_repository.default us-central1/my-repo
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

1. **KMS encryption**: The `kms_key_name` cannot be modified after repository creation. Plan encryption requirements carefully.

2. **Labels**: This resource uses non-authoritative labels. The `effective_labels` attribute shows all labels including those set outside Terraform.

3. **Cleanup policies**: Test with `cleanup_policy_dry_run = true` before enabling to verify which versions will be deleted.

4. **Remote repository credentials**: Store passwords in Secret Manager and reference them via `password_secret_version`. Never hardcode credentials.

5. **Repository URL format**: Access repositories using the `registry_uri` attribute:
   ```bash
   docker pull us-central1-docker.pkg.dev/my-project/my-repo/image:tag
   ```

6. **Multi-region**: Use multi-region locations ("us", "europe", "asia") for higher availability, or specific regions for lower latency.

7. **Virtual repositories**: Useful for aggregating multiple repositories under a single endpoint (e.g., combining snapshot and release repositories).
