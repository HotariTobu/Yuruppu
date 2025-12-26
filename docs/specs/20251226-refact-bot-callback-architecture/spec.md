# Refactor: Bot Callback Architecture

> Restructure bot package to use callback pattern for proper layer separation.

## Overview

Refactor the bot package to separate infrastructure concerns (webhook handling, message sending) from application logic (LLM calls, response generation) using a callback-based architecture.

## Background & Purpose

The current bot package has the following problems:

1. **Mixed responsibilities**: Webhook handling, LLM calls, and message sending are all in one function (`processMessageEvent`)
2. **Duplicate interfaces**: `bot.LLMProvider` and `llm.Provider` have identical definitions
3. **Misplaced layer**: `SystemPrompt` (domain knowledge) exists in llm package (infrastructure layer)
4. **Hard to test**: Relies on global variables, making `t.Parallel()` impossible

By introducing the callback pattern, we invert dependencies and clarify each layer's responsibilities.

## Current Structure

```
main.go
├─ initBot() → bot.New()
├─ initLLM() → llm.NewVertexAIClient()
├─ bot.SetDefaultLLMProvider(llmProvider)
└─ http.HandleFunc("/webhook", bot.HandleWebhook)

internal/bot/bot.go
├─ HandleWebhook() - HTTP handler
├─ processMessageEvent() - Directly calls LLM and sends reply
├─ LLMProvider interface (duplicate)
├─ Global variables: llmProvider, messageSender, defaultBot
└─ Directly references llm.SystemPrompt

internal/llm/
├─ provider.go - Provider interface
├─ prompt.go - SystemPrompt (domain knowledge in infrastructure layer)
└─ vertexai.go - Implementation
```

## Proposed Structure

```
main.go
├─ bot.New() creates webhook server
├─ llm.NewVertexAIClient() creates LLM client
├─ app.New(llmClient, systemPrompt, replier) creates application handler
├─ bot.OnMessage(app.HandleMessage) registers callback
└─ bot.Start() starts server

internal/bot/bot.go (Infrastructure layer - LINE communication only)
├─ New(channelSecret, channelToken) → *Server
├─ Server.OnMessage(callback MessageHandler)
├─ Server.HandleWebhook() - Calls callback only
├─ Server.SendReply(replyToken, text) - Sends reply
├─ Message struct - Defined here, used by callback
└─ Replier interface - For app layer to send replies

internal/app/handler.go (Application layer - NEW)
├─ New(llmProvider, systemPrompt, replier) → *Handler
├─ Handler.HandleMessage(ctx, msg) - Calls LLM, sends reply via replier
└─ SystemPrompt held here

internal/llm/ (Infrastructure layer - LLM communication only)
├─ provider.go - Provider interface
├─ vertexai.go - Implementation
└─ prompt.go - DELETED

internal/yuruppu/prompt.go (Domain layer - NEW)
└─ SystemPrompt - Yuruppu character definition
```

## Scope

- [ ] SC-001: Create `internal/app` package with Handler
- [ ] SC-002: Refactor `internal/bot` to callback-based architecture
- [ ] SC-003: Move SystemPrompt to `internal/yuruppu` package
- [ ] SC-004: Remove duplicate LLMProvider interface from bot
- [ ] SC-005: Update `main.go` to wire new architecture
- [ ] SC-006: Update bot package tests to use constructor injection
- [ ] SC-007: Define Message struct and Replier interface in `internal/bot`

## Breaking Changes

**External API**: None - HTTP webhook endpoint remains unchanged.

**Internal API** (for documentation purposes):
- `bot.SetDefaultLLMProvider()` removed
- `bot.SetDefaultMessageSender()` removed
- `bot.SetDefaultBot()` removed
- `bot.HandleWebhook` signature may change to method on Server

## Acceptance Criteria

### AC-001: Webhook Behavior Unchanged [SC-002, SC-005]

