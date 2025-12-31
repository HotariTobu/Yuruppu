# Design: line-media-api

## Overview

Enable the bot to download media content (images, videos, audio, files) from LINE's Get Content API using message IDs.

## File Structure

| File | Purpose |
|------|---------|
| `internal/line/client.go` | Add `blobAPI` field, initialize in `NewClient` |
| `internal/line/media.go` | `GetMessageContent` method implementation |
| `internal/line/media_integration_test.go` | Integration test for `GetMessageContent` |

## Interfaces

### Client (modified)

```go
type Client struct {
    api     *messaging_api.MessagingApiAPI
    blobAPI *messaging_api.MessagingApiBlobAPI
    logger  *slog.Logger
}
```

### MediaContent

```go
type MediaContent struct {
    Data     []byte
    MIMEType string
}
```

### GetMessageContent

```go
func (c *Client) GetMessageContent(messageID string) (*MediaContent, error)
```

## Data Flow

1. Input: `messageID` (string) - LINE media message ID
2. Process: Call `blobAPI.GetMessageContent(messageID)`
   - Extract `Content-Type` header → `MIMEType`
   - Read body with `io.ReadAll` → `Data`
3. Output: `*MediaContent{Data, MIMEType}` or `error`

## Implementation Notes

- `Client` holds `blobAPI` field (initialized in `NewClient`)
- `GetMessageContent` does not take `context.Context` (LINE SDK does not support it)
- Integration test only, no unit test (thin wrapper over SDK)
