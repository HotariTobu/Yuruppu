# ko apply Command

## Overview

The `ko apply` command builds Go import path references into container images, publishes them, and applies the resulting YAML through kubectl.

## Synopsis

```bash
ko apply -f FILENAME [flags]
```

## Description

This command finds import path references within the provided files, builds them into Go binaries, containerizes them, publishes them, and then feeds the resulting yaml into `kubectl apply`.

## Usage Examples

### Standard deployment

```bash
ko apply -f config/
```

Builds and publishes images to a Docker registry as `${KO_DOCKER_REPO}/<package name>-<hash>`.

### Preserve import paths

```bash
ko apply --preserve-import-paths -f config/
```

Images are published as `${KO_DOCKER_REPO}/<import path>`.

### Local Docker daemon

```bash
ko apply --local -f config/
```

Loads images as `ko.local/<import path>`.

### Stdin input

```bash
cat config.yaml | ko apply -f -
```

### Pass flags to kubectl

```bash
ko apply -f config -- --namespace=foo --kubeconfig=cfg.yaml
```

## Key Flags

| Flag | Description |
|------|-------------|
| `-f, --filename` | Input file, directory, or URL |
| `-L, --local` | Load to local Docker daemon |
| `-P, --preserve-import-paths` | Keep full import path structure |
| `-t, --tags` | Custom image tags (default: latest) |
| `--push` | Enable pushing to registry (default: true) |
| `-j, --jobs` | Concurrent build limit |
| `-R, --recursive` | Process directory recursively |
| `--platform string` | Specify target platforms |

## Related Commands

- [ko build](cli-build.md) - Build container images only
- [ko resolve](#ko-resolve) - Build and resolve YAML without applying
- [ko delete](#ko-delete) - Delete Kubernetes resources

## ko resolve

The `ko resolve` command builds images and resolves `ko://` references in YAML without applying to Kubernetes:

```bash
ko resolve -f config/ > release.yaml
```

This is useful for:
- Generating deployment YAML for version control
- Reviewing changes before applying
- Using with other tools like ArgoCD

## ko delete

Remove deployed resources:

```bash
ko delete -f config/
```

This provides a convenient wrapper around `kubectl delete` without performing builds or image cleanup.
