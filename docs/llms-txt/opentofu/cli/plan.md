# tofu plan

The `tofu plan` command creates an execution plan, which lets you preview the changes that OpenTofu plans to make to your infrastructure.

## Usage

```bash
tofu plan [options]
```

## What plan Does

1. **Refreshes State** - Synchronizes state with real infrastructure
2. **Compares Configuration** - Compares desired state (config) with actual state
3. **Creates Execution Plan** - Determines what actions are needed
4. **Displays Changes** - Shows what will be created, updated, or destroyed

## Basic Example

```bash
tofu plan
```

Output shows:
- `+` Resources to be created
- `~` Resources to be updated
- `-` Resources to be destroyed
- `-/+` Resources to be replaced (destroyed then created)

Example output:
```
OpenTofu will perform the following actions:

  # google_cloud_run_service.api will be updated in-place
  ~ resource "google_cloud_run_service" "api" {
        name     = "api-service"
      ~ template {
          ~ spec {
              ~ containers {
                  ~ image = "gcr.io/project/api:v1.0" -> "gcr.io/project/api:v1.1"
                }
            }
        }
    }

Plan: 0 to add, 1 to change, 0 to destroy.
```

## Planning Modes

### Normal Mode (default)

```bash
tofu plan
```

Shows infrastructure changes needed to match configuration.

### Destroy Mode

```bash
tofu plan -destroy
```

Creates a plan to eliminate all remote objects and empty the state.

### Refresh-Only Mode

```bash
tofu plan -refresh-only
```

Creates a plan whose goal is only to update the OpenTofu state and any root module output values to match changes made to remote objects outside of OpenTofu.

## Important Options

### Saving Plans

```bash
tofu plan -out=tfplan
```

Saves the plan to a file for later execution with `tofu apply tfplan`.

**Benefits:**
- Ensures exactly what was reviewed gets applied
- Required for CI/CD pipelines
- Prevents drift between plan and apply

### Detailed Exit Code

```bash
tofu plan -detailed-exitcode
```

Returns granular exit codes:
- `0` - No changes needed (infrastructure matches configuration)
- `1` - Error occurred
- `2` - Changes are present in the plan

Useful in scripts and CI/CD:

```bash
if tofu plan -detailed-exitcode -out=tfplan; then
  echo "No changes needed"
else
  if [ $? -eq 2 ]; then
    echo "Changes detected, review required"
  fi
fi
```

### JSON Output

```bash
tofu plan -json
```

Produces machine-readable JSON output suitable for automation and parsing.

```bash
tofu plan -json | jq '.resource_changes[] | select(.change.actions[] | contains("create"))'
```

## Refresh Options

### Skip Refresh

```bash
tofu plan -refresh=false
```

Skips synchronization with remote objects. Improves speed but might miss external changes.

Use when:
- You're certain no external changes occurred
- Speed is critical
- You're testing configuration changes

## Targeting Options

### Target Specific Resources

```bash
tofu plan -target=google_cloud_run_service.api
```

Focuses planning on specific resources and their dependencies.

```bash
tofu plan \
  -target=google_cloud_run_service.api \
  -target=google_service_account.api
```

**Warning:** Use sparingly. Can lead to inconsistent infrastructure state.

### Target File

```bash
tofu plan -target-file=targets.txt
```

**targets.txt:**
```
google_cloud_run_service.api
google_service_account.api
module.network
```

### Exclude Resources

```bash
tofu plan -exclude=google_cloud_run_service.worker
```

Excludes specific resources from the plan.

```bash
tofu plan -exclude-file=excludes.txt
```

## Resource Replacement

### Replace Specific Resources

```bash
tofu plan -replace=google_cloud_run_service.api
```

Plans resource replacement instead of updates. OpenTofu will destroy and recreate the resource.

Useful when:
- Resource is in a bad state
- Update isn't working correctly
- You need to force recreation

```bash
tofu plan -replace=google_cloud_run_service.api -out=tfplan
tofu apply tfplan
```

## Variable Options

### Set Variables

```bash
tofu plan -var="project_id=my-project" -var="region=us-central1"
```

### Variable Files

```bash
tofu plan -var-file="production.tfvars"
```

```bash
tofu plan \
  -var-file="common.tfvars" \
  -var-file="production.tfvars" \
  -var="override_value=special"
```

## Output Options

### No Color

```bash
tofu plan -no-color
```

Removes color formatting. Useful for logs and CI/CD output.

### Compact Warnings

```bash
tofu plan -compact-warnings
```

Shows warnings in compact format.

## Performance Options

### Parallelism

```bash
tofu plan -parallelism=20
```

Limits concurrent operations (default: 10). Increase for faster plans on large infrastructures, decrease to reduce API rate limiting.

### Lock Options

```bash
tofu plan -lock=false
```

