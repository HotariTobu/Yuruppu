# Deploying Container Images

> Comprehensive guide to deploying containerized applications to Cloud Run using various methods and registries.

## Deployment Methods

### Console
Navigate to Cloud Run and select "Deploy container," then specify a container image URL and configure service settings before clicking "Create."

### gcloud CLI
```bash
gcloud run deploy SERVICE --image IMAGE_URL
```

Optional flags:
- `--allow-unauthenticated`: Enable public access
- `--region`: Specify deployment region
- `--platform managed`: Use fully managed Cloud Run

### Infrastructure as Code
- YAML service definitions
- Terraform configurations
- Docker Compose specifications

### REST API
Direct HTTP requests to the Cloud Run Admin API's service endpoint support programmatic deployment.

## Supported Container Registries

The platform accepts images from:

- **Artifact Registry** (recommended)
- **Docker Hub** (with up to one-hour caching)
- **Remote repositories** through Artifact Registry setup
- Other public/private registries via Artifact Registry remote repositories

## Image Requirements

Container images must comply with Cloud Run's runtime contract. Key constraints:

- Container image layers larger than 9.9 GB are not supported when deploying from Docker Hub or an Artifact Registry remote repository
- Images can be specified using tags (e.g., `image:latest`) or exact digests
- Revisions resolve tags to immutable digests upon deployment
- Must be 64-bit Linux compatible

## Service Configuration

Essential settings during deployment:

### Basic Settings
- Service name (49 characters maximum, unique per region)
- Region selection
- Authentication (public or restricted)

### Resource Allocation
- Memory limits (128 MiB to 32 GiB)
- CPU (0.08 to 8 vCPU)
- Concurrency limits (1 to 1000 concurrent requests per instance)

### Advanced Configuration
- Environment variables and secrets
- VPC networking and Cloud SQL connections
- Execution environment selection (first or second generation)
- Volume mounts
- Service accounts and IAM permissions

## Deployment from Source

Deploy directly from source code without a pre-built container:

```bash
gcloud run deploy --source .
```

This automatically builds a container image and deploys it. Requires a buildable project structure (e.g., Go with `go.mod`).

## Multi-Container Deployments

Services can include up to 10 containers total - one ingress container handling HTTPS requests and up to nine sidecars communicating via localhost ports for monitoring, proxying, or authentication purposes.
