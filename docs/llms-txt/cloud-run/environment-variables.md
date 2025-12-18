# Environment Variables

> Configuration guide for using environment variables in Cloud Run services and jobs.

## Overview

Environment variables in Cloud Run allow service configuration without storing sensitive data.

**Important**: Do not use environment variables to store secrets such as database credentials or API keys. Use Secret Manager instead.

## Setting Environment Variables

### Console
Add variables via the Container tab's Variables & Secrets section

### gcloud CLI

```bash
# Set multiple variables
gcloud run deploy SERVICE \
  --set-env-vars KEY1=VALUE1,KEY2=VALUE2

# Update without replacing others
gcloud run services update SERVICE \
  --update-env-vars KEY3=VALUE3

# Remove specific variables
gcloud run services update SERVICE \
  --remove-env-vars KEY1,KEY2

# Clear all variables
gcloud run services update SERVICE \
  --clear-env-vars
```

### YAML Configuration

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: SERVICE
spec:
  template:
    spec:
      containers:
      - image: IMAGE
        env:
        - name: KEY1
          value: VALUE1
        - name: KEY2
          value: VALUE2
```

### Terraform

```hcl
resource "google_cloud_run_v2_service" "default" {
  name     = "service-name"
  location = "us-central1"

  template {
    containers {
      image = "gcr.io/project/image"
      env {
        name  = "KEY1"
        value = "VALUE1"
      }
    }
  }
}
```

### Dockerfile

Set defaults using `ENV` statements:

```dockerfile
ENV PORT=8080
ENV LOG_LEVEL=info
```

## Built-in Environment Variables

Cloud Run automatically injects certain variables:

- `PORT`: The container's listening port (default 8080, user-configurable)
- `K_SERVICE`: Service name (services only)
- `K_REVISION`: Revision identifier (services only)
- `K_CONFIGURATION`: Configuration name (services only)

For functions:
- `FUNCTION_TARGET`
- `FUNCTION_SIGNATURE_TYPE`

## Limitations

- **Maximum count**: 1000 environment variables per service
- **Maximum length**: 32 KB per variable
- **Reserved prefixes**: Cannot use keys that are empty, contain `=`, or start with `X_GOOGLE_`

## Important Security Notes

**Never set** `GOOGLE_APPLICATION_CREDENTIALS` as an environment variable when using service identity authentication. This can interfere with Cloud Run's built-in authentication mechanisms.

## Managing Updates

Changes to environment variables create new service revisions automatically. This means:

- Previous revisions retain their original variable values
- Traffic can be split between revisions with different configurations
- Rollbacks are possible by routing traffic to earlier revisions

## Best Practices

1. **Use Secret Manager for sensitive data**: Environment variables are visible in console, logs, and to anyone with appropriate IAM permissions
2. **Update non-destructively**: Use `--update-env-vars` to modify specific variables without affecting others
3. **Document variable requirements**: Maintain a list of required environment variables for your service
4. **Validate at startup**: Check for required variables early in application initialization
5. **Use descriptive names**: Follow naming conventions like `DATABASE_URL`, `API_TIMEOUT_SECONDS`
6. **Avoid hardcoding defaults**: Let Cloud Run configuration be the source of truth

## Go Example

```go
package main

import (
    "log"
    "os"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
        log.Printf("Defaulting to port %s", port)
    }

    logLevel := os.Getenv("LOG_LEVEL")
    if logLevel == "" {
        logLevel = "info"
    }

    // Use environment variables throughout your application
    log.Printf("Starting server on port %s with log level %s", port, logLevel)
}
```
