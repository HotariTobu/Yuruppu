# Refactor: Bot Callback Architecture

> Restructure bot package to use callback pattern for proper layer separation.

## Overview

Refactor the bot package (renamed to `line`) to separate infrastructure concerns (webhook handling, message sending) from application logic (LLM calls, response generation) using a callback-based architecture with asynchronous event processing.

## Background & Purpose

The current bot package has the following problems:

1. **Mixed responsibilities**: Webhook handling, LLM calls, and message sending are all in one function (`processMessageEvent`)
2. **Duplicate interfaces**: `bot.LLMProvider` and `llm.Provider` have identical definitions
3. **Misplaced layer**: `SystemPrompt` (domain knowledge) exists in llm package (infrastructure layer)
4. **Hard to test**: Relies on global variables, making `t.Parallel()` impossible

By introducing the callback pattern, we invert dependencies and clarify each layer's responsibilities.

**LINE Platform Recommendation**: "We recommend that you process webhook events asynchronously. This is to prevent subsequent requests to wait until the current request is processed."

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
├─ line.NewServer() creates webhook server
├─ line.NewClient() creates LINE messaging client
├─ llm.NewVertexAIClient() creates LLM client
├─ yuruppu.NewHandler(llmClient, lineClient) creates handler
├─ server.OnMessage(handler.HandleMessage) registers callback
└─ http.ListenAndServe()

internal/line/ (Infrastructure layer - LINE communication)
├─ server.go
│   ├─ NewServer(channelSecret) → *Server
│   ├─ Server.OnMessage(callback MessageHandler)
│   └─ Server.HandleWebhook() - Verifies, returns 200, calls callback async
├─ client.go
│   ├─ NewClient(channelToken) → *Client
│   └─ Client.SendReply(replyToken, text) - Sends reply via LINE API
└─ types.go
    └─ Message struct - ReplyToken, Type, Text, UserID

internal/yuruppu/ (Application + Domain layer)
├─ handler.go
│   ├─ NewHandler(llmProvider, lineClient) → *Handler
│   └─ Handler.HandleMessage(ctx, msg) - Calls LLM, sends reply
└─ prompt.go
    └─ SystemPrompt - Yuruppu character definition

internal/llm/ (Infrastructure layer - LLM communication)
├─ provider.go - Provider interface
└─ vertexai.go - Implementation (prompt.go DELETED)
```

## Scope

- [ ] SC-001: Create `internal/yuruppu` package with Handler and SystemPrompt
- [ ] SC-002: Rename `internal/bot` to `internal/line` and refactor to callback-based architecture
- [ ] SC-003: Split line package into Server (webhook) and Client (messaging)
- [ ] SC-004: Remove duplicate LLMProvider interface from bot
- [ ] SC-005: Delete `internal/llm/prompt.go`
- [ ] SC-006: Update `main.go` to wire new architecture
- [ ] SC-007: Update tests for new architecture

## Breaking Changes

**External API**: None - HTTP webhook endpoint remains unchanged.

**Internal API** (for documentation purposes):
- `internal/bot` package renamed to `internal/line`
- `bot.SetDefaultLLMProvider()` removed
- `bot.SetDefaultMessageSender()` removed
- `bot.SetDefaultBot()` removed
- `bot.HandleWebhook` becomes `line.Server.HandleWebhook()`

## Acceptance Criteria

### AC-001: Webhook Behavior Unchanged [SC-002, SC-006]

- **Given**: LINE platform sends a webhook request with a text message
- **When**: The webhook is processed
- **Then**:
  - LLM is called with the user message
  - Reply is sent back via LINE API
  - HTTP 200 is returned

### AC-002: Asynchronous Callback Execution [SC-002]

- **Given**: line.Server receives a webhook request with valid signature
- **When**: Request is processed
- **Then**:
  - Signature is verified synchronously
  - Events are parsed synchronously
  - HTTP 200 is returned synchronously (response written to client)
  - **After** HTTP response is sent, callback is invoked in a goroutine for each MessageEvent
  - HTTP response time does not depend on callback execution time

### AC-003: Callback Flow Works [SC-001, SC-002, SC-003]

- **Given**: yuruppu.Handler is created with LLM provider and LINE client
- **When**: server.HandleWebhook receives an HTTP request containing a MessageEvent
- **Then**:
  - line.Server extracts message and calls the registered callback asynchronously
  - Callback receives `line.Message` containing ReplyToken, Type, Text, UserID
  - Callback calls `lineClient.SendReply(replyToken, text)` to send response

### AC-004: Layer Separation Achieved [SC-001, SC-002, SC-003, SC-005]

- **Given**: Refactoring is complete
- **When**: Import dependencies are analyzed
- **Then**:
  - `internal/line` does NOT import `internal/llm`
  - `internal/line` does NOT import `internal/yuruppu`
  - `internal/yuruppu` imports `internal/line` (for Message, Client types)
  - `internal/yuruppu` imports `internal/llm` (for Provider interface)
  - SystemPrompt is defined in `internal/yuruppu`, not `internal/llm`
  - Circular dependency is avoided: line.Server holds a callback function of type MessageHandler

### AC-005: No Duplicate Interfaces [SC-004]

- **Given**: Refactoring is complete
- **When**: Codebase is searched for LLM provider interfaces
- **Then**:
  - Only `llm.Provider` exists
  - `bot.LLMProvider` is removed

### AC-006: Tests Run Without Global State [SC-007]

- **Given**: Refactored test code
- **When**: Tests are run
- **Then**:
  - All tests pass
  - No global variable manipulation required (SetDefaultLLMProvider etc.)
  - line package tests use constructor injection (passing mocks to New())

### AC-007: All Tests Pass [SC-007]

- **Given**: All scope items are implemented
- **When**: `go test ./...` is run
- **Then**:
  - All tests pass
  - No regression in functionality

### AC-008: Callback Error Handling [SC-001, SC-002]

- **Given**: Callback returns an error or panics
- **When**: Error occurs during async callback execution
- **Then**:
  - Panics are recovered using defer/recover
  - Errors are logged at ERROR level with context (replyToken, userID)
  - No reply is sent to user
  - HTTP 200 was already returned (not affected by callback errors)
  - Goroutine exits cleanly (no leak)

## Implementation Notes

### Separation of Concerns

**line.Server** handles:
- Receiving webhook HTTP requests
- Verifying X-Line-Signature
- Parsing webhook payload into Message structs
- Returning HTTP 200 synchronously
- Calling registered callback asynchronously (goroutine)

**line.Client** handles:
- Sending messages via LINE Messaging API
- Managing channel access token
- HTTP communication with LINE API

**yuruppu.Handler** handles:
- Receiving Message from callback
- Calling LLM with SystemPrompt + user message
- Calling line.Client to send reply
- Error handling and logging

### Goroutine Execution Model

- **One goroutine per MessageEvent**: Each MessageEvent spawns a new goroutine
- **No concurrency limits**: Unbounded goroutines (accepted risk for simplicity; this bot has low traffic)
- **No worker pool**: Not needed for current scale
- **Panic recovery**: Each goroutine wraps callback in defer/recover

### Context Propagation

- Callback receives `context.Background()` with configured timeout (not HTTP request context)
- HTTP request context is not used because HTTP response has already been sent
- Timeout is configurable (default: 30 seconds, matching current LLM_TIMEOUT_SECONDS)

### Type Definitions

```go
// internal/line/types.go

