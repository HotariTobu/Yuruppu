# Design: group-member-count

## Overview

Track the number of members in LINE groups and pass this information to the LLM as conversation context. The member count is retrieved when the bot joins a group, updated on member join/leave events, and included in the context for group messages.

## File Structure

| File | Purpose |
|------|---------|
| `internal/groupprofile/groupprofile.go` | Add `UserCount` field to `GroupProfile` struct |
| `internal/line/client/profile.go` | Add `GetGroupMemberCount` method to fetch count from LINE API |
| `internal/bot/handler.go` | Add `GetGroupMemberCount` to `LineClient` interface |
| `internal/bot/join.go` | Implement join/member-joined/member-left handlers with count updates |
| `internal/bot/message.go` | Include user count in LLM context for group messages |
| `internal/bot/template/chat_context.txt` | Add `user_count` field to context template |
| `internal/line/server/server.go` | Add `HandleMemberLeft` to Handler interface, dispatch MemberLeftEvent |
| `internal/line/server/join.go` | Add `dispatchMemberLeft` and handler invocation |
| `internal/bot/handler_test.go` | Update mock to include `GetGroupMemberCount` |
| `internal/line/server/server_test.go` | Update mock to include `HandleMemberLeft` |
| `cmd/cli/mock/lineclient.go` | Add `groupsim.Service` dependency, implement `GetGroupMemberCount` via `len(GetMembers())` |

## Interfaces

### GroupProfile Struct

```go
// GroupProfile contains LINE group profile information.
type GroupProfile struct {
    DisplayName     string `json:"displayName"`
    PictureURL      string `json:"pictureUrl,omitempty"`
    PictureMIMEType string `json:"pictureMimeType,omitempty"`
    UserCount       int    `json:"userCount,omitempty"`
}
```

### LineClient Interface (bot package)

```go
type LineClient interface {
    GetMessageContent(messageID string) (data []byte, mimeType string, err error)
    GetUserProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error)
    GetGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error)
    GetGroupMemberCount(ctx context.Context, groupID string) (int, error)
    ShowLoadingAnimation(ctx context.Context, chatID string, timeout time.Duration) error
}
```

### LINE Client Method

```go
// GetGroupMemberCount fetches the member count of a group from LINE API.
func (c *Client) GetGroupMemberCount(ctx context.Context, groupID string) (int, error)
```

### Handler Interface (server package)

```go
type Handler interface {
    HandleFollow(ctx context.Context) error
    HandleJoin(ctx context.Context) error
    HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error
    HandleMemberLeft(ctx context.Context, leftUserIDs []string) error
    HandleText(ctx context.Context, text string) error
    HandleImage(ctx context.Context, messageID string) error
    HandleSticker(ctx context.Context, packageID, stickerID string) error
    HandleVideo(ctx context.Context, messageID string) error
    HandleAudio(ctx context.Context, messageID string) error
    HandleLocation(ctx context.Context, latitude, longitude float64) error
    HandleUnknown(ctx context.Context) error
}
```

### Bot Handler Methods

```go
// HandleJoin handles the bot being added to a group.
// Retrieves member count from LINE API and saves to group profile.
func (h *Handler) HandleJoin(ctx context.Context) error

// HandleMemberJoined handles members joining a group.
// Increments stored member count by len(joinedUserIDs).
func (h *Handler) HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error

// HandleMemberLeft handles members leaving a group.
// Decrements stored member count by len(leftUserIDs).
func (h *Handler) HandleMemberLeft(ctx context.Context, leftUserIDs []string) error
```

## Data Flow

### Flow 1: Bot Joins Group (FR-001, FR-004)

```
JoinEvent
    │
    ▼
Server.dispatchJoin()
    │
    ▼
Handler.HandleJoin()
    │
    ├──► LineClient.GetGroupSummary(groupID)
    │
    ├──► LineClient.GetGroupMemberCount(groupID)
    │         │
    │         ├── success: use returned count
    │         └── error: log warning, use fallback=1 (AC-006)
    │
    ▼
GroupProfile{DisplayName, PictureURL, UserCount}
    │
    ▼
GroupProfileService.SetGroupProfile()
```

### Flow 2: Member Joins (FR-002, FR-004)

```
MemberJoinedEvent
    │
    ▼
Server.dispatchMemberJoined()
    │
    ▼
Handler.HandleMemberJoined(joinedUserIDs)
    │
    ├──► GroupProfileService.GetGroupProfile()
    │         │
    │         └── error: log warning, return nil (graceful)
    │
    ▼
profile.UserCount += len(joinedUserIDs)
    │
    ▼
GroupProfileService.SetGroupProfile()
```

### Flow 3: Member Leaves (FR-003, FR-004)

```
MemberLeftEvent
    │
    ▼
Server.dispatchMemberLeft()
    │
    ▼
Handler.HandleMemberLeft(leftUserIDs)
    │
    ├──► GroupProfileService.GetGroupProfile()
    │         │
    │         └── error: log warning, return nil (graceful)
    │
    ▼
profile.UserCount -= len(leftUserIDs)
    │
    ▼
GroupProfileService.SetGroupProfile()
```

### Flow 4: LLM Context (FR-005)

```
MessageEvent (group chat)
    │
    ▼
Handler.HandleText()
    │
    ▼
buildContextParts()
    │
    ├──► chatType == group?
    │         │
    │         └── yes: GroupProfileService.GetGroupProfile()
    │                       │
    │                       ├── success: userCount = profile.UserCount
    │                       └── error: userCount = 0 (AC-005)
    │
    ▼
chatContextTemplate.Execute({
    CurrentLocalTime: "2026 Jan 18(Sat) 4:30PM",
    ChatType: "group",
    UserCount: 15,
})
    │
    ▼
Output:
[context]
current_local_time: 2026 Jan 18(Sat) 4:30PM
chat_type: group
user_count: 15
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| GetGroupMemberCount fails on join | Log warning, save profile with fallback UserCount=1 |
| GetGroupProfile fails on member join/leave | Log warning, skip count update |
| GetGroupProfile fails in message context | Use UserCount=0 in template |
| SetGroupProfile fails | Log warning, continue (no crash) |

## CLI Mock

```go
type GroupSim interface {
    GetMembers(ctx context.Context, groupID string) ([]string, error)
}

type LineClient struct {
    fetcher  Fetcher
    groupSim GroupSim
}

func NewLineClient(fetcher Fetcher, groupSim GroupSim) *LineClient

func (c *LineClient) GetGroupMemberCount(ctx context.Context, groupID string) (int, error)
```

## Notes

- Field name `UserCount` (not `MemberCount`) aligns with LINE API terminology
- Fallback value of 1: at least 1 user must exist to trigger any event (bot is not counted)
- No minimum bound check on decrement (out of scope per spec)
