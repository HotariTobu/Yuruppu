# Frequently Asked Questions

## How can I set ldflags?

Use the `GOFLAGS` environment variable instead of direct flags:

```bash
GOFLAGS="-ldflags=-X=main.version=1.2.3" ko build .
```

Note: Multiple ldflags arguments via `GOFLAGS` have limitations. The `.ko.yaml` configuration file's `builds` section works better for complex scenarios.

Example in `.ko.yaml`:

```yaml
builds:
- id: app
  main: ./cmd/app
  ldflags:
  - -s -w
  - -X main.version={{.Env.VERSION}}
  - -X main.commit={{.Git.ShortCommit}}
```

## Why are my images all created in 1970?

In order to support reproducible builds, ko doesn't embed timestamps in the images it produces by default.

Set timestamps using environment variables:

- `SOURCE_DATE_EPOCH=$(date +%s)` for current timestamp
- `KO_DATA_DATE_EPOCH=$(git log -1 --format='%ct')` for latest git commit time

Example:

```bash
SOURCE_DATE_EPOCH=$(date +%s) ko build ./cmd/app
```

## Can I build Windows containers?

Yes, but it's experimental. Update `.ko.yaml` with a Windows base image:

```yaml
defaultBaseImage: mcr.microsoft.com/windows/nanoserver:ltsc2022
```

Then build:

```bash
ko build ./ --platform=windows/amd64
```

**Known issues**: Symlinks in `kodata` are ignored; only regular files and directories are included.

## Does ko support autocompletion?

Yes. Generate completion scripts with:

```bash
ko completion [bash|zsh|fish|powershell] --help
```

Or source directly:

```bash
source <(ko completion bash)
```

Add to your shell profile for persistent autocompletion.

## Does ko work with Kustomize?

Yes:

```bash
kustomize build config | ko resolve -f -
```

This allows you to use kustomize for YAML templating and ko for building and resolving container images.

## Does ko integrate with other build tools?

Yes, ko integrates with various build and deployment tools:

- **Skaffold** - For local Kubernetes development
- **goreleaser** - For release automation
- **Tekton catalog** - For CI/CD pipelines
- **Carvel's kbld** - For Kubernetes image management
- **Tilt** - Via Tilt extension for local development

## Does ko work with OpenShift Internal Registry?

Yes. Steps include:

1. Connect to OpenShift cluster
2. Expose the internal registry
3. Export credentials to `$HOME/.docker/config.json`
4. Set `KO_DOCKER_REPO` to the registry URL

Example:

```bash
oc login
oc registry login --insecure=true
export KO_DOCKER_REPO=$(oc registry info)/my-namespace
ko build ./cmd/app
```

## Can I use ko with private Go modules?

Yes, set the `GOPRIVATE` environment variable:

```bash
export GOPRIVATE=github.com/myorg/*
ko build ./cmd/app
```

Or configure it in `.ko.yaml`:

```yaml
builds:
- id: app
  main: ./cmd/app
  env:
  - GOPRIVATE=github.com/myorg/*
```

## How do I use a custom base image?

Set it in `.ko.yaml`:

```yaml
defaultBaseImage: gcr.io/distroless/base-debian11
```

Or per-package:

```yaml
baseImageOverrides:
  github.com/myorg/myapp/cmd/app: ubuntu:22.04
```

Or via environment variable:

```bash
KO_DEFAULTBASEIMAGE=ubuntu:22.04 ko build ./cmd/app
```
