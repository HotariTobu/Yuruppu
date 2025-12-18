# Handler Interface API Reference

Complete API reference for handlers and handler-related types.

## Handler Interface

```go
type Handler interface {
    Enabled(ctx context.Context, level Level) bool
    Handle(ctx context.Context, r Record) error
    WithAttrs(attrs []Attr) Handler
    WithGroup(name string) Handler
}
```

### Enabled

```go
Enabled(ctx context.Context, level Level) bool
```

Report whether the handler will process records at the given level.

### Handle

```go
Handle(ctx context.Context, r Record) error
```

Process a log record. The handler should not retain the record or any of its attributes after returning.

### WithAttrs

```go
WithAttrs(attrs []Attr) Handler
```

Return a new handler with the given attributes added. Attributes from this handler will be prepended to those from the Record.

### WithGroup

```go
WithGroup(name string) Handler
```

Return a new handler that groups subsequent attributes under the given name.

## TextHandler

```go
type TextHandler struct {
    // contains filtered or unexported fields
}
```

Human-readable, space-separated output format.

### NewTextHandler

```go
func NewTextHandler(w io.Writer, opts *HandlerOptions) *TextHandler
```

Create a TextHandler that writes to w using the given options.

Example:

```go
handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level:     slog.LevelDebug,
    AddSource: true,
})
```

### TextHandler Methods

```go
func (h *TextHandler) Enabled(_ context.Context, level Level) bool
func (h *TextHandler) Handle(_ context.Context, r Record) error
func (h *TextHandler) WithAttrs(attrs []Attr) Handler
func (h *TextHandler) WithGroup(name string) Handler
```

### Output Format

```
time=2024-01-15T10:30:00.000-05:00 level=INFO msg="user action" user_id=123 action=login
```

## JSONHandler

```go
type JSONHandler struct {
    // contains filtered or unexported fields
}
```

JSON output format for machine parsing.

### NewJSONHandler

```go
func NewJSONHandler(w io.Writer, opts *HandlerOptions) *JSONHandler
```

Create a JSONHandler that writes to w using the given options.

Example:

```go
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level:     slog.LevelInfo,
    AddSource: true,
})
```

### JSONHandler Methods

```go
func (h *JSONHandler) Enabled(_ context.Context, level Level) bool
func (h *JSONHandler) Handle(_ context.Context, r Record) error
func (h *JSONHandler) WithAttrs(attrs []Attr) Handler
func (h *JSONHandler) WithGroup(name string) Handler
```

### Output Format

```json
{"time":"2024-01-15T10:30:00.000000000-05:00","level":"INFO","msg":"user action","user_id":123,"action":"login"}
```

## HandlerOptions

```go
type HandlerOptions struct {
    AddSource   bool
    Level       Leveler
    ReplaceAttr func(groups []string, a Attr) Attr
}
```

Options for built-in handlers.

### AddSource

```go
AddSource bool
```

If true, include source code position in the output.

Example:

```go
opts := &slog.HandlerOptions{
    AddSource: true,
}
```

Output includes:
```json
{
  "source": {
    "function": "main.main",
    "file": "/path/to/main.go",
    "line": 42
  }
}
```

### Level

```go
Level Leveler
```

Minimum level to log. Can be a `Level` or `*LevelVar` for dynamic levels.

Example:

```go
opts := &slog.HandlerOptions{
    Level: slog.LevelDebug,
}

// Or with dynamic level:
var programLevel = new(slog.LevelVar)
opts := &slog.HandlerOptions{
    Level: programLevel,
}
```

### ReplaceAttr

```go
ReplaceAttr func(groups []string, a Attr) Attr
```

Function called on each attribute to transform or filter it. Return an empty `Attr` to remove the attribute.

Parameters:
- `groups`: Path of groups enclosing the attribute (empty for top-level)
- `a`: The attribute to transform

Example:

```go
opts := &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        // Remove time
        if a.Key == slog.TimeKey {
            return slog.Attr{}
        }

        // Redact passwords
        if a.Key == "password" {
            return slog.String("password", "REDACTED")
        }

        // Customize level display
        if a.Key == slog.LevelKey {
            level := a.Value.Any().(slog.Level)
            return slog.String("severity", level.String())
        }

        return a
    },
}
```

## DiscardHandler

```go
var DiscardHandler Handler
```

A handler that discards all log records.

Example:

```go
logger := slog.New(slog.DiscardHandler)
logger.Info("not logged anywhere")
```

## Leveler Interface

```go
type Leveler interface {
    Level() Level
}
```

Implemented by `Level` and `LevelVar` to allow dynamic log levels.