// Message represents an incoming LINE message
type Message struct {
    ReplyToken string
    Type       string  // "text", "image", "sticker", etc.
    Text       string  // For text messages; formatted for others
    UserID     string
}

// MessageHandler is the callback signature for message processing
type MessageHandler func(ctx context.Context, msg Message) error
```

```go
// internal/line/client.go

// Client sends messages via LINE Messaging API
type Client struct {
    // ...
}

func (c *Client) SendReply(replyToken string, text string) error {
    // Calls LINE ReplyMessage API
}
```

### Sequence Flow

```
HTTP POST /webhook
    │
    ▼
line.Server.HandleWebhook(w, r)
    │
    ├─ 1. Verify X-Line-Signature (sync)
    ├─ 2. Parse webhook events into []Message (sync)
    ├─ 3. w.WriteHeader(http.StatusOK) (sync) ← HTTP response sent
    │
    └─ 4. for each MessageEvent:
           go func() {                         ← New goroutine
               defer recover()                 ← Panic protection
               ctx := context.WithTimeout(background, timeout)
               err := callback(ctx, msg)
               if err != nil {
                   log.Error(...)
               }
           }()
```

### Note on Reply Token

Reply tokens are valid for a limited time (typically around 30 seconds). If LLM takes longer than this, the reply will fail. In this case:
- Error is logged with "reply token expired" context
- No retry is attempted (reply tokens are single-use)

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-26 | 1.0 | Initial version | - |
| 2025-12-26 | 1.1 | Address spec-reviewer feedback: clarify callback responsibility, add Message/Replier types, add AC-007 for error handling, specify import dependencies, translate Background to English | - |
| 2025-12-26 | 1.2 | Rename bot→line, merge app+yuruppu→yuruppu, separate Server/Client responsibilities, add AC-008 for HTTP/LINE separation | - |
| 2025-12-26 | 1.3 | Clarify async processing: HTTP 200 returned immediately, callback executed in goroutine. Add AC-002 for async behavior. Reference LINE Platform recommendation. | - |
| 2025-12-26 | 1.4 | Address spec-reviewer feedback: clarify AC-002 timing (sync/async steps), add goroutine execution model, add panic recovery to AC-008, add context propagation details, improve sequence flow diagram | - |
