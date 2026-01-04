# ADR: CLI User Prompt Library

> Date: 2026-01-04
> Status: **Adopted**

## Context

The mock external services CLI needs to prompt users for profile information (display name, picture URL, status message) when creating a new user profile. This decision determines which library to use for interactive prompts.

## Decision Drivers

- Simple text input prompts (display name, URL, status message)
- Consistent with REPL input approach
- Minimal dependencies
- Basic validation (empty display name rejection)

## Options Considered

- **Option 1:** fmt.Print + bufio.Scanner (standard library)
- **Option 2:** AlecAivazis/survey
- **Option 3:** manifoldco/promptui

## Evaluation

| Criterion | Weight | fmt/bufio | survey | promptui |
|-----------|--------|-----------|--------|----------|
| Functional Fit | 25% | 4 (1.00) | 5 (1.25) | 5 (1.25) |
| Go Compatibility | 20% | 5 (1.00) | 4 (0.80) | 4 (0.80) |
| Lightweight | 15% | 5 (0.75) | 3 (0.45) | 3 (0.45) |
| Security | 15% | 5 (0.75) | 4 (0.60) | 4 (0.60) |
| Documentation | 15% | 5 (0.75) | 4 (0.60) | 4 (0.60) |
| Ecosystem | 10% | 5 (0.50) | 4 (0.40) | 3 (0.30) |
| **Total** | 100% | **4.75** | **4.10** | **4.00** |

## Decision

Adopt **fmt.Print + bufio.Scanner (standard library)**.

## Rationale

- Consistent with REPL input decision (ADR 20260104-cli-repl-input.md)
- Zero external dependencies
- Profile creation is a one-time operation per user; advanced prompt features (select menus, validation UI) are unnecessary
- Simple loop for empty name rejection is trivial to implement

## Consequences

**Positive:**
- No external dependencies
- Consistent input handling throughout CLI
- Simple implementation

**Negative:**
- No fancy UI features (colored prompts, select menus)
- No built-in validation feedback

**Risks:**
- None significant; the prompt flow is simple enough that standard library is sufficient

## Related Decisions

- [20260104-cli-repl-input.md](./20260104-cli-repl-input.md)
- [20260104-cli-flag-parsing.md](./20260104-cli-flag-parsing.md)

## Resources

| Option | Documentation | Repository |
|--------|---------------|------------|
| fmt/bufio | [pkg.go.dev](https://pkg.go.dev/bufio) | stdlib |
| survey | [pkg.go.dev](https://pkg.go.dev/github.com/AlecAivazis/survey) | [GitHub](https://github.com/AlecAivazis/survey) |
| promptui | [pkg.go.dev](https://pkg.go.dev/github.com/manifoldco/promptui) | [GitHub](https://github.com/manifoldco/promptui) |
