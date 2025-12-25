# Refactor: Centralize Environment Variable Loading

> Consolidate all environment variable reading into a single location.

## Overview

Refactor environment variable loading so that all `os.Getenv` calls occur in one place (`loadConfig` in main.go), and other packages receive values as parameters instead of reading environment variables directly.

## Background & Purpose

Currently, environment variables are read in multiple locations:
- `main.go`: LINE credentials, GCP project ID, LLM timeout, PORT
- `internal/llm/vertexai.go`: GCP_REGION (as fallback in `GetRegion`)

This scattered approach makes it difficult to:
- Understand all configuration options at a glance
- Test components with different configurations
- Ensure consistent handling (trimming, validation, defaults)

## Current Structure

- `main.go`: Contains `loadConfig()` that reads most environment variables and returns a `Config` struct
- `main.go`: Contains `getPort()` that reads `PORT` environment variable separately
- `internal/llm/vertexai.go`: Contains `GetRegion()` that reads `GCP_REGION` as a fallback, and defines `defaultRegion` constant

## Proposed Structure

- `main.go`: Consolidate all environment variable reading into `loadConfig()`, including `PORT` and `GCP_REGION`
- `main.go`: Define all default values as constants (including `defaultPort` and `defaultRegion`)
- `internal/llm/vertexai.go`: Remove `os.Getenv("GCP_REGION")` call from `GetRegion()`, accept region as a parameter instead
- `internal/llm/vertexai.go`: Remove `defaultRegion` constant (moved to main.go)

## Scope

- [ ] SC-001: Move PORT reading into loadConfig and add to Config struct
- [ ] SC-002: Move GCP_REGION reading into loadConfig and add to Config struct
- [ ] SC-003: Update NewVertexAIClient to accept region as parameter
- [ ] SC-004: Update GetRegion to accept fallback region as parameter (remove os.Getenv call)
- [ ] SC-005: Move defaultRegion constant to main.go
- [ ] SC-006: Remove getPort function (replaced by Config.Port)

## Breaking Changes

None - all changes are internal implementation details.

## Acceptance Criteria

### AC-001: PORT loaded via loadConfig [Linked to SC-001, SC-006]

- **Given**: Application starts
- **When**: `loadConfig()` is called
- **Then**:
  - `Config` struct contains `Port` field
  - PORT environment variable is read and trimmed
  - If empty after trimming, defaults to "8080"
  - `getPort()` function no longer exists
  - All existing tests pass

### AC-002: GCP_REGION loaded via loadConfig [Linked to SC-002]

- **Given**: Application starts
- **When**: `loadConfig()` is called
- **Then**:
  - `Config` struct contains `GCPRegion` field
  - GCP_REGION environment variable is read and trimmed
  - If empty after trimming, defaults to "us-central1"
  - All existing tests pass

### AC-003: NewVertexAIClient accepts fallback region parameter [Linked to SC-003]

- **Given**: Refactoring is complete
- **When**: `NewVertexAIClient` is called with a fallback region
- **Then**:
  - Uses metadata server region if available
  - Falls back to the provided region if metadata server is unavailable
  - All existing tests pass

### AC-004: GetRegion no longer reads environment variables [Linked to SC-004]

- **Given**: Refactoring is complete
- **When**: `GetRegion` is called
- **Then**:
  - No `os.Getenv` calls exist in the function
  - Returns metadata server region if available, otherwise returns provided fallback
  - All existing tests pass

### AC-005: Default constants centralized in main.go [Linked to SC-005]

- **Given**: Refactoring is complete
- **When**: Code is reviewed
- **Then**:
  - `defaultRegion` constant exists in main.go
  - `defaultPort` constant exists in main.go
  - `defaultRegion` constant no longer exists in vertexai.go
  - All existing tests pass

### AC-006: No os.Getenv calls outside main.go [Linked to SC-001, SC-002, SC-003, SC-004]

- **Given**: Refactoring is complete
- **When**: Codebase is searched for `os.Getenv`
- **Then**:
  - Only `main.go` contains `os.Getenv` calls (files with `_test.go` suffix are excluded)
  - All configuration values are passed as parameters to other packages

## Implementation Notes

- Metadata server check happens at client creation time, not at config load time
- Test files (`*_test.go`) are excluded from the `os.Getenv` restriction

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-25 | 1.0 | Initial version | - |
| 2025-12-25 | 1.1 | Removed implementation details from acceptance criteria | - |
