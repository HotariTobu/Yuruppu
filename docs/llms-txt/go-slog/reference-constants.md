# Constants and Variables Reference

Package-level constants and variables in the log/slog package.

## Default Key Constants

These constants define the default keys used by built-in handlers:

```go
const (
    TimeKey    = "time"
    LevelKey   = "level"
    MessageKey = "msg"
    SourceKey  = "source"
)
```

### TimeKey

```go
const TimeKey = "time"
```

The default key for the timestamp in log records.

Example output:
```json
{"time":"2024-01-15T10:30:00.000Z",...}
```

### LevelKey

```go
const LevelKey = "level"
```

The default key for the log level.

Example output:
```json
{"level":"INFO",...}
```

### MessageKey

```go
const MessageKey = "msg"
```

The default key for the log message.

Example output:
```json
{"msg":"user logged in",...}
```

### SourceKey

```go
const SourceKey = "source"
```

The default key for source code location (when `AddSource: true`).

Example output:
```json
{
  "source": {
    "function": "main.main",
    "file": "/path/to/main.go",
    "line": 42
  },
  ...
}
```

## Customizing Default Keys

Use `ReplaceAttr` to customize keys:

```go
opts := &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        // Rename "msg" to "message"
        if a.Key == slog.MessageKey {
            a.Key = "message"
            return a
        }

        // Rename "level" to "severity"
        if a.Key == slog.LevelKey {
            a.Key = "severity"
            return a
        }

        // Rename "time" to "timestamp"
        if a.Key == slog.TimeKey {
            a.Key = "timestamp"
            return a
        }

        return a
    },
}

handler := slog.NewJSONHandler(os.Stdout, opts)
logger := slog.New(handler)

logger.Info("test")
// Output: {"timestamp":"...","severity":"INFO","message":"test"}
```

## Log Levels

```go
const (
    LevelDebug Level = -4
    LevelInfo  Level = 0
    LevelWarn  Level = 4
    LevelError Level = 8
)
```

### LevelDebug

```go
const LevelDebug Level = -4
```

Debug level for detailed diagnostic information. Usually disabled in production.

### LevelInfo

```go
const LevelInfo Level = 0
```

Info level for general informational messages. The default level.

### LevelWarn

```go
const LevelWarn Level = 4
```

Warn level for warning conditions that should be investigated.

### LevelError

```go
const LevelError Level = 8
```

Error level for error conditions requiring attention.

## Custom Levels

Define custom levels between standard values:

```go
const (
    LevelTrace    = slog.LevelDebug - 4  // -8
    LevelNotice   = slog.LevelInfo + 2   // 2
    LevelCritical = slog.LevelError + 4  // 12
    LevelFatal    = slog.LevelError + 8  // 16
)

func Trace(msg string, args ...any) {
    slog.Log(context.Background(), LevelTrace, msg, args...)
}

func Critical(msg string, args ...any) {
    slog.Log(context.Background(), LevelCritical, msg, args...)
}

func Fatal(msg string, args ...any) {
    slog.Log(context.Background(), LevelFatal, msg, args...)
    os.Exit(1)
}
```

## Global Variables

### DiscardHandler

```go
var DiscardHandler Handler
```

A handler that discards all log records. Useful for testing or disabling logging.

Example:

```go
// Disable all logging
logger := slog.New(slog.DiscardHandler)
logger.Info("this is not logged anywhere")
```

Performance testing:

```go
func BenchmarkLogging(b *testing.B) {
    logger := slog.New(slog.DiscardHandler)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        logger.Info("benchmark", "iteration", i)
    }
}
```

## Value Kinds

```go
const (
    KindAny       Kind = iota
    KindBool
    KindDuration
    KindFloat64
    KindInt64
    KindString
    KindTime
    KindUint64
    KindGroup
    KindLogValuer
)
```

### KindAny

```go
const KindAny Kind = iota
```

Represents any Go value (uses reflection).

### KindBool

```go
const KindBool Kind
```

Represents a boolean value.

### KindDuration

```go
const KindDuration Kind
```

Represents a `time.Duration` value.

### KindFloat64

```go
const KindFloat64 Kind
```

Represents a float64 value.

### KindInt64

```go
const KindInt64 Kind
```

Represents an int64 value (also used for int).

### KindString

```go
const KindString Kind
```

Represents a string value.

### KindTime

```go
const KindTime Kind
```

Represents a `time.Time` value.

### KindUint64

```go
const KindUint64 Kind
```

Represents a uint64 value.

### KindGroup

```go
const KindGroup Kind
```

Represents a group of attributes ([]Attr).

### KindLogValuer

```go
const KindLogValuer Kind
```

Represents a value implementing the `LogValuer` interface.

## Usage Examples

### Checking Value Kind

```go
func processValue(v slog.Value) {
    switch v.Kind() {
    case slog.KindString:
        fmt.Println("String:", v.String())
    case slog.KindInt64:
        fmt.Println("Int:", v.Int64())
    case slog.KindBool:
        fmt.Println("Bool:", v.Bool())
    case slog.KindDuration:
        fmt.Println("Duration:", v.Duration())
    case slog.KindGroup:
        fmt.Println("Group with", len(v.Group()), "attributes")
    case slog.KindLogValuer:
        resolved := v.Resolve()
        fmt.Println("LogValuer resolved to:", resolved)
    default:
        fmt.Println("Other type:", v.Any())
    }
}
```

### Custom Key Names in Handler

```go
type CustomKeyHandler struct {
    handler slog.Handler
}

func (h *CustomKeyHandler) Handle(ctx context.Context, r slog.Record) error {
    // Create a new record with custom keys
    newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

    r.Attrs(func(a slog.Attr) bool {
        // Transform standard keys
        switch a.Key {
        case slog.TimeKey:
            newRecord.AddAttrs(slog.Attr{Key: "timestamp", Value: a.Value})
        case slog.LevelKey:
            newRecord.AddAttrs(slog.Attr{Key: "severity", Value: a.Value})
        case slog.MessageKey:
            newRecord.AddAttrs(slog.Attr{Key: "message", Value: a.Value})
        default:
            newRecord.AddAttrs(a)
        }
        return true
    })

    return h.handler.Handle(ctx, newRecord)
}
```

## Package Information

- Package: `log/slog`
- Since: Go 1.21
- Import: `import "log/slog"`

## See Also

- [Level and Record Types](api-level-record.md) - Level and Record API reference
- [Attr and Value Types](api-attr-value.md) - Attr and Value API reference
- [Handler Interface](api-handler.md) - Handler API reference
