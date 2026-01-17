# ADR: Prohibit nolint Directives

> Date: 2026-01-17
> Status: **Adopted**

## Context

Go's `//nolint` directives suppress linter warnings. This is not a fix—it hides problems.

## Decision Drivers

- `//nolint` is a workaround, not a solution
- Suppressing warnings creates technical debt
- Linter warnings exist for a reason

## Decision

**Never use `//nolint` directives. No exceptions.**

When a linter reports a warning, fix the code to satisfy the linter.

## Rationale

If code triggers a linter warning, the code is wrong. Fix it.

## Consequences

**Positive:**
- All code meets linter standards without exceptions
- No hidden suppressions scattered across the codebase

**Negative:**
- None

## Related Decisions

- [20251226-linting-strategy.md](./20251226-linting-strategy.md)
