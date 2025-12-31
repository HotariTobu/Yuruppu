# Design: image-reaction

## Overview

Enable the bot to receive image messages, download and store the image, and include it in the AI agent context.

## File Structure

| File | Purpose |
|------|---------|
| `internal/bot/media.go` | MediaDownloader interface, uploadMedia method |
| `internal/bot/handler.go` | Add mediaDownloader field, modify HandleImage |
| `internal/line/media.go` | GetMessageContent implementation |
| `main.go` | Wire mediaDownloader dependency |

## Interfaces

### bot/media.go

```go
type MediaDownloader interface {
    GetMessageContent(messageID string) (data []byte, mimeType string, err error)
}

func (h *Handler) uploadMedia(ctx context.Context, sourceID, messageID string) (storageKey, mimeType string, err error)
```

### line/media.go

```go
func (c *Client) GetMessageContent(messageID string) ([]byte, string, error)
```

## Data Flow

1. LINE sends image message → HandleImage(messageID)
2. uploadMedia calls GetMessageContent(messageID) → (data, mimeType)
3. Generate UUIDv7, create storageKey = `{sourceID}/{uuidv7}`
4. Write to storage → storageKey
5. Create UserFileDataPart with storageKey, mimeType
6. Save to history, convert to agent format with signed URL
7. Agent receives image in context

## Implementation Notes

- UUIDv7: use `github.com/google/uuid` (existing indirect dependency)
- Storage key format: `{sourceID}/{uuidv7}` (per ADR)
- Fallback to placeholder on any error (per spec NFR-001): `[User sent an image, but an error occurred while loading]`
