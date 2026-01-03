# Design: pass-user-name-to-llm

## Overview

Fetch LINE user profile on Follow event, store with caching and persistence, and include profile information in LLM requests for personalized responses.

## File Structure

| File | Purpose |
|------|---------|
| `internal/line/profile.go` | LINE API GetProfile method |
| `internal/profile/profile.go` | Profile service with sync.Map cache + GCS storage |
| `internal/bot/handler.go` | Add ProfileService dependency, HandleFollow, build context message |
| `internal/bot/template/user_profile.txt` | Template for context message with user profile |
| `internal/bot/template/user_header.txt` | Template for message header `[UserName\|LocalTime]` |
| `internal/line/server.go` | Add HandleFollow dispatch for Follow events |
| `main.go` | Wire profile bucket and profile service |

## Interfaces

### `internal/line/profile.go`

```go
// UserProfile contains LINE user profile information.
type UserProfile struct {
    DisplayName   string
    PictureURL    string
    StatusMessage string
}

// GetProfile fetches user profile from LINE API.
func (c *Client) GetProfile(ctx context.Context, userID string) (*UserProfile, error)
```

### `internal/profile/profile.go`

```go
// UserProfile contains user profile information for storage.
type UserProfile struct {
    DisplayName     string `json:"displayName"`
    PictureURL      string `json:"pictureUrl,omitempty"`
    PictureMIMEType string `json:"pictureMimeType,omitempty"`
    StatusMessage   string `json:"statusMessage,omitempty"`
}

// Service provides user profile management with caching and persistence.
type Service struct {
    storage storage.Storage
    logger  *slog.Logger
    cache   sync.Map // userID -> *UserProfile
}

func NewService(storage storage.Storage, logger *slog.Logger) *Service

func (s *Service) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error)

func (s *Service) SetUserProfile(ctx context.Context, userID string, profile *UserProfile) error
```

### `internal/bot/handler.go`

```go
// LineClient provides access to LINE API.
type LineClient interface {
    GetMessageContent(messageID string) (data []byte, mimeType string, err error)
    GetProfile(ctx context.Context, userID string) (*line.UserProfile, error)
}

// ProfileService provides access to user profiles.
type ProfileService interface {
    GetUserProfile(ctx context.Context, userID string) (*profile.UserProfile, error)
    SetUserProfile(ctx context.Context, userID string, profile *profile.UserProfile) error
}

func (h *Handler) HandleFollow(ctx context.Context) error
```

### `internal/line/server.go`

```go
// Handler interface (add HandleFollow)
type Handler interface {
    HandleText(ctx context.Context, text string) error
    HandleImage(ctx context.Context, messageID string) error
    HandleSticker(ctx context.Context, packageID, stickerID string) error
    HandleVideo(ctx context.Context, messageID string) error
    HandleAudio(ctx context.Context, messageID string) error
    HandleLocation(ctx context.Context, latitude, longitude float64) error
    HandleUnknown(ctx context.Context) error
    HandleFollow(ctx context.Context) error
}
```

## Data Flow

### Profile Fetch Flow (on Follow event)

```
1. User follows bot
2. LINE Webhook -> Server.dispatchFollow()
3. Handler.HandleFollow(ctx)
   +-- lineClient.GetProfile(userID)
   +-- fetchPictureMIMEType(pictureURL)  // HEAD request for MIME type
   +-- profileService.SetUserProfile(userID, profile)
       +-- cache.Store(userID, profile)
       +-- storage.Write(userID, json)
```

### Profile Usage Flow (on message)

```
1. User sends message
2. Handler.handleMessage(ctx, userMsg)
   +-- history.GetHistory(sourceID)
   +-- history.PutHistory(sourceID, messages)
   +-- (parallel)
       +-- buildContextMessage(userID)
       |   +-- profileService.GetUserProfile(userID)
       |       +-- cache.Load(userID) -> hit: return
       |       +-- storage.Read(userID) -> cache.Store -> return
       +-- convertToAgentHistory(hist, getUsername)
           +-- getUsername(userID)
               +-- profileService.GetUserProfile(userID)
3. agent.Generate([contextMsg, ...agentHistory])
```

### LLM Input Format

```
Message 1 (context):
  [[context.user_profiles]]
  display_name: Alice
  description: Hello!
  [User's avatar image]

Message 2 (history):
  [Alice|Jan 3(Fri) 8:30PM]
  Hello

Message 3 (history):
  [Assistant response]

Message 4 (history):
  [Alice|Jan 3(Fri) 8:31PM]
  How are you?
```

## Templates

### `user_profile.txt`

```
[[context.user_profiles]]
display_name: {{.DisplayName}}
description: {{.StatusMessage}}
```

### `user_header.txt`

```
[{{.UserName}}|{{.LocalTime}}]
```

## Notes

- Profile is only fetched on Follow event (1:1 chat)
- For group/room chats, if user hasn't followed the bot, profile won't exist and "Unknown User" is used
- Profile service uses dedicated profile bucket (separate from media bucket)
- Missing profile fields (e.g., no status message) are treated as empty strings
