# Installation

## GitHub Releases

Download from [GitHub Releases](https://github.com/ko-build/ko/releases). Available for Linux/Darwin on x86_64/arm64/i386/s390x architectures.

The project generates SLSA3 provenance for verification. After downloading the tarball:

```bash
tar xzf ko.tar.gz ko
chmod +x ./ko
```

## Package Managers

### Homebrew

```bash
brew install ko
```

### MacPorts

```bash
sudo port install ko
```

See [MacPorts details](https://ports.macports.org/port/ko/)

### Windows (Scoop)

```bash
scoop install ko
```

### Alpine Linux

Enable the testing repository first:

```bash
echo https://dl-cdn.alpinelinux.org/alpine/edge/testing/ >> /etc/apk/repositories
apk update
apk add ko
```

## Build from Source

Requires Go 1.16 or newer:

```bash
go install github.com/google/ko@latest
```

## GitHub Actions

Integrate ko into your workflow using the official action:

```yaml
steps:
- uses: ko-build/setup-ko@v0.6
```

This installs ko and configures authentication to GitHub Container Registry.
