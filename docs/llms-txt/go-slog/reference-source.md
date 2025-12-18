# Source Type Reference

Reference for the `Source` type used to track source code location in log records.

## Source Type

```go
type Source struct {
    Function string `json:"function"`
    File     string `json:"file"`
    Line     int    `json:"line"`
}
```

Represents the source code location of a log statement.

## Fields

### Function

```go
Function string `json:"function"`
```

The fully qualified function name where the log was called.

Example: `"main.handleRequest"`, `"github.com/user/pkg.(*Handler).Process"`

### File

```go
File string `json:"file"`
```

The full path to the source file.

Example: `"/Users/user/project/main.go"`

### Line

```go
Line int `json:"line"`
```

The line number in the source file.

Example: `42`

## Enabling Source Tracking

Set `AddSource: true` in `HandlerOptions`:

```go
opts := &slog.HandlerOptions{
    AddSource: true,
}
handler := slog.NewJSONHandler(os.Stdout, opts)
logger := slog.New(handler)

logger.Info("message")
```

Output:

```json
{
  "time": "2024-01-15T10:30:00.000Z",
  "level": "INFO",
  "source": {
    "function": "main.main",
    "file": "/Users/user/project/main.go",
    "line": 42
  },
  "msg": "message"
}
```

## Accessing Source from Record

```go
func (r Record) Source() *Source
```

Get the source location from a log record. Returns `nil` if PC is zero or invalid.

Example:

```go
type CustomHandler struct {
    handler slog.Handler
}

func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
    if src := r.Source(); src != nil {
        fmt.Printf("Log from %s:%d in %s\n", src.File, src.Line, src.Function)
    }
    return h.handler.Handle(ctx, r)
}
```

## Customizing Source Output

### Shorten File Paths

Use `ReplaceAttr` to customize source formatting:

```go
import "path/filepath"

opts := &slog.HandlerOptions{
    AddSource: true,
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == slog.SourceKey {
            source := a.Value.Any().(*slog.Source)
            // Show only filename, not full path
            source.File = filepath.Base(source.File)
            return slog.Any(slog.SourceKey, source)
        }
        return a
    },
}

handler := slog.NewJSONHandler(os.Stdout, opts)
logger := slog.New(handler)

logger.Info("message")
// Output: {"source":{"function":"main.main","file":"main.go","line":42},...}
```

### Remove Package Path from Function

```go
import "strings"

opts := &slog.HandlerOptions{
    AddSource: true,
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == slog.SourceKey {
            source := a.Value.Any().(*slog.Source)

            // Shorten function name
            if idx := strings.LastIndex(source.Function, "/"); idx >= 0 {
                source.Function = source.Function[idx+1:]
            }

            // Shorten file path
            source.File = filepath.Base(source.File)

            return slog.Any(slog.SourceKey, source)
        }
        return a
    },
}
```

### Custom Format

```go
opts := &slog.HandlerOptions{
    AddSource: true,
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == slog.SourceKey {
            source := a.Value.Any().(*slog.Source)
            // Format as single string
            formatted := fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line)
            return slog.String("location", formatted)
        }
        return a
    },
}

// Output: {"location":"main.go:42",...}
```

## Performance Considerations

### Runtime Cost

Enabling source tracking has a small performance cost:

```go
// Without source tracking
handler := slog.NewJSONHandler(os.Stdout, nil)

// With source tracking (slightly slower)
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    AddSource: true,
})
```

The overhead comes from:
1. Calling `runtime.Callers()` to get the program counter
2. Resolving the PC to function/file/line using `runtime.CallersFrames()`

### When to Enable

Enable source tracking when:
- Debugging production issues
- Development and testing
- Error logs (use filtering to only add source for errors)

Disable when:
- Maximum performance is required
- Logs are high-volume
- Source location is not useful

### Conditional Source Tracking

Only add source for errors:

```go
type ConditionalSourceHandler struct {
    handler slog.Handler
}

func (h *ConditionalSourceHandler) Handle(ctx context.Context, r slog.Record) error {
    // Only add source for error and above
    if r.Level >= slog.LevelError {
        if src := r.Source(); src != nil {
            r.AddAttrs(slog.Any("source", src))
        }
    }
    return h.handler.Handle(ctx, r)
}
```

