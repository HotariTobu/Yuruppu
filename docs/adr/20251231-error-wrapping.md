# ADR: Error Wrapping Method

> Date: 2025-12-31
> Status: **Adopted**

<!--
ADR records decisions only. Do NOT add:
- Configuration examples or code snippets
- Version numbers
- Setup instructions or commands
-->

## Context

The unified error handling refactoring requires a consistent error wrapping method across the codebase. Currently, error handling is inconsistent: some functions log errors AND return them, causing duplicate log entries. The refactoring establishes a pattern where only top-level callers log errors, while called functions wrap errors with context and return them.

The spec states "Error wrapping method: To be decided in tech-research phase" and explicitly prohibits custom error types.

## Decision Drivers

- Must wrap errors with descriptive context messages
- No custom error types allowed (per spec)
- Preserve original error for inspection with errors.Is/As
- Zero or minimal external dependencies preferred
- Consistent with existing codebase patterns

## Options Considered

- **Option 1:** fmt.Errorf with %w (Go standard library)
- **Option 2:** github.com/pkg/errors
- **Option 3:** github.com/cockroachdb/errors
- **Option 4:** emperror.dev/errors

## Evaluation

See `evaluation-criteria.md` for criteria definitions.

| Criterion | Weight | fmt.Errorf %w | pkg/errors | cockroachdb/errors | emperror/errors |
|-----------|--------|---------------|------------|--------------------|-----------------|
| Functional Fit | 25% | 5 (1.25) | 4 (1.00) | 5 (1.25) | 4 (1.00) |
| Go Compatibility | 20% | 5 (1.00) | 3 (0.60) | 5 (1.00) | 4 (0.80) |
| Lightweight | 15% | 5 (0.75) | 5 (0.75) | 2 (0.30) | 4 (0.60) |
| Security | 15% | 5 (0.75) | 4 (0.60) | 4 (0.60) | 3 (0.45) |
| Documentation | 15% | 5 (0.75) | 4 (0.60) | 5 (0.75) | 4 (0.60) |
| Ecosystem | 10% | 5 (0.50) | 2 (0.20) | 5 (0.50) | 2 (0.20) |
| **Total** | 100% | **4.00** | **3.75** | **4.40** | **3.65** |

## Decision

Adopt **fmt.Errorf with %w** (Go standard library).

## Rationale

1. **Zero dependencies**: Part of Go standard library, no external packages required
2. **Already established in codebase**: The project already uses this pattern in storage, history, and agent packages
3. **Spec compliance**: No custom error types needed, just string messages with wrapped errors
4. **Full errors.Is/As support**: Wrapped errors are fully accessible for error chain inspection
5. **Future-proof**: Guaranteed long-term support as part of Go standard library

While cockroachdb/errors scored higher (4.40 vs 4.00), its benefits (automatic stack traces, network portability) are unnecessary for this LINE bot:
- Stack traces are not needed when errors are logged once at the top level with sufficient context
- Network portability is irrelevant for a single-server application
- Additional dependencies add unnecessary weight

## Consequences

**Positive:**
- Consistent error handling pattern across the codebase
- No new dependencies introduced
- Simpler mental model for developers

**Negative:**
- No automatic stack traces (must add context manually at each layer)

**Risks:**
- If debugging becomes difficult without stack traces, can migrate to cockroachdb/errors later (API is similar)

## Related Decisions

- [20251217-logging.md](./20251217-logging.md)

## Resources

| Option | Documentation | Repository |
|--------|---------------|------------|
| fmt.Errorf %w | [pkg.go.dev/fmt](https://pkg.go.dev/fmt) | Go standard library |
| pkg/errors | [pkg.go.dev](https://pkg.go.dev/github.com/pkg/errors) | [GitHub](https://github.com/pkg/errors) (Archived) |
| cockroachdb/errors | [pkg.go.dev](https://pkg.go.dev/github.com/cockroachdb/errors) | [GitHub](https://github.com/cockroachdb/errors) |
| emperror/errors | [pkg.go.dev](https://pkg.go.dev/emperror.dev/errors) | [GitHub](https://github.com/emperror/errors) |

## Sources

- [Working with Errors in Go](https://go.dev/blog/go1.13-errors)
- [Error Wrapping with Go's Standard Library](https://medium.com/@AlexanderObregon/error-wrapping-with-gos-standard-library-0a345eeea019)
- [A practical guide to error handling in Go | Datadog](https://www.datadoghq.com/blog/go-error-handling/)
- [The performance of Go error handling](https://g4s8.wtf/posts/go-errors-performance/)
