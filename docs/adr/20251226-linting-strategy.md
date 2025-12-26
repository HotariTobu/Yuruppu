# ADR: Linting Strategy

> Date: 2025-12-26
> Status: **Amended** (2025-12-27)

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
- **Option 4:** Explicit comprehensive approach (explicitly list all desired linters)

## Evaluation

| Criterion | Weight | Enable-all | Selective | Preset | Explicit |
|-----------|--------|------------|-----------|--------|----------|
| Functional Fit | 25% | 5 (1.25) | 3 (0.75) | 3 (0.75) | 5 (1.25) |
| Go Compatibility | 20% | 5 (1.00) | 5 (1.00) | 5 (1.00) | 5 (1.00) |
| Lightweight | 15% | 2 (0.30) | 4 (0.60) | 4 (0.60) | 2 (0.30) |
| Security | 15% | 5 (0.75) | 3 (0.45) | 3 (0.45) | 5 (0.75) |
| Documentation | 15% | 4 (0.60) | 4 (0.60) | 5 (0.75) | 5 (0.75) |
| Ecosystem | 10% | 4 (0.40) | 4 (0.40) | 4 (0.40) | 5 (0.50) |
| **Total** | 100% | **4.30** | **3.80** | **3.95** | **4.55** |

**Functional Fit:** Enable-all and Explicit ensure no valuable linter is overlooked. Selective/Preset risk missing important checks.

**Lightweight:** Enable-all and Explicit run all linters, slower execution. Selective/Preset are faster.

**Security:** Enable-all and Explicit include security-focused linters like gosec. Others may miss them.

**Ecosystem:** Explicit approach aligns with golangci-lint v2's recommended configuration pattern.

## Decision

~~Adopt **Enable-all approach** with golangci-lint.~~ (Original)

**Amended:** Adopt **Explicit comprehensive approach** with golangci-lint v2.

## Rationale

1. **Discovery over convenience:** For a small, new project, the upfront cost of reviewing all linter output is worthwhile to establish comprehensive quality standards from day one.

2. **No legacy burden:** With only 15 Go files and no existing technical debt, addressing all issues immediately is feasible.

3. **Learning opportunity:** Comprehensive linting surfaces linters and best practices that selective approaches might miss entirely.

4. **Explicit configuration:** When specific linters are too strict or impractical, the explicit list documents exactly which linters are enabled and why others are excluded.

5. **Predictable updates:** Unlike enable-all, new linters added in future golangci-lint releases won't automatically break CI. Linter additions are deliberate.

## Consequences

**Positive:**
- Comprehensive code quality coverage from day one
- No risk of missing valuable linters
- Explicit documentation of enabled linters
- Predictable behavior across golangci-lint version upgrades

**Negative:**
- Initial setup requires reviewing all linter output and curating the list
- Slower lint execution compared to selective approaches
- Longer configuration file with explicit linter list

**Risks:**
- Some linters may produce excessive false positives - mitigated by removing them from the enable list with comments explaining why
- New useful linters may be missed - mitigated by periodic review of available linters

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

## Amendment (2025-12-27)

**Trigger:** golangci-lint v2 release changed configuration best practices.

**Change:** Migrated from enable-all approach (Option 1) to explicit comprehensive approach (Option 4).

**Reason:**
- golangci-lint v2 introduced new configuration format requiring migration
- Explicit list provides predictable behavior - new linters won't auto-enable on upgrades
- Better alignment with v2 best practices while maintaining comprehensive linting philosophy

The core philosophy (comprehensive linting) remains unchanged; only the implementation approach changed to align with v2 best practices.
