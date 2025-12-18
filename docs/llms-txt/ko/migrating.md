# Migrating from Dockerfile

## Overview

The `ko` tool provides a streamlined alternative to traditional Docker-based builds for Go applications. This guide explains how to transition from a multi-stage Dockerfile approach to using `ko`.

## Traditional Dockerfile Approach

A typical multi-stage Go Dockerfile follows this pattern:

1. **Build stage**: Uses a `golang` base image, copies source files, downloads dependencies, and compiles the application
2. **Deploy stage**: Copies the compiled binary into a minimal distroless base image with appropriate metadata

Example Dockerfile:

```dockerfile
# Build stage
FROM golang:1.21 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o app ./cmd/app

# Deploy stage
FROM gcr.io/distroless/static-debian11
COPY --from=builder /app/app /
ENTRYPOINT ["/app"]
```

This approach requires running `docker build` followed by pushing the image to a registry for deployment.

## Migration to ko

### Simplification Benefits

Instead of maintaining a Dockerfile, developers can simply execute `ko build ./` to build and push container images to a registry:

```bash
ko build ./cmd/app
```

You can then delete your `Dockerfile` and uninstall `docker`.

### Key Advantages

- **Automatic optimization**: `ko` leverages local Go build caches without explicit configuration
- **Sensible defaults**: Automatically sets `ENTRYPOINT` and uses a nonroot distroless base image
- **Multi-architecture support**: Adding `--platform=all` creates multi-arch images without additional complexity

Example multi-platform build:

```bash
ko build --platform=all ./cmd/app
```

### Requirements

The approach assumes:

- Go source follows standard project layout conventions
- `ko` is installed and properly configured
- Environment setup is complete (see [Quick Start](quick-start.md))

## Comparison

| Feature | Dockerfile | ko |
|---------|-----------|-----|
| Docker required | Yes | No |
| Build caching | Manual configuration | Automatic |
| Multi-arch builds | Complex | `--platform=all` |
| Base image | Manual specification | Sensible defaults |
| SBOM generation | Manual | Automatic |
| Reproducible builds | Manual configuration | Built-in |

## When to Use Dockerfile

Consider keeping a Dockerfile if your application:

- Requires C bindings (cgo)
- Needs OS package dependencies
- Uses complex base image configurations
- Includes non-Go components

For pure Go applications, `ko` provides a simpler, faster, and more secure alternative.
