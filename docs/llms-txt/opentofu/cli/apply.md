# tofu apply

The `tofu apply` command executes the actions proposed in an OpenTofu plan.

## Usage

```bash
tofu apply [options] [plan file]
```

## Two Operating Modes

### Automatic Plan Mode

Running `tofu apply` without a saved plan file triggers OpenTofu to automatically generate an execution plan, request approval, and then proceed with the indicated actions.

```bash
tofu apply
```

OpenTofu will:
1. Generate a plan
2. Display proposed changes
3. Prompt for confirmation
4. Execute the changes after approval

### Saved Plan Mode

When you provide a previously-saved plan file, OpenTofu implements those specific actions without requesting confirmation.

```bash
tofu plan -out=tfplan
tofu apply tfplan
```

This is suitable for automated workflows and ensures exactly what was planned gets applied.

## Important Options

### Auto-Approve

```bash
tofu apply -auto-approve
```

Bypasses interactive approval prompt. Use with caution.

**Warning:** If you use `-auto-approve`, make sure that no one can change your infrastructure outside of your OpenTofu workflow.

### JSON Output

```bash
tofu apply -json
```

Enables machine-readable JSON output for automation.

### Parallelism

```bash
tofu apply -parallelism=20
```

Controls concurrent operations (default: 10).

### Lock Options

```bash
tofu apply -lock=false
```

Disables state locking. **Not recommended.**

```bash
tofu apply -lock-timeout=5m
```

Overrides default state lock timeout.

### Show Sensitive Values

```bash
tofu apply -show-sensitive
```

Displays redacted sensitive values in output.

## Planning Modes

Apply supports all planning mode flags from `tofu plan`:

### Destroy Mode

```bash
tofu apply -destroy
```

Destroys all managed infrastructure. Equivalent to `tofu destroy`.

### Refresh-Only Mode

```bash
tofu apply -refresh-only
```

Updates state to match real infrastructure without making changes.

### Refresh Control

```bash
tofu apply -refresh=false
```

Skips state refresh before applying.

## Targeting and Filtering

### Target Specific Resources

```bash
tofu apply -target=google_cloud_run_service.api
```

### Exclude Resources

```bash
tofu apply -exclude=google_monitoring_dashboard.optional
```

### Replace Resources

```bash
tofu apply -replace=google_cloud_run_service.api
```

## Variable Options

```bash
tofu apply -var="project_id=my-project" -var="region=us-central1"
```

```bash
tofu apply -var-file="production.tfvars"
```

## Common Workflows

### Development

```bash
# Interactive workflow
tofu apply

# Review plan, type 'yes' to confirm
```

### Production with Saved Plans

```bash
# Create plan
tofu plan -out=prod.tfplan

# Review plan
tofu show prod.tfplan

# Apply after approval (no confirmation needed)
tofu apply prod.tfplan
```

### CI/CD Pipeline

```bash
# Non-interactive apply with saved plan
tofu apply -input=false -auto-approve tfplan
```

### Automated Apply Without Plan File

```bash
# Dangerous: auto-approve without review
tofu apply -auto-approve -var-file="prod.tfvars"
```

## Understanding Apply Output

```
google_cloud_run_service.api: Creating...
google_cloud_run_service.api: Still creating... [10s elapsed]
google_cloud_run_service.api: Creation complete after 23s [id=projects/my-project/locations/us-central1/services/api-service]

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.

Outputs:

service_url = "https://api-service-abc123-uc.a.run.app"
```

Status messages:
- `Creating...` - Resource is being created
- `Modifying...` - Resource is being updated
- `Destroying...` - Resource is being deleted
- `Still creating/modifying/destroying...` - Operation in progress
- `Creation complete` - Resource successfully created

## Error Handling

If apply fails partway through:

```bash
# State is updated with what succeeded
# Review what succeeded
tofu show

# Fix the error in configuration or infrastructure
# Re-run apply
tofu apply
```

OpenTofu tracks state after each resource operation, so partial applies are safe to retry.

## State Backups

OpenTofu automatically creates state backups:

```bash
ls -la
# terraform.tfstate
# terraform.tfstate.backup
```

If something goes wrong:

```bash
# Restore from backup
cp terraform.tfstate.backup terraform.tfstate
```

With remote backends (like GCS), versioning provides better recovery.

## Best Practices

1. **Use saved plans in production** - `tofu plan -out=tfplan` then `tofu apply tfplan`
2. **Review before applying** - Always review the plan
3. **Avoid auto-approve** - Use sparingly, only in trusted automated environments
4. **Test in non-prod first** - Apply changes to staging before production
5. **Use version control** - Track all configuration changes in git
6. **Monitor state** - Enable state versioning and backups
7. **Apply small changes** - Make incremental changes rather than large batches
8. **Coordinate with team** - Use state locking to prevent conflicts
9. **Document changes** - Include reason for changes in commit messages
10. **Have rollback plan** - Know how to revert changes if needed

## CI/CD Pattern

```bash
#!/bin/bash
set -e

# Initialize
tofu init -input=false \
  -backend-config="bucket=${STATE_BUCKET}" \
  -backend-config="prefix=${ENVIRONMENT}"

# Plan
tofu plan -input=false -no-color -out=tfplan

# Show plan for review
tofu show -no-color tfplan

# Wait for approval (manual step in CI/CD)
# ...

# Apply
tofu apply -input=false -no-approve tfplan
```

## Troubleshooting

### Concurrent Modification Error

```
Error: Error acquiring the state lock

Lock Info:
  ID:        abc123...
  Path:      my-bucket/my-prefix
  Operation: OperationTypeApply
  Who:       user@hostname
  Created:   2025-12-19 10:30:15
```

**Solution:**
```bash
# Wait for other operation to complete
# Or if stuck, force unlock (dangerous)
tofu force-unlock <lock-id>
```

### Resource Already Exists

```
Error: A resource with the ID "..." already exists
```

**Solution:**
```bash
# Import existing resource
tofu import google_cloud_run_service.api projects/my-project/locations/us-central1/services/api-service

# Or remove from state if managed elsewhere
tofu state rm google_cloud_run_service.api
```

### API Rate Limiting

**Solution:**
```bash
# Reduce parallelism
tofu apply -parallelism=5
```

## Exit Codes

- `0` - Success
- `1` - Error occurred

## Related Commands

- `tofu plan` - Preview changes before applying
- `tofu destroy` - Destroy managed infrastructure
- `tofu refresh` - Update state to match reality
- `tofu show` - Display state or plan
