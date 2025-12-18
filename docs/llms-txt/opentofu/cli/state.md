# tofu state

The `tofu state` command provides advanced state management capabilities for inspecting and modifying OpenTofu state.

## Usage

```bash
tofu state <subcommand> [options] [args]
```

## Key Characteristics

The OpenTofu state subcommands all work with remote state just as if it was local state. You can use these commands whether your state is stored locally or in a remote backend like GCS.

## Backup Protection

All modification subcommands automatically generate backup files, which cannot be disabled. This safeguard protects the sensitive state file. Read-only operations like `list` don't create backups since they don't modify the state.

## Subcommands

### list

List resources in the state:

```bash
tofu state list
```

Output:
```
google_cloud_run_service.api
google_service_account.api
google_secret_manager_secret.api_key
module.network.google_compute_network.vpc
```

Filter by resource or module:

```bash
tofu state list google_cloud_run_service
tofu state list module.network
```

### show

Display detailed information about a specific resource:

```bash
tofu state show google_cloud_run_service.api
```

Output shows all attributes:
```
# google_cloud_run_service.api:
resource "google_cloud_run_service" "api" {
    id       = "projects/my-project/locations/us-central1/services/api-service"
    location = "us-central1"
    name     = "api-service"
    project  = "my-project"

    template {
        spec {
            containers {
                image = "gcr.io/my-project/api:latest"
                ...
            }
        }
    }

    status {
        url = "https://api-service-abc123-uc.a.run.app"
        ...
    }
}
```

### mv

Move or rename resources in state:

```bash
# Rename a resource
tofu state mv google_cloud_run_service.old google_cloud_run_service.new

# Move to a module
tofu state mv google_cloud_run_service.api module.api.google_cloud_run_service.api

# Move from a module
tofu state mv module.api.google_cloud_run_service.api google_cloud_run_service.api

# Move between count indices
tofu state mv 'google_cloud_run_service.services[0]' 'google_cloud_run_service.services[1]'

# Move to for_each
tofu state mv 'google_cloud_run_service.services[0]' 'google_cloud_run_service.services["api"]'
```

**Use cases:**
- Refactoring module structure
- Renaming resources
- Converting between count and for_each
- Reorganizing resource hierarchy

### rm

Remove resources from state without destroying them:

```bash
tofu state rm google_cloud_run_service.api
```

The resource continues to exist in the cloud but is no longer managed by OpenTofu.

**Use cases:**
- Stop managing a resource with OpenTofu
- Resource was deleted outside OpenTofu
- Migrating to another OpenTofu configuration
- Removing duplicates after accidental imports

Remove multiple resources:

```bash
tofu state rm google_cloud_run_service.api google_service_account.api
```

Remove entire modules:

```bash
tofu state rm module.network
```

### pull

Download and display remote state:

```bash
tofu state pull
```

Outputs state in JSON format. Useful for:
- Inspecting remote state
- Backing up state
- Debugging state issues
- Parsing with jq

```bash
# Save state backup
tofu state pull > state-backup.json

# Query state with jq
tofu state pull | jq '.resources[] | select(.type == "google_cloud_run_service")'
```

### push

Upload local state to remote backend:

```bash
tofu state push terraform.tfstate
```

**Warning:** Use with extreme caution. This can overwrite remote state and cause data loss or state corruption.

**Use cases:**
- Restoring from backup
- Migrating state to new backend
- Recovering from state corruption

Always backup before pushing:

```bash
tofu state pull > state-before-push.json
tofu state push restored-state.tfstate
```

### replace-provider

Update provider references in state:

```bash
tofu state replace-provider hashicorp/google registry.opentofu.org/hashicorp/google
```

**Use cases:**
- Migrating to different provider registry
- Updating provider source after fork
- Fixing provider source inconsistencies

## Common Workflows

### Inspecting State