Disables state locking. **Not recommended** unless you're certain no concurrent operations are occurring.

```bash
tofu plan -lock-timeout=5m
```

Overrides default state lock timeout.

## Common Workflows

### Development Workflow

```bash
# Make changes to configuration
vim main.tf

# Preview changes
tofu plan

# If satisfied, apply
tofu apply
```

### Production Workflow

```bash
# Create saved plan
tofu plan -out=prod.tfplan

# Review the plan
tofu show prod.tfplan

# Apply exact plan after approval
tofu apply prod.tfplan
```

### CI/CD Pipeline

```bash
# Generate plan
tofu plan -input=false -no-color -out=tfplan -detailed-exitcode

# Save plan for review
tofu show -json tfplan > plan.json

# Later, after approval
tofu apply -input=false -auto-approve tfplan
```

### Drift Detection

```bash
# Check for external changes
tofu plan -refresh-only

# Or check regularly in monitoring
tofu plan -detailed-exitcode -no-color | grep "Plan:"
```

### Targeted Planning

```bash
# Plan changes for specific module
tofu plan -target=module.cloud_run

# Plan excluding certain resources
tofu plan -exclude=google_monitoring_dashboard.optional
```

## Understanding Plan Output

### Symbols

- `+` Create
- `-` Destroy
- `~` Update in-place
- `-/+` Replace (destroy then create)
- `+/~` Create then update (rare)
- `<=` Read during apply

### Attributes

- `(known after apply)` - Value will be determined during apply
- `(sensitive value)` - Value marked as sensitive
- `# (config)` - Value from configuration
- `# forces replacement` - Change requires resource recreation

### Example

```
# google_cloud_run_service.api will be updated in-place
~ resource "google_cloud_run_service" "api" {
      id       = "locations/us-central1/namespaces/my-project/services/api-service"
      name     = "api-service"
    ~ template {
        ~ metadata {
            ~ annotations = {
                ~ "autoscaling.knative.dev/maxScale" = "10" -> "20"
            }
        }
        ~ spec {
            ~ containers {
                ~ image = "gcr.io/project/api:v1.0" -> "gcr.io/project/api:v1.1" # forces replacement
            }
        }
    }
}

Plan: 0 to add, 1 to change, 0 to destroy.
```

## Speculative vs. Saved Plans

### Speculative Plan

```bash
tofu plan
```

- Not saved
- For review and validation
- Cannot be applied
- Useful for code reviews and exploration

### Saved Plan

```bash
tofu plan -out=tfplan
```

- Saved to file
- Can be applied exactly
- Includes all computed values
- Required for production pipelines

## Plan Files

### Inspecting Plans

```bash
# Human-readable format
tofu show tfplan

# JSON format
tofu show -json tfplan

# Specific output
tofu show -json tfplan | jq '.resource_changes'
```

### Security Considerations

Plan files may contain sensitive data. Protect them:

```bash
# Encrypt plan files
openssl enc -aes-256-cbc -salt -in tfplan -out tfplan.enc

# Decrypt before apply
openssl enc -aes-256-cbc -d -in tfplan.enc -out tfplan
tofu apply tfplan
```

## Best Practices

1. **Always plan before apply** - Review changes before executing
2. **Save plans in CI/CD** - Use `-out` for production pipelines
3. **Use detailed-exitcode** - Automate change detection
4. **Review plan output** - Don't blindly apply changes
5. **Check for "forces replacement"** - Be aware of destructive changes
6. **Use refresh-only** - Detect drift regularly
7. **Limit targeting** - Use `-target` sparingly
8. **Version control configs** - Keep configurations in git
9. **Document plan decisions** - Note why changes are being made
10. **Test in non-prod first** - Always test plans in staging

## Troubleshooting

### Plan Takes Too Long

```bash
# Increase parallelism
tofu plan -parallelism=20

# Skip refresh if safe
tofu plan -refresh=false
```

### State Lock Errors

```bash
# Increase timeout
tofu plan -lock-timeout=10m

# Last resort: force unlock (dangerous)
tofu force-unlock <lock-id>
```

### Too Much Output

```bash
# Save to file
tofu plan -no-color > plan-output.txt

# Filter in JSON
tofu plan -json | jq '.resource_changes[] | select(.change.actions[] | contains("delete"))'
```

### Plan Doesn't Match Expectations

```bash
# Refresh state explicitly
tofu refresh

# Then plan again
tofu plan

# Or use refresh-only to see what changed
tofu plan -refresh-only
```

## Exit Codes

- `0` - Success, no changes (with `-detailed-exitcode`)
- `1` - Error occurred
- `2` - Success, changes present (with `-detailed-exitcode`)

## Related Commands

- `tofu apply` - Executes the planned changes
- `tofu show` - Displays plan or state
- `tofu refresh` - Updates state to match real infrastructure
- `tofu validate` - Validates configuration syntax
