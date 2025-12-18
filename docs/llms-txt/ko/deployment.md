# Deployment

## Overview

Because the output of `ko build` is an image reference, you can easily pass it to other tools that expect to take an image reference.

## Docker Run

Execute containers locally using:

```bash
docker run -p 8080:8080 $(ko build ./cmd/app)
```

## Google Cloud Run

Deploy to Google Cloud with:

```bash
gcloud run deploy --image=$(ko build ./cmd/app)
```

**Requirement:** Images must be pushed to Google Container Registry or Artifact Registry.

## Fly.io

Launch applications via:

```bash
flyctl launch --image=$(ko build ./cmd/app)
```

**Requirements:** Images should be pushed to `registry.fly.io`, publicly available elsewhere, or require prior authentication using `flyctl auth docker`.

## AWS Lambda

Update Lambda functions with:

```bash
aws lambda update-function-code \
  --function-name=my-function-name \
  --image-uri=$(ko build ./cmd/app)
```

**Requirements:** Images must be pushed to ECR, based on AWS-provided base images, and utilize the `aws-lambda-go` framework.

## Azure Container Apps

Update container apps via:

```bash
az containerapp update \
  --name my-container-app \
  --resource-group my-resource-group \
  --image $(ko build ./cmd/app)
```

**Requirement:** Images must be pushed to ACR or alternative registry services.

## Kubernetes

For Kubernetes deployments, consult the [Kubernetes Integration](kubernetes.md) documentation.
