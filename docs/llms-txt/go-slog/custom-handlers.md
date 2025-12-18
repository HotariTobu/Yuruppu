# Custom Handlers

Guide to implementing custom handlers for specialized logging requirements.

## Handler Interface

```go
type Handler interface {
    Enabled(ctx context.Context, level Level) bool
    Handle(ctx context.Context, r Record) error
    WithAttrs(attrs []Attr) Handler
    WithGroup(name string) Handler
}
```

## Basic Custom Handler

### Minimal Implementation

```go
type SimpleHandler struct {
    w     io.Writer
    level slog.Level
    attrs []slog.Attr
    group string
}

func NewSimpleHandler(w io.Writer, level slog.Level) *SimpleHandler {
    return &SimpleHandler{
        w:     w,
        level: level,
    }
}

func (h *SimpleHandler) Enabled(_ context.Context, level slog.Level) bool {
    return level >= h.level
}

func (h *SimpleHandler) Handle(_ context.Context, r slog.Record) error {
    buf := make([]byte, 0, 1024)

    // Add timestamp
    buf = append(buf, r.Time.Format(time.RFC3339)...)
    buf = append(buf, ' ')

    // Add level
    buf = append(buf, r.Level.String()...)
    buf = append(buf, ' ')

    // Add message
    buf = append(buf, r.Message...)

    // Add pre-configured attributes
    for _, attr := range h.attrs {
        buf = append(buf, ' ')
        buf = appendAttr(buf, attr)
    }

    // Add record attributes
    r.Attrs(func(a slog.Attr) bool {
        buf = append(buf, ' ')
        buf = appendAttr(buf, a)
        return true
    })

    buf = append(buf, '\n')
    _, err := h.w.Write(buf)
    return err
}

func appendAttr(buf []byte, attr slog.Attr) []byte {
    buf = append(buf, attr.Key...)
    buf = append(buf, '=')
    buf = append(buf, attr.Value.String()...)
    return buf
}

func (h *SimpleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
    copy(newAttrs, h.attrs)
    copy(newAttrs[len(h.attrs):], attrs)

    return &SimpleHandler{
        w:     h.w,
        level: h.level,
        attrs: newAttrs,
        group: h.group,
    }
}

func (h *SimpleHandler) WithGroup(name string) slog.Handler {
    return &SimpleHandler{
        w:     h.w,
        level: h.level,
        attrs: h.attrs,
        group: name,
    }
}
```

## Multi-Output Handler

Write logs to multiple destinations:

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
        if !handler.Enabled(ctx, r.Level) {
            continue
        }
        // Clone record for each handler
        if err := handler.Handle(ctx, r.Clone()); err != nil {
            return err
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
errorFileHandler := slog.NewJSONHandler(errorFile, &slog.HandlerOptions{
    Level: slog.LevelError,
})

multiHandler := NewMultiHandler(consoleHandler, fileHandler, errorFileHandler)
logger := slog.New(multiHandler)
```

## Filtering Handler

Filter logs based on custom criteria:

```go
type FilterHandler struct {
    handler slog.Handler
    filter  func(context.Context, slog.Record) bool
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

// Example filters
func NoHealthCheckFilter(ctx context.Context, r slog.Record) bool {
    isHealthCheck := false
    r.Attrs(func(a slog.Attr) bool {
        if a.Key == "path" && a.Value.String() == "/health" {
            isHealthCheck = true
            return false
        }
        return true
    })
    return !isHealthCheck
}

// Usage
baseHandler := slog.NewJSONHandler(os.Stdout, nil)
filteredHandler := NewFilterHandler(baseHandler, NoHealthCheckFilter)
logger := slog.New(filteredHandler)
```

## Sampling Handler

Sample logs to reduce volume:

```go
type SamplingHandler struct {
    handler    slog.Handler
    sampleRate float64 // 0.0 to 1.0
    mu         sync.Mutex
    counter    uint64
}

func NewSamplingHandler(h slog.Handler, sampleRate float64) *SamplingHandler {
    return &SamplingHandler{
        handler:    h,
        sampleRate: sampleRate,
    }
}

func (h *SamplingHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *SamplingHandler) Handle(ctx context.Context, r slog.Record) error {
    // Always log errors and above
    if r.Level >= slog.LevelError {
        return h.handler.Handle(ctx, r)
    }

    // Sample other levels
    h.mu.Lock()
    h.counter++
    count := h.counter
    h.mu.Unlock()

    if float64(count%100)/100.0 < h.sampleRate {
        return h.handler.Handle(ctx, r)
    }

    return nil
}

func (h *SamplingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &SamplingHandler{
        handler:    h.handler.WithAttrs(attrs),
        sampleRate: h.sampleRate,
    }
}

func (h *SamplingHandler) WithGroup(name string) slog.Handler {
    return &SamplingHandler{
        handler:    h.handler.WithGroup(name),
        sampleRate: h.sampleRate,
    }
}

// Usage: log 10% of info/debug, all errors
handler := NewSamplingHandler(
    slog.NewJSONHandler(os.Stdout, nil),
    0.1, // 10% sample rate
)
logger := slog.New(handler)
```

## Buffered Handler

Buffer logs for batch writing:

```go
type BufferedHandler struct {
    handler slog.Handler
    buf     []slog.Record
    mu      sync.Mutex
    maxSize int
    ticker  *time.Ticker
    stop    chan struct{}
}

func NewBufferedHandler(h slog.Handler, maxSize int, flushInterval time.Duration) *BufferedHandler {
    bh := &BufferedHandler{
        handler: h,
        buf:     make([]slog.Record, 0, maxSize),
        maxSize: maxSize,
        ticker:  time.NewTicker(flushInterval),
        stop:    make(chan struct{}),
    }

    go bh.periodicFlush()

    return bh
}

func (h *BufferedHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *BufferedHandler) Handle(ctx context.Context, r slog.Record) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.buf = append(h.buf, r.Clone())

    if len(h.buf) >= h.maxSize {
        return h.flushLocked(ctx)
    }

    return nil
}

func (h *BufferedHandler) flushLocked(ctx context.Context) error {
    for _, r := range h.buf {
        if err := h.handler.Handle(ctx, r); err != nil {
            return err
        }
    }
    h.buf = h.buf[:0]
    return nil
}

func (h *BufferedHandler) Flush(ctx context.Context) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    return h.flushLocked(ctx)
}