```bash
# List all resources
tofu state list

# Show specific resource
tofu state show google_cloud_run_service.api

# Pull and analyze with jq
tofu state pull | jq '.resources[].type' | sort | uniq -c
```

### Refactoring Modules

```bash
# Move resources into new module
tofu state mv google_cloud_run_service.api module.api.google_cloud_run_service.api
tofu state mv google_service_account.api module.api.google_service_account.api

# Update configuration to match
# Then verify with plan
tofu plan
```

### Removing Resources

```bash
# Resource deleted outside OpenTofu
tofu state rm google_cloud_run_service.deleted

# Stop managing resource
tofu state rm google_monitoring_dashboard.external
```

### Converting count to for_each

```bash
# Original: count-based resources
# google_cloud_run_service.services[0]
# google_cloud_run_service.services[1]

# Move to for_each keys
tofu state mv \
  'google_cloud_run_service.services[0]' \
  'google_cloud_run_service.services["api"]'

tofu state mv \
  'google_cloud_run_service.services[1]' \
  'google_cloud_run_service.services["worker"]'

# Update configuration to use for_each
# Then verify
tofu plan
```

### Backup and Restore

```bash
# Backup
tofu state pull > state-$(date +%Y%m%d-%H%M%S).json

# Restore if needed
tofu state push state-20251219-103000.json
```

## Best Practices

1. **Backup before modifications** - Always backup state before running state commands
2. **Use pull for inspection** - Prefer `pull` over direct state file access
3. **Test in non-prod** - Practice state operations in development first
4. **Verify with plan** - Run `tofu plan` after state modifications
5. **Document changes** - Note why state was modified
6. **Coordinate with team** - Ensure no concurrent operations during state changes
7. **Use version control** - Track configuration changes that accompany state changes
8. **Avoid push when possible** - Let OpenTofu manage state naturally
9. **Use rm carefully** - Understand that removed resources won't be destroyed
10. **Check backups** - State commands create `.backup` files automatically

## CLI Integration

The commands are designed for compatibility with standard Unix tools like grep and awk:

```bash
# Find all Cloud Run services
tofu state list | grep google_cloud_run_service

# Count resources by type
tofu state list | cut -d. -f1 | sort | uniq -c

# Get all service URLs
for resource in $(tofu state list | grep google_cloud_run_service); do
  tofu state show $resource | grep "url.*="
done

# Complex queries with jq
tofu state pull | jq -r '.resources[] |
  select(.type == "google_cloud_run_service") |
  .instances[].attributes.status[0].url'
```

## State File Safety

### Automatic Backups

When you modify state, OpenTofu creates backups:

```bash
ls -la
# terraform.tfstate
# terraform.tfstate.backup
# terraform.tfstate.1234567890.backup
```

### Remote State Versioning

With GCS backend:

```bash
# List versions
gsutil ls -la gs://my-terraform-state/production/cloudrun/default.tfstate

# Restore specific version
gsutil cp gs://my-terraform-state/production/cloudrun/default.tfstate#1234567890 terraform.tfstate
tofu state push terraform.tfstate
```

## Troubleshooting

### State Lock Errors

```bash
# If another operation is running
tofu force-unlock <lock-id>
```

### Corrupted State

```bash
# Restore from backup
cp terraform.tfstate.backup terraform.tfstate

# Or from remote backend version
tofu state pull > corrupted-state.json
# Restore previous version from GCS
tofu state push restored-state.tfstate
```

### Resource Not Found

```bash
# List to see actual resource names
tofu state list

# Check for typos in resource address
tofu state show google_cloud_run_service.api
```

### Move Failed

```bash
# If move fails, state is unchanged
# Check error message
# Verify source and destination addresses
# Try again with corrected addresses
```

## Exit Codes

- `0` - Success
- `1` - Error occurred

## Related Commands

- `tofu show` - Display state or plan
- `tofu import` - Import existing resources
- `tofu refresh` - Update state to match reality
- `tofu plan` - Preview changes
