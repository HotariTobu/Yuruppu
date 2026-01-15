# ADR: Tool Response Pattern

> Date: 2026-01-16
> Status: **Adopted**

## Context

Tools need to communicate both success and failure states to the caller. A consistent pattern is needed across all tools in the codebase.

## Decision Drivers

- Consistency across all tools
- Clear separation between success data and error handling
- Leverage Go's native error handling
- Security: avoid exposing internal structure

## Options Considered

- **Option 1:** Success/Error fields in response - Include `success: bool` and `error: string` in response schema
- **Option 2:** Go error for failures - Return Go `error` for failures, response schema defines success data only

## Evaluation

| Criterion | Success/Error Fields | Go Error |
|-----------|---------------------|----------|
| Consistency with Go idioms | Low - duplicates error handling | High - uses native error |
| Schema simplicity | Low - extra fields | High - success data only |
| Error information | Limited to string | Full error chain with context |

## Decision

Adopt **Go error for failures**.

## Rationale

- Response schema defines success data only
- Errors are returned as Go `error` from the tool callback
- Leverages Go's native error handling and wrapping
- For security, returned errors must not include messages that reveal internal structure; detailed information should be output via logger

## Consequences

**Positive:**
- Simpler response schemas
- Consistent with Go idioms
- Full error context available through error wrapping

**Negative:**
- Must refactor event tools to remove success/error fields

## Related Decisions

- [20260102-reply-tool.md](./20260102-reply-tool.md)
