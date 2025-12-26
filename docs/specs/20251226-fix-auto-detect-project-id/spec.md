# Fix: GCP_PROJECT_ID not auto-detected on Cloud Run

## Overview

The application fails to start on Cloud Run because `GCP_PROJECT_ID` is required as an environment variable, but it should be automatically detected from Cloud Run metadata server like `GCP_REGION`.

## Current Behavior (Bug)

- Application requires `GCP_PROJECT_ID` environment variable to be set
- On Cloud Run, if this env var is not configured, the app exits with error: `GCP_PROJECT_ID is required`
- Container fails health check and deployment fails

## Expected Behavior

- On Cloud Run, `GCP_PROJECT_ID` should be automatically detected from metadata server
- `GCP_PROJECT_ID` environment variable should be optional (used as fallback for local development)
- Application should start successfully on Cloud Run without explicit `GCP_PROJECT_ID` env var

## Root Cause

`loadConfig()` in `main.go` requires `GCP_PROJECT_ID` as a mandatory environment variable, but unlike `GCP_REGION`, it does not attempt to fetch from Cloud Run metadata server.

The metadata server provides project ID at:
```
http://metadata.google.internal/computeMetadata/v1/project/project-id
```

## Proposed Fix

- [ ] FX-001: Add `GetProjectID` function following the `GetRegion()` pattern and use it in `NewVertexAIClient`

## Acceptance Criteria

### AC-001: Auto-detect project ID on Cloud Run [Linked to FX-001]

- **Given**: Application running on Cloud Run (metadata server available)
- **When**: `loadConfig()` is called without `GCP_PROJECT_ID` env var
- **Then**:
  - Project ID is fetched from `http://metadata.google.internal/computeMetadata/v1/project/project-id`
  - Application starts successfully
  - Vertex AI client is initialized with correct project ID

### AC-002: Fallback to env var when metadata unavailable [Linked to FX-001]

- **Given**: Application running locally (metadata server unavailable)
- **When**: `loadConfig()` is called with `GCP_PROJECT_ID` env var set
- **Then**:
  - Project ID from env var is used
  - Application starts successfully

### AC-003: Error when no project ID available [Linked to FX-001]

- **Given**: Application running locally without metadata server
- **When**: Application starts without `GCP_PROJECT_ID` env var
- **Then**: Application fails with appropriate error message

### AC-004: Regression - existing functionality preserved [Linked to FX-001]

- **Given**: `GCP_PROJECT_ID` env var is set
- **When**: Application starts (regardless of metadata server availability)
- **Then**:
  - Env var value takes precedence over metadata detection
  - Existing behavior is unchanged

## Implementation Notes

- Follow existing pattern from `GetRegion()` in `internal/llm/vertexai.go`
- Project ID response is plain text (no path parsing needed, unlike region)

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-26 | 1.0 | Initial version | - |
| 2025-12-26 | 1.1 | Remove FX-002 (main.go change unnecessary), simplify implementation notes | - |
