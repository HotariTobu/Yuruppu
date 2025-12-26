# ADR: Fake Time Testing Strategy

> Date: 2025-12-27
> Status: **Adopted**

## Context

Integration tests for timeout behavior (`TestGetRegion_Timeout`, `TestGetProjectID_Timeout`) use real `time.Sleep()` delays, adding ~23 seconds to CI runs. The spec `fix-integration-test-issues` FX-003 requires replacing real timers with fake timers for deterministic, fast tests.

## Decision Drivers

- Must eliminate ~23 seconds of real-time delays in timeout tests
- Must support HTTP client timeout testing scenarios
- Prefer standard library solutions over external dependencies
- Project uses Go 1.25+ (synctest is stable)

## Options Considered

- **Option 1:** testing/synctest (Go standard library)
- **Option 2:** jonboulle/clockwork
- **Option 3:** coder/quartz
- **Option 4:** benbjohnson/clock

## Evaluation

See `evaluation-criteria.md` for criteria definitions.

| Criterion | Weight | synctest | clockwork | quartz | benbjohnson/clock |
|-----------|--------|----------|-----------|--------|-------------------|
| Functional Fit | 25% | 5 (1.25) | 4 (1.00) | 4 (1.00) | 4 (1.00) |
| Go Compatibility | 20% | 5 (1.00) | 5 (1.00) | 5 (1.00) | 5 (1.00) |
| Lightweight | 15% | 5 (0.75) | 5 (0.75) | 5 (0.75) | 5 (0.75) |
| Security | 15% | 5 (0.75) | 4 (0.60) | 4 (0.60) | 3 (0.45) |
| Documentation | 15% | 5 (0.75) | 4 (0.60) | 4 (0.60) | 4 (0.60) |
| Ecosystem | 10% | 5 (0.50) | 4 (0.40) | 3 (0.30) | 2 (0.20) |
| **Total** | 100% | **5.00** | **4.35** | **4.25** | **4.00** |

## Decision

Adopt **testing/synctest** (Go standard library).

## Rationale

1. **Standard library**: No external dependencies required, full Go 1 compatibility guarantee
2. **Designed for HTTP timeout testing**: Official blog demonstrates exact use case with `net.Pipe()` and `http.Transport`
3. **Zero refactoring of production code**: Unlike clock injection libraries, synctest wraps tests without modifying production code
4. **Go 1.25+ compatibility**: Project uses Go 1.25+ where synctest is stable (not experimental)
5. **benbjohnson/clock archived**: The spec-mentioned library was archived May 2023 with no maintenance

## Consequences

**Positive:**
- Test suite runs in milliseconds instead of ~23 seconds
- No external dependency added
- Deterministic tests (no timing flakiness)

**Negative:**
- Must use `net.Pipe()` instead of real network connections in tests
- Cannot use `t.Run()`, `t.Parallel()` within synctest bubbles
- All goroutines must exit before test completes

**Risks:**
- Learning curve for "durable blocking" concept - mitigated by official Go blog documentation

## Related Decisions

- [20251217-testing-strategy.md](./20251217-testing-strategy.md)

## Resources

| Option | Documentation | Repository |
|--------|---------------|------------|
| testing/synctest | [pkg.go.dev](https://pkg.go.dev/testing/synctest) | [golang/go](https://github.com/golang/go) |
| jonboulle/clockwork | [pkg.go.dev](https://pkg.go.dev/github.com/jonboulle/clockwork) | [GitHub](https://github.com/jonboulle/clockwork) |
| coder/quartz | [pkg.go.dev](https://pkg.go.dev/github.com/coder/quartz) | [GitHub](https://github.com/coder/quartz) |
| benbjohnson/clock | [pkg.go.dev](https://pkg.go.dev/github.com/benbjohnson/clock) | [GitHub](https://github.com/benbjohnson/clock) (archived) |

## Sources

- [Testing concurrent code with testing/synctest - Go Blog](https://go.dev/blog/synctest)
- [Testing Time (and other asynchronicities) - Go Blog](https://go.dev/blog/testing-time)
- [synctest package - Go Packages](https://pkg.go.dev/testing/synctest)
