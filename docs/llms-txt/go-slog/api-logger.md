# Logger Type API Reference

Complete API reference for the `Logger` type.

## Type Definition

```go
type Logger struct {
    // contains filtered or unexported fields
}
```

## Creating Loggers

### New

```go
func New(h Handler) *Logger
```

Create a new Logger with the given handler.

Example:

```go
handler := slog.NewJSONHandler(os.Stdout, nil)
logger := slog.New(handler)
```

### Default

```go
func Default() *Logger
```

Return the default logger used by top-level functions.

Example:

```go
logger := slog.Default()
logger.Info("using default logger")
```

## Logging Methods

### Info

```go
func (l *Logger) Info(msg string, args ...any)
```

Log at Info level with alternating key-value pairs.

Example:

```go
logger.Info("user logged in", "user_id", 123, "username", "alice")
```

### InfoContext

```go
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any)
```

Log at Info level with context support.

Example:

```go
logger.InfoContext(ctx, "request processed", "duration_ms", 45)
```

### Debug

```go
func (l *Logger) Debug(msg string, args ...any)
```

Log at Debug level.

Example:

```go
logger.Debug("variable state", "count", count, "active", active)
```

### DebugContext

```go
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any)
```

Log at Debug level with context.

### Warn

```go
func (l *Logger) Warn(msg string, args ...any)
```

Log at Warn level.

Example:

```go
logger.Warn("slow operation detected", "duration_ms", duration)
```

### WarnContext

```go
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any)
```

Log at Warn level with context.

### Error

```go
func (l *Logger) Error(msg string, args ...any)
```

Log at Error level.

Example:

```go
logger.Error("database connection failed", "error", err, "retry_count", 3)
```

### ErrorContext

```go
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any)
```

Log at Error level with context.

### Log

```go
func (l *Logger) Log(ctx context.Context, level Level, msg string, args ...any)
```

Log at any level with alternating key-value pairs.

Example:

```go
logger.Log(ctx, slog.LevelInfo, "custom message", "key", "value")
```

### LogAttrs

```go
func (l *Logger) LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr)
```

Log at any level using pre-constructed Attr values (more efficient).

Example:

```go
logger.LogAttrs(ctx, slog.LevelInfo, "message",
    slog.Int("count", 42),
    slog.String("status", "ok"),
)
```

## Logger Configuration

### With

```go
func (l *Logger) With(args ...any) *Logger
```

Return a new Logger with additional attributes included in all logs.

Example:

```go
requestLogger := logger.With(
    "request_id", requestID,
    "user_id", userID,
)
requestLogger.Info("processing") // Includes request_id and user_id
```

### WithGroup

```go
func (l *Logger) WithGroup(name string) *Logger
```

Return a new Logger that groups all subsequent attributes under the given name.

Example:

```go
dbLogger := logger.WithGroup("database")
dbLogger.Info("connected", "host", "localhost")
// Output: ... database.host=localhost
```

## Logger Properties

### Handler

```go
func (l *Logger) Handler() Handler
```

Return the logger's handler.

Example:

```go
handler := logger.Handler()
```

### Enabled

```go
func (l *Logger) Enabled(ctx context.Context, level Level) bool
```

Check if the logger will log at the given level.

Example:

```go
if logger.Enabled(ctx, slog.LevelDebug) {
    // Only compute expensive debug data if needed
    debugData := computeExpensiveData()
    logger.Debug("debug info", "data", debugData)
}
```

## Top-Level Functions

These functions use the default logger.

### SetDefault

```go
func SetDefault(l *Logger)
```

Set the default logger used by top-level functions.

Example:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
slog.SetDefault(logger)

slog.Info("now uses JSON handler")
```

### Info

```go
func Info(msg string, args ...any)
```

Log at Info level using the default logger.

### InfoContext

```go
func InfoContext(ctx context.Context, msg string, args ...any)
```

### Debug

```go
func Debug(msg string, args ...any)
```

### DebugContext

```go
func DebugContext(ctx context.Context, msg string, args ...any)
```

### Warn

```go
func Warn(msg string, args ...any)
```

### WarnContext

```go
func WarnContext(ctx context.Context, msg string, args ...any)
```

### Error

```go
func Error(msg string, args ...any)
```

### ErrorContext

```go
func ErrorContext(ctx context.Context, msg string, args ...any)
```

### Log

```go
func Log(ctx context.Context, level Level, msg string, args ...any)
```

### LogAttrs

```go
func LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr)
```

## Standard Library Integration

### NewLogLogger

```go
func NewLogLogger(h Handler, level Level) *log.Logger
```

Create a `*log.Logger` that writes to the given handler at the specified level.

Example:

```go
handler := slog.NewJSONHandler(os.Stdout, nil)
stdLogger := slog.NewLogLogger(handler, slog.LevelInfo)

stdLogger.Println("message") // Uses slog handler
```

### SetLogLoggerLevel

```go
func SetLogLoggerLevel(level Level) (oldLevel Level)
```

Set the log level for the default logger's `log.Logger` compatibility.

## Usage Patterns

### Basic Logging

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("application started", "port", 8080)
```

### Request-Scoped Logger

```go
func HandleRequest(ctx context.Context, req Request) {
    logger := slog.Default().With(
        "request_id", req.ID,
        "method", req.Method,
        "path", req.Path,
    )

    logger.Info("request received")
    // ... process request ...
    logger.Info("request completed", "status", 200)
}
```

### Component Logger

```go
type Database struct {
    logger *slog.Logger
}

func NewDatabase() *Database {
    return &Database{
        logger: slog.Default().WithGroup("database"),
    }
}

func (db *Database) Connect() error {
    db.logger.Info("connecting", "host", "localhost")
    // ...
}
```

### High-Performance Logging

```go
if logger.Enabled(ctx, slog.LevelDebug) {
    logger.LogAttrs(ctx, slog.LevelDebug, "detailed info",
        slog.Int("count", count),
        slog.Duration("elapsed", elapsed),
        slog.Any("data", expensiveData()),
    )
}
```

### Error Logging with Context

```go
if err != nil {
    logger.ErrorContext(ctx, "operation failed",
        "error", err,
        "operation", "database_query",
        "query", query,
        "retry_count", retryCount,
    )
    return err
}
```

## Performance Notes

1. **Use LogAttrs**: For maximum performance, use `LogAttrs` with pre-constructed `Attr` values
2. **Check Enabled**: Check `Enabled()` before computing expensive log values
3. **Reuse Loggers**: Create loggers with `With()` once and reuse them
4. **Avoid String Formatting**: Use structured key-value pairs instead of formatted strings

## See Also

- [Handler Interface](api-handler.md) - Handler API reference
- [Attr and Value Types](api-attr-value.md) - Attribute and value API
- [Best Practices](best-practices.md) - Performance and usage patterns
