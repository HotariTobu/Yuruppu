# ADR: Error Creation Style

> Date: 2026-01-17
> Status: **Adopted**

<!--
ADR records decisions only. Do NOT add:
- Configuration examples or code snippets
- Version numbers
- Setup instructions or commands
-->

## Context

This project uses `fmt.Errorf` with `%w` for error wrapping (see 20251231-error-wrapping.md). However, a separate decision is needed for creating new errors that have no underlying error to wrap.

Go provides two common approaches:
- `errors.New("message")` - simple, no context
- `fmt.Errorf("operation %s: reason", param)` - includes context information

## Decision Drivers

- Errors should be debuggable without stack traces
- Context information (function parameters, state) aids debugging
- Consistency across the codebase
- No sentinel errors needed in application code

## Options Considered

- **Option 1:** Use `errors.New` for simple errors, `fmt.Errorf` for contextual errors
- **Option 2:** Always use `fmt.Errorf` with context information

## Decision

Adopt **Option 2**: Always use `fmt.Errorf` with context information.

Do not define sentinel errors.

**Exceptions** where `errors.New` is acceptable:
- Environment variable validation (variable name is the context)
- Argument nil/empty checks (programming error detection)
- Errors returned to LLM (toolset validation and operation failures)

## Rationale

1. **Debuggability**: Context information (parameters, identifiers) in error messages makes debugging easier without relying on stack traces
2. **Consistency**: Single pattern for all error creation reduces cognitive load
3. **Sentinel errors are unnecessary**: Application code rarely needs `errors.Is` checks against package-level errors. When error type differentiation is needed, custom error types are more flexible
4. **Performance is acceptable**: `fmt.Errorf` is slightly slower than `errors.New`, but error paths are not hot paths

## Consequences

**Positive:**
- All errors carry useful context for debugging
- Consistent error creation pattern
- No proliferation of sentinel error variables

**Negative:**
- Slightly more verbose error creation
- Cannot use `errors.Is` with sentinel errors (use custom error types if needed)

## Related Decisions

- [20251231-error-wrapping.md](./20251231-error-wrapping.md)
