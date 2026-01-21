# Design: event-flex-message

## Overview

`list_events` tool sends Flex Message directly to LINE. `get_event` tool does not exist.

## File Structure

| File | Purpose |
|------|---------|
| `internal/line/client/message.go` | `SendFlexReply` accepts `[]byte` for flex JSON |
| `internal/toolset/event/list/list.go` | Sends Flex Message via LineClient |
| `internal/toolset/event/list/flex.json` | Go template for event carousel |
| `internal/toolset/event/list/response.json` | Status-based response schema |
| `internal/toolset/event/event.go` | Provides create, list, update, remove tools |
| `cmd/cli/mock/lineclient.go` | Mock for CLI testing |

### Tests

| File | Purpose |
|------|---------|
| `internal/line/client/message_test.go` | Tests for `SendFlexReply` |
| `internal/toolset/event/list/list_test.go` | Tests for list tool behavior |
| `internal/toolset/event/event_test.go` | Tests for event toolset |

## Interfaces

### LineClient.SendFlexReply

```go
SendFlexReply(replyToken string, altText string, flexJSON []byte) error
```

### UserProfileService (for list tool)

```go
type UserProfileService interface {
    GetUserProfile(ctx context.Context, userID string) (*userprofile.UserProfile, error)
}
```

## Data Flow

### list_events (events exist)

1. **Input**: Tool receives `args` (created_by_me, start, end) + context (userID, replyToken)
2. **Process**:
   - Build `ListOptions` from args
   - `EventService.List()` → `[]*event.Event`
   - For each event with `ShowCreator=true`: `UserProfileService.GetUserProfile()` → creator name
   - Render `flex.json` template → `[]byte`
   - `LineClient.SendFlexReply(replyToken, altText, flexJSON)`
3. **Output**: `{"status": "sent"}` + `IsFinal()` returns true

### list_events (no events)

1. **Input**: Same as above
2. **Process**:
   - Build `ListOptions` from args
   - `EventService.List()` → empty slice
3. **Output**: `{"status": "no_events"}` + `IsFinal()` returns false (LLM continues)
