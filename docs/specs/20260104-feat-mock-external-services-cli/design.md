# Design: mock-external-services-cli

## Overview

CLI tool for local conversation testing. Mocks LINE API and GCS while using real LLM and Weather API.

## File Structure

| File | Purpose |
|------|---------|
| `cmd/cli/main.go` | Entry point: flag parsing, DI, startup |
| `cmd/cli/repl/repl.go` | REPL loop, Ctrl+C/`/quit` handling, Handler invocation |
| `cmd/cli/setup/setup.go` | Data directory existence check and creation prompt |
| `cmd/cli/profile/profile.go` | New user profile input prompts |
| `cmd/cli/mock/lineclient.go` | Mock implementation of `bot.LineClient` + `reply.LineClient` |
| `cmd/cli/mock/storage.go` | Mock implementation of `storage.Storage` using local filesystem |

## Interfaces

### setup/setup.go

```go
// EnsureDataDir checks if dataDir exists. If not, prompts user for confirmation.
// Returns error if user declines or directory creation fails.
func EnsureDataDir(dataDir string, stdin io.Reader, stderr io.Writer) error
```

### profile/profile.go

```go
// PromptNewProfile prompts user to enter profile fields.
// Display name is required (re-prompts if empty).
// Picture URL and status message are optional.
// If picture URL is provided, MIME type is fetched automatically.
func PromptNewProfile(ctx context.Context, stdin io.Reader, stderr io.Writer) (*profile.UserProfile, error)
```

### repl/repl.go

```go
// Config holds REPL configuration.
type Config struct {
    UserID  string
    Handler *bot.Handler
    Logger  *slog.Logger
}

// Run starts the REPL loop.
// Exits on /quit or Ctrl+C twice.
func Run(ctx context.Context, cfg Config) error
```

### mock/lineclient.go

Implements existing interfaces:
- `bot.LineClient`: `GetMessageContent` returns error (media not supported), `GetProfile` returns error (profile created via CLI prompts)
- `reply.LineClient`: `SendReply` prints message to stdout

### mock/storage.go

Implements `storage.Storage`:
- Uses local filesystem with file mtime as generation
- `GetSignedURL` returns `file://` URL

## Data Flow

### Startup Flow

```
main.go
  |
  +-- Flag parsing (--user-id, --data-dir, --message)
  |
  +-- user-id validation (pattern: [0-9a-z_]+)
  |     +-- Failure -> error output, exit(1)
  |
  +-- setup.EnsureDataDir(dataDir)
  |     +-- Does not exist -> prompt -> "n" exits
  |
  +-- Create mock.FileStorage (profiles/, history/, media/)
  |
  +-- Get profile via profile.Service
  |     +-- Does not exist -> profile.PromptNewProfile -> save
  |
  +-- Create LLM Agent (Gemini)
  |
  +-- Create mock.LineClient
  |
  +-- Create Tools (reply, weather, skip)
  |
  +-- Create bot.Handler
  |
  +-- --message provided -> handler.HandleText -> exit
      --message not provided -> repl.Run
```

### REPL Flow

```
repl.Run
  |
  +-- Setup signal handler (SIGINT)
  |
  +-- Loop
       |
       +-- Display "> " prompt
       |
       +-- Wait for input
       |     +-- Ctrl+C first -> display "Press Ctrl+C again to exit"
       |     +-- Ctrl+C second -> return
       |     +-- "/quit" -> return
       |     +-- Empty line -> continue
       |     +-- Text -> handler.HandleText
       |
       +-- handler.HandleText
            |
            +-- Set userID, sourceID, replyToken in context
            |
            +-- (internal) LLM invocation
            |
            +-- (internal) reply tool -> mock.LineClient.SendReply -> stdout
```

### Storage Structure

```
.yuruppu/              (configurable via --data-dir)
+-- profiles/
|   +-- {userID}.json
+-- history/
|   +-- {sourceID}.json    (sourceID = userID)
+-- media/
    +-- (unused in CLI)
```

## Design Decisions

- **No changes to internal/**: CLI adapts to existing interfaces
- **All mock implementations in cmd/cli/mock/**: Does not pollute internal/
- **Single message mode in main.go**: Too simple to warrant a separate package
- **Standard library only**: flag, bufio.Scanner, fmt (per ADRs)
