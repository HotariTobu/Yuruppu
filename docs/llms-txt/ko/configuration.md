# Configuration

## Overview

Ko uses a `.ko.yaml` configuration file to manage build behavior. The file location can be overridden using `KO_CONFIG_PATH`.

## Base Images

Ko defaults to `cgr.dev/chainguard/static` as the base image.

### Global Base Image

Add to `.ko.yaml`:

```yaml
defaultBaseImage: registry.example.com/base/image
```

Or use the environment variable:

```bash
export KO_DEFAULTBASEIMAGE=registry.example.com/base/image
ko build .
```

### Per Import Path

Specify overrides for individual packages:

```yaml
baseImageOverrides:
  github.com/my-user/my-repo/cmd/app: registry.example.com/base/for/app
```

## Go Build Settings

Configure build flags and linker flags using a GoReleaser-style `builds` section:

```yaml
builds:
- id: foo
  dir: .
  main: ./foobar/foo
  env:
  - GOPRIVATE=git.internal.example.com
  flags:
  - -tags
  - netgo
  ldflags:
  - -s -w
  - -X main.version={{.Env.VERSION}}
```

The `dir` field is essential for multi-module repositories with multiple `go.mod` files.

## Template Parameters

Build configurations support templating for `flags` and `ldflags`:

| Parameter | Purpose |
|-----------|---------|
| `Env` | Environment variables |
| `GoEnv` | Go environment variables |
| `Date` | UTC build date (RFC 3339) |
| `Git.Tag`, `Git.Branch`, `Git.ShortCommit` | Git information |
| `Git.TreeState` | Either `clean` or `dirty` |

## Platform Configuration

Set default build platforms in `.ko.yaml`:

```yaml
defaultPlatforms:
- linux/arm64
- linux/amd64
```

Or via environment variable:

```bash
export KO_DEFAULTPLATFORMS=linux/arm64,linux/amd64
```

## Build Environment Variables

Configure variables globally or per-build:

```yaml
defaultEnv:
- FOO=foo
builds:
- id: foo
  env:
  - FOO=bar  # Overrides defaultEnv
```

Precedence: system environment → `defaultEnv` → per-build `env`

## Build Flags and Ldflags

Set defaults and per-build overrides:

```yaml
defaultFlags:
- -v
defaultLdflags:
- -s
```

## Environment Variables Reference

| Variable | Purpose |
|----------|---------|
| `KO_DOCKER_REPO` | Container registry for pushing images (required) |
| `KO_GO_PATH` | Path to go binary |
| `KO_CONFIG_PATH` | Path to configuration file |
| `KOCACHE` | Enable local build caching |

## Image Naming Strategies

Given `KO_DOCKER_REPO=registry.example.com/repo`:

- **Default**: `registry.example.com/repo/app-<md5>` (includes MD5 hash)
- **`--preserve-import-paths`**: Full import path included
- **`--base-import-paths`**: Omits MD5 hash
- **`--bare`**: Only repository name

## Local Publishing

### Docker daemon

Set `KO_DOCKER_REPO=ko.local` or use `--local` flag

### KinD clusters

Set `KO_DOCKER_REPO=kind.local` (defaults to cluster named `kind`)

Local images can serve as base images for other Ko builds.
