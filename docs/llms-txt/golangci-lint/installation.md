# Installation

golangci-lint can be installed locally on your development machine or in CI/CD environments. Binary installation is the recommended approach.

## Quick Install (Recommended)

Use the install script to download the binary:

```bash
curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.7.2
```

## Package Managers

### macOS

**Homebrew:**
```bash
brew install golangci-lint
```

**MacPorts:**
```bash
sudo port install golangci-lint
```

### Linux

golangci-lint is packaged in most major Linux package managers. Check availability via Repology.

### Windows

**Chocolatey:**
```bash
choco install golangci-lint
```

**Scoop:**
```bash
scoop install main/golangci-lint
```

## Docker

Run golangci-lint in a container:

```bash
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v2.7.2 golangci-lint run
```

## Version Manager (mise)

Install via mise tool version manager:

```bash
mise use -g golangci-lint@v2.7.2
```

## Building from Source (Not Recommended)

Building from source is not guaranteed to work due to compilation variability and dependency issues. However, if needed:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2
```

## CI/CD Installation

For continuous integration environments, run golangci-lint and check the exit code. A non-zero result should fail the build. See the [CI installation documentation](https://golangci-lint.run/docs/welcome/install/ci/) for platform-specific instructions.

## Verification

After installation, verify the installation:

```bash
golangci-lint version
```
