# ADR: Project Structure

> Date: 2025-12-17
> Status: **Adopted**

## Context

Need to define the Go project structure for the Yuruppu LINE bot. The echo feature has 4 functions: `NewBot()`, `HandleWebhook()`, `HandleTextMessage()`, `FormatEchoMessage()`.

## Decision Drivers

- Small codebase initially (4 functions)
- Testability (avoid main package for logic)
- Simplicity over ceremony
- May grow with Gemini integration later

## Options Considered

- **Option 1:** Flat - all in `main.go`
- **Option 2:** cmd + internal - `cmd/yuruppu/main.go` + `internal/bot/`
- **Option 3:** Simple internal - `main.go` + `internal/bot/`

## Decision

Adopt **Option 3: Simple internal**.

```
yuruppu/
├── main.go                  # entry point only
├── internal/
│   └── bot/
│       ├── bot.go           # NewBot, HandleWebhook, HandleTextMessage
│       ├── message.go       # FormatEchoMessage
│       └── bot_test.go      # tests
├── go.mod
└── docs/
```

## Rationale

- Separates entry point from logic (testable)
- Simpler than full `cmd/` layout
- `internal/` prevents external imports
- Easy to add more packages later (e.g., `internal/gemini/`)

## Consequences

**Positive:**
- Logic is testable (not in main package)
- Clear separation of concerns
- Room to grow

**Negative:**
- Slightly more structure than flat layout

## Related Decisions

- [20241214-line-bot-architecture.md](./20241214-line-bot-architecture.md)
