# Design: save-group-info

## Overview

When the bot is invited to a LINE group (join event), it retrieves the group information from LINE API and saves it to persistent storage. This enables the bot to have group context for future conversations.

## File Structure

| File | Purpose |
|------|---------|
| `internal/groupprofile/groupprofile.go` | Group profile struct and service with cache + persistence |
| `internal/groupprofile/groupprofile_test.go` | Unit tests for group profile service |
| `internal/line/client/profile.go` | Add `GetGroupSummary` method to fetch group info from LINE API |
| `internal/bot/handler.go` | Add `GroupProfileService` interface and dependency |
| `internal/bot/join.go` | Add `HandleJoin` to fetch and save group profile |
| `internal/bot/message_test.go` | Update `NewHandler` calls with new argument |
| `main.go` | Wire up group profile service with GCS storage |

## Interfaces

### GroupProfile Struct

```go
// internal/groupprofile/groupprofile.go
type GroupProfile struct {
    DisplayName string `json:"displayName"`
    PictureURL  string `json:"pictureUrl,omitempty"`
}
```

### GroupProfile Service

```go
// internal/groupprofile/groupprofile.go
type Service struct {
    storage storage.Storage
    logger  *slog.Logger
    cache   sync.Map // groupID -> *GroupProfile
}

func NewService(storage storage.Storage, logger *slog.Logger) (*Service, error)
func (s *Service) GetGroupProfile(ctx context.Context, groupID string) (*GroupProfile, error)
func (s *Service) SetGroupProfile(ctx context.Context, groupID string, profile *GroupProfile) error
```

### GroupSummary (LINE API response)

```go
// internal/line/client/profile.go
type GroupSummary struct {
    GroupID    string
    GroupName  string
    PictureURL string
}

func (c *Client) GetGroupSummary(ctx context.Context, groupID string) (*GroupSummary, error)
```

### Bot Handler Interfaces

```go
// internal/bot/handler.go
type GroupProfileService interface {
    GetGroupProfile(ctx context.Context, groupID string) (*groupprofile.GroupProfile, error)
    SetGroupProfile(ctx context.Context, groupID string, profile *groupprofile.GroupProfile) error
}

type LineClient interface {
    // ... existing methods ...
    GetGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error)
}
```

## Data Flow

### Join Event Flow

```
1. LINE sends join event to webhook
         ↓
2. lineserver.HandleWebhook parses event
         ↓
3. Handler.HandleJoin called with context (sourceID from event)
         ↓
4. lineClient.GetGroupSummary(groupID)
         ↓
5. LINE API returns GroupSummary
   {GroupID, GroupName, PictureURL}
         ↓
6. Create GroupProfile from summary
   {DisplayName, PictureURL}
         ↓
7. groupProfileService.SetGroupProfile
         ↓
8. Cache in sync.Map + persist to GCS
   (key: groupID, prefix: groupprofile/)
         ↓
9. HandleJoin returns nil
```

### Error Handling (AC-002)

- If LINE API fails: Log error, return nil (do not crash)
- If storage fails: Log error, return nil (do not crash)
- No partial data is saved on failure

### Missing Picture URL (AC-003)

- If `GroupSummary.PictureURL` is empty, `GroupProfile.PictureURL` is stored as empty string
- JSON serialization uses `omitempty`, so empty PictureURL is omitted from stored JSON

## Storage

- Storage prefix: `groupprofile/`
- Key format: `{groupID}` (e.g., `groupprofile/C1234567890abcdef`)
- Content type: `application/json`
