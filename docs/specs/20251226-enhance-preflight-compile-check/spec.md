# Enhancement: Strengthen Preflight Checks

## Overview

Enhance `make preflight` to catch more issues before CI by adding compile checks for all build tags and comprehensive linting with golangci-lint.

## Background & Purpose

Currently, `make preflight` runs `go test ./...` which only compiles and tests files without build tags. Files with `//go:build integration` are completely skipped during compilation, allowing type errors to pass CI undetected.

This was discovered when `NewVertexAIClient` signature changed but `vertexai_integration_test.go` was not updated. The type error was only caught when manually running `make test-integration`.

Additionally, the current `check` target only runs `go fmt` and `go vet`, missing opportunities to catch:
- Unused or missing imports
- Common code issues that staticcheck detects
- Code that can be simplified

## Current Behavior

```makefile
check:
	go fmt ./...
	go vet ./...

preflight: check test
```

- Files with build tags (e.g., `//go:build integration`) are not compiled
- No import organization enforcement
- No static analysis beyond `go vet`

## Proposed Changes

- [ ] CH-001: Add compile check with `-tags=integration` to verify all files compile
- [ ] CH-002: Add golangci-lint with explicit linter list for comprehensive linting
- [ ] CH-003: Add `govulncheck` for known vulnerability detection
- [ ] CH-004: Separate `fix` target for auto-formatting
- [ ] CH-005: Update GitHub Actions workflow to install tools and run new preflight

## Acceptance Criteria

### AC-001: Compile check catches type errors in tagged files [Linked to CH-001]

- **Given**: A file with `//go:build integration` tag containing a type error
- **When**: `make preflight` is executed
- **Then**:
  - The compile check fails with the type error message
  - CI blocks the PR from merging

### AC-002: golangci-lint catches lint issues [Linked to CH-002]

- **Given**: Code with lint issues (formatting, imports, staticcheck violations, etc.)
- **When**: `make preflight` is executed
- **Then**:
  - golangci-lint reports the issues
  - Preflight fails until issues are fixed

### AC-003: govulncheck detects known vulnerabilities [Linked to CH-003]

- **Given**: A dependency with a known vulnerability
- **When**: `make preflight` is executed
- **Then**:
  - govulncheck reports the vulnerability
  - Preflight fails until the dependency is updated

### AC-004: fix target auto-formats code [Linked to CH-004]

- **Given**: Go files with formatting or lint issues
- **When**: `make fix` is executed
- **Then**:
  - Files are auto-formatted by golangci-lint

### AC-005: GitHub Actions runs new preflight [Linked to CH-005]

- **Given**: A PR is opened
- **When**: GitHub Actions workflow runs
- **Then**:
  - golangci-lint and govulncheck are installed
  - `make preflight` passes with all new checks

### AC-006: Backward Compatibility

- **Given**: Existing usage of `make test` and `make test-integration`
- **When**: User runs these commands
- **Then**:
  - Behavior remains unchanged

## Implementation Notes

Expected Makefile after implementation:

```makefile
.PHONY: fix check compile-all test test-integration preflight

fix:
	golangci-lint run --fix ./...

check:
	golangci-lint run ./...
	govulncheck ./...

compile-all:
	go build -tags=integration ./...

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

preflight: check compile-all test
```

Expected `.golangci.yml` (v2 format):

```yaml
version: "2"

formatters:
  enable:
    - gofmt
    - gofumpt
    - goimports

linters:
  default: none
  enable:
    - errcheck
    - govet
    - staticcheck
    # ... 75 linters explicitly listed
```

Tool installation (for local development):
```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-26 | 1.0 | Initial version | - |
| 2025-12-27 | 1.1 | Upgrade to golangci-lint v2 with explicit linter list | - |
