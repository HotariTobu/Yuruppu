# Managing Secrets

> Secure handling of sensitive information like API keys and passwords through Google Cloud's Secret Manager integration with Cloud Run.

## Overview

Cloud Run enables secure handling of sensitive information through Google Cloud's Secret Manager. The service offers flexible deployment methods for secret access.

**Important**: Never store secrets directly in environment variables or commit them to source control. Always use Secret Manager.

## Access Methods

### Volume Mounting

Cloud Run makes the secret available to the container as files when mounted as volumes. This approach automatically fetches the latest secret version, supporting secret rotation effectively.

**Advantages**:
- Automatic updates with secret rotation
- Supports large secrets
- Can mount multiple secrets to different paths

**Example mount path**: `/etc/secrets/api-key`

### Environment Variables

Secrets can be passed as environment variables. Google recommends pinning to specific versions rather than using "latest" to ensure consistency at startup.

**Considerations**:
- Retrieved before instance startup
- If retrieval fails, the instance won't start
- Should pin to specific versions for stability

## Configuration Methods

Secrets are configurable via:

- **Google Cloud Console**: UI-based configuration
- **gcloud CLI**: `--update-secrets` flag
- **YAML**: Service configurations
- **Terraform**: Infrastructure-as-code

### gcloud Example

```bash
# Mount as volume
gcloud run deploy SERVICE \
  --update-secrets=/secrets/api-key=my-secret:latest

# Set as environment variable
gcloud run deploy SERVICE \
  --update-secrets=API_KEY=my-secret:1
```

## Cross-Project References

Services can access secrets from other projects using the resource ID format:

```
projects/PROJECT_NUMBER/secrets/SECRET_NAME:VERSION
```

## Required Permissions

Service accounts need the Secret Manager Secret Accessor role (`roles/secretmanager.secretAccessor`) to access secrets.

**IAM Setup**:
```bash
gcloud secrets add-iam-policy-binding SECRET_NAME \
  --member="serviceAccount:SERVICE_ACCOUNT@PROJECT.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

## Key Limitations

- Regional secrets are not supported
- Mounting at `/dev`, `/proc`, `/sys` directories is prohibited
- Multiple secrets cannot share the same mount path
- Mounting on existing directories overwrites their contents - use new subdirectories like `/etc/app_data/secrets` instead

## Best Practices

1. **Use volume mounts for rotation**: Volume-mounted secrets automatically update when rotated
2. **Pin versions for environment variables**: Ensures consistent behavior across instance restarts
3. **Separate mount paths**: Use unique subdirectories for each secret to avoid conflicts
4. **Verify permissions during deployment**: Cloud Run verifies service account access at deployment time
5. **Monitor secret access**: Use Cloud Audit Logs to track secret access patterns