## Wrapping Functions and Source Tracking

When wrapping slog functions, you need to adjust the skip parameter:

```go
func InfoWithSource(logger *slog.Logger, msg string, args ...any) {
    if !logger.Enabled(context.Background(), slog.LevelInfo) {
        return
    }

    var pcs [1]uintptr
    // Skip 2: runtime.Callers, InfoWithSource
    runtime.Callers(2, pcs[:])

    r := slog.NewRecord(time.Now(), slog.LevelInfo, msg, pcs[0])
    r.Add(args...)

    _ = logger.Handler().Handle(context.Background(), r)
}

// Usage
InfoWithSource(logger, "message", "key", "value")
// Source will point to the caller of InfoWithSource, not InfoWithSource itself
```

## Text Handler Source Format

TextHandler formats source inline:

```
time=2024-01-15T10:30:00.000Z level=INFO source=/path/to/main.go:42 msg=message
```

## JSON Handler Source Format

JSONHandler formats source as an object:

```json
{
  "time": "2024-01-15T10:30:00.000Z",
  "level": "INFO",
  "source": {
    "function": "main.main",
    "file": "/path/to/main.go",
    "line": 42
  },
  "msg": "message"
}
```

## Example: Source-Aware Error Handler

```go
type ErrorSourceHandler struct {
    handler slog.Handler
}

func NewErrorSourceHandler(h slog.Handler) *ErrorSourceHandler {
    return &ErrorSourceHandler{handler: h}
}

func (h *ErrorSourceHandler) Handle(ctx context.Context, r slog.Record) error {
    // Add detailed source info for errors
    if r.Level >= slog.LevelError {
        if src := r.Source(); src != nil {
            // Add as separate, easy-to-read attributes
            r.AddAttrs(
                slog.String("error_file", src.File),
                slog.Int("error_line", src.Line),
                slog.String("error_func", src.Function),
            )
        }
    }

    return h.handler.Handle(ctx, r)
}

func (h *ErrorSourceHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *ErrorSourceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &ErrorSourceHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *ErrorSourceHandler) WithGroup(name string) slog.Handler {
    return &ErrorSourceHandler{handler: h.handler.WithGroup(name)}
}
```

## Filtering by Source

```go
type SourceFilterHandler struct {
    handler     slog.Handler
    excludePaths []string
}

func (h *SourceFilterHandler) Handle(ctx context.Context, r slog.Record) error {
    if src := r.Source(); src != nil {
        // Skip logs from certain paths
        for _, path := range h.excludePaths {
            if strings.Contains(src.File, path) {
                return nil
            }
        }
    }

    return h.handler.Handle(ctx, r)
}

// Usage: exclude logs from test files
handler := &SourceFilterHandler{
    handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        AddSource: true,
    }),
    excludePaths: []string{"_test.go", "/vendor/"},
}
```

## Best Practices

### 1. Enable Selectively

```go
// Development: enable source
if os.Getenv("ENV") == "development" {
    opts.AddSource = true
}
```

### 2. Shorten Paths in Production

```go
opts := &slog.HandlerOptions{
    AddSource: true,
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == slog.SourceKey {
            source := a.Value.Any().(*slog.Source)
            // Relative to project root
            source.File = strings.TrimPrefix(source.File, "/path/to/project/")
            return slog.Any(slog.SourceKey, source)
        }
        return a
    },
}
```

### 3. Only for Errors

```go
// Use separate handlers for different levels
errorHandler := slog.NewJSONHandler(errorFile, &slog.HandlerOptions{
    Level:     slog.LevelError,
    AddSource: true, // Source only for errors
})

infoHandler := slog.NewJSONHandler(infoFile, &slog.HandlerOptions{
    Level:     slog.LevelInfo,
    AddSource: false, // No source for info
})

multiHandler := NewMultiHandler(errorHandler, infoHandler)
```

## See Also

- [Logger Configuration](logger-configuration.md) - Setting up source tracking
- [Handler Interface](api-handler.md) - Handler implementation
- [Level and Record Types](api-level-record.md) - Record API
