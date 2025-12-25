# Fix: CI Test Failures Due to Google Client ADC Requirement

> Fix for CI test failures caused by `genai.NewClient` requiring Application Default Credentials.

## Overview

CI tests for `NewVertexAIClient` fail because `genai.NewClient` attempts to authenticate using Application Default Credentials (ADC), which are not available in the CI environment.

## Current Behavior (Bug)

- `TestNewVertexAIClient_ValidProjectID` and related tests call `llm.NewVertexAIClient`
- `NewVertexAIClient` internally calls `genai.NewClient` (line 57-61 in `internal/llm/vertexai.go`)
- `genai.NewClient` attempts to read ADC from the environment
- In CI, no ADC is configured, causing test failures

**Affected Tests** (tests that call `NewVertexAIClient` with valid project ID):
- `TestNewVertexAIClient_ValidProjectID`
- `TestNewVertexAIClient_ContextSupport`
- `TestNewVertexAIClient_FromEnvironment` (partial)
- `TestNewVertexAIClient_ADCAuthentication`
- `TestNewVertexAIClient_ModelConfiguration`
- `TestNewVertexAIClient_RegionConfiguration`
- `TestNewVertexAIClient_InterfaceCompliance`
- `TestNewVertexAIClient_Concurrency`
- `TestNewVertexAIClient_InitializationFailure` (partial - nil context test)

## Expected Behavior

- Unit tests should pass without requiring real Google Cloud credentials
- Tests should verify the client creation logic without hitting the real Google API
- The production code should remain unchanged and continue to work with ADC

## Root Cause

The `NewVertexAIClient` function directly creates a `genai.Client`, which is tightly coupled to the Google API authentication. There is no abstraction layer to inject a mock client for testing.

## Proposed Fix

- [ ] FX-001: Introduce dependency injection for `genai.Client` creation in `NewVertexAIClient` using functional options pattern
- [ ] FX-002: Create mock implementation for unit tests that can be configured to return nil client (success) or error (failure)
- [ ] FX-003: Update affected tests to use mock implementation

## Acceptance Criteria

### AC-001: Tests Pass in CI Without Credentials [Linked to FX-001, FX-002, FX-003]

- **Given**: CI environment without Google Cloud credentials
- **When**: Running `go test ./internal/llm/...`
- **Then**:
  - All `TestNewVertexAIClient_*` tests pass
  - No authentication errors occur
  - Tests complete quickly without network calls (< 1 second)
  - Mock implementation is verified to be invoked in tests

### AC-002: Backward Compatibility [Linked to FX-001, Regression]

- **Given**: Production environment with ADC configured
- **When**: `NewVertexAIClient` is called with valid project ID and no options
- **Then**:
  - Client is created successfully using real `genai.Client`
  - Existing functionality remains intact
  - Callers without options continue to work unchanged

### AC-003: Mock Error Propagation [Linked to FX-002]

- **Given**: Test with mock configured to return an error
- **When**: `NewVertexAIClient` is called
- **Then**:
  - `NewVertexAIClient` returns the error correctly
  - Error message is preserved

### AC-004: Mock Success Case [Linked to FX-002]

- **Given**: Test with mock configured to return nil error
- **When**: `NewVertexAIClient` is called
- **Then**:
  - `NewVertexAIClient` returns valid Provider without error

## Technical Requirements

### TR-001: ClientFactory Abstraction

Define an interface for creating `genai.Client` that supports dependency injection for testing. The interface should:
- Accept context and configuration parameters
- Return client and error
- Have a default production implementation that calls `genai.NewClient`

### TR-002: Functional Options for NewVertexAIClient

Modify `NewVertexAIClient` to accept optional configuration using Go's functional options pattern:
- Add variadic options parameter to existing signature
- When no options provided, use default production factory
- Allow tests to inject mock factory via option
- Maintain backward compatibility (existing callers without options continue to work)

### TR-003: Mock Implementation for Tests

Create a test-only mock that:
- Implements the ClientFactory interface
- Can be configured to return nil client and nil error (success case)
- Can be configured to return nil client and custom error (error case)
- Lives in test file (not production code)

## Out of Scope

- Integration tests with real Google Cloud credentials (separate test suite)
- Mocking the `GenerateText` functionality (already uses `Provider` interface)
- Tests that call `GenerateText` method - those are integration tests and not affected by this fix since they would fail at `GenerateText` call, not at client creation

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-25 | 1.0 | Initial version | - |
| 2025-12-25 | 1.1 | Address spec-reviewer feedback: remove implementation code, clarify requirements | - |
