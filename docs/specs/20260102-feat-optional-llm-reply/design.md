# Design: 20260102-feat-optional-llm-reply

## Overview

Enable LLM to decide whether to send a reply by using a dedicated `reply` tool. If the tool is not called, no message is sent.

## File Structure

| File | Action | Purpose |
|------|--------|---------|
| `internal/line/context.go` | Create | Context keys and accessor functions for replyToken, sourceID, userID |
| `internal/line/server.go` | Modify | Remove `MessageContext`, set ctx values via `WithValue` |
| `internal/line/server_test.go` | Modify | Update tests for new handler signatures |
| `internal/bot/handler.go` | Modify | New handler signatures, use `line.*FromContext()`, delegate reply to tool, DEBUG log response contents |
| `internal/bot/handler_test.go` | Modify | Update tests for new signatures |
| `internal/bot/media_test.go` | Modify | Update tests for new signatures |
| `internal/toolset/reply/reply.go` | Create | Reply tool implementation |
| `internal/toolset/reply/reply_test.go` | Create | Test reply tool (table-driven, testify) |
| `internal/toolset/reply/parameters.json` | Create | Strict schema validation |
| `internal/toolset/reply/response.json` | Create | Strict schema validation |
| `internal/yuruppu/prompt/system.txt` | Modify | Add reply tool usage instructions |
| `main.go` | Modify | Register reply tool with dependencies |

Note: Prototype commit (`49e3551`) will be reverted before implementation.

---

## internal/line/context.go (Create)

Both `bot` and `reply` packages import `line` for context accessors, keeping them independent of each other.

```go
package line

type ctxKey int

const (
	ctxKeyReplyToken ctxKey = iota
	ctxKeySourceID
	ctxKeyUserID
)

func WithReplyToken(ctx context.Context, token string) context.Context
func WithSourceID(ctx context.Context, id string) context.Context
func WithUserID(ctx context.Context, id string) context.Context
func ReplyTokenFromContext(ctx context.Context) (string, bool)
func SourceIDFromContext(ctx context.Context) (string, bool)
func UserIDFromContext(ctx context.Context) (string, bool)
```

---

## internal/line/server.go (Modify)

- Delete `MessageContext` struct
- Update `Handler` interface: remove `msgCtx MessageContext` parameter from all methods
- Update `invokeHandler`: set context values via `WithReplyToken`, `WithSourceID`, `WithUserID` before calling handler

---

## internal/toolset/reply/reply.go (Create)

```go
type Sender interface {
	SendReply(replyToken string, text string) error
}

type HistoryRepository interface {
	GetHistory(ctx context.Context, sourceID string) ([]history.Message, int64, error)
	PutHistory(ctx context.Context, sourceID string, messages []history.Message, expectedGeneration int64) (int64, error)
}

type Tool struct {
	sender  Sender
	history HistoryRepository
	logger  *slog.Logger
}

func NewTool(sender Sender, historyRepo HistoryRepository, logger *slog.Logger) *Tool
func (t *Tool) Name() string
func (t *Tool) Description() string
func (t *Tool) ParametersJsonSchema() []byte
func (t *Tool) ResponseJsonSchema() []byte
func (t *Tool) Callback(ctx context.Context, validatedArgs map[string]any) (map[string]any, error)
```

### Callback processing:
1. Get `replyToken` and `sourceID` from context (error: "internal error")
2. `GetHistory` (error: "failed to load conversation")
3. `SendReply` (error: "failed to send reply")
4. Append `AssistantMessage` with text
5. `PutHistory` (error: "failed to save message")
6. Return `{"status": "sent"}`

Note: Log detailed errors internally, return LLM-friendly messages.

---

## internal/toolset/reply/parameters.json (Create)

```json
{
  "type": "object",
  "properties": {
    "message": {
      "type": "string",
      "description": "The reply message to send to the user",
      "minLength": 1,
      "maxLength": 5000
    }
  },
  "required": ["message"],
  "additionalProperties": false
}
```

---

## internal/toolset/reply/response.json (Create)

```json
{
  "type": "object",
  "properties": {
    "status": {
      "type": "string",
      "enum": ["sent"],
      "description": "The status of the reply"
    }
  },
  "required": ["status"],
  "additionalProperties": false
}
```

---

## internal/bot/handler.go (Modify)

- Update all `Handle*` method signatures: remove `msgCtx line.MessageContext` parameter
- Update `handleMessage`: use `line.SourceIDFromContext(ctx)` instead of `msgCtx.SourceID`
- Remove reply logic: don't call `SendReply`, don't save assistant message
- Add DEBUG log for response contents (no longer used for reply)
- Remove `Sender` field from `Handler` struct
- Update `NewHandler`: remove `sender` parameter
- Delete unused code

---

## internal/yuruppu/prompt/system.txt (Modify)

Add section explaining:
- Use `reply` tool to send messages
- Don't call tool if no reply needed
- Call `reply` tool only once

---

## main.go (Modify)

- Add import `yuruppu/internal/toolset/reply`
- Create reply tool: `reply.NewTool(lineClient, historyRepo, logger)`
- Add `replyTool` to agent tools
- Remove `lineClient` (Sender) from `bot.NewHandler` call

---

## Data Flow

```
1. LINE Webhook
   |
   v
2. line.Server.invokeHandler()
   - ctx = WithReplyToken(ctx, replyToken)
   - ctx = WithSourceID(ctx, sourceID)
   - ctx = WithUserID(ctx, userID)
   - handler.HandleText(ctx, text)
   |
   v
3. bot.Handler.handleMessage()
   - sourceID = line.SourceIDFromContext(ctx)
   - GetHistory(sourceID) -> hist, gen
   - Append user message to hist
   - PutHistory(sourceID, hist, gen)
   - Convert hist to agent format
   - response = agent.Generate(ctx, agentHistory)
   - DEBUG log response contents
   |
   v
4. agent.GeminiAgent.Generate()
   - LLM decides: reply or not?
   |
   +--> [Reply needed] LLM calls reply tool
   |      - tool.Use() -> Callback(ctx, validatedArgs)
   |        - replyToken = line.ReplyTokenFromContext(ctx)
   |        - sourceID = line.SourceIDFromContext(ctx)
   |        - GetHistory(sourceID) -> hist, gen
   |        - SendReply(replyToken, message)
   |          (2nd call fails with LINE API error)
   |        - Append assistant message to hist
   |        - PutHistory(sourceID, hist, gen)
   |      - Return {"status": "sent"}
   |
   +--> [No reply needed] LLM does not call reply tool
          - No message sent
          - No assistant message in history
   |
   v
5. Handler returns nil (success)
```

