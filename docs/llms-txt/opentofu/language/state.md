# State Management

OpenTofu maintains state to map real world resources to your configuration, keep track of metadata, and improve performance for large infrastructures.

## Purpose

The state file serves as the source of truth for your infrastructure. It:

1. **Maps Configuration to Reality** - Establishes one-to-one bindings between configured resource instances and actual remote objects
2. **Tracks Metadata** - Stores resource dependencies and other metadata
3. **Improves Performance** - Caches resource attributes to avoid querying providers for every operation
4. **Enables Collaboration** - Allows teams to share infrastructure state

## State File

### Default Storage

By default, state is stored locally as `terraform.tfstate` in your working directory. This file uses JSON format.

### State Contents

The state file contains:
- Resource mappings and attributes
- Output values
- Dependency information
- Provider configurations
- Workspace information

**Warning**: State files can contain sensitive data like passwords and private keys. Protect state files appropriately.

## Remote State

For team environments, store state remotely using backends like GCS (Google Cloud Storage).

### Benefits of Remote State

- **Collaboration** - Multiple team members can work together
- **Locking** - Prevents concurrent modifications
- **Versioning** - Track state changes over time
- **Security** - Encrypted storage and access controls
- **Backup** - Automatic backups and recovery

### GCS Backend Example

```hcl
terraform {
  backend "gcs" {
    bucket = "my-terraform-state"
    prefix = "production/cloudrun"
  }
}
```

See the [GCS Backend documentation](../gcs-backend.md) for detailed configuration.

## State Commands

OpenTofu provides the `tofu state` CLI command for safe state modifications. Rather than directly editing JSON state files, use these commands:

### List Resources

```bash
tofu state list
```

Lists all resources in the state:

```
google_cloud_run_service.api
google_service_account.api
google_secret_manager_secret.api_key
```

### Show Resource Details

```bash
tofu state show google_cloud_run_service.api
```

Displays detailed information about a specific resource.

### Move Resources

Rename resources or move them between modules:

```bash
# Rename a resource
tofu state mv google_cloud_run_service.old google_cloud_run_service.new

# Move to a module
tofu state mv google_cloud_run_service.api module.api.google_cloud_run_service.api
```

### Remove Resources

Remove resources from state without destroying them:

```bash
tofu state rm google_cloud_run_service.api
```

Useful when:
- Resources were deleted outside of OpenTofu
- You want to stop managing a resource with OpenTofu
- Migrating resources to another configuration

### Pull State

Download and display remote state:

```bash
tofu state pull
```

### Push State

Upload local state to remote backend:

```bash
tofu state push terraform.tfstate
```

**Warning**: Use with caution; can overwrite remote state.

### Replace Provider

Update provider references in state:

```bash
tofu state replace-provider hashicorp/google registry.opentofu.org/hashicorp/google
```

## State Inspection

### Viewing Outputs

```bash
# Show all outputs
tofu output

# Show specific output
tofu output service_url

# JSON format
tofu output -json

# Raw value (no quotes)
tofu output -raw service_url
```

### Show Command

Display human-readable state or plan:

```bash
# Show current state
tofu show

# Show saved plan
tofu show tfplan

# JSON format
tofu show -json
```

## State Management Best Practices

### 1. Never Manually Edit State

Always use `tofu state` commands or the OpenTofu workflow. Direct JSON editing can corrupt state.

### 2. Use Remote State for Teams

Configure remote backends like GCS for team collaboration:

```hcl
terraform {
  backend "gcs" {
    bucket = "company-terraform-state"
    prefix = "team/project"
  }
}
```

### 3. Enable State Locking

Use backends that support locking (like GCS) to prevent concurrent modifications.

### 4. Version Your State

Enable versioning on your state storage:

```bash
# For GCS buckets
gcloud storage buckets update gs://my-terraform-state --versioning
```

### 5. Encrypt State

Use encrypted backends and access controls:

```hcl
terraform {
  backend "gcs" {
    bucket             = "my-terraform-state"
    prefix             = "production"
    kms_encryption_key = "projects/my-project/locations/global/keyRings/terraform/cryptoKeys/state"
  }
}
```

### 6. Backup State Files

Regularly backup state files, especially before major operations.

```bash
# Before major changes
cp terraform.tfstate terraform.tfstate.backup
```

### 7. One-to-One Mapping

When manually altering state bindings (via `tofu import` or `tofu state rm`), verify the one-to-one mapping remains valid.

### 8. Protect Sensitive Data

State files contain sensitive data. Use:
- Encrypted backends
- IAM access controls
- Audit logging
- Secure CI/CD pipelines

### 9. Organize State Files

Use separate state files for different environments:

```
# Production
terraform {
  backend "gcs" {
    bucket = "company-terraform-state"
    prefix = "production/cloudrun"
  }
}

# Staging
terraform {
  backend "gcs" {
    bucket = "company-terraform-state"
    prefix = "staging/cloudrun"
  }
}
```

### 10. Don't Commit State to Version Control

Add to `.gitignore`:

```
# .gitignore
*.tfstate
*.tfstate.*
.terraform/
```

## State Locking

State locking prevents concurrent state modifications that could corrupt state.

### How It Works

1. Before operations, OpenTofu acquires a lock
2. Performs the operation
3. Releases the lock

### Lock Timeout

Configure lock timeout:

```bash
tofu apply -lock-timeout=10m
```

### Force Unlock

If a lock gets stuck (e.g., process crash):

```bash
tofu force-unlock LOCK_ID
```

**Warning**: Only use if you're certain no other process is using the state.

## State Refresh

OpenTofu refreshes state to detect changes made outside of OpenTofu.

### Automatic Refresh

By default, `tofu plan` and `tofu apply` automatically refresh state:

```bash
tofu plan  # Refreshes state, then plans
```

### Skip Refresh

```bash
tofu plan -refresh=false
```

Improves speed but might miss external changes.

### Refresh-Only Mode

Update state without making infrastructure changes:

```bash
tofu apply -refresh-only
```

## Importing Existing Resources

Import resources created outside OpenTofu:

```bash
# Import a Cloud Run service
tofu import google_cloud_run_service.api projects/my-project/locations/us-central1/services/api-service

# Import a service account
tofu import google_service_account.api projects/my-project/serviceAccounts/api-sa@my-project.iam.gserviceaccount.com
```

After importing:
1. Verify with `tofu state show`
2. Run `tofu plan` to check for drift
3. Update configuration to match imported resource

## Workspaces

Workspaces allow multiple state files for the same configuration:

```bash
# Create and switch to workspace
tofu workspace new staging

# List workspaces
tofu workspace list

# Select workspace
tofu workspace select production

# Show current workspace
tofu workspace show
```

State files are stored separately per workspace:
```
terraform.tfstate.d/
  staging/
    terraform.tfstate
  production/
    terraform.tfstate
```

Access workspace name in configuration:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-${terraform.workspace}"
  location = var.region
}
```

## State Outputs for Integration

Export state for external consumption:

```bash
# JSON output
tofu output -json > outputs.json

# Specific output
tofu output -raw service_url > service_url.txt

# Show state in JSON
tofu show -json > state.json
```

These enable automation workflows where state artifacts are captured post-deployment for downstream system integration.

## Disaster Recovery

If state is lost or corrupted:

1. **Restore from backup** - Use versioned state from GCS
2. **Import resources** - Re-import resources into new state
3. **Rebuild state** - In worst case, manually reconstruct state

Prevention is key:
- Use remote backends with versioning
- Regular backups
- Access logging and monitoring
- Test restore procedures
