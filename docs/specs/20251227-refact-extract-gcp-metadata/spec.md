# Refactor: Extract GCP Metadata Fetching from LLM Package

## Overview

Extract GCP metadata fetching logic from `internal/llm/vertexai.go` into a dedicated `internal/gcp` package with a MetadataClient.

## Background & Purpose

The current `internal/llm/vertexai.go` file mixes two responsibilities:

1. LLM Client: Creating and using the Vertex AI client for text generation
2. GCP Metadata: Fetching project ID and region from Cloud Run metadata server

This violates the Single Responsibility Principle. The metadata fetching is a Cloud Run infrastructure concern, not an LLM concern.

## Current Structure

The `internal/llm/vertexai.go` file contains metadata fetching logic (constants, HTTP client, fetch functions, parse functions) alongside LLM client logic. The test file mixes tests for both concerns.

## Proposed Structure

Create a dedicated `internal/gcp` package with a MetadataClient that handles fetching project ID and region from the Cloud Run metadata server. The LLM package will use pre-resolved values passed by the caller. The caller (main.go) will create the MetadataClient and resolve values before creating the LLM client.

## Scope

- [x] SC-001: Create `internal/gcp` package with MetadataClient
- [x] SC-002: Create tests for MetadataClient
- [x] SC-003: Update `internal/llm/vertexai.go` to accept resolved values
- [x] SC-004: Update LLM tests to remove metadata-related tests
- [x] SC-005: Update `main.go` to use MetadataClient
- [x] SC-006: Update all other callers of NewVertexAIClient

## Breaking Changes

- `NewVertexAIClient` signature changes: no longer accepts fallback values or timeout, only resolved project ID and region
- Callers must resolve project ID and region using MetadataClient before calling NewVertexAIClient

## Acceptance Criteria

### AC-001: MetadataClient Created [Linked to SC-001]

- **Given**: The refactoring is complete
- **When**: Examining `internal/gcp` package
- **Then**:
  - MetadataClient can be created with configurable timeout and logger
  - MetadataClient can fetch project ID with fallback
  - MetadataClient can fetch region with fallback
  - Default metadata server URL is `http://metadata.google.internal`
  - Default timeout is 2 seconds

### AC-002: MetadataClient Tests [Linked to SC-002]

- **Given**: The refactoring is complete
- **When**: Running `go test ./internal/gcp/...`
- **Then**:
  - All metadata-related tests pass
  - Timeout tests using synctest work correctly

### AC-003: VertexAI Client Simplified [Linked to SC-003]

- **Given**: The refactoring is complete
- **When**: Examining `internal/llm/vertexai.go`
- **Then**:
  - No metadata-related code remains
  - NewVertexAIClient accepts resolved project ID and region
  - Validation errors are returned for empty/whitespace values

### AC-004: LLM Tests Focused [Linked to SC-004]

- **Given**: The refactoring is complete
- **When**: Running `go test ./internal/llm/...`
- **Then**:
  - Only LLM-specific tests remain
  - All tests pass

### AC-005: Main Uses MetadataClient [Linked to SC-005]

- **Given**: The refactoring is complete
- **When**: Examining `main.go`
- **Then**:
  - Creates MetadataClient
  - Resolves project ID and region before creating LLM client

### AC-006: All Callers Updated [Linked to SC-006]

- **Given**: The refactoring is complete
- **When**: Building the project
- **Then**:
  - No compilation errors

### AC-007: Behavior Unchanged

- **Given**: The refactoring is complete
- **When**: Running the application
- **Then**:
  - Auto-detection from metadata server works on Cloud Run
  - Fallback to environment variables works when metadata unavailable
  - All existing functionality works identically

## Implementation Notes

- The `gcp` package must not import from `llm` package

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-27 | 1.0 | Initial version | - |
| 2025-12-27 | 2.0 | Simplify spec, remove implementation details, use MetadataClient approach | - |
