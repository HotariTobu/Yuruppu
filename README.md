# Yuruppu

A LINE bot that responds as the character "Yuruppu". Written in Go.

## Prerequisites

- Go 1.23+
- [ko](https://ko.build/) for container image builds
- [OpenTofu](https://opentofu.org/) (or Terraform) for infrastructure
- GCP project with billing enabled
- LINE Messaging API channel

## Local Development

```bash
# Run tests
make test

# Run preflight check (fmt, vet, test)
make preflight

# Run locally
export LINE_CHANNEL_SECRET="your-channel-secret"
export LINE_CHANNEL_ACCESS_TOKEN="your-channel-access-token"
go run .
```

## Deployment

### 1. Set Up GCP Infrastructure

```bash
cd infra

# Copy and configure variables
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values:
#   project_id   = "your-gcp-project-id"
#   region       = "asia-northeast1"
#   github_owner = "your-github-username"
#   github_repo  = "yuruppu"

# Create state bucket (one-time setup)
gcloud storage buckets create gs://yuruppu-tfstate \
  --project=YOUR_PROJECT_ID \
  --location=asia-northeast1

# Initialize and apply
tofu init
tofu plan
tofu apply
```

### 2. Configure LINE Secrets

After infrastructure is created, add LINE credentials to Secret Manager:

```bash
# Add LINE channel secret
echo -n "your-channel-secret" | \
  gcloud secrets versions add LINE_CHANNEL_SECRET --data-file=-

# Add LINE channel access token
echo -n "your-channel-access-token" | \
  gcloud secrets versions add LINE_CHANNEL_ACCESS_TOKEN --data-file=-
```

### 3. Manual Deployment

For manual deployment (without CI/CD):

```bash
# Install ko
go install github.com/google/ko@latest

# Set the container registry
export KO_DOCKER_REPO=asia-northeast1-docker.pkg.dev/YOUR_PROJECT_ID/yuruppu

# Build and push image
ko build . --bare -t latest

# Deploy to Cloud Run
gcloud run deploy yuruppu \
  --image=asia-northeast1-docker.pkg.dev/YOUR_PROJECT_ID/yuruppu:latest \
  --region=asia-northeast1 \
  --platform=managed \
  --update-secrets=LINE_CHANNEL_SECRET=LINE_CHANNEL_SECRET:latest \
  --update-secrets=LINE_CHANNEL_ACCESS_TOKEN=LINE_CHANNEL_ACCESS_TOKEN:latest
```

### 4. Configure LINE Webhook

1. Get the Cloud Run URL:
   ```bash
   gcloud run services describe yuruppu --region=asia-northeast1 --format='value(status.url)'
   ```

2. In [LINE Developers Console](https://developers.line.biz/):
   - Go to your channel settings
   - Set Webhook URL to: `https://YOUR_CLOUD_RUN_URL/webhook`
   - Enable "Use webhook"
   - Verify the webhook connection

### Automatic Deployment (CI/CD)

After initial setup, pushing to the `main` branch triggers automatic deployment via Cloud Build.

## Project Structure

```
.
├── main.go              # Application entry point
├── internal/
│   └── bot/             # Bot core logic
├── infra/               # OpenTofu/Terraform configuration
├── cloudbuild.yaml      # CI/CD pipeline
├── .ko.yaml             # ko build configuration
└── docs/
    ├── adr/             # Architecture Decision Records
    └── specs/           # Feature specifications
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LINE_CHANNEL_SECRET` | Yes | - | LINE channel secret for signature verification |
| `LINE_CHANNEL_ACCESS_TOKEN` | Yes | - | LINE channel access token for API calls |
| `PORT` | No | `8080` | HTTP server port |

## License

Private
