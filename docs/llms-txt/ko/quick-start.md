# Quick Start

## Setup

### Installation

Install ko following the [installation guide](installation.md).

### Authentication

Ko relies on Docker configuration credentials, typically stored in `~/.docker/config.json`. If you can already push images with Docker, you're ready to use ko.

The tool offers `ko login` for registry authentication without requiring Docker, supporting:

- Google Container Registry/Artifact Registry (via Application Default Credentials)
- Amazon ECR (AWS credentials)
- Azure Container Registry (environment variables)
- GitHub Container Registry (GITHUB_TOKEN)

### Configure Destination

Set the `KO_DOCKER_REPO` environment variable to specify where built images should be pushed:

```bash
export KO_DOCKER_REPO=gcr.io/my-project
export KO_DOCKER_REPO=ghcr.io/my-org/my-repo
export KO_DOCKER_REPO=my-dockerhub-user
```

## Build an Image

Execute `ko build ./cmd/app` to build and push a container image. The command requires `./cmd/app` to be a Go `package main` with a `func main()` definition.

```bash
ko build ./cmd/app
```

The compiled binary becomes available at `/ko-app/app` within the image and serves as the entrypoint.

Note: versions before v0.10 used `ko publish`; both commands remain functionally equivalent.

## Next Steps

- Learn about [configuration options](configuration.md)
- Explore [deployment targets](deployment.md)
- Integrate with [Kubernetes](kubernetes.md)
