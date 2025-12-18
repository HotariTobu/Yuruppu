# ko

> ko is a streamlined container image builder for Go applications that makes building Go container images easy, fast, and secure by default. No Docker required. A Cloud Native Computing Foundation Sandbox project.

ko executes `go build` locally, eliminating the need for Docker installation, while providing multi-platform builds, automatic SBOM generation, and Kubernetes YAML templating capabilities. It's optimal for container images containing single Go applications without C bindings or OS package dependencies. Used by prominent projects including Knative, Tekton, Karpenter, Kyverno, and Sigstore.

## Getting Started

- [Installation](installation.md): Install ko via GitHub releases, package managers, or build from source
- [Quick Start](quick-start.md): Set up authentication, configure destination, and build your first image

## Configuration

- [Configuration](configuration.md): Configure base images, build settings, platforms, and environment variables using .ko.yaml

## Deployment

- [Deployment](deployment.md): Deploy to Docker, Cloud Run, Fly.io, AWS Lambda, and Azure Container Apps
- [Kubernetes Integration](kubernetes.md): Use ko:// references in YAML, ko resolve, ko apply, and ko delete commands

## Features

- [Multi-Platform Builds](multi-platform.md): Build images for multiple architectures using --platform flag
- [Static Assets](static-assets.md): Bundle static files via kodata/ directory convention
- [SBOMs](sboms.md): Automatic SPDX-format Software Bill of Materials generation
- [Build Cache](build-cache.md): Leverage Go build cache and registry blob optimization with KOCACHE

## CLI Reference

- [ko build](cli-build.md): Build and publish container images from Go import paths
- [ko apply](cli-apply.md): Build images and apply resulting YAML via kubectl

## Advanced Topics

- [Migrating from Dockerfile](migrating.md): Transition from multi-stage Dockerfile to ko
- [Debugging](debugging.md): Use --debug flag to include delve debugger in images
- [FAQ](faq.md): Common questions about ldflags, timestamps, Windows support, and integrations

## Optional

- [Community](https://ko.build/community/): Join the ko community and contribute
- [GitHub Repository](https://github.com/ko-build/ko): Source code and issue tracker
