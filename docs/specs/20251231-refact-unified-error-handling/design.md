# Design: unified-error-handling

## Overview

Unify error handling patterns: only top-level callers log errors at ERROR level; called functions wrap errors with context and return them.

## File Structure

| File | Purpose | Change |
|------|---------|--------|
| `internal/bot/handler.go` | Message handler | Remove ERROR logging, add error wrapping |
| `internal/line/client.go` | LINE API client | Remove ERROR logging, add error wrapping with x-line-request-id |
| `internal/agent/gemini.go` | LLM agent | Remove ERROR logging from response validation |

### Files Excluded

| File | Reason |
|------|--------|
| `main.go` | Initialization errors cause process termination - remain as-is |
| `internal/line/server.go` | Top-level caller for request processing - remain as-is |

## Error Wrapping Pattern

Per [ADR 20251231-error-wrapping](../../adr/20251231-error-wrapping.md): Use `fmt.Errorf` with `%w`.

```go
// Pattern: wrap and return, do not log
if err != nil {
    return fmt.Errorf("context message: %w", err)
}
```

## Data Flow

```
User Request
    ↓
server.go (top-level) ← Logs errors at ERROR level
    ↓
handler.go ← Wraps errors, returns to caller
    ↓
├── history repo ← Wraps errors, returns
├── agent (gemini.go) ← Wraps errors, returns
└── sender (client.go) ← Wraps errors with x-line-request-id, returns
```

## Implementation Notes

- Do NOT create custom error types
- Keep logger field in structs for non-error logging (Debug, Info, Warn)
- Include x-line-request-id in LINE API error messages
- msgCtx is already logged by server.go - no need to include in error messages
