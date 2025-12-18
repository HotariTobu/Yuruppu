# Basic Usage

The `log/slog` package provides convenient top-level functions for quick logging without explicit logger creation.

## Quick Start

### Simple Logging

```go
import "log/slog"

slog.Info("application started")
slog.Debug("debug information")
slog.Warn("warning message")
slog.Error("error occurred")
```

### Structured Logging with Key-Value Pairs

The key feature of slog is structured logging using key-value pairs:

```go
slog.Info("user logged in", "user_id", 123, "username", "alice")
// Output: time=2024-01-15T10:30:00.000Z level=INFO msg="user logged in" user_id=123 username=alice

slog.Error("database connection failed", "error", err, "retry_count", 3)
```

**Arguments must alternate between keys (strings) and values (any type).**

## Top-Level Functions

### Basic Logging Methods

```go
func Debug(msg string, args ...any)
func Info(msg string, args ...any)
func Warn(msg string, args ...any)
func Error(msg string, args ...any)
```

### Context-Aware Logging

For tracing and request tracking:

```go
func DebugContext(ctx context.Context, msg string, args ...any)
func InfoContext(ctx context.Context, msg string, args ...any)
func WarnContext(ctx context.Context, msg string, args ...any)
func ErrorContext(ctx context.Context, msg string, args ...any)
```

Example:

```go
ctx := context.WithValue(context.Background(), "request_id", "abc123")
slog.InfoContext(ctx, "processing request", "path", "/api/users")
```

## Log Levels

Four standard log levels in increasing severity:

```go
const (
    LevelDebug Level = -4  // Detailed debugging information
    LevelInfo  Level = 0   // General informational messages
    LevelWarn  Level = 4   // Warning conditions
    LevelError Level = 8   // Error conditions
)
```

By default, `Debug` messages are not logged. Configure the handler to enable debug logging.

## Default Logger

Top-level functions use the default logger, which outputs to stderr in text format:

```go
// These are equivalent:
slog.Info("message", "key", "value")
slog.Default().Info("message", "key", "value")
```

## Setting the Default Logger

Replace the default logger globally:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
slog.SetDefault(logger)

// Now top-level functions use the JSON handler
slog.Info("message", "key", "value")
```

## Common Types for Values

Use any Go type as a value:

```go
slog.Info("event",
    "string", "value",
    "int", 42,
    "float", 3.14,
    "bool", true,
    "duration", 5*time.Second,
    "time", time.Now(),
    "error", err,
    "struct", user,
)
```

## Next Steps

- Learn about creating custom loggers in [Logger Creation](logger-creation.md)
- Explore structured logging patterns in [Structured Logging](structured-logging.md)
- Configure output formats in [Handlers](handlers.md)
