# Refactor: Integration Test Strategy

> Separate unit tests and integration tests using Go build tags.

## Overview

Introduce build tags to separate unit tests (run in CI) from integration tests (run locally with real credentials).

## Background & Purpose

### Current Problems

1. **Mixed test types**: Unit tests and integration tests are mixed in the same files without clear separation
2. **ADC dependency**: Tests that require Application Default Credentials (ADC) fail in CI
3. **Coverage gaps**: ADC-dependent tests were removed (20251225-fix-google-client-mock), leaving no coverage for real API integration

### Goals

- Clear separation between unit tests and integration tests
- CI runs only unit tests (no credentials required)
- Developers can run integration tests locally with real credentials
- Restore coverage for external service integration

## Current Structure

All tests run with `go test ./...`:

- `main_test.go`: Configuration and initialization tests
- `internal/bot/bot_test.go`: Bot functionality tests with mocks
- `internal/llm/provider_test.go`: LLM provider interface tests
- `internal/llm/vertexai_test.go`: Vertex AI client tests (error cases only)

## Proposed Structure

Tests separated by build tags:

- `*_test.go`: Unit tests (run in CI with `make test`)
- `*_integration_test.go`: Integration tests (run locally with `make test-integration`)

New integration test files:

- `internal/llm/vertexai_integration_test.go`: Vertex AI integration tests
- `internal/bot/bot_integration_test.go`: LINE API integration tests

Makefile targets:

- `make test`: Unit tests only (CI)
- `make test-integration`: All tests including integration (local)

## Scope

- [ ] SC-001: Create `internal/llm/vertexai_integration_test.go`
- [ ] SC-002: Create `internal/bot/bot_integration_test.go`
- [ ] SC-003: Add Makefile targets for test separation
- [ ] SC-004: Verify all existing `*_test.go` files pass without credentials (no code changes expected)

## Breaking Changes

None. Existing `go test ./...` continues to run unit tests only. Integration tests require explicit `-tags=integration` flag.

## Acceptance Criteria

### AC-001: Unit Tests Pass Without Credentials [SC-004]

- **Given**: CI environment without ADC or LINE credentials
- **When**: Running `make test`
- **Then**: All unit tests pass (integration tests are not executed)

### AC-002: Integration Tests Verify Vertex AI [SC-001]

- **Given**: Local environment with ADC and `GCP_PROJECT_ID` configured
- **When**: Running `make test-integration`
- **Then**:
  - `NewVertexAIClient()` creates client successfully
  - `GenerateText()` returns response from Vertex AI

### AC-003: Integration Tests Skip Without Credentials [SC-001, SC-002]

- **Given**: Local environment without required credentials
- **When**: Running `make test-integration`
- **Then**: Integration tests skip with message indicating missing credentials (using `t.Skip()`)

### AC-004: Integration Tests Verify LINE API [SC-002]

- **Given**: Local environment with LINE credentials configured
- **When**: Running `make test-integration`
- **Then**:
  - `GetBotInfo()` returns bot information from LINE API

### AC-005: Makefile Provides Test Targets [SC-003]

- **Given**: Developer wants to run unit tests only
- **When**: Running `make test`
- **Then**: Only unit tests run (no credentials required)

- **Given**: Developer wants to run all tests including integration
- **When**: Running `make test-integration`
- **Then**: Both unit tests and integration tests run

### AC-006: Integration Tests Clearly Marked [SC-001, SC-002]

- **Given**: Developer reading test files
- **When**: Looking at `*_integration_test.go` files
- **Then**:
  - File starts with `//go:build integration`
  - Test function names follow pattern `Test<Component>_Integration_<Behavior>`

## Implementation Notes

- Integration test files must start with the build constraint `//go:build integration`
- Test function names follow the pattern `Test<Component>_Integration_<Behavior>`
- Integration tests must check for required environment variables at startup and call `t.Skip()` with a descriptive message if credentials are missing
- Vertex AI tests require `GCP_PROJECT_ID` and ADC (Application Default Credentials)
- LINE API tests require `LINE_CHANNEL_SECRET` and `LINE_CHANNEL_ACCESS_TOKEN`

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-26 | 1.0 | Initial version | - |
| 2025-12-26 | 1.1 | Address spec-reviewer feedback: add credential skip behavior, clarify AC numbering | - |
| 2025-12-26 | 1.2 | Remove code blocks from Implementation Notes per REFACTOR template | - |
