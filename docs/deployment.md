# Deployment Guide

## Initial Setup (One-time)

### 1. Create State Bucket

```bash
gcloud storage buckets create gs://yuruppu-tfstate \
  --project=YOUR_PROJECT_ID \
  --location=asia-northeast1 \
  --enable-autoclass
```

### 2. Connect GitHub to Cloud Build

Connect your repository in [Cloud Build Repositories](https://console.cloud.google.com/cloud-build/repositories/2nd-gen).

### 3. Configure Variables

In `infra/`:

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` with your values.

### 4. Create Secret Manager Secrets

In `infra/`:

```bash
tofu init
tofu apply -target=google_secret_manager_secret.secrets
```

### 5. Configure LINE Secrets

Add secret values in [Secret Manager](https://console.cloud.google.com/security/secret-manager) for each secret.

### 6. Provision Remaining Infrastructure

In `infra/`:

```bash
tofu apply
```

### 7. Configure LINE Webhook

1. Get the Cloud Run URL:
   ```bash
   tofu output webhook_url
   ```

2. In [LINE Developers Console](https://developers.line.biz/):
   - Set Webhook URL to the output value
   - Enable "Use webhook"

## Regular Deployment

Push to `main` branch to trigger automatic deployment via Cloud Build.

For manual deployment, see [manual-deployment.md](manual-deployment.md).
