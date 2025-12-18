# Google Cloud Run

> Google Cloud Run is a fully managed serverless platform for deploying and scaling containerized applications. It automatically handles infrastructure management, scales from zero to many instances based on traffic, and charges only for resources used during request processing.

Cloud Run is ideal for stateless HTTP services, APIs, webhooks, and event-driven applications. It supports any language or framework that can be containerized and listens on a configurable port (default 8080). The platform provides automatic HTTPS, traffic splitting, gradual rollouts, and integration with Google Cloud services.

**Key features**: Sub-second scale-up, built-in traffic management, VPC networking support, secret management, request/instance-based billing, and support for up to 32 GiB memory and 8 vCPU per container.

## Getting Started

- [Go Quickstart](go-quickstart.md): Build and deploy Go applications with code examples and best practices

## Core Concepts

- [Container Runtime Contract](container-contract.md): Port configuration, environment variables, startup requirements, and security constraints
- [Deploying Container Images](deploying.md): Deployment methods, supported registries, image requirements, and service configuration
- [Environment Variables](environment-variables.md): Setting and managing environment variables for service configuration
- [Managing Secrets](secrets.md): Secure handling of sensitive information using Secret Manager integration

## Observability

- [Logging and Monitoring](logging.md): Writing structured logs, viewing logs, log correlation, and severity levels
- [Troubleshooting and Debugging](troubleshooting.md): Common issues, diagnostic techniques, and solutions for deployment and runtime errors

## Configuration

- [Health Checks](health-checks.md): Configure startup, liveness, and readiness probes for reliable service operation
- [Best Practices](best-practices.md): Performance optimization, cost management, security recommendations, and operational best practices

## Optional

Additional resources available in the official documentation:

- **IAM and Authentication**: Configure service-to-service authentication and user access control
- **VPC Networking**: Connect to private VPC resources and Cloud SQL databases
- **Traffic Management**: Split traffic between revisions for gradual rollouts and A/B testing
- **Cloud Run Jobs**: Run batch workloads and scheduled tasks
- **Custom Domains**: Map custom domains with managed SSL certificates
- **Cloud Code Integration**: Deploy and debug directly from IDEs (VS Code, IntelliJ)
- **CI/CD Integration**: Automate deployments with Cloud Build, GitHub Actions, and GitLab CI
- **Pricing Details**: Understand billing for CPU, memory, requests, and networking at https://cloud.google.com/run/pricing
- **Quotas and Limits**: Review service limits and request quota increases at https://cloud.google.com/run/quotas
- **Release Notes**: Stay updated on new features and changes at https://cloud.google.com/run/docs/release-notes

For the complete official documentation, visit https://cloud.google.com/run/docs
