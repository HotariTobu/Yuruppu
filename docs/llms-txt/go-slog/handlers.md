# Handlers

Handlers control how log records are processed, formatted, and output. The `log/slog` package provides two built-in handlers and an interface for creating custom handlers.

## Handler Interface

```go
type Handler interface {
    Enabled(ctx context.Context, level Level) bool
    Handle(ctx context.Context, r Record) error
    WithAttrs(attrs []Attr) Handler
    WithGroup(name string) Handler
}
```

### Methods

- **`Enabled`**: Returns whether the handler will process records at the given level
- **`Handle`**: Processes and outputs a log record
- **`WithAttrs`**: Returns a new handler with additional attributes
- **`WithGroup`**: Returns a new handler that adds a group namespace

## TextHandler

Outputs logs in a human-readable, space-separated format.

### Creating a TextHandler

```go
func NewTextHandler(w io.Writer, opts *HandlerOptions) *TextHandler
```

Example:

```go
handler := slog.NewTextHandler(os.Stderr, nil)
logger := slog.New(handler)

logger.Info("server started", "port", 8080, "env", "production")
```

Output:
```
time=2024-01-15T10:30:00.000-05:00 level=INFO msg="server started" port=8080 env=production
```

### TextHandler Options

```go
opts := &slog.HandlerOptions{
    Level:     slog.LevelDebug,
    AddSource: true,
}
handler := slog.NewTextHandler(os.Stdout, opts)
```

## JSONHandler

Outputs logs in JSON format, ideal for log aggregation systems (e.g., ELK, Splunk).

### Creating a JSONHandler

```go
func NewJSONHandler(w io.Writer, opts *HandlerOptions) *JSONHandler
```

Example:

```go
handler := slog.NewJSONHandler(os.Stdout, nil)
logger := slog.New(handler)

logger.Info("server started", "port", 8080, "env", "production")
```

Output:
```json
{"time":"2024-01-15T10:30:00.000000000-05:00","level":"INFO","msg":"server started","port":8080,"env":"production"}
```

## HandlerOptions

Configure handler behavior:

```go
type HandlerOptions struct {
    AddSource   bool
    Level       Leveler
    ReplaceAttr func(groups []string, a Attr) Attr
}
```

### AddSource

Include source file location in logs:

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
    "file": "/path/to/main.go",
    "line": 42
  },
  "msg": "message"
}
```

### Level

Set minimum log level:

```go
opts := &slog.HandlerOptions{
    Level: slog.LevelWarn, // Only Warn and Error
}
handler := slog.NewTextHandler(os.Stderr, opts)
logger := slog.New(handler)

logger.Info("not logged")  // Filtered out
logger.Warn("logged")      // Visible
logger.Error("logged")     // Visible
```

### ReplaceAttr

Transform or filter attributes before output:

```go
opts := &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        // Remove time from logs
        if a.Key == slog.TimeKey {
            return slog.Attr{}
        }

        // Customize level display
        if a.Key == slog.LevelKey {
            level := a.Value.Any().(slog.Level)
            return slog.String("severity", level.String())
        }

        // Redact sensitive fields
        if a.Key == "password" || a.Key == "token" {
            return slog.String(a.Key, "***REDACTED***")
        }

        return a
    },
}
handler := slog.NewJSONHandler(os.Stdout, opts)
```

#### ReplaceAttr with Groups

The `groups` parameter shows the group path:

```go
opts := &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        // groups = [] for top-level attributes
        // groups = ["request"] for attributes in request group
        // groups = ["request", "headers"] for nested groups

        if len(groups) > 0 && groups[0] == "secrets" {
            // Redact all attributes in "secrets" group
            return slog.String(a.Key, "REDACTED")
        }

        return a
    },
}
```

## DiscardHandler

A handler that discards all log records (no output):

```go
var DiscardHandler Handler
```

Useful for testing or disabling logging:

```go
logger := slog.New(slog.DiscardHandler)
logger.Info("this will not appear anywhere")
```

## Handler Methods

### WithAttrs

Add persistent attributes to a handler:

```go
handler := slog.NewJSONHandler(os.Stdout, nil)
handlerWithAttrs := handler.WithAttrs([]slog.Attr{
    slog.String("service", "api"),
    slog.String("version", "1.0.0"),
})

logger := slog.New(handlerWithAttrs)
logger.Info("started") // Includes service and version
```

### WithGroup

Add a group namespace:

```go
handler := slog.NewJSONHandler(os.Stdout, nil)
groupedHandler := handler.WithGroup("database")

logger := slog.New(groupedHandler)
logger.Info("connected", "host", "localhost")
// Output: {"level":"INFO","msg":"connected","database":{"host":"localhost"}}
```

## Custom Handler Example

Implement the `Handler` interface for custom behavior:

```go
type CustomHandler struct {
    handler slog.Handler
}

func NewCustomHandler(w io.Writer) *CustomHandler {
    return &CustomHandler{
        handler: slog.NewJSONHandler(w, nil),
    }
}

func (h *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
    // Add custom logic here
    // For example, send to external service, filter, etc.

    // Add a custom attribute to every log
    r.AddAttrs(slog.String("hostname", os.Hostname()))

    return h.handler.Handle(ctx, r)
}

func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &CustomHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *CustomHandler) WithGroup(name string) slog.Handler {
    return &CustomHandler{handler: h.handler.WithGroup(name)}
}
```

## Wrapping Handlers

Create handler chains for multiple outputs:

```go
type MultiHandler struct {
    handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
    return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
    for _, handler := range h.handlers {
        if handler.Enabled(ctx, level) {
            return true
        }
    }
    return false
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
    for _, handler := range h.handlers {
        if handler.Enabled(ctx, r.Level) {
            if err := handler.Handle(ctx, r); err != nil {
                return err
            }
        }
    }
    return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    handlers := make([]slog.Handler, len(h.handlers))
    for i, handler := range h.handlers {
        handlers[i] = handler.WithAttrs(attrs)
    }
    return NewMultiHandler(handlers...)
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
    handlers := make([]slog.Handler, len(h.handlers))
    for i, handler := range h.handlers {
        handlers[i] = handler.WithGroup(name)
    }
    return NewMultiHandler(handlers...)
}

// Usage
consoleHandler := slog.NewTextHandler(os.Stdout, nil)
fileHandler := slog.NewJSONHandler(file, nil)
multiHandler := NewMultiHandler(consoleHandler, fileHandler)
logger := slog.New(multiHandler)
```

## Handler Selection Guide

| Use Case | Handler | Reason |
|----------|---------|--------|
| Development | TextHandler | Human-readable output |
| Production | JSONHandler | Machine-parseable, structured |
| Log aggregation systems | JSONHandler | Compatible with ELK, Splunk, etc. |
| Testing | DiscardHandler | No output, fast |
| Multiple destinations | Custom MultiHandler | Write to multiple targets |
| Cloud logging | JSONHandler | Integrates with CloudWatch, Stackdriver |

## Next Steps

- Learn about custom handler implementation in [Custom Handlers](custom-handlers.md)
- Explore handler configuration in [Logger Configuration](logger-configuration.md)
- See the Handler API reference in [Handler Interface](api-handler.md)
