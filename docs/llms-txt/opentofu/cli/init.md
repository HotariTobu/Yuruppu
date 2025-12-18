# tofu init

The `tofu init` command initializes a working directory containing OpenTofu configuration files. This should be executed first after writing new configuration or cloning existing code from version control.

## Usage

```bash
tofu init [options]
```

It's safe to run multiple times - the command is idempotent.

## What init Does

1. **Backend Initialization** - Configures the backend for storing state
2. **Module Installation** - Downloads and installs modules referenced in the configuration
3. **Provider Installation** - Downloads and installs provider plugins

## Basic Example

```bash
cd my-opentofu-project
tofu init
```

Output:
```
Initializing the backend...
Initializing provider plugins...
- Finding hashicorp/google versions matching "~> 5.0"...
- Installing hashicorp/google v5.10.0...

OpenTofu has been successfully initialized!
```

## General Options

### -input

```bash
tofu init -input=false
```

When set to `false`, disables interactive prompts. Errors if input is required. Useful for automation.

### -upgrade

```bash
tofu init -upgrade
```

Upgrades modules and plugins to the latest versions allowed by version constraints.

### -var and -var-file

```bash
tofu init -var="project_id=my-project" -var-file="prod.tfvars"
```

Sets input variables. Use multiple times for several variables.

### -json

```bash
tofu init -json
```

Produces machine-readable JSON output suitable for automation.

### -lock-timeout

```bash
tofu init -lock-timeout=30s
```

Overrides the default state lock wait time.

### -no-color

```bash
tofu init -no-color
```

Removes color formatting from output.

## Backend Initialization

During init, OpenTofu processes the backend configuration and sets up state storage.

### Initial Backend Setup

```hcl
# main.tf
terraform {
  backend "gcs" {
    bucket = "my-terraform-state"
    prefix = "production/cloudrun"
  }
}
```

```bash
tofu init
```

### Reconfiguring Backend

Use when changing backend settings:

```bash
tofu init -reconfigure
```

Forces reinitialization of the backend, ignoring any existing configuration.

### Migrating State

```bash
tofu init -migrate-state
```

Attempts to copy existing state to the new backend configuration. You'll be prompted for confirmation.

### Force Copy

```bash
tofu init -migrate-state -force-copy
```

Suppresses confirmation prompts during state migration.

### Skip Backend

```bash
tofu init -backend=false
```

Skips backend initialization. Not recommended for new setups.

### Partial Backend Configuration

Provide backend configuration dynamically:

```bash
tofu init \
  -backend-config="bucket=my-state-bucket" \
  -backend-config="prefix=production/app"
```

Or from a file:

```bash
tofu init -backend-config=backend.tfvars
```

**backend.tfvars:**
```hcl
bucket = "my-terraform-state"
prefix = "production/cloudrun"
```

Useful for:
- Keeping sensitive values out of version control
- Using different backends per environment
- CI/CD pipelines with dynamic configuration

## Module Installation

OpenTofu downloads modules specified in `module` blocks.

### Basic Module Init

```hcl
module "cloud_run" {
  source = "./modules/cloud-run"

  project_id = var.project_id
  region     = var.region
}
```

```bash
tofu init
```

Downloads modules to `.terraform/modules/`.

### Upgrading Modules

```bash
tofu init -upgrade
```

Updates modules to the latest versions within version constraints.

### Module from Registry

```hcl
module "gke" {
  source  = "terraform-google-modules/kubernetes-engine/google"
  version = "~> 30.0"

  project_id = var.project_id
  name       = "my-cluster"
}
```

```bash
tofu init
```

Downloads from the OpenTofu Registry.

## Provider Installation

OpenTofu downloads provider plugins specified in `required_providers`.

### Basic Provider Init

```hcl
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}
```

```bash
tofu init
```

Downloads providers to `.terraform/providers/`.

### Upgrading Providers

```bash
tofu init -upgrade
```

Updates providers to the latest versions within constraints.

### Provider Plugin Cache

Enable caching to avoid re-downloading providers:

```hcl
# ~/.tofurc or ~/.terraformrc
plugin_cache_dir = "$HOME/.terraform.d/plugin-cache"
```

```bash
mkdir -p ~/.terraform.d/plugin-cache
tofu init
```

Subsequent inits in other projects will use cached providers.

### Lock File

After init, OpenTofu creates `.terraform.lock.hcl`:

```hcl
provider "registry.opentofu.org/hashicorp/google" {
  version     = "5.10.0"
  constraints = "~> 5.0"
  hashes = [
    "h1:abc123...",
    "zh:def456...",
  ]
}
```

**Commit this file** to version control to ensure consistent provider versions across team members and CI/CD.

## Common Workflows

### New Project

```bash
# Create configuration files
vim main.tf variables.tf

# Initialize
tofu init

# Validate
tofu validate

# Plan
tofu plan
```

### Cloned Project

```bash
# Clone repository
git clone https://github.com/myorg/my-infra.git
cd my-infra

# Initialize
tofu init

# Plan to see current state
tofu plan
```

### Adding New Provider

```hcl
# Add to versions.tf
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}
```

```bash
tofu init
```

### Changing Backend

```hcl
# Change from local to GCS
terraform {
  backend "gcs" {
    bucket = "my-terraform-state"
    prefix = "production"
  }
}
```

```bash
tofu init -migrate-state
```

Answer `yes` when prompted to copy state.

### CI/CD Pipeline

```bash
# Non-interactive, with backend config
tofu init \
  -input=false \
  -backend-config="bucket=${STATE_BUCKET}" \
  -backend-config="prefix=${ENVIRONMENT}"
```

## Troubleshooting

### Provider Download Issues

```bash
# Increase timeout
tofu init -lock-timeout=10m

# Use specific provider mirror
terraform {
  required_providers {
    google = {
      source  = "registry.terraform.io/hashicorp/google"
      version = "~> 5.0"
    }
  }
}
```

### Module Not Found

```bash
# Verify module source is correct
# For local modules, ensure path is relative to root module

# Reinitialize
tofu init -upgrade
```

### Backend Configuration Errors

```bash
# Reconfigure from scratch
tofu init -reconfigure

# Or migrate state
tofu init -migrate-state
```

### Permission Errors

Ensure service account or user has:
- Storage bucket access for GCS backend
- Provider registry access for downloading providers
- Module source access (Git, HTTP, etc.)

## Best Practices

1. **Run init first** - Always run init before other OpenTofu commands
2. **Commit lock file** - Commit `.terraform.lock.hcl` to version control
3. **Use version constraints** - Specify provider and module versions
4. **Enable plugin cache** - Reduce bandwidth usage and initialization time
5. **Automate in CI/CD** - Use `-input=false` and `-backend-config` for automation
6. **Upgrade regularly** - Run `init -upgrade` periodically to get bug fixes
7. **Review lock file changes** - Check lock file updates in code reviews
8. **Separate environments** - Use different backend prefixes per environment
9. **Test migrations** - Test state migrations in non-production first
10. **Document backend config** - Document backend configuration in README

## Exit Codes

- `0` - Success
- `1` - Error occurred
- `2` - Partial success (used with some flags)

## Related Commands

- `tofu get` - Downloads modules only (subset of init)
- `tofu validate` - Validates configuration after init
- `tofu providers` - Shows provider requirements
- `tofu version` - Shows OpenTofu and provider versions
