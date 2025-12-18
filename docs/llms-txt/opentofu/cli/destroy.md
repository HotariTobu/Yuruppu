# tofu destroy

The `tofu destroy` command is a convenient way to destroy all remote objects managed by a particular OpenTofu configuration.

## Usage

```bash
tofu destroy [options]
```

## What destroy Does

The command accepts most of the options that `tofu apply` accepts, although it does not accept a plan file argument and forces the selection of the 'destroy' planning mode.

Equivalent to:
```bash
tofu apply -destroy
```

## Safety Considerations

**Warning:** You will typically not want to destroy long-lived objects in a production environment.

However, the tool serves a valid purpose for ephemeral infrastructure, particularly in development contexts where temporary resources need cleanup.

## Preview Destroy

Before executing destruction, you can preview changes:

```bash
tofu plan -destroy
```

This will run `tofu plan` in destroy mode, showing you the proposed destroy changes without executing them.

## Common Options

### Auto-Approve

```bash
tofu destroy -auto-approve
```

Skip interactive approval. **Use with extreme caution.**

### Target Specific Resources

```bash
tofu destroy -target=google_cloud_run_service.api
```

Destroy only specific resources.

### Variable Options

```bash
tofu destroy -var="project_id=my-project"
tofu destroy -var-file="production.tfvars"
```

## Common Workflows

### Preview Then Destroy

```bash
# Preview what will be destroyed
tofu plan -destroy

# If satisfied, destroy
tofu destroy
```

### Destroy Development Environment

```bash
# Destroy dev resources
tofu destroy -var="environment=dev"
```

### Cleanup After Testing

```bash
# Create test infrastructure
tofu apply -var-file="test.tfvars"

# Run tests
./run-tests.sh

# Cleanup
tofu destroy -auto-approve -var-file="test.tfvars"
```

### Partial Destroy

```bash
# Destroy specific resources
tofu destroy \
  -target=google_cloud_run_service.temp \
  -target=google_service_account.temp
```

## Safety Measures

### 1. Use prevent_destroy

Protect critical resources:

```hcl
resource "google_sql_database_instance" "main" {
  name             = "production-db"
  database_version = "POSTGRES_15"

  lifecycle {
    prevent_destroy = true
  }
}
```

This will cause `tofu destroy` to fail if it attempts to destroy this resource.

### 2. Manual Approval

Always review before destroying:

```bash
# Preview
tofu plan -destroy

# Confirm you want to destroy
tofu destroy  # Type 'yes' when prompted
```

### 3. Backup State

Before destroying:

```bash
# Backup state
cp terraform.tfstate terraform.tfstate.before-destroy

# Or with remote state
tofu state pull > state-backup.json
```

### 4. Use Separate Environments

Keep production separate:

```
environments/
  dev/
    main.tf
  staging/
    main.tf
  prod/
    main.tf
```

```bash
cd environments/dev
tofu destroy  # Only affects dev
```

## Understanding Destroy Output

```
google_cloud_run_service.api: Refreshing state... [id=projects/my-project/locations/us-central1/services/api-service]

OpenTofu will perform the following actions:

  # google_cloud_run_service.api will be destroyed
  - resource "google_cloud_run_service" "api" {
      - id       = "projects/my-project/locations/us-central1/services/api-service" -> null
      - name     = "api-service" -> null
      - location = "us-central1" -> null
    }

Plan: 0 to add, 0 to change, 1 to destroy.

Do you really want to destroy all resources?
  OpenTofu will destroy all your managed infrastructure, as shown above.
  There is no undo. Only 'yes' will be accepted to confirm.

  Enter a value:
```

Type `yes` to proceed.

## Destroy Order

OpenTofu automatically determines the correct order to destroy resources based on dependencies:

```hcl
resource "google_service_account" "api" {
  account_id = "api-service"
}

resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = var.region

  template {
    spec {
      service_account_name = google_service_account.api.email
    }
  }
}
```

OpenTofu will:
1. Destroy `google_cloud_run_service.api` first (depends on service account)
2. Destroy `google_service_account.api` second (no dependencies)

## Handling Errors

If destroy fails:

```bash
# Some resources destroyed, some failed
# State is updated

# Check what remains
tofu show

# Fix the issue
# Re-run destroy
tofu destroy
```

## Alternatives to Destroy

### 1. Remove from Configuration

Instead of destroying, remove from OpenTofu management:

```bash
tofu state rm google_cloud_run_service.api
```

Resource continues to exist but is no longer managed by OpenTofu.

### 2. Conditional Resources

Use variables to control resource creation:

```hcl
variable "create_monitoring" {
  type    = bool
  default = false
}

resource "google_monitoring_dashboard" "main" {
  count = var.create_monitoring ? 1 : 0

  dashboard_json = file("dashboard.json")
}
```

```bash
# Destroy monitoring by setting variable
tofu apply -var="create_monitoring=false"
```

### 3. Workspace-Based Isolation

```bash
# Create dev workspace
tofu workspace new dev
tofu apply

# Destroy dev workspace
tofu destroy

# Production workspace unaffected
tofu workspace select prod
```

## Best Practices

1. **Always preview** - Run `tofu plan -destroy` before destroying
2. **Never auto-approve in prod** - Require manual confirmation
3. **Protect critical resources** - Use `prevent_destroy` lifecycle argument
4. **Backup before destroy** - Save state and configuration
5. **Test destroy in non-prod** - Verify destroy works as expected
6. **Use targeted destroy sparingly** - Understand dependency implications
7. **Document destroy actions** - Note why resources were destroyed
8. **Separate environments** - Isolate dev, staging, and prod
9. **Check for external dependencies** - Ensure no other systems depend on resources
10. **Have recovery plan** - Know how to recreate resources if needed

## CI/CD Pattern

```bash
#!/bin/bash
# Destroy ephemeral test environment

# Initialize
tofu init -input=false

# Preview destroy
tofu plan -destroy -input=false -no-color

# Destroy with approval
if [ "$AUTO_APPROVE" = "true" ]; then
  tofu destroy -auto-approve -input=false
else
  tofu destroy -input=false
fi
```

## Troubleshooting

### Resources Can't Be Destroyed

Some resources may have protection:

```
Error: Instance cannot be destroyed

  lifecycle {
    prevent_destroy = true
  }
```

**Solution:** Remove the `prevent_destroy` setting.

### Dependent Resources Exist

```
Error: Cannot destroy resource because it is referenced by another resource
```

**Solution:** Destroy dependent resources first or use `-target` to specify order.

### API Errors

```
Error: Error waiting for service to be deleted: timeout while waiting for state to become 'DELETED'
```

**Solution:**
- Manually delete in console if stuck
- Use `tofu state rm` to remove from state
- Try again with longer timeout

## Exit Codes

- `0` - Success
- `1` - Error occurred

## Related Commands

- `tofu plan -destroy` - Preview destroy changes
- `tofu apply -destroy` - Alternative destroy command
- `tofu state rm` - Remove resource from state without destroying
