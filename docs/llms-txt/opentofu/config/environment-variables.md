# Environment Variables

OpenTofu supports configuration through environment variables, allowing you to customize behavior without modifying configuration files.

## Variable Assignment

### TF_VAR_name

Supply variable values using the format `TF_VAR_name`. These are checked last in the precedence order.

```bash
export TF_VAR_project_id="my-gcp-project"
export TF_VAR_region="us-central1"
export TF_VAR_environment="production"
export TF_VAR_enable_monitoring="true"

tofu apply
```

For complex types:

```bash
# List
export TF_VAR_regions='["us-central1", "us-east1"]'

# Map
export TF_VAR_labels='{"environment":"prod","managed_by":"opentofu"}'

# Object (JSON)
export TF_VAR_config='{"cpu":"2000m","memory":"1Gi","replicas":3}'
```

## Logging and Debugging

### TF_LOG

Enables detailed stderr logs. Set to log level or `off` to disable.

**Levels:**
- `TRACE` - Maximum verbosity
- `DEBUG` - Detailed debugging information
- `INFO` - General information
- `WARN` - Warning messages only
- `ERROR` - Error messages only
- `OFF` - Disable logging

```bash
# Maximum verbosity
export TF_LOG=TRACE
tofu apply

# Errors only
export TF_LOG=ERROR
tofu plan
```

### TF_LOG_PATH

Specifies where log output persists to a file.

```bash
export TF_LOG=TRACE
export TF_LOG_PATH="./opentofu.log"
tofu apply

# View logs
tail -f opentofu.log
```

## Input and Automation

### TF_INPUT

When set to "false" or "0", disables interactive prompts for unspecified variable values.

```bash
export TF_INPUT=false
tofu apply  # Errors if variables are missing
```

Useful for CI/CD pipelines:

```bash
export TF_INPUT=false
export TF_IN_AUTOMATION=true
tofu apply -auto-approve
```

### TF_IN_AUTOMATION

When set to any non-empty value, adjusts output for CI/automation systems.

```bash
export TF_IN_AUTOMATION=true
tofu plan
```

Effects:
- Adjusts output formatting
- Reduces interactivity
- Optimizes for log parsing

## Command Behavior

### TF_CLI_ARGS

Adds arguments to all commands.

```bash
export TF_CLI_ARGS="-no-color -lock-timeout=5m"
tofu plan  # Automatically uses -no-color -lock-timeout=5m
```

### TF_CLI_ARGS_name

Targets specific commands only.

```bash
export TF_CLI_ARGS_plan="-out=tfplan -detailed-exitcode"
export TF_CLI_ARGS_apply="-auto-approve"

tofu plan   # Uses -out=tfplan -detailed-exitcode
tofu apply  # Uses -auto-approve
```

## Workspace Selection

### TF_WORKSPACE

Selects a workspace without using the `workspace select` command.

```bash
export TF_WORKSPACE=production
tofu plan  # Uses production workspace
```

Useful in CI/CD:

```bash
export TF_WORKSPACE=${ENVIRONMENT}
tofu init
tofu apply
```

## File and Directory Management

### TF_DATA_DIR

Changes where OpenTofu stores per-working-directory data (defaults to `.terraform`).

```bash
export TF_DATA_DIR="/tmp/terraform-data"
tofu init
```

### TF_CLI_CONFIG_FILE

Specifies custom CLI configuration file location.

```bash
export TF_CLI_CONFIG_FILE="$HOME/.config/opentofu/config.tfrc"
tofu init
```

### TF_PLUGIN_CACHE_DIR

Sets provider plugin cache directory.

```bash
export TF_PLUGIN_CACHE_DIR="$HOME/.terraform.d/plugin-cache"
mkdir -p $TF_PLUGIN_CACHE_DIR
tofu init
```

Subsequent inits will use cached providers, reducing download time.

## Registry and Provider Configuration

### TF_REGISTRY_DISCOVERY_RETRY

Configures max retry attempts for registry client requests.

```bash
export TF_REGISTRY_DISCOVERY_RETRY=5
tofu init
```

### TF_REGISTRY_CLIENT_TIMEOUT

Sets remote registry request timeout (default: 10 seconds).

```bash
export TF_REGISTRY_CLIENT_TIMEOUT=30s
tofu init
```

### TF_PROVIDER_DOWNLOAD_RETRY

Configures max retry attempts for provider downloads.

```bash
export TF_PROVIDER_DOWNLOAD_RETRY=3
tofu init
```

## Debugging

### TF_IGNORE

Set to "trace" to debug `.terraformignore` file processing.

```bash
export TF_IGNORE=trace
tofu plan
```

## State Management

### TF_STATE_PERSIST_INTERVAL

Adjusts state persistence interval in seconds (minimum: 20s).

```bash
export TF_STATE_PERSIST_INTERVAL=30
tofu apply
```

## Security

### TF_ENCRYPTION

Provides encryption configuration as HCL or JSON, overriding config files.

