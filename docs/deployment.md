# Deployment Guide

## Prerequisites

- [OpenTofu](https://opentofu.org/) (or Terraform)
- GCP project with billing enabled
- LINE Messaging API channel

## Initial Setup (One-time)

### 1. Create State Bucket

```bash
gcloud storage buckets create gs://yuruppu-tfstate \
  --project=YOUR_PROJECT_ID \
  --location=asia-northeast1
```

### 2. Configure Variables

In `infra/`:

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` with your values.

### 3. Connect GitHub to Cloud Build

1. Go to [Cloud Build - Repositories](https://console.cloud.google.com/cloud-build/repositories)
2. Click "Link a repository"
3. Select "GitHub" and authenticate
4. Select your repository and connect

### 4. Provision Infrastructure

In `infra/`:

```bash
tofu init
tofu plan
tofu apply
```

### 5. Configure LINE Secrets

```bash
echo -n "your-channel-secret" | \
  gcloud secrets versions add LINE_CHANNEL_SECRET --data-file=-

echo -n "your-channel-access-token" | \
  gcloud secrets versions add LINE_CHANNEL_ACCESS_TOKEN --data-file=-
```

### 6. Configure LINE Webhook

1. Get the Cloud Run URL:
   ```bash
   gcloud run services describe yuruppu --region=asia-northeast1 --format='value(status.url)'
   ```

2. In [LINE Developers Console](https://developers.line.biz/):
   - Set Webhook URL to: `https://YOUR_CLOUD_RUN_URL/webhook`
   - Enable "Use webhook"

## Regular Deployment

Push to `main` branch to trigger automatic deployment via Cloud Build.

For manual deployment, see [manual-deployment.md](manual-deployment.md).
