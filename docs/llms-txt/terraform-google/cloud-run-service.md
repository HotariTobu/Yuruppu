# google_cloud_run_service

Manages a Cloud Run service with unique endpoints and container autoscaling capabilities.

**Note**: Google recommends using `google_cloud_run_v2_service` for improved developer experience and broader feature support.

## Example Usage

```hcl
resource "google_cloud_run_service" "default" {
  name     = "my-service"
  location = "us-central1"

  template {
    spec {
      containers {
        image = "gcr.io/my-project/my-image:latest"

        resources {
          limits = {
            cpu    = "1000m"
            memory = "512Mi"
          }
        }

        ports {
          container_port = 8080
        }
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}
```

## Argument Reference

### Required Arguments

- **name**: Service identifier within a Google Cloud project and region
- **location**: Geographic deployment region (e.g., "us-central1")

### Optional Arguments

- **template**: Container specifications and revision metadata (see Template section)
- **traffic**: Traffic distribution across revisions (see Traffic section)
- **metadata**: Service-level annotations and labels
- **project**: GCP project ID (defaults to provider project)
- **autogenerate_revision_name**: Enables automatic revision naming (default: false)

## Template Configuration

The template block defines container specifications:

```hcl
template {
  metadata {
    annotations = {
      "autoscaling.knative.dev/maxScale" = "10"
      "run.googleapis.com/cloudsql-instances" = "project:region:instance"
      "run.googleapis.com/client-name" = "terraform"
    }
  }

  spec {
    containers {
      image = "gcr.io/my-project/my-image:latest"

      resources {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
      }

      env {
        name  = "ENV_VAR"
        value = "value"
      }

      env {
        name = "SECRET_ENV"
        value_from {
          secret_key_ref {
            name = "secret-name"
            key  = "latest"
          }
        }
      }

      ports {
        name           = "http1"
        container_port = 8080
      }

      volume_mounts {
        name       = "secret-volume"
        mount_path = "/secrets"
      }
    }

    volumes {
      name = "secret-volume"
      secret {
        secret_name = "my-secret"
        items {
          key  = "latest"
          path = "secret-file"
        }
      }
    }

    service_account_name = "my-service-account@my-project.iam.gserviceaccount.com"
  }
}
```

### Template Metadata Annotations

Important annotations for Cloud Run:

- `autoscaling.knative.dev/maxScale`: Maximum number of instances (default: 100)
- `autoscaling.knative.dev/minScale`: Minimum number of instances (default: 0)
- `run.googleapis.com/cloudsql-instances`: Cloud SQL connections (comma-separated)
- `run.googleapis.com/vpc-access-connector`: VPC connector name
- `run.googleapis.com/vpc-access-egress`: VPC egress setting ("all-traffic" or "private-ranges-only")
- `run.googleapis.com/execution-environment`: "gen1" or "gen2"
- `run.googleapis.com/cpu-throttling`: "true" or "false" (CPU always allocated if false)

### Container Specifications

- **image**: Container registry reference (required)
- **resources**: CPU/memory limits and requests
  - `limits`: Maximum resources (e.g., "1000m" CPU, "512Mi" memory)
  - `requests`: Reserved resources
- **env**: Environment variables (name/value or name/value_from)
- **ports**: Exposed container ports (one container port required)
- **command**: Override image entrypoint
- **args**: Override image CMD
- **working_dir**: Container working directory
- **volume_mounts**: Storage attachment points

### Health Checks

Configure startup, liveness, and readiness probes:

```hcl
startup_probe {
  http_get {
    path = "/health"
    port = 8080
  }
  initial_delay_seconds = 0
  timeout_seconds       = 1
  period_seconds        = 10
  failure_threshold     = 3
}

liveness_probe {
  http_get {
    path = "/healthz"
  }
}

readiness_probe {
  tcp_socket {
    port = 8080
  }
}
```

Probe types:
- **http_get**: HTTP GET request
- **tcp_socket**: TCP socket connection
- **grpc**: gRPC health check

### Volumes

Supported volume types:

1. **Secret volumes**: Mount Secret Manager secrets
2. **emptyDir**: Ephemeral storage (memory-backed)
3. **CSI**: Cloud Storage FUSE (gcsfuse) mounts
4. **NFS**: Network file system mounts

```hcl
volumes {
  name = "gcs-bucket"
  csi {
    driver = "gcsfuse.run.googleapis.com"
    volume_attributes = {
      bucketName = "my-bucket"
    }
  }
}
```

## Traffic Management

Distribute traffic across revisions:

```hcl
traffic {
  percent         = 100
  latest_revision = true
}

# Or target specific revision
traffic {
  percent       = 100
  revision_name = "my-service-abc123"
}

# Split traffic between revisions
traffic {
  percent       = 90
  revision_name = "my-service-v2"
}

traffic {
  percent       = 10
  revision_name = "my-service-v1"
}

# Tagged traffic (creates dedicated URL)
traffic {
  percent       = 0
  revision_name = "my-service-canary"
  tag           = "canary"
}
```

Traffic attributes:
- **percent**: Traffic percentage (0-100)
- **latest_revision**: Route to newest ready revision (boolean)
- **revision_name**: Target specific revision
- **tag**: Create dedicated URL (format: tag-[hash]-[region]-[project].a.run.app)

## Attributes Reference

- **id**: Resource identifier (format: `locations/{location}/namespaces/{project}/services/{name}`)
- **status.url**: Service endpoint URL
- **status.latest_ready_revision_name**: Current active revision
- **status.latest_created_revision_name**: Most recently created revision
- **status.observed_generation**: Configuration generation number
- **status.conditions**: Service readiness conditions
- **status.traffic**: Current traffic assignments

## Import

Import formats:

```bash
# Full path
terraform import google_cloud_run_service.default locations/us-central1/namespaces/my-project/services/my-service

# Abbreviated
terraform import google_cloud_run_service.default us-central1/my-project/my-service
terraform import google_cloud_run_service.default us-central1/my-service
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

1. **Automatic annotations**: The Cloud Run API automatically adds annotations. Use `lifecycle.ignore_changes` to prevent drift:

```hcl
lifecycle {
  ignore_changes = [
    metadata[0].annotations,
    template[0].metadata[0].annotations,
  ]
}
```

2. **Restricted annotation prefixes**: Avoid manually setting annotations with prefixes:
   - `run.googleapis.com/`
   - `autoscaling.knative.dev/`
   - `serving.knative.dev/`

3. **Service account**: Cloud Run uses the Compute Engine default service account by default. Use a dedicated service account with minimal permissions for better security.

4. **Versioning**: Consider using `google_cloud_run_v2_service` for new deployments (better features and developer experience).
