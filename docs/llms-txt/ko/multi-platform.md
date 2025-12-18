# Multi-Platform Images

## Overview

The `ko` tool leverages Go's cross-compilation capabilities to efficiently create multi-platform container images.

## Building Multi-Platform Images

### All Supported Platforms

To build across every platform your base image supports, use the `--platform=all` flag. This instructs `ko` to:

- Identify all supported platforms in the base image
- Execute `GOOS=<os> GOARCH=<arch> GOARM=<variant> go build` for each platform
- Generate a manifest list containing images for all platforms

Example:

```bash
ko build --platform=all ./cmd/app
```

### Specific Platforms

For targeted builds, specify platforms explicitly:

```bash
ko build --platform=linux/amd64,linux/arm64 ./cmd/app
```

## Windows Support

ko also has experimental support for building Windows images. See the [FAQ](faq.md) for additional details regarding Windows container builds.

## Key Benefit

Go's native cross-compilation support makes `ko` particularly well-suited for producing multi-architecture container images without requiring platform-specific tooling or emulation for each target architecture.
