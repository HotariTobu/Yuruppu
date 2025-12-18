# Kubernetes Integration

## Overview

The `ko` tool streamlines Kubernetes deployments by integrating Go binary building with YAML generation, eliminating manual image specification steps.

## YAML Changes

Traditional Kubernetes deployments require explicit image references:

```yaml
image: registry.example.com/my-app:v1.2.3
```

With `ko`, reference Go binaries using the `ko://` prefix followed by their import path:

```yaml
image: ko://github.com/my-user/my-repo/cmd/app
```

## ko resolve

The `ko resolve` command automates image handling:

1. Scans YAML files for `ko://`-prefixed values
2. Builds and pushes images for each unique reference
3. Replaces prefixed strings with fully-specified image references
4. Outputs resolved YAML to stdout

Example usage:

```bash
ko resolve -f config/ > release.yaml
```

This approach makes container image management transparent within your deployment workflow.

## ko apply

Combine resolution with Kubernetes application in one step:

```bash
ko apply -f config/
```

Pass additional kubectl flags after `--`:

```bash
ko apply -f config/ -- --context=foo --kubeconfig=cfg.yaml
```

## ko delete

Remove deployed resources using:

```bash
ko delete -f config/
```

This provides a convenient wrapper around `kubectl delete` without performing builds or image cleanup.

## Integration with Kustomize

ko works with Kustomize:

```bash
kustomize build config | ko resolve -f -
```

## Additional Integrations

ko integrates with various build tools including Skaffold, goreleaser, Tekton catalog, Carvel's kbld, and Tilt extension.
