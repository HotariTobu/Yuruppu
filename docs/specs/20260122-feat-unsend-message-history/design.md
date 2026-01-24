# Design: unsend-message-history

## Overview

Handle LINE unsend events to remove recalled messages from conversation history. When a user unsends a message, the corresponding message is identified by its LINE message ID and removed from storage.

## Prototype Learnings

- UnsendEvent can be dispatched following the existing event handler pattern
- MessageID field on UserMessage struct is backward compatible with `omitempty` JSON tag
- Filter-based removal works with GCS optimistic locking
- Existing generation-based concurrency control applies to delete operations
- MessageID should be passed as handler argument, not via context (consistency with existing patterns)

## Data Flow

```
1. LINE Platform sends UnsendEvent webhook
      ↓
2. Server receives and parses UnsendEvent
      ↓
3. dispatchUnsend extracts source info (chatType, sourceID, userID)
      ↓
4. invokeUnsendHandler calls HandleUnsend(ctx, messageID)
      ↓
5. HandleUnsend loads history from storage (GetHistory)
      ↓
6. Filter removes message matching messageID from history
      ↓
7. Save updated history to storage (PutHistory with generation)
```

**Key points:**
- MessageID is passed as handler argument (not via context)
- UserMessage struct needs new `MessageID` field to enable lookup
- Filter-based removal is idempotent (missing message = no-op with warning log)
- Optimistic locking via generation parameter handles concurrency (NFR-001)

## Interfaces

### UnsendHandler (internal/line/server/unsend.go)

```go
// UnsendHandler handles LINE unsend events.
type UnsendHandler interface {
    HandleUnsend(ctx context.Context, messageID string) error
}
```

### Handler (internal/line/server/server.go)

```go
type Handler interface {
    FollowHandler
    JoinHandler
    MessageHandler
    UnsendHandler
}
```

### UserMessage (internal/history/message.go)

```go
type UserMessage struct {
    MessageID string     // LINE message ID for unsend tracking (empty for legacy messages)
    UserID    string
    Parts     []UserPart
    Timestamp time.Time
}
```

### JSON message struct (internal/history/message.go)

```go
type message struct {
    Role      string    `json:"role"`
    MessageID string    `json:"messageId,omitempty"`
    UserID    string    `json:"userId,omitempty"`
    ModelName string    `json:"modelName,omitempty"`
    Parts     []part    `json:"parts"`
    Timestamp time.Time `json:"timestamp"`
}
```

### HandleUnsend (internal/bot/unsend.go)

```go
// HandleUnsend removes a message from history when the user unsends it.
// Returns nil if the message is not found (idempotent operation).
func (h *Handler) HandleUnsend(ctx context.Context, messageID string) error
```

## File Structure

| File | Purpose |
|------|---------|
| `internal/line/server/unsend.go` | UnsendHandler interface and dispatch logic |
| `internal/line/server/server.go` | Add UnsendHandler to Handler interface, dispatch UnsendEvent |
| `internal/line/server/server_test.go` | Add HandleUnsend to mock handler |
| `internal/bot/unsend.go` | HandleUnsend implementation with filter logic |
| `internal/bot/message.go` | Pass messageID when creating UserMessage |
| `internal/history/message.go` | Add MessageID field to UserMessage and message struct |
| `internal/history/serialize.go` | Serialize MessageID to JSON |
| `internal/history/parse.go` | Parse MessageID from JSON |

## Backward Compatibility

- `MessageID` uses `omitempty` JSON tag - existing history without message IDs will parse correctly
- Old messages without MessageID cannot be unsent (acceptable limitation)
- No migration required for existing data