## Custom Handler Example

Implement the Handler interface for custom behavior:

```go
type CustomHandler struct {
    handler slog.Handler
    prefix  string
}

func NewCustomHandler(w io.Writer, prefix string) *CustomHandler {
    return &CustomHandler{
        handler: slog.NewJSONHandler(w, nil),
        prefix:  prefix,
    }
}

func (h *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
    // Add custom prefix to message
    r2 := slog.NewRecord(r.Time, r.Level, h.prefix+r.Message, r.PC)
    r.Attrs(func(a slog.Attr) bool {
        r2.AddAttrs(a)
        return true
    })
    return h.handler.Handle(ctx, r2)
}

func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &CustomHandler{
        handler: h.handler.WithAttrs(attrs),
        prefix:  h.prefix,
    }
}

func (h *CustomHandler) WithGroup(name string) slog.Handler {
    return &CustomHandler{
        handler: h.handler.WithGroup(name),
        prefix:  h.prefix,
    }
}

// Usage
handler := NewCustomHandler(os.Stdout, "[MYAPP] ")
logger := slog.New(handler)
logger.Info("started") // Output: {"msg":"[MYAPP] started",...}
```

## Multi-Output Handler Example

Write to multiple destinations:

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
            if err := handler.Handle(ctx, r.Clone()); err != nil {
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
logger := slog.New(NewMultiHandler(consoleHandler, fileHandler))
```

## Filtering Handler Example

Filter logs based on custom criteria:

```go
type FilterHandler struct {
    handler slog.Handler
    filter  func(ctx context.Context, r slog.Record) bool
}

func NewFilterHandler(h slog.Handler, filter func(context.Context, slog.Record) bool) *FilterHandler {
    return &FilterHandler{
        handler: h,
        filter:  filter,
    }
}

func (h *FilterHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *FilterHandler) Handle(ctx context.Context, r slog.Record) error {
    if !h.filter(ctx, r) {
        return nil // Skip this record
    }
    return h.handler.Handle(ctx, r)
}

func (h *FilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &FilterHandler{
        handler: h.handler.WithAttrs(attrs),
        filter:  h.filter,
    }
}

func (h *FilterHandler) WithGroup(name string) slog.Handler {
    return &FilterHandler{
        handler: h.handler.WithGroup(name),
        filter:  h.filter,
    }
}

// Usage: only log errors
filter := func(ctx context.Context, r slog.Record) bool {
    return r.Level >= slog.LevelError
}
handler := NewFilterHandler(slog.NewJSONHandler(os.Stdout, nil), filter)
logger := slog.New(handler)
```

## Handler Design Guidelines

### 1. Don't Retain Records

Handlers must not retain the `Record` or its attributes after `Handle` returns:

```go
// Bad: retains record
func (h *BadHandler) Handle(ctx context.Context, r slog.Record) error {
    h.lastRecord = r // Don't do this!
    return nil
}

// Good: processes immediately
func (h *GoodHandler) Handle(ctx context.Context, r slog.Record) error {
    // Process record immediately
    json.NewEncoder(h.w).Encode(r)
    return nil
}
```

### 2. Clone Records if Needed

If you must retain a record, clone it first:

```go
func (h *AsyncHandler) Handle(ctx context.Context, r slog.Record) error {
    r2 := r.Clone()
    h.queue <- r2 // Safe: r2 is independent
    return nil
}
```

### 3. Preserve WithAttrs and WithGroup Semantics

```go
func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    // Create new handler with combined attributes
    return &CustomHandler{
        handler: h.handler.WithAttrs(attrs),
        // ... copy other fields
    }
}
```

### 4. Handle Errors Gracefully

```go
func (h *RobustHandler) Handle(ctx context.Context, r slog.Record) error {
    if err := h.handler.Handle(ctx, r); err != nil {
        // Log to stderr as fallback
        fmt.Fprintf(os.Stderr, "logging error: %v\n", err)
        return err
    }
    return nil
}
```

## Handler Selection

| Scenario | Handler | Reason |
|----------|---------|--------|
| Development | TextHandler | Human-readable |
| Production | JSONHandler | Machine-parseable |
| Testing | DiscardHandler | No output, fast |
| Multiple outputs | Custom MultiHandler | Write to many destinations |
| Filtering | Custom FilterHandler | Skip certain logs |
| Cloud logging | JSONHandler | Compatible with services |
| High performance | Custom with buffering | Batch writes |

## See Also

- [Handlers Guide](handlers.md) - Detailed handler usage
- [Custom Handlers](custom-handlers.md) - Building custom handlers
- [Record Type](api-level-record.md) - Record structure
