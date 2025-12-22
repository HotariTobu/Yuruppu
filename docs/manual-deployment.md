# Manual Deployment

## Prerequisites

- [ko](https://ko.build/)
- Infrastructure provisioned (see [deployment.md](deployment.md))

## Steps

```bash
# Set the container registry
export KO_DOCKER_REPO=asia-northeast1-docker.pkg.dev/YOUR_PROJECT_ID/yuruppu

# Build and push image
ko build . --bare -t latest

# Deploy to Cloud Run
gcloud run deploy yuruppu \
  --image=asia-northeast1-docker.pkg.dev/YOUR_PROJECT_ID/yuruppu:latest \
  --region=asia-northeast1 \
  --update-secrets=LINE_CHANNEL_SECRET=LINE_CHANNEL_SECRET:latest \
  --update-secrets=LINE_CHANNEL_ACCESS_TOKEN=LINE_CHANNEL_ACCESS_TOKEN:latest
```
