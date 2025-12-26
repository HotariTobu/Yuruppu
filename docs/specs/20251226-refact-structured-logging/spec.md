# Refactor: Unified Structured Logging

> Template for code refactoring.

## Overview

Refactor logging throughout the codebase to use slog consistently with JSON format.

## Background & Purpose

ADR `20251217-logging.md` specifies using `log/slog` for all logging. However, the current implementation violates this decision:
- `main.go` uses `log.Fatal()` and `log.Printf()` instead of slog
- Nil logger checks exist, allowing logging to be silently disabled
- Some packages have no logging coverage

This refactoring enforces the ADR decision.

## Current Structure

The codebase currently uses two different logging approaches:

**main.go**: Uses `log.Fatal()` and `log.Printf()` for startup/fatal errors.

**internal/yuruppu/handler.go**: Uses `slog` with context-aware methods (`DebugContext`, `ErrorContext`) with structured attributes. Contains nil logger checks.

**internal/line/server.go**: Uses `slog.Error()` with structured attributes. Contains nil logger checks.

**internal/llm/vertexai.go**: No logging at all.

**internal/line/client.go**: No logging at all.

Logger is instantiated directly in main.go: `slog.New(slog.NewTextHandler(os.Stdout, nil))`.

## Proposed Structure

**Logger Initialization in main.go**: Create `*slog.Logger` with JSON handler. No environment variable configuration - log level filtering is handled by Cloud Logging.

**main.go**: Replace all `log.Fatal()`/`log.Printf()` calls with structured slog calls.

**All internal packages**: Accept `*slog.Logger` as a required dependency (no nil checks). Add logging to packages that currently lack it.

## Scope

- [ ] SC-001: Refactor `main.go` to use JSON logger and unified structured logging
- [ ] SC-002: Change `NewServer` to accept logger in constructor, remove `SetLogger` method
- [ ] SC-003: Change `NewClient` to accept logger in constructor
- [ ] SC-004: Change `NewVertexAIClient` to accept logger in constructor
- [ ] SC-005: Remove nil logger checks from `internal/yuruppu/handler.go`
- [ ] SC-006: Update all tests to provide logger instances

## Breaking Changes

None. External API remains unchanged. Only internal logging behavior is modified.

## Acceptance Criteria

### AC-001: Main Logger Uses JSON [Linked to SC-001]

- **Given**: The application starts up
- **When**: Logger is initialized in main.go
- **Then**:
  - Logger uses `slog.NewJSONHandler`
  - All startup messages use structured slog format
  - Fatal errors are logged with `slog.Error()` before `os.Exit(1)`

### AC-002: Server Constructor Injection [Linked to SC-002]

- **Given**: `internal/line/server.go` is refactored
- **When**: `NewServer` is called
- **Then**:
  - `NewServer` accepts `*slog.Logger` as a required parameter
  - `SetLogger` method is removed
  - No `if logger != nil` checks exist

### AC-003: Client Constructor Injection [Linked to SC-003]

- **Given**: `internal/line/client.go` is refactored
- **When**: `NewClient` is called
- **Then**:
  - `NewClient` accepts `*slog.Logger` as a required parameter
  - Reply operations are logged at DEBUG level
  - Errors are logged at ERROR level

### AC-004: VertexAI Constructor Injection [Linked to SC-004]

- **Given**: `internal/llm/vertexai.go` is refactored
- **When**: `NewVertexAIClient` is called
- **Then**:
  - `NewVertexAIClient` accepts `*slog.Logger` as a required parameter
  - API calls are logged at DEBUG level
  - Errors are logged at ERROR level

### AC-005: Handler Nil Checks Removed [Linked to SC-005]

- **Given**: `internal/yuruppu/handler.go` is refactored
- **When**: Code is reviewed
- **Then**:
  - No `if logger != nil` checks exist
  - Logger is used directly without nil guards

### AC-006: Tests Provide Loggers [Linked to SC-006]

- **Given**: All tests are updated
- **When**: Tests are executed
- **Then**:
  - All constructors receive valid logger instances
  - Tests use discard handler to suppress logs
  - All existing tests pass

## DEBUG Log Insertion

Add DEBUG logs throughout the codebase following the log level guidelines. For code that already has logger access, add DEBUG logs without changing function signatures or structure.

## Log Level Guidelines

| Level | Purpose | Examples |
|-------|---------|----------|
| ERROR | System problems requiring investigation | Config errors, API auth failures, unexpected panics |
| WARN | External factors / temporary issues | Webhook signature validation failure, reply token expired, rate limiting |
| INFO | Request flow | Server startup, message received, reply sent |
| DEBUG | Detailed info for troubleshooting | Message content, API request/response |

**Principles:**
- 1 request = at least 1 INFO log (for traceability)
- Use ERROR vs WARN based on root cause
- Sensitive/detailed data goes in DEBUG

## Implementation Notes

- Use `slog.NewJSONHandler(os.Stdout, nil)` for production
- For tests, use `slog.New(slog.NewJSONHandler(io.Discard, nil))` to suppress logs

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-26 | 1.2 | Add DEBUG log insertion section | - |
| 2025-12-26 | 1.1 | Add log level guidelines | - |
| 2025-12-26 | 1.0 | Initial version | - |