func (h *BufferedHandler) periodicFlush() {
    for {
        select {
        case <-h.ticker.C:
            h.Flush(context.Background())
        case <-h.stop:
            return
        }
    }
}

func (h *BufferedHandler) Close() error {
    close(h.stop)
    h.ticker.Stop()
    return h.Flush(context.Background())
}

func (h *BufferedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &BufferedHandler{
        handler: h.handler.WithAttrs(attrs),
        maxSize: h.maxSize,
    }
}

func (h *BufferedHandler) WithGroup(name string) slog.Handler {
    return &BufferedHandler{
        handler: h.handler.WithGroup(name),
        maxSize: h.maxSize,
    }
}

// Usage
baseHandler := slog.NewJSONHandler(os.Stdout, nil)
bufferedHandler := NewBufferedHandler(baseHandler, 100, 5*time.Second)
defer bufferedHandler.Close()

logger := slog.New(bufferedHandler)
```

## Async Handler

Handle logs asynchronously:

```go
type AsyncHandler struct {
    handler slog.Handler
    queue   chan asyncRecord
    wg      sync.WaitGroup
}

type asyncRecord struct {
    ctx context.Context
    rec slog.Record
}

func NewAsyncHandler(h slog.Handler, queueSize int) *AsyncHandler {
    ah := &AsyncHandler{
        handler: h,
        queue:   make(chan asyncRecord, queueSize),
    }

    ah.wg.Add(1)
    go ah.processQueue()

    return ah
}

func (h *AsyncHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *AsyncHandler) Handle(ctx context.Context, r slog.Record) error {
    select {
    case h.queue <- asyncRecord{ctx: ctx, rec: r.Clone()}:
        return nil
    default:
        // Queue full, handle synchronously
        return h.handler.Handle(ctx, r)
    }
}

func (h *AsyncHandler) processQueue() {
    defer h.wg.Done()

    for ar := range h.queue {
        _ = h.handler.Handle(ar.ctx, ar.rec)
    }
}

func (h *AsyncHandler) Close() error {
    close(h.queue)
    h.wg.Wait()
    return nil
}

func (h *AsyncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &AsyncHandler{
        handler: h.handler.WithAttrs(attrs),
        queue:   h.queue,
    }
}

func (h *AsyncHandler) WithGroup(name string) slog.Handler {
    return &AsyncHandler{
        handler: h.handler.WithGroup(name),
        queue:   h.queue,
    }
}

// Usage
baseHandler := slog.NewJSONHandler(os.Stdout, nil)
asyncHandler := NewAsyncHandler(baseHandler, 1000)
defer asyncHandler.Close()

logger := slog.New(asyncHandler)
```

## Context Enrichment Handler

Automatically add context values to all logs:

```go
type ContextEnricher struct {
    handler slog.Handler
    keys    []string
}

