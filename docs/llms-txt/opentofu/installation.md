# Installing OpenTofu

OpenTofu offers multiple installation methods across various operating systems. Select your platform and preferred installation method.

## Supported Platforms

OpenTofu supports 9 installation methods:

1. **Alpine Linux** - Install via .apk package manager
2. **Debian/Ubuntu** - Use .deb package manager for Debian-based distributions
3. **FreeBSD** - Run without formal installation
4. **Fedora** - Native Fedora support
5. **RHEL & Derivatives** - Install via .rpm package manager (includes openSUSE, AlmaLinux)
6. **Ubuntu/Manjaro** - Snap package installation
7. **macOS/Linux** - Homebrew installation method
8. **Windows** - Native Windows support
9. **Container** - Use official OCI container image available on GitHub Container Registry

## Quick Install Methods

### Homebrew (macOS/Linux)

```bash
brew install opentofu
```

### Snap (Ubuntu/Manjaro)

```bash
snap install opentofu --classic
```

### Container

Use the official OCI container image from GitHub Container Registry:

```bash
docker pull ghcr.io/opentofu/opentofu:latest
```

## Standalone Installation

OpenTofu supports a standalone option that enables you to use OpenTofu without installation on Linux, macOS, Windows, or FreeBSD, making it accessible across multiple environments with minimal setup requirements.

1. Download the appropriate binary for your platform
2. Extract the archive
3. Move the binary to a location in your PATH
4. Verify the installation: `tofu version`

## Verification

After installation, verify OpenTofu is available:

```bash
tofu version
```

You should see output indicating the installed version.

## Next Steps

After installing OpenTofu, initialize a working directory with `tofu init` to start managing infrastructure.
