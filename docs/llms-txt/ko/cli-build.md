# ko build Command

## Overview

The `ko build` command constructs container images from Go import paths, containerizes them, and publishes the results.

## Synopsis

```bash
ko build IMPORTPATH... [flags]
```

## Description

This subcommand converts provided import paths into Go binaries, packages them into containers, and distributes them to registries.

## Usage Examples

### Multiple import paths

Build and publish references to a Docker Registry using the pattern `${KO_DOCKER_REPO}/<package name>-<hash of import path>`:

```bash
ko build github.com/foo/bar/cmd/baz github.com/foo/bar/cmd/blah
```

### Relative paths

```bash
ko build ./cmd/blah
```

### Preserve full import paths

```bash
ko build --preserve-import-paths ./cmd/blah
```

### Local Docker daemon

Load images locally as `ko.local/<import path>` and always preserve import paths:

```bash
ko build --local github.com/foo/bar/cmd/baz
```

When `KO_DOCKER_REPO` is set to `ko.local`, behavior mirrors using `--local` with `--preserve-import-paths`.

## Key Options

| Flag | Purpose |
|------|---------|
| `--bare` | Use only `KO_DOCKER_REPO` without additional context |
| `-B, --base-import-paths` | Omit MD5 hash from image naming |
| `--debug` | Include Delve debugger; listens on port 40000 |
| `--disable-optimizations` | Skip Go build optimizations for interactive debugging |
| `--image-label strings` | Add labels to the image |
| `-L, --local` | Load images into local Docker daemon |
| `-P, --preserve-import-paths` | Maintain full import path after registry |
| `--sbom string` | Specify SBOM media type (default: "spdx") |
| `-t, --tags strings` | Custom image tags instead of 'latest' |
| `--push` | Push to registry (enabled by default) |
| `--platform string` | Specify target platforms (e.g., linux/amd64,linux/arm64 or all) |
| `-j, --jobs int` | Number of concurrent builds (default: GOMAXPROCS) |

## Related Commands

- [ko apply](cli-apply.md) - Build and apply to Kubernetes
- [ko resolve](cli-apply.md) - Build and resolve YAML
