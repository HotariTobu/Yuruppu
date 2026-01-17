# Design: cli-group-chat

## Overview

Enable group chat simulation in the CLI using `-group-id` flag. This design covers group membership management, REPL commands for user switching and invitation, and bot join event handling.

## File Structure

| File | Purpose |
|------|---------|
| `cmd/cli/groupsim/groupsim.go` | GroupSim struct and Service for persistence and member management |
| `cmd/cli/groupsim/groupsim_test.go` | Unit tests for Service |
| `cmd/cli/main.go` | Add `-group-id` flag, initialize GroupSimService |
| `cmd/cli/repl/repl.go` | Add `/switch`, `/users`, `/invite`, `/invite-bot` commands |
| `cmd/cli/repl/repl_test.go` | Unit tests for REPL commands |
| `internal/bot/join.go` | Add `HandleJoin`, `HandleMemberJoined` handlers |
| `internal/bot/join_test.go` | Unit tests for handlers |

## Interfaces

### cmd/cli/groupsim/groupsim.go

```go
package groupsim

import (
    "context"
    "yuruppu/internal/storage"
)

// groupSim is internal storage structure.
type groupSim struct {
    Members    []string `json:"members"`
    BotInGroup bool     `json:"botInGroup"`
}

// Service provides group simulation operations.
type Service struct {
    storage storage.Storage
}

func NewService(s storage.Storage) (*Service, error)

// Group lifecycle
func (s *Service) Exists(ctx context.Context, groupID string) (bool, error)
func (s *Service) Create(ctx context.Context, groupID, firstMemberID string) error

// Member management
func (s *Service) GetMembers(ctx context.Context, groupID string) ([]string, error)
func (s *Service) IsMember(ctx context.Context, groupID, userID string) (bool, error)
func (s *Service) AddMember(ctx context.Context, groupID, userID string) error

// Bot management
func (s *Service) IsBotInGroup(ctx context.Context, groupID string) (bool, error)
func (s *Service) AddBot(ctx context.Context, groupID string) error
```

### cmd/cli/repl/repl.go

```go
package repl

import (
    "context"
    "io"
    "log/slog"

    "yuruppu/cmd/cli/groupsim"
    "yuruppu/internal/profile"
)

// MessageHandler defines the interface for handling messages and events.
type MessageHandler interface {
    HandleText(ctx context.Context, text string) error
    HandleJoin(ctx context.Context) error
    HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error
}

// Config holds REPL configuration.
type Config struct {
    UserID         string
    Handler        MessageHandler
    Logger         *slog.Logger
    Stdin          io.Reader
    Stdout         io.Writer

    // Group mode (optional)
    GroupID         string
    GroupSimService *groupsim.Service
    ProfileService  *profile.Service
}
```

### internal/bot/join.go

```go
package bot

import "context"

// HandleJoin handles the bot being added to a group.
// Currently logs only (FR-020).
func (h *Handler) HandleJoin(ctx context.Context) error

// HandleMemberJoined handles members joining a group.
// Currently logs only (FR-020).
func (h *Handler) HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error
```

## Data Flow

### 1. CLI Startup (Group Mode)

```
User: ./cli -user-id alice -group-id mygroup
                    |
                    v
            Parse flags (-group-id)
                    |
                    v
        GroupSimService.Exists(groupID)
                    |
            +-------+-------+
            |               |
        not exists        exists
            |               |
            v               v
    Create(groupID,     IsMember(groupID, userID)
           userID)          |
            |         +-----+-----+
            |         |           |
            v       member     not member
      Start REPL      |           |
                      v           v
                 Start REPL   Error & Exit
```

### 2. Message Sending (Group Mode)

```
User: "Hello" (in REPL)
            |
            v
    IsBotInGroup(groupID)
            |
      +-----+-----+
      |           |
    false       true
      |           |
      v           v
   Ignore    BuildContext:
  (no-op)     - ChatType: "group"
              - SourceID: groupID
              - UserID: activeUserID
                    |
                    v
            Handler.HandleText()
                    |
                    v
               Bot Response
```

### 3. /invite Command

```
User: /invite bob
            |
            v
    AddMember(groupID, "bob")
            |
      +-----+-----+
      |           |
   already      success
   member         |
      |           v
      v     IsBotInGroup(groupID)
   Error          |
            +-----+-----+
            |           |
          false       true
            |           |
            v           v
        (done)    HandleMemberJoined(["bob"])
```

### 4. /invite-bot Command

```
User: /invite-bot
            |
            v
    IsBotInGroup(groupID)
            |
      +-----+-----+
      |           |
    true        false
      |           |
      v           v
   Error     AddBot(groupID)
                  |
                  v
          HandleJoin()
```

## REPL Commands

| Command | Description | Group Mode Only |
|---------|-------------|-----------------|
| `/quit` | Exit REPL | No |
| `/switch <user-id>` | Switch active user | Yes |
| `/users` | List group members | Yes |
| `/invite <user-id>` | Invite user to group | Yes |
| `/invite-bot` | Invite bot to group | Yes |

## Prompt Format

| Mode | Has Profile | Prompt |
|------|-------------|--------|
| 1-on-1 | Yes | `DisplayName(user-id)> ` |
| 1-on-1 | No | `(user-id)> ` |
| Group | Yes | `DisplayName(user-id)> ` |
| Group | No | `(user-id)> ` |

## Error Messages

| Scenario | Message (stderr) |
|----------|------------------|
| Non-member startup | `user '<user-id>' is not a member of group '<group-id>'` |
| /switch unknown user | `'<user-id>' is not a member of this group` |
| /invite existing member | `<user-id> is already a member of this group` |
| /invite-bot when already in | `bot is already in the group` |
| Unavailable command | `/<command> is not available` |

## Storage

- Key prefix: `groupsim/`
- Storage key: `groupsim/<group-id>`
- Format: JSON

```json
{
  "members": ["alice", "bob"],
  "botInGroup": true
}
```
