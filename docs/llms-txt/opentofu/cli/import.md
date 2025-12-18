# tofu import

The `tofu import` command imports existing resources into OpenTofu state, allowing you to bring resources created outside OpenTofu under management.

## Usage

```bash
tofu import [options] ADDRESS ID
```

Import will find the existing resource from ID and import it into your OpenTofu state at the given ADDRESS.

## Parameters

- **ADDRESS** - Valid resource address (supports modules, count, for_each)
- **ID** - Provider-specific resource identifier

## Important Considerations

OpenTofu expects that each remote object it is managing will be bound to only one resource address. Importing the same object multiple times can cause unwanted behavior.

## Basic Examples

### Cloud Run Service

```bash
tofu import google_cloud_run_service.api \
  projects/my-project/locations/us-central1/services/api-service
```

### Service Account

```bash
tofu import google_service_account.api \
  projects/my-project/serviceAccounts/api-sa@my-project.iam.gserviceaccount.com
```

### Secret Manager Secret

```bash
tofu import google_secret_manager_secret.api_key \
  projects/my-project/secrets/api-key
```

### Artifact Registry Repository

```bash
tofu import google_artifact_registry_repository.images \
  projects/my-project/locations/us-central1/repositories/docker-images
```

### Cloud Build Trigger

```bash
tofu import google_cloudbuild_trigger.deploy \
  projects/my-project/locations/global/triggers/abc-123-def
```

## Import with count

```bash
tofu import 'google_cloud_run_service.services[0]' \
  projects/my-project/locations/us-central1/services/api-service
```

Note the single quotes to prevent shell interpretation of brackets.

## Import with for_each

```bash
tofu import 'google_cloud_run_service.services["api"]' \
  projects/my-project/locations/us-central1/services/api-service
```

## Import into Modules

```bash
tofu import module.cloud_run.google_cloud_run_service.api \
  projects/my-project/locations/us-central1/services/api-service
```

## Common Options

### Configuration Path

```bash
tofu import -config=../other-dir google_cloud_run_service.api <ID>
```

### Variable Assignment

```bash
tofu import \
  -var="project_id=my-project" \
  -var="region=us-central1" \
  google_cloud_run_service.api <ID>
```

### Skip State Lock

```bash
tofu import -lock=false google_cloud_run_service.api <ID>
```

## Import Workflow

### 1. Write Configuration

First, write the resource configuration:

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api-service"
  location = "us-central1"

  template {
    spec {
      containers {
        image = "gcr.io/my-project/api:latest"
      }
    }
  }
}
```

### 2. Import Resource

```bash
tofu import google_cloud_run_service.api \
  projects/my-project/locations/us-central1/services/api-service
```

### 3. Verify Import

```bash
tofu state show google_cloud_run_service.api
```

### 4. Check for Drift

```bash
tofu plan
```

If the plan shows changes, update your configuration to match the actual resource.

### 5. Iterate Configuration

Adjust configuration until `tofu plan` shows no changes:

```bash
# Plan should show: No changes. Infrastructure is up-to-date.
tofu plan
```

## Finding Resource IDs

Each provider has different ID formats. Check provider documentation or use cloud console.

### Google Cloud

**Cloud Run:**
```
projects/{project}/locations/{location}/services/{service}
```

**Service Account:**
```
projects/{project}/serviceAccounts/{email}
```

**Secret:**
```
projects/{project}/secrets/{secret_id}
```

**Artifact Registry:**
```
projects/{project}/locations/{location}/repositories/{repository}
```

You can often find IDs in the cloud console URL or using gcloud:

```bash
# List Cloud Run services
gcloud run services list --format="value(metadata.name,metadata.namespace,metadata.location)"

# Get service account email
gcloud iam service-accounts list
```

## Import Multiple Resources

```bash
#!/bin/bash

# Import multiple Cloud Run services
services=(
  "api:projects/my-project/locations/us-central1/services/api"
  "worker:projects/my-project/locations/us-central1/services/worker"
  "frontend:projects/my-project/locations/us-central1/services/frontend"
)

for entry in "${services[@]}"; do
  IFS=: read -r name id <<< "$entry"
  tofu import "google_cloud_run_service.services[\"$name\"]" "$id"
done
```

## Limitations

1. **Configuration Required** - You must write configuration before importing
2. **One Resource at a Time** - Cannot import multiple resources in one command
3. **No Bulk Import** - Must import each resource individually
4. **Provider-Specific IDs** - ID format varies by provider and resource type
5. **State Only** - Import only affects state, not configuration

## Common Use Cases

### Migrating to OpenTofu

Import existing infrastructure created manually or by other tools.

### Recovering from State Loss

If state is lost, reimport resources:

```bash
# Reimport all resources
tofu import google_cloud_run_service.api <ID>
tofu import google_service_account.api <ID>
# ... etc
```

### Adopting Existing Resources

Bring resources created by other teams under OpenTofu management.

## Best Practices

1. **Write configuration first** - Always write config before importing
2. **Import one at a time** - Import and verify each resource individually
3. **Verify with plan** - Run `tofu plan` after import to check drift
4. **Update configuration** - Adjust config to match actual state
5. **Document IDs** - Keep a record of resource IDs for reference
6. **Test in non-prod** - Practice import workflow before production
7. **Backup state** - Backup state before importing many resources
8. **Use version control** - Track configuration changes during import
9. **Check dependencies** - Import dependent resources in correct order
10. **Validate imports** - Use `tofu state show` to verify import

## Troubleshooting

### Resource Not Found

```
Error: Cannot import non-existent remote object
```

**Solution:**
- Verify resource exists in cloud console
- Check ID format is correct
- Ensure correct project/region

### Configuration Mismatch

After import, `tofu plan` shows many changes.

**Solution:**
- Use `tofu state show` to see actual attributes
- Update configuration to match
- Run `tofu plan` again
- Iterate until no changes shown

### Import to Wrong Address

```bash
# Remove from state
tofu state rm google_cloud_run_service.wrong

# Reimport to correct address
tofu import google_cloud_run_service.correct <ID>
```

### Duplicate Resource

```
Error: Resource already managed by OpenTofu
```

**Solution:**
- Check if already imported
- Use `tofu state list` to see managed resources
- Don't import the same resource twice

## Alternative: Generated Configuration

Some providers support generating configuration during import:

```bash
# Future feature in OpenTofu
tofu import -generate-config google_cloud_run_service.api <ID>
```

Check OpenTofu documentation for the latest features.

## Exit Codes

- `0` - Success
- `1` - Error occurred

## Related Commands

- `tofu state show` - View imported resource details
- `tofu state list` - List all managed resources
- `tofu plan` - Check for drift after import
- `tofu state rm` - Remove resource from state