func NewContextEnricher(h slog.Handler, keys ...string) *ContextEnricher {
    return &ContextEnricher{
        handler: h,
        keys:    keys,
    }
}

func (h *ContextEnricher) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *ContextEnricher) Handle(ctx context.Context, r slog.Record) error {
    for _, key := range h.keys {
        if val := ctx.Value(key); val != nil {
            r.AddAttrs(slog.Any(key, val))
        }
    }
    return h.handler.Handle(ctx, r)
}

func (h *ContextEnricher) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &ContextEnricher{
        handler: h.handler.WithAttrs(attrs),
        keys:    h.keys,
    }
}

func (h *ContextEnricher) WithGroup(name string) slog.Handler {
    return &ContextEnricher{
        handler: h.handler.WithGroup(name),
        keys:    h.keys,
    }
}

// Usage
baseHandler := slog.NewJSONHandler(os.Stdout, nil)
enricher := NewContextEnricher(baseHandler, "request_id", "user_id", "trace_id")
logger := slog.New(enricher)

ctx := context.WithValue(context.Background(), "request_id", "abc-123")
logger.InfoContext(ctx, "message")
// Output includes request_id automatically
```

## Syslog Handler

Send logs to syslog:

```go
import "log/syslog"

type SyslogHandler struct {
    writer *syslog.Writer
    attrs  []slog.Attr
}

func NewSyslogHandler(network, raddr string, priority syslog.Priority, tag string) (*SyslogHandler, error) {
    w, err := syslog.Dial(network, raddr, priority, tag)
    if err != nil {
        return nil, err
    }

    return &SyslogHandler{writer: w}, nil
}

func (h *SyslogHandler) Enabled(_ context.Context, level slog.Level) bool {
    return true
}

func (h *SyslogHandler) Handle(_ context.Context, r slog.Record) error {
    msg := formatRecord(r, h.attrs)

    switch r.Level {
    case slog.LevelDebug:
        return h.writer.Debug(msg)
    case slog.LevelInfo:
        return h.writer.Info(msg)
    case slog.LevelWarn:
        return h.writer.Warning(msg)
    case slog.LevelError:
        return h.writer.Err(msg)
    default:
        return h.writer.Info(msg)
    }
}

func formatRecord(r slog.Record, attrs []slog.Attr) string {
    buf := new(strings.Builder)
    buf.WriteString(r.Message)

    for _, attr := range attrs {
        buf.WriteString(" ")
        buf.WriteString(attr.Key)
        buf.WriteString("=")
        buf.WriteString(attr.Value.String())
    }

    r.Attrs(func(a slog.Attr) bool {
        buf.WriteString(" ")
        buf.WriteString(a.Key)
        buf.WriteString("=")
        buf.WriteString(a.Value.String())
        return true
    })

    return buf.String()
}

func (h *SyslogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
    copy(newAttrs, h.attrs)
    copy(newAttrs[len(h.attrs):], attrs)

    return &SyslogHandler{
        writer: h.writer,
        attrs:  newAttrs,
    }
}

func (h *SyslogHandler) WithGroup(name string) slog.Handler {
    // Simplified: prepend group name to keys
    return h
}

func (h *SyslogHandler) Close() error {
    return h.writer.Close()
}
```

## Handler Design Best Practices

### 1. Don't Retain Records

```go
// Bad
func (h *BadHandler) Handle(ctx context.Context, r slog.Record) error {
    h.lastRecord = r // Don't retain!
    return nil
}

// Good
func (h *GoodHandler) Handle(ctx context.Context, r slog.Record) error {
    // Process immediately or clone
    return h.processRecord(r)
}
```

### 2. Clone Records for Async Processing

```go
func (h *AsyncHandler) Handle(ctx context.Context, r slog.Record) error {
    h.queue <- r.Clone() // Clone before sending
    return nil
}
```

### 3. Preserve Handler Semantics

```go
func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    // Return new handler, don't modify existing
    return &CustomHandler{
        // Copy fields and add new attrs
    }
}
```

### 4. Handle Errors Gracefully

```go
func (h *ResilientHandler) Handle(ctx context.Context, r slog.Record) error {
    if err := h.primary.Handle(ctx, r); err != nil {
        // Fallback to secondary
        return h.fallback.Handle(ctx, r)
    }
    return nil
}
```

## See Also

- [Handler Interface API](api-handler.md) - Complete handler API reference
- [Handlers Guide](handlers.md) - Built-in handlers
- [Context Support](context-support.md) - Context-aware handlers
