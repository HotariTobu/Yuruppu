# Refactor: Unified Error Handling and Logger Usage

> Template for code refactoring.

## Overview

Unify error handling patterns across the codebase. Only top-level callers log errors; called functions wrap and return errors without logging.

## Background & Purpose

Currently, handler.go logs errors at ERROR level for each operation (GetHistory, PutHistory, Generate, SendReply) AND returns them. When server.go receives these errors, it logs them again, creating duplicate log entries for the same failure. Similarly, client.go and gemini.go log errors and return them.

This refactoring establishes a clear pattern:
- **Top-level callers**: Log errors at ERROR level
- **Called functions**: Wrap errors with context and return them (no ERROR logging)
- **Non-error logs**: Continue logging normally (Debug, Info, Warn)

## Scope

- [ ] SC-001: `internal/bot/handler.go` - Remove ERROR logging, add error wrapping
- [ ] SC-002: `internal/line/client.go` - Remove ERROR logging, add error wrapping
- [ ] SC-003: `internal/agent/gemini.go` - Remove ERROR logging, add error wrapping

## Breaking Changes

None. External behavior is unchanged; only internal logging patterns change.

## Acceptance Criteria

### AC-001: handler.go error handling [Linked to SC-001]

- **Given**: A request that causes an error in `handleMessage`
- **When**: The error propagates to `server.go`
- **Then**:
  - Error is logged exactly once (at server layer)
  - Error message contains context showing the call path
  - No `logger.ErrorContext` calls remain in handler.go
- **Verification**: Search handler.go for `ErrorContext` - should return no matches

### AC-002: client.go error handling [Linked to SC-002]

- **Given**: A LINE API call that fails
- **When**: The error is returned to the caller
- **Then**:
  - Error is wrapped with context (including x-line-request-id)
  - No `logger.Error` calls remain in client.go
  - Debug logs preserved: "sending reply", "reply sent successfully"
- **Verification**: Search client.go for `logger.Error` - should return no matches

### AC-003: gemini.go error handling [Linked to SC-003]

- **Given**: An LLM operation that fails (including response validation)
- **When**: The error is returned to the caller
- **Then**:
  - Error is wrapped with context showing the call path
  - No `logger.Error` calls remain in gemini.go
  - Preserved logs: Debug (system prompt token count, cache skipped, cache created, cache refreshed, generating text), Info (response generated successfully), Warn (cache warnings)
- **Verification**: Search gemini.go for `logger.Error` - should return no matches

### AC-004: No duplicate error logging [Linked to SC-001, SC-002, SC-003]

- **Given**: Any error that occurs during request processing
- **When**: Error propagates through the call stack
- **Then**:
  - Error is logged at ERROR level exactly once (at server.go or main.go)
  - Error message is traceable through the call path via wrapping
- **Verification**: Trigger error in test, verify single ERROR log entry with full context chain

### AC-005: Existing tests pass [Linked to SC-001, SC-002, SC-003]

- **Given**: Refactoring is complete
- **When**: `go test ./...` is executed
- **Then**:
  - All existing tests pass
  - No compilation errors
- **Verification**: Run `go test ./...` and confirm exit code 0

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-31 | 1.0 | Initial version | - |
| 2025-12-31 | 1.1 | Added verification methods, excluded files section | - |
| 2025-12-31 | 1.2 | Defer error wrapping method decision to tech-research phase | - |
| 2025-12-31 | 1.3 | Remove implementation order from notes | - |
| 2025-12-31 | 1.4 | Move design sections to design.md | - |
