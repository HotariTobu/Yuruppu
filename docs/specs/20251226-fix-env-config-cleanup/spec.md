# Fix: Environment Configuration Cleanup

## Overview

Fix environment variable handling: reorder fields for consistency and move GCP credential validation to the LLM client layer where it belongs.

## Current Behavior (Bug)

1. **Field ordering is inconsistent**: The `Config` struct orders fields as `ChannelSecret`, `ChannelAccessToken`, `GCPProjectID`, `LLMTimeoutSeconds`, `Port`, `GCPRegion` - mixing logical groupings.

2. **Validation happens in wrong layer**: `GCP_PROJECT_ID` is validated in `loadConfig()` (main.go), but this is redundant since:
   - The LLM client already validates it (`vertexai.go:47-49`)
   - On Cloud Run, project ID can be auto-detected from metadata server, so environment variable is optional

3. **Unnecessary default value**: `GCP_REGION` has a hardcoded default (`asia-northeast1`) in config layer, but:
   - On Cloud Run, region is auto-detected from metadata server
   - The LLM client layer should decide the fallback behavior

4. **NewVertexAIClient internal order is inconsistent**: The nil context check is placed between projectID and region handling, breaking the logical flow

## Expected Behavior

1. **Consistent field ordering**: Fields should be ordered logically:
   - Server config first (`Port`)
   - LINE credentials (`ChannelSecret`, `ChannelAccessToken`)
   - GCP config (`GCPProjectID`, `GCPRegion`)
   - Timeout config (`LLMTimeoutSeconds`)

2. **Validation in appropriate layer**: GCP-related validation should only happen in the LLM client layer

3. **No default values for GCP config**: `GCPProjectID` and `GCPRegion` should not have defaults in config layer

4. **Consistent NewVertexAIClient internal order**: Operations should follow logical order:
   - Context normalization first
   - Credential retrieval (projectID, region)
   - Validation
   - Client creation

## Root Cause

Initial implementation added validation and defaults at the config layer for convenience, but this creates:
- Redundant validation
- Incorrect assumption that `GCP_PROJECT_ID` env var is always required (not true on Cloud Run)
- Hardcoded region default that may not be appropriate

## Proposed Fix

- [ ] FX-001: Reorder `Config` struct fields to: `Port`, `ChannelSecret`, `ChannelAccessToken`, `GCPProjectID`, `GCPRegion`, `LLMTimeoutSeconds`
- [ ] FX-002: Remove `GCP_PROJECT_ID` validation from `loadConfig()` in main.go
- [ ] FX-003: Remove `GCP_REGION` default value (`defaultRegion` constant and related code) from main.go
- [ ] FX-004: After removing validation/defaults, verify `Config.GCPProjectID` and `Config.GCPRegion` are empty strings when env vars are not set (natural behavior after FX-002/FX-003)
- [ ] FX-005: Verify LLM client validates `GCPProjectID` - no changes needed (existing code in `vertexai.go:47-49`)
- [ ] FX-006: Add `GCPRegion` validation in `NewVertexAIClient` - return error if region is empty after metadata detection and fallback
- [ ] FX-007: Update tests in `main_test.go` to reflect new behavior (see AC-005 for specific test changes)
- [ ] FX-008: Reorder `NewVertexAIClient` internal logic: (1) nil context check, (2) projectID retrieval, (3) region retrieval, (4) projectID validation, (5) region validation, (6) client creation

## Acceptance Criteria

### AC-001: [Linked to FX-001]

- **Given**: The `Config` struct in main.go
- **When**: Developer reads the struct definition
- **Then**:
  - Fields are ordered: `Port`, `ChannelSecret`, `ChannelAccessToken`, `GCPProjectID`, `GCPRegion`, `LLMTimeoutSeconds`
  - Related fields are grouped together

### AC-002: [Linked to FX-002, FX-004]

- **Given**: `GCP_PROJECT_ID` environment variable is not set
- **When**: `loadConfig()` is called
- **Then**:
  - No error is returned from `loadConfig()`
  - `Config.GCPProjectID` is empty string
  - Error will be returned later by LLM client if project ID cannot be detected from metadata

### AC-003: [Linked to FX-003, FX-004]

- **Given**: `GCP_REGION` environment variable is not set
- **When**: `loadConfig()` is called
- **Then**:
  - No error is returned from `loadConfig()`
  - `Config.GCPRegion` is empty string
  - LLM client will attempt metadata detection, then use empty string as fallback

