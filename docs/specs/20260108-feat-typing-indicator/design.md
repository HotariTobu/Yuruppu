# Design: Typing Indicator

## Overview

Display a loading indicator on LINE only when message processing takes longer than a configured delay time.

## File Structure

| File | Purpose |
|------|---------|
| `internal/bot/handler.go` | Add HandlerConfig, update LineClient interface |
| `internal/bot/message.go` | Add delayed loading logic in handleMessage |
| `internal/line/client/client.go` | No changes |
| `internal/line/client/loading.go` | ShowLoadingAnimation method (add timeout parameter) |
| `main.go` | Load and validate TypingIndicatorDelay/TypingIndicatorTimeout |
| `cmd/cli/mock/lineclient.go` | Update mock implementation |

## Interfaces

### HandlerConfig

```go
// HandlerConfig holds handler configuration.
type HandlerConfig struct {
    TypingIndicatorDelay   time.Duration  // time to wait before showing indicator (default 3s)
    TypingIndicatorTimeout time.Duration  // indicator display duration (5-60s)
}
```

### Handler

```go
type Handler struct {
    lineClient     LineClient
    profileService ProfileService
    history        HistoryService
    media          MediaService
    agent          Agent
    config         HandlerConfig
    logger         *slog.Logger
}

func NewHandler(
    lineClient LineClient,
    profileService ProfileService,
    history HistoryService,
    media MediaService,
    agent Agent,
    config HandlerConfig,
    logger *slog.Logger,
) *Handler
```

### LineClient Interface

```go
type LineClient interface {
    GetMessageContent(messageID string) (data []byte, mimeType string, err error)
    GetProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error)
    ShowLoadingAnimation(ctx context.Context, chatID string, timeout time.Duration) error
}
```

### ShowLoadingAnimation

```go
// ShowLoadingAnimation displays a loading animation in a 1:1 chat.
// timeout is converted to seconds (5-60) for LINE API.
func (c *Client) ShowLoadingAnimation(ctx context.Context, chatID string, timeout time.Duration) error {
    loadingSeconds := int32(timeout.Seconds())
    req := &messaging_api.ShowLoadingAnimationRequest{
        ChatId:         chatID,
        LoadingSeconds: loadingSeconds,
    }
    _, err := c.api.ShowLoadingAnimation(req)
    return err
}
```

## Data Flow

```
handleMessage starts
       │
       ▼
Extract sourceID, userID
       │
       ▼
Is 1:1 chat? (sourceID == userID)
       │
       ├─ No ──────────────────────────┐
       │                               │
       ▼ Yes                           │
done := make(chan struct{})            │
defer close(done)                      │
       │                               │
       ▼                               │
go func() {                            │
    select {                           │
    case <-time.After(delay):          │
        // Still processing → show     │
        ShowLoadingAnimation(...)      │
    case <-done:                       │
        // Completed → do nothing      │
        return                         │
    }                                  │
}()                                    │
       │                               │
       ▼◄──────────────────────────────┘
Normal message processing
(history load, agent.Generate, etc.)
       │
       ▼
Processing complete (reply/skip/error)
       │
       ▼
defer close(done) executes
       │
       ▼
goroutine terminates
```

## Error Handling

- ShowLoadingAnimation failure: Log at WARN level, continue processing
- Panic in goroutine: Recover and log, continue processing

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            h.logger.WarnContext(ctx, "loading indicator goroutine panicked", slog.Any("panic", r))
        }
    }()
    // ...
}()
```

## Testing Strategy

- Unit tests: HandlerConfig validation, 1:1 chat detection logic
- Mock: Mock LineClient.ShowLoadingAnimation
- Integration: Manual testing with actual LINE API

## Environment Variables

| Variable | Description | Default | Validation |
|----------|-------------|---------|------------|
| `TYPING_INDICATOR_DELAY_SECONDS` | Delay before showing indicator (seconds) | 3 | > 0 |
| `TYPING_INDICATOR_TIMEOUT_SECONDS` | Indicator display duration (seconds) | 30 | 5-60 |
