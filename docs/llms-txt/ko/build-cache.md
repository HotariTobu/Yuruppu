# Build Cache

## Overview

The `ko` build tool leverages Go's native caching mechanisms to accelerate container builds during development workflows.

## Go Build Cache Integration

`ko` automatically utilizes the standard `go build` cache from your development environment. This means that repeated builds benefit from cached compilation results, significantly speeding up iterative development cycles without requiring additional configuration.

## Registry Blob Optimization

Beyond local compilation caching, `ko` implements intelligent blob management. When pushing images to a remote registry, it avoids re-uploading layers that already exist in the target registry, reducing push times and bandwidth consumption.

## Enhanced Performance with KOCACHE

For maximum build performance, developers can set the `KOCACHE` environment variable:

```bash
export KOCACHE=true
ko build ./cmd/app
```

This feature enables `ko` to maintain a local index mapping Go build inputs to their corresponding container image layers. When this mapping is enabled, the build system can skip Go compilation entirely if the resulting layer is already available in the image registry, providing substantial time savings for unchanged code.
