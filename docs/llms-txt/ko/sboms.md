# SBOMs (Software Bill of Materials)

## Overview

An SBOM (Software Bill of Materials) is a comprehensive list of software components and dependencies used to build a software artifact. This information is valuable for identifying potentially vulnerable components in your software.

## Key Features

### Default Generation

From v0.9+, `ko` generates and uploads an SBOM for every image it produces by default.

### Format

ko generates SBOMs using the SPDX format as its default standard.

### Disabling SBOMs

To prevent SBOM generation, users can pass the `--sbom=none` flag when running ko commands:

```bash
ko build --sbom=none ./cmd/app
```

## Retrieving SBOMs

Generated SBOMs can be downloaded using the cosign tool:

```bash
cosign download sbom <image-reference>
```

See the [cosign download sbom](https://github.com/sigstore/cosign/blob/main/doc/cosign_download_sbom.md) command documentation for more details.