- **Given**: LINE platform sends a webhook request with a text message
- **When**: The webhook is processed
- **Then**:
  - LLM is called with the user message
  - Reply is sent back via LINE API
  - HTTP 200 is returned

### AC-002: Callback Flow Works [SC-001, SC-002, SC-007]

- **Given**: app.Handler is created with LLM provider, system prompt, and replier
- **When**: bot.OnMessage(handler.HandleMessage) is called and a message arrives
- **Then**:
  - bot.Server extracts message and calls the registered callback
  - Callback receives `bot.Message` containing ReplyToken, Type, Text, UserID
  - Callback calls `replier.SendReply(replyToken, text)` to send response
  - bot.Server returns HTTP 200 regardless of callback result

### AC-003: Layer Separation Achieved [SC-001, SC-002, SC-003, SC-007]

- **Given**: Refactoring is complete
- **When**: Import dependencies are analyzed
- **Then**:
  - `internal/bot` does NOT import `internal/llm` at all
  - `internal/bot` does NOT import `internal/app`
  - `internal/app` imports `internal/bot` (for Message, Replier types)
  - `internal/app` imports `internal/llm` (for Provider interface)
  - `internal/app` imports `internal/yuruppu` (for SystemPrompt)
  - SystemPrompt is defined in `internal/yuruppu`, not `internal/llm`

### AC-004: No Duplicate Interfaces [SC-004]

- **Given**: Refactoring is complete
- **When**: Codebase is searched for LLM provider interfaces
- **Then**:
  - Only `llm.Provider` exists
  - `bot.LLMProvider` is removed

### AC-005: Tests Run Without Global State [SC-006]

- **Given**: Refactored test code
- **When**: Tests are run
- **Then**:
  - All tests pass
  - No global variable manipulation required (SetDefaultLLMProvider etc.)
  - bot package tests use constructor injection (passing mocks to New())

### AC-006: All Tests Pass [SC-006]

- **Given**: All scope items are implemented
- **When**: `go test ./...` is run
- **Then**:
  - All tests pass
  - No regression in functionality

### AC-007: Callback Error Handling [SC-001, SC-002]

- **Given**: Callback returns an error (e.g., LLM timeout)
- **When**: bot.Server handles the callback result
- **Then**:
  - Error is logged at ERROR level
  - No reply is sent to user
  - HTTP 200 is still returned (to prevent LINE retry)

## Implementation Notes

### Callback Responsibility Decision

The callback (app.HandleMessage) is responsible for calling `replier.SendReply()` directly. This design:
- Gives the application layer full control over reply content and timing
- Allows for future scenarios like multiple replies or no reply
- bot.Server only provides the Replier interface, not the logic

### Type Definitions

```go
// internal/bot/types.go

// Message represents an incoming LINE message
type Message struct {
    ReplyToken string
    Type       string  // "text", "image", "sticker", etc.
    Text       string  // For text messages; formatted for others
    UserID     string
}

// MessageHandler is the callback signature for message processing
type MessageHandler func(ctx context.Context, msg Message) error

// Replier sends replies via LINE API
type Replier interface {
    SendReply(replyToken string, text string) error
}
```

### Sequence Flow

```
HTTP POST /webhook
    │
    ▼
bot.Server.HandleWebhook()
    │
    ├─ Verify signature
    ├─ Parse webhook events
    │
    ▼ (for each MessageEvent)
bot.Server calls registered callback
    │
    ▼
app.Handler.HandleMessage(ctx, msg)
    │
    ├─ llm.Provider.GenerateText(systemPrompt, msg.Text)
    ├─ replier.SendReply(msg.ReplyToken, response)
    │
    ▼ (returns error or nil)
bot.Server
    │
    ├─ If error: log at ERROR level
    └─ Return HTTP 200
```

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-26 | 1.0 | Initial version | - |
| 2025-12-26 | 1.1 | Address spec-reviewer feedback: clarify callback responsibility, add Message/Replier types, add AC-007 for error handling, specify import dependencies, translate Background to English | - |
