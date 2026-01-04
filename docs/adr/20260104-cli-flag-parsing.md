# ADR: CLI Flag Parsing Library

> Date: 2026-01-04
> Status: **Adopted**

## Context

The mock external services CLI needs to parse command-line flags (--user-id, --data-dir, --message). This decision determines which library to use for flag parsing.

## Decision Drivers

- Simple flag parsing (only 3-4 flags)
- Help text generation (-h/--help)
- Minimal dependencies
- Not building a complex multi-command CLI

## Options Considered

- **Option 1:** flag (standard library)
- **Option 2:** spf13/pflag
- **Option 3:** spf13/cobra
- **Option 4:** urfave/cli

## Evaluation

| Criterion | Weight | flag | pflag | cobra | urfave/cli |
|-----------|--------|------|-------|-------|------------|
| Functional Fit | 25% | 4 (1.00) | 5 (1.25) | 5 (1.25) | 5 (1.25) |
| Go Compatibility | 20% | 5 (1.00) | 4 (0.80) | 4 (0.80) | 4 (0.80) |
| Lightweight | 15% | 5 (0.75) | 4 (0.60) | 2 (0.30) | 3 (0.45) |
| Security | 15% | 5 (0.75) | 4 (0.60) | 4 (0.60) | 4 (0.60) |
| Documentation | 15% | 5 (0.75) | 4 (0.60) | 5 (0.75) | 4 (0.60) |
| Ecosystem | 10% | 5 (0.50) | 5 (0.50) | 5 (0.50) | 4 (0.40) |
| **Total** | 100% | **4.75** | **4.35** | **4.20** | **4.10** |

## Decision

Adopt **flag (standard library)**.

## Rationale

- Zero external dependencies for a simple local testing tool
- Standard library is guaranteed stable and maintained
- The CLI only needs 3-4 simple flags; advanced features (subcommands, POSIX-style --flags) are unnecessary
- Built-in help text generation via -h flag is sufficient

## Consequences

**Positive:**
- No external dependencies
- Guaranteed compatibility with all Go versions
- Simplest possible implementation
- Automatic help generation

**Negative:**
- Uses `-flag` style instead of POSIX `--flag` style
- No shorthand flags (e.g., -u for --user-id)
- No subcommand support (not needed for this CLI)

**Risks:**
- None significant; if more complex flag parsing is needed later, migration to pflag is straightforward (mostly API-compatible)

## Related Decisions

- [20260104-cli-repl-input.md](./20260104-cli-repl-input.md)

## Resources

| Option | Documentation | Repository |
|--------|---------------|------------|
| flag | [pkg.go.dev](https://pkg.go.dev/flag) | stdlib |
| pflag | [pkg.go.dev](https://pkg.go.dev/github.com/spf13/pflag) | [GitHub](https://github.com/spf13/pflag) |
| cobra | [cobra.dev](https://cobra.dev/) | [GitHub](https://github.com/spf13/cobra) |
| urfave/cli | [pkg.go.dev](https://pkg.go.dev/github.com/urfave/cli/v2) | [GitHub](https://github.com/urfave/cli) |
