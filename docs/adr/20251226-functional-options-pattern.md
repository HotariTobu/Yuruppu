# ADR: Functional Options Pattern for Dependency Injection

> Date: 2025-12-26
> Status: **Adopted**

## Context

The `NewVertexAIClient` function directly calls `genai.NewClient`, which requires Application Default Credentials (ADC). This prevents unit testing in CI environments without credentials. We need a way to inject a mock client factory for testing while maintaining backward compatibility.

## Decision Drivers

- Need to inject test dependencies without breaking existing callers
- Must maintain backward compatibility (zero changes to production call sites)
- Follow established Go idioms
- Align with existing testing strategy (manual mocks, no mock libraries)

## Options Considered

- **Option 1:** Constructor parameter (explicit factory argument)
- **Option 2:** Functional options pattern (variadic options)
- **Option 3:** Package-level variable (swap factory in tests)

## Decision

Adopt **Option 2: Functional options pattern**.

## Rationale

- **Backward compatible**: Variadic `...Option` parameter allows existing callers to continue without changes
- **Idiomatic Go**: Used by standard library (`http.Server`) and major Go projects (`grpc-go`, `go-cloud`)
- **Extensible**: Can add more options later without signature changes
- **Explicit**: Dependencies are clearly passed at call site, no hidden global state

Option 1 (constructor parameter) would break all existing callers. Option 3 (package-level variable) introduces global state and potential race conditions in parallel tests.

## Naming Conventions

For this codebase:

| Element | Convention | Example |
|---------|------------|---------|
| Option type | `<Type>Option` | `VertexAIOption` |
| Config struct | `<type>Config` (unexported) | `vertexAIConfig` |
| Option function | `With<Dependency>` | `WithClientFactory` |

## Consequences

**Positive:**
- Tests can run without credentials
- No breaking changes to production code
- Pattern can be reused for future optional dependencies

**Negative:**
- Slightly more code than direct constructor
- New pattern to learn for contributors

## Related Decisions

- [20251217-testing-strategy.md](./20251217-testing-strategy.md) - Manual mock approach
