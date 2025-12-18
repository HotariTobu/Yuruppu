# GCS Backend Configuration

The GCS backend stores OpenTofu state as an object in Google Cloud Storage. This is the recommended backend for managing GCP infrastructure with OpenTofu.

## Prerequisites

The bucket must exist prior to configuring the backend. Create it manually or through another OpenTofu configuration.

## Basic Configuration

```hcl
terraform {
  backend "gcs" {
    bucket  = "tf-state-prod"
    prefix  = "tofu/state"
  }
}
```

## Configuration Variables

### Required

- `bucket` - The globally unique GCS bucket name

### Important Optional Variables

- `prefix` - GCS folder path within the bucket for organizing state files (e.g., "tofu/state", "environments/production")
- `credentials` - Path to a service account JSON file (uses Application Default Credentials if omitted)
- `encryption_key` - For customer-supplied encryption
- `kms_encryption_key` - For Cloud KMS managed encryption (recommended)

## Authentication Methods

### 1. Local Workstation

Use User Application Default Credentials:

```bash
gcloud auth application-default login
```

### 2. Google Cloud Instances

Configure a service account with appropriate scopes on the instance. The backend will automatically use the instance's service account credentials.

### 3. External Environments (CI/CD)

Set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable pointing to a service account key file:

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
```

## Example with KMS Encryption

```hcl
terraform {
  backend "gcs" {
    bucket              = "my-tofu-state"
    prefix              = "prod/cloudrun"
    kms_encryption_key  = "projects/my-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key"
  }
}
```

## Best Practices

### Enable Object Versioning

Enable Object Versioning on your bucket for disaster recovery and rollback capabilities:

```bash
gcloud storage buckets update gs://my-tofu-state --versioning
```

### Use Environment Variables for Credentials

Avoid hardcoding credentials in your configuration. Use environment variables instead:

```bash
export GOOGLE_CREDENTIALS=$(cat /path/to/key.json)
```

### State Locking

The GCS backend supports state locking for concurrent operation safety. OpenTofu automatically creates and manages locks during operations to prevent conflicts when multiple users or processes attempt to modify state simultaneously.

### Organize State Files by Environment

Use the `prefix` parameter to organize state files for different environments:

```hcl
# Production
terraform {
  backend "gcs" {
    bucket = "company-tofu-state"
    prefix = "production/cloudrun"
  }
}

# Staging
terraform {
  backend "gcs" {
    bucket = "company-tofu-state"
    prefix = "staging/cloudrun"
  }
}
```

## IAM Permissions Required

The service account or user accessing the backend needs:

- `storage.objects.create`
- `storage.objects.delete`
- `storage.objects.get`
- `storage.objects.list`
- `storage.objects.update`

Assign the `roles/storage.objectUser` role or a custom role with these permissions.

## Security Considerations

- If you use customer-supplied encryption keys, you must securely manage your keys
- Use Cloud KMS for encryption key management when possible
- Store service account keys securely and rotate them regularly
- Limit bucket access to only necessary users and service accounts
- Enable audit logging on the state bucket to track access

## Initialization

After configuring the backend, initialize it:

```bash
tofu init
```

If migrating from a local backend, OpenTofu will prompt you to copy existing state to GCS.

## Migrating State

To migrate from another backend to GCS:

```bash
tofu init -migrate-state
```

OpenTofu will prompt for confirmation before copying state.
