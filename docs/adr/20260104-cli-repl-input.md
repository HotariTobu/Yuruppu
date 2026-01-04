# ADR: CLI REPL Input Library

> Date: 2026-01-04
> Status: **Adopted**

## Context

The mock external services CLI needs to read user input in a REPL (Read-Eval-Print Loop) for interactive conversation testing. This decision determines which library to use for reading line input from stdin.

## Decision Drivers

- Simple text input for chat messages
- Cross-platform compatibility (Linux, macOS, Windows)
- Minimal dependencies for a local testing tool
- Arrow key support (nice-to-have, not critical)

## Options Considered

- **Option 1:** bufio.Scanner (standard library)
- **Option 2:** ergochat/readline
- **Option 3:** peterh/liner
- **Option 4:** c-bata/go-prompt

## Evaluation

| Criterion | Weight | bufio.Scanner | ergochat/readline | peterh/liner | c-bata/go-prompt |
|-----------|--------|---------------|-------------------|--------------|------------------|
| Functional Fit | 25% | 3 (0.75) | 5 (1.25) | 4 (1.00) | 5 (1.25) |
| Go Compatibility | 20% | 5 (1.00) | 4 (0.80) | 4 (0.80) | 4 (0.80) |
| Lightweight | 15% | 5 (0.75) | 4 (0.60) | 5 (0.75) | 4 (0.60) |
| Security | 15% | 5 (0.75) | 4 (0.60) | 3 (0.45) | 3 (0.45) |
| Documentation | 15% | 5 (0.75) | 4 (0.60) | 3 (0.45) | 3 (0.45) |
| Ecosystem | 10% | 5 (0.50) | 3 (0.30) | 2 (0.20) | 2 (0.20) |
| **Total** | 100% | **4.50** | **4.15** | **3.65** | **3.75** |

## Decision

Adopt **bufio.Scanner (standard library)**.

## Rationale

- Zero external dependencies aligns with the goal of a simple local testing tool
- Standard library is guaranteed stable and maintained
- Arrow key support was determined to be nice-to-have, not essential for the CLI's primary purpose (testing LLM conversations)
- The readline alternatives (ergochat, liner, go-prompt) add dependencies for features that aren't critical to the use case

## Consequences

**Positive:**
- No external dependencies for input handling
- Guaranteed compatibility with all Go versions
- Simplest possible implementation

**Negative:**
- No arrow key navigation (up/down for history, left/right for cursor)
- No input history between commands
- Users must retype messages if they make mistakes

**Risks:**
- If arrow key support becomes essential later, migration to ergochat/readline is straightforward

## Related Decisions

- [20260104-cli-flag-parsing.md](./20260104-cli-flag-parsing.md)

## Resources

| Option | Documentation | Repository |
|--------|---------------|------------|
| bufio.Scanner | [pkg.go.dev](https://pkg.go.dev/bufio#Scanner) | stdlib |
| ergochat/readline | [pkg.go.dev](https://pkg.go.dev/github.com/ergochat/readline) | [GitHub](https://github.com/ergochat/readline) |
| peterh/liner | [pkg.go.dev](https://pkg.go.dev/github.com/peterh/liner) | [GitHub](https://github.com/peterh/liner) |
| c-bata/go-prompt | [pkg.go.dev](https://pkg.go.dev/github.com/c-bata/go-prompt) | [GitHub](https://github.com/c-bata/go-prompt) |
