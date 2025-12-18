# ADR: Configuration

> Date: 2025-12-17
> Status: **Adopted**

## Context

Need to load configuration for the LINE bot. FR-005 requires loading LINE channel secret and access token from environment variables.

## Decision Drivers

- Only 2-3 environment variables needed
- Minimal dependencies preferred
- Simple validation (check if empty)

## Options Considered

- **Option 1:** `os.Getenv()` directly
- **Option 2:** envconfig (struct-based)
- **Option 3:** viper (full-featured)

## Decision

Adopt **`os.Getenv()` directly**.

No formal evaluation performed - the choice is obvious given the constraints.

## Rationale

- Only 3 environment variables: `LINE_CHANNEL_SECRET`, `LINE_CHANNEL_ACCESS_TOKEN`, `PORT`
- No external dependencies needed
- LINE Bot SDK examples use `os.Getenv()` directly
- Simple validation can be done with basic if-checks

Example:
```go
channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
if channelSecret == "" {
    log.Fatal("LINE_CHANNEL_SECRET is required")
}
```

## Consequences

**Positive:**
- No additional dependencies
- Simple and explicit
- Matches LINE Bot SDK examples

**Negative:**
- No automatic struct binding
- Manual validation required (acceptable for 3 variables)

## Related Decisions

- [20241214-line-bot-architecture.md](./20241214-line-bot-architecture.md)

## Resources

| Option | Documentation |
|--------|---------------|
| os.Getenv | [pkg.go.dev](https://pkg.go.dev/os#Getenv) |