### AC-004: [Linked to FX-002, Regression]

- **Given**: `LINE_CHANNEL_SECRET` or `LINE_CHANNEL_ACCESS_TOKEN` is not set
- **When**: `loadConfig()` is called
- **Then**:
  - Error is still returned for missing LINE credentials
  - Existing validation for required LINE env vars remains intact

### AC-005: [Linked to FX-007] Test Updates

The following test changes are required:

1. **Remove or modify**: `TestLoadConfig_MissingGCPProjectID`
   - Current: Expects error when `GCP_PROJECT_ID` is not set
   - After: Should expect success with empty string in `Config.GCPProjectID`

2. **Modify**: `TestLoadConfig_GCPRegion`
   - Current: Expects default `asia-northeast1` when `GCP_REGION` is not set
   - After: Should expect empty string when `GCP_REGION` is not set

3. **Modify**: `TestLoadConfig_GCPRegion_TrimsWhitespace`
   - Current: Expects default `asia-northeast1` for whitespace-only input
   - After: Should expect empty string for whitespace-only input

4. **Keep unchanged**: `TestInitLLM_EmptyGCPProjectID`
   - This test validates LLM client layer correctly rejects empty project ID
   - Should continue to pass (validation happens in LLM layer)

5. **Modify**: `TestLoadConfig_ErrorMessages`
   - Remove the test case "missing GCP_PROJECT_ID error mentions GCP_PROJECT_ID"

6. **Add new test**: `TestLoadConfig_GCPConfigOptional`
   - Given: `GCP_PROJECT_ID` and `GCP_REGION` are not set, LINE credentials are set
   - When: `loadConfig()` is called
   - Then: No error, both `GCPProjectID` and `GCPRegion` are empty strings

7. **Add new test in `vertexai_test.go`**: `TestNewVertexAIClient_EmptyGCPRegion`
   - Given: Empty fallbackRegion and metadata server unavailable
   - When: `NewVertexAIClient` is called
   - Then: Error "GCP_REGION is missing or empty" is returned

### AC-006: [Linked to FX-006, FX-008]

- **Given**: The `NewVertexAIClient` function in `vertexai.go`
- **When**: Developer reads the function body
- **Then**:
  - Operations are ordered: (1) nil context check, (2) projectID retrieval, (3) region retrieval, (4) projectID validation, (5) region validation, (6) client creation
  - Related operations are grouped together
  - Region validation added: returns error "GCP_REGION is missing or empty" if region is empty

### AC-007: [Linked to FX-006]

- **Given**: `GCP_REGION` environment variable is not set AND not running on Cloud Run (metadata unavailable)
- **When**: `NewVertexAIClient` is called
- **Then**:
  - Error is returned: "GCP_REGION is missing or empty"
  - No Vertex AI client is created

## Implementation Notes

### Existing Validation in LLM Client

The LLM client (`vertexai.go:47-49`) already validates `GCPProjectID`:

```go
// Validate projectID is not empty or whitespace
if strings.TrimSpace(projectID) == "" {
    return nil, errors.New("GCP_PROJECT_ID is missing or empty")
}
```

### Region Handling in LLM Client

The `GetRegion()` function retrieves region from metadata or fallback. After FX-006, validation will be added:

```go
// Get region (existing)
region := GetRegion(metadataServerURL, fallbackRegion)

// Validate region (new - FX-006)
if strings.TrimSpace(region) == "" {
    return nil, errors.New("GCP_REGION is missing or empty")
}
```

### Safety of Changes

- Removing the `defaultRegion` constant is safe because the client uses metadata detection first
- `GCP_PROJECT_ID` validation is preserved in the LLM client layer where it belongs

### Go Type Changes

```go
// main.go - Updated Config struct ordering
type Config struct {
    Port               string // Server port (default: 8080)
    ChannelSecret      string
    ChannelAccessToken string
    GCPProjectID       string // Optional: auto-detected on Cloud Run
    GCPRegion          string // Optional: auto-detected on Cloud Run
    LLMTimeoutSeconds  int    // LLM API timeout in seconds (default: 30)
}
```

### Constants to Remove

```go
// Remove from main.go:
// defaultRegion = "asia-northeast1"
```

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-26 | 1.0 | Initial version | - |
