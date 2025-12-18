# Logger Creation

Creating custom loggers allows you to control output format, destination, and logging behavior.

## Creating a Logger

### Basic Logger Creation

```go
import (
    "log/slog"
    "os"
)

// Create a logger with a handler
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
logger.Info("hello", "count", 3)
```

The `New` function signature:

```go
func New(h Handler) *Logger
```

## Built-in Handlers

### TextHandler

Outputs logs in a human-readable format:

```go
handler := slog.NewTextHandler(os.Stderr, nil)
logger := slog.New(handler)

logger.Info("user action", "action", "login", "user_id", 123)
// Output: time=2024-01-15T10:30:00.000-05:00 level=INFO msg="user action" action=login user_id=123
```

### JSONHandler

Outputs logs in JSON format for machine parsing:

```go
handler := slog.NewJSONHandler(os.Stdout, nil)
logger := slog.New(handler)

logger.Info("user action", "action", "login", "user_id", 123)
// Output: {"time":"2024-01-15T10:30:00.000000000-05:00","level":"INFO","msg":"user action","action":"login","user_id":123}
```

## Handler Options

Configure handler behavior with `HandlerOptions`:

```go
type HandlerOptions struct {
    AddSource   bool                                             // Include source code location
    Level       Leveler                                          // Minimum log level
    ReplaceAttr func(groups []string, a Attr) Attr              // Modify attributes
}
```

### Setting Log Level

```go
opts := &slog.HandlerOptions{
    Level: slog.LevelDebug, // Enable debug logging
}
handler := slog.NewJSONHandler(os.Stdout, opts)
logger := slog.New(handler)

logger.Debug("debug message") // Now visible
```

### Adding Source Location

Include file, line number, and function name:

```go
opts := &slog.HandlerOptions{
    AddSource: true,
}
handler := slog.NewTextHandler(os.Stderr, opts)
logger := slog.New(handler)

logger.Info("message")
// Output includes: source=/path/to/file.go:42
```

### Dynamic Log Levels

Use `LevelVar` to change log level at runtime:

```go
var programLevel = new(slog.LevelVar) // Defaults to Info

opts := &slog.HandlerOptions{
    Level: programLevel,
}
handler := slog.NewJSONHandler(os.Stderr, opts)
logger := slog.New(handler)

logger.Debug("not visible") // LevelVar is at Info

programLevel.Set(slog.LevelDebug) // Change level dynamically
logger.Debug("now visible") // This will be logged
```

### Customizing Output with ReplaceAttr

Transform or filter attributes:

```go
opts := &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        // Remove time attribute
        if a.Key == slog.TimeKey {
            return slog.Attr{}
        }
        // Redact sensitive values
        if a.Key == "password" {
            return slog.String("password", "REDACTED")
        }
        return a
    },
}
handler := slog.NewTextHandler(os.Stdout, opts)
logger := slog.New(handler)

logger.Info("login", "username", "alice", "password", "secret123")
// Output: level=INFO msg=login username=alice password=REDACTED
```

## Writing to Files

```go
file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
    panic(err)
}
defer file.Close()

handler := slog.NewJSONHandler(file, nil)
logger := slog.New(handler)
```

## Multiple Loggers

Create different loggers for different purposes:

```go
// Application logger
appLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

// Error logger to separate file
errorFile, _ := os.OpenFile("errors.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
errorLogger := slog.New(slog.NewJSONHandler(errorFile, &slog.HandlerOptions{
    Level: slog.LevelError,
}))

appLogger.Info("application started")
errorLogger.Error("critical error", "error", err)
```

## Getting the Default Logger

```go
func Default() *Logger
```

Retrieve the current default logger:

```go
logger := slog.Default()
logger.Info("using default logger")
```

## Interoperability with log Package

Create a `log.Logger` from an slog handler:

```go
func NewLogLogger(h Handler, level Level) *log.Logger
```

Example:

```go
handler := slog.NewJSONHandler(os.Stdout, nil)
oldStyleLogger := slog.NewLogLogger(handler, slog.LevelInfo)

oldStyleLogger.Println("message") // Uses slog handler but log.Logger interface
```

## Next Steps

- Learn about structured logging in [Structured Logging](structured-logging.md)
- Explore handler implementations in [Handlers](handlers.md)
- See configuration best practices in [Logger Configuration](logger-configuration.md)
