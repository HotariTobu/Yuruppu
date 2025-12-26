# ADR: Linting Strategy

> Date: 2025-12-26
> Status: **Adopted**

<!--
ADR records decisions only. Do NOT add:
- Configuration examples or code snippets
- Version numbers
- Setup instructions or commands
-->

## Context

The project needs comprehensive linting to catch code quality issues before CI. Currently, `make preflight` only runs `go fmt` and `go vet`, missing opportunities to catch unused imports, staticcheck violations, and code simplification opportunities.

golangci-lint is the de facto standard for Go linting, aggregating 100+ linters into a single tool. The key decision is the configuration strategy: how to select which linters to enable.

## Decision Drivers

- Small codebase (15 Go files) - can address all issues upfront
- New project - no legacy code burden requiring gradual adoption
- Single developer context - no team consensus overhead
- Discovery-oriented goal - want to catch issues we might not know about

## Options Considered

- **Option 1:** Enable-all approach (enable all linters, disable specific problematic ones)
- **Option 2:** Selective approach (start with defaults, enable specific linters deliberately)
- **Option 3:** Preset approach (use `standard` preset with minimal customization)

## Evaluation

| Criterion | Weight | Enable-all | Selective | Preset |
|-----------|--------|------------|-----------|--------|
| Functional Fit | 25% | 5 (1.25) | 3 (0.75) | 3 (0.75) |
| Go Compatibility | 20% | 5 (1.00) | 5 (1.00) | 5 (1.00) |
| Lightweight | 15% | 2 (0.30) | 4 (0.60) | 4 (0.60) |
| Security | 15% | 5 (0.75) | 3 (0.45) | 3 (0.45) |
| Documentation | 15% | 4 (0.60) | 4 (0.60) | 5 (0.75) |
| Ecosystem | 10% | 4 (0.40) | 4 (0.40) | 4 (0.40) |
| **Total** | 100% | **4.30** | **3.80** | **3.95** |

**Functional Fit:** Enable-all ensures no valuable linter is overlooked. Selective/Preset risk missing important checks.

**Lightweight:** Enable-all runs all linters, slower execution. Selective/Preset are faster.

**Security:** Enable-all includes security-focused linters like gosec by default. Others may miss them.

## Decision

Adopt **Enable-all approach** with golangci-lint.

## Rationale

1. **Discovery over convenience:** For a small, new project, the upfront cost of reviewing all linter output is worthwhile to establish comprehensive quality standards from day one.

2. **No legacy burden:** With only 15 Go files and no existing technical debt, addressing all issues immediately is feasible.

3. **Learning opportunity:** Enable-all surfaces linters and best practices that selective approaches might miss entirely.

4. **Explicit disables:** When specific linters are too strict or impractical, disabling them explicitly documents the decision (vs never knowing they existed).

## Consequences

**Positive:**
- Comprehensive code quality coverage from day one
- No risk of missing valuable linters
- Explicit documentation of any disabled linters

**Negative:**
- Initial setup requires reviewing all linter output and deciding what to disable
- Slower lint execution compared to selective approaches
- May need to disable linters that conflict or are overly strict

**Risks:**
- Some linters may produce excessive false positives - mitigated by explicitly disabling them with comments explaining why

## Related Decisions

- [20251217-testing-strategy.md](./20251217-testing-strategy.md)

## Resources

| Option | Documentation | Repository |
|--------|---------------|------------|
| golangci-lint | [docs](https://golangci-lint.run/) | [repo](https://github.com/golangci/golangci-lint) |

## Sources

- [golangci-lint Configuration Guide](https://golangci-lint.run/docs/configuration/)
- [Golden config for golangci-lint](https://gist.github.com/maratori/47a4d00457a92aa426dbd48a18776322)
- [Go linters configuration, the right version](https://olegk.dev/go-linters-configuration-the-right-version)
