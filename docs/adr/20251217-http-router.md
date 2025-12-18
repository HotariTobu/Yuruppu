# ADR: HTTP Router

> Date: 2025-12-17
> Status: **Adopted**

## Context

Need an HTTP router/framework to handle LINE webhook requests for the Yuruppu bot. The bot only needs a single endpoint (`/webhook`).

## Decision Drivers

- Single endpoint only
- Minimal dependencies preferred
- LINE Bot SDK provides webhook parsing/validation

## Options Considered

- **Option 1:** Standard library `net/http`
- **Option 2:** Chi
- **Option 3:** Gin
- **Option 4:** Echo

## Decision

Adopt **Standard library `net/http`**.

No formal evaluation performed - the choice is obvious given the constraints.

## Rationale

- LINE Bot SDK examples use `net/http` directly
- SDK provides `webhook.ParseRequest()` for signature validation
- SDK provides `webhook.NewWebhookHandler()` as handler pattern
- Only one endpoint needed - no routing complexity
- Zero external dependencies

## Consequences

**Positive:**
- No additional dependencies
- Direct compatibility with LINE Bot SDK examples
- Simple and maintainable

**Negative:**
- Less convenient if multiple endpoints are added later (can migrate then)

## Related Decisions

- [20241214-line-bot-architecture.md](./20241214-line-bot-architecture.md)

## Resources

| Option | Documentation |
|--------|---------------|
| net/http | [pkg.go.dev](https://pkg.go.dev/net/http) |