```bash
export TF_ENCRYPTION='
  key_provider "pbkdf2" "my_passphrase" {
    passphrase = "my-secret-passphrase"
  }
'
tofu init
```

## Google Cloud Specific

### GOOGLE_APPLICATION_CREDENTIALS

Points to GCP service account key file.

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
tofu apply
```

### GOOGLE_PROJECT

Sets default GCP project.

```bash
export GOOGLE_PROJECT="my-gcp-project"
tofu plan
```

### GOOGLE_REGION

Sets default GCP region.

```bash
export GOOGLE_REGION="us-central1"
tofu plan
```

### GOOGLE_CREDENTIALS

Service account key content (as JSON string).

```bash
export GOOGLE_CREDENTIALS=$(cat /path/to/key.json)
tofu apply
```

## CI/CD Configuration Example

```bash
#!/bin/bash
# CI/CD pipeline environment setup

# Logging
export TF_LOG=INFO
export TF_LOG_PATH="./logs/tofu-$(date +%Y%m%d-%H%M%S).log"

# Automation
export TF_INPUT=false
export TF_IN_AUTOMATION=true

# Performance
export TF_PLUGIN_CACHE_DIR="/cache/terraform-plugins"
export TF_REGISTRY_CLIENT_TIMEOUT=60s
export TF_PROVIDER_DOWNLOAD_RETRY=5

# Google Cloud
export GOOGLE_APPLICATION_CREDENTIALS="${GOOGLE_CREDENTIALS_FILE}"
export GOOGLE_PROJECT="${GCP_PROJECT_ID}"

# Variables
export TF_VAR_project_id="${GCP_PROJECT_ID}"
export TF_VAR_region="${GCP_REGION}"
export TF_VAR_environment="${ENVIRONMENT}"

# Backend configuration
export TF_CLI_ARGS_init="-backend-config=bucket=${STATE_BUCKET} -backend-config=prefix=${ENVIRONMENT}"

# Apply configuration
export TF_CLI_ARGS_apply="-auto-approve"

# Execute
tofu init
tofu plan -out=tfplan
tofu apply tfplan
```

## Local Development Setup

```bash
# .envrc (use with direnv)

# GCP Authentication
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.config/gcloud/application_default_credentials.json"

# Project variables
export TF_VAR_project_id="my-dev-project"
export TF_VAR_region="us-central1"
export TF_VAR_environment="dev"

# Development settings
export TF_LOG=DEBUG
export TF_LOG_PATH="./dev.log"

# Plugin cache
export TF_PLUGIN_CACHE_DIR="$HOME/.terraform.d/plugin-cache"

# Fast iteration
export TF_CLI_ARGS_plan="-compact-warnings"
```

## Best Practices

1. **Use for secrets** - Pass sensitive values via environment variables, not in files
2. **Set in CI/CD** - Configure environment variables in pipeline settings
3. **Document requirements** - List required environment variables in README
4. **Use consistent names** - Follow TF_VAR_ convention for variables
5. **Separate environments** - Use different values per environment
6. **Enable logging in CI** - Set TF_LOG for troubleshooting
7. **Cache plugins** - Use TF_PLUGIN_CACHE_DIR to reduce downloads
8. **Set timeouts** - Increase timeouts for slow networks
9. **Validate in CI** - Verify all required variables are set before running
10. **Use direnv** - Automatically load environment per directory

## Precedence Order

Variables are evaluated in this order (highest to lowest):

1. Command-line `-var` flags
2. `-var-file` flags
3. `*.auto.tfvars` files
4. `terraform.tfvars`
5. **TF_VAR_* environment variables** (lowest precedence)

## Security Considerations

1. **Never commit secrets** - Don't hardcode secrets in environment files
2. **Use secret management** - Store secrets in vault, not environment variables
3. **Rotate credentials** - Regularly rotate service account keys
4. **Limit exposure** - Only set variables where needed
5. **Audit access** - Track who can set CI/CD environment variables
6. **Use encryption** - Encrypt sensitive environment variables in CI/CD
7. **Clear after use** - Unset sensitive variables after operations
8. **Use IAM** - Prefer workload identity over service account keys when possible

## Troubleshooting

### Variables Not Being Set

```bash
# Verify environment variable is set
echo $TF_VAR_project_id

# Check if OpenTofu sees it
tofu console
> var.project_id
```

### Authentication Errors

```bash
# Verify credentials file exists
ls -la $GOOGLE_APPLICATION_CREDENTIALS

# Test credentials
gcloud auth activate-service-account --key-file=$GOOGLE_APPLICATION_CREDENTIALS
```

### Plugin Cache Issues

```bash
# Clear cache
rm -rf $TF_PLUGIN_CACHE_DIR/*

# Reinitialize
tofu init -upgrade
```

## Related Documentation

- [Variable Declaration](../language/variables.md)
- [Provider Configuration](../language/providers.md)
- [GCS Backend](../gcs-backend.md)
