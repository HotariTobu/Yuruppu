# Refactor: Simplify ko Configuration

## Overview

Remove `defaultBaseImage` from `.ko.yaml` and use a more appropriate build image for Cloud Build.

## Background & Purpose

- `defaultBaseImage` in `.ko.yaml` is unnecessary because ko has a suitable default
- Current Cloud Build installs ko every build, which is inefficient
- Simplify configuration by removing unnecessary settings

## Current Structure

- `.ko.yaml`: Contains `defaultBaseImage` setting that duplicates ko's default behavior
- `cloudbuild.yaml`: Uses `golang:1` and installs ko on every build

## Proposed Structure

- `.ko.yaml`: Remove `defaultBaseImage`, keep only `builds` section
- `cloudbuild.yaml`: Use a build image with ko pre-installed

## Scope

- [ ] SC-001: Remove `defaultBaseImage` from `.ko.yaml`
- [ ] SC-002: Update Cloud Build step to use appropriate image

## Breaking Changes

None.

## Acceptance Criteria

### AC-001: defaultBaseImage Removed [Linked to SC-001]

- **Given**: `.ko.yaml` exists with `defaultBaseImage` setting
- **When**: Refactoring is complete
- **Then**: `.ko.yaml` no longer contains `defaultBaseImage`

### AC-002: Cloud Build Uses Pre-installed ko [Linked to SC-002]

- **Given**: `cloudbuild.yaml` installs ko on every build
- **When**: Refactoring is complete
- **Then**: Build uses an image with ko pre-installed

### AC-003: Build Succeeds

- **Given**: Refactoring is complete
- **When**: Cloud Build is triggered
- **Then**: Container image is built and pushed successfully

## Implementation Notes

- Specific image choice to be decided in design phase

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2024-12-24 | 1.0 | Initial version | - |
