# Context Support

The `log/slog` package provides built-in support for `context.Context`, enabling integration with tracing, request tracking, and cancellation.

## Context-Aware Logging Methods

All logging methods have Context variants:

```go
func InfoContext(ctx context.Context, msg string, args ...any)
func DebugContext(ctx context.Context, msg string, args ...any)
func WarnContext(ctx context.Context, msg string, args ...any)
func ErrorContext(ctx context.Context, msg string, args ...any)
func Log(ctx context.Context, level Level, msg string, args ...any)
func LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr)
```

## Why Use Context?

### 1. Distributed Tracing

Pass trace IDs through the call stack:

```go
func HandleRequest(ctx context.Context, req Request) error {
    // Context contains trace ID from middleware
    slog.InfoContext(ctx, "processing request",
        "method", req.Method,
        "path", req.Path,
    )

    if err := processRequest(ctx, req); err != nil {
        slog.ErrorContext(ctx, "request failed", "error", err)
        return err
    }

    slog.InfoContext(ctx, "request completed")
    return nil
}
```

### 2. Request Correlation

Correlate logs across function calls:

```go
func middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := uuid.New().String()
        ctx := context.WithValue(r.Context(), "request_id", requestID)

        slog.InfoContext(ctx, "request started", "method", r.Method, "path", r.URL.Path)

        next.ServeHTTP(w, r.WithContext(ctx))

        slog.InfoContext(ctx, "request completed")
    })
}
```

### 3. Cancellation

Respect context cancellation:

```go
func LongRunningTask(ctx context.Context) error {
    for i := 0; i < 100; i++ {
        select {
        case <-ctx.Done():
            slog.WarnContext(ctx, "task cancelled", "iteration", i)
            return ctx.Err()
        default:
            // Do work
            if i%10 == 0 {
                slog.DebugContext(ctx, "progress", "iteration", i)
            }
        }
    }
    return nil
}
```

## Extracting Values from Context

### Basic Approach

```go
func LogWithRequestID(ctx context.Context, msg string, args ...any) {
    if requestID, ok := ctx.Value("request_id").(string); ok {
        args = append([]any{"request_id", requestID}, args...)
    }
    slog.InfoContext(ctx, msg, args...)
}

// Usage
LogWithRequestID(ctx, "processing", "step", 1)
// Output: ... request_id=abc-123 step=1 msg=processing
```

### Using Custom Handler

```go
type ContextHandler struct {
    handler slog.Handler
}

func NewContextHandler(h slog.Handler) *ContextHandler {
    return &ContextHandler{handler: h}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
    // Extract values from context and add as attributes
    if traceID, ok := ctx.Value("trace_id").(string); ok {
        r.AddAttrs(slog.String("trace_id", traceID))
    }
    if userID, ok := ctx.Value("user_id").(int); ok {
        r.AddAttrs(slog.Int("user_id", userID))
    }

    return h.handler.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &ContextHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
    return &ContextHandler{handler: h.handler.WithGroup(name)}
}

// Usage
baseHandler := slog.NewJSONHandler(os.Stdout, nil)
contextHandler := NewContextHandler(baseHandler)
logger := slog.New(contextHandler)

ctx := context.WithValue(context.Background(), "trace_id", "abc-123")
logger.InfoContext(ctx, "message")
// Output: {"trace_id":"abc-123","msg":"message",...}
```

## Context with Logger

### Storing Logger in Context

```go
type contextKey string

const loggerKey contextKey = "logger"

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
    return context.WithValue(ctx, loggerKey, logger)
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
    if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
        return logger
    }
    return slog.Default()
}

// Usage
func main() {
    logger := slog.Default().With("service", "api")
    ctx := WithLogger(context.Background(), logger)

    handleRequest(ctx)
}

func handleRequest(ctx context.Context) {
    logger := LoggerFromContext(ctx)
    logger.Info("handling request")
}
```

### Context with Pre-configured Attributes

```go
func WithRequestContext(ctx context.Context, req *http.Request) context.Context {
    logger := slog.Default().With(
        "request_id", generateRequestID(),
        "method", req.Method,
        "path", req.URL.Path,
        "remote_addr", req.RemoteAddr,
    )
    return WithLogger(ctx, logger)
}

// Usage
func middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := WithRequestContext(r.Context(), r)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func handler(w http.ResponseWriter, r *http.Request) {
    logger := LoggerFromContext(r.Context())
    logger.Info("processing") // Includes request_id, method, path, remote_addr
}
```

## OpenTelemetry Integration

### Basic Integration

```go
import (
    "go.opentelemetry.io/otel/trace"
)

type OtelHandler struct {
    handler slog.Handler
}

func NewOtelHandler(h slog.Handler) *OtelHandler {
    return &OtelHandler{handler: h}
}

func (h *OtelHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *OtelHandler) Handle(ctx context.Context, r slog.Record) error {
    // Extract trace context
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        r.AddAttrs(
            slog.String("trace_id", span.SpanContext().TraceID().String()),
            slog.String("span_id", span.SpanContext().SpanID().String()),
        )
    }

    return h.handler.Handle(ctx, r)
}

func (h *OtelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &OtelHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *OtelHandler) WithGroup(name string) slog.Handler {
    return &OtelHandler{handler: h.handler.WithGroup(name)}
}

// Usage
baseHandler := slog.NewJSONHandler(os.Stdout, nil)
otelHandler := NewOtelHandler(baseHandler)
logger := slog.New(otelHandler)
slog.SetDefault(logger)

// All logs now include trace_id and span_id from context
```

## HTTP Request Context Pattern

### Complete Example

```go
type contextKey string

const (
    requestIDKey contextKey = "request_id"
    userIDKey    contextKey = "user_id"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        ctx := context.WithValue(r.Context(), requestIDKey, requestID)
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := authenticateUser(r)
        ctx := context.WithValue(r.Context(), userIDKey, userID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        slog.InfoContext(r.Context(), "request started",
            "method", r.Method,
            "path", r.URL.Path,
        )

        next.ServeHTTP(w, r)

        slog.InfoContext(r.Context(), "request completed",
            "method", r.Method,
            "path", r.URL.Path,
            "duration_ms", time.Since(start).Milliseconds(),
        )
    })
}

// Handler with custom context handler
type ContextExtractorHandler struct {
    handler slog.Handler
}

func NewContextExtractorHandler(h slog.Handler) *ContextExtractorHandler {
    return &ContextExtractorHandler{handler: h}
}

func (h *ContextExtractorHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.handler.Enabled(ctx, level)
}

func (h *ContextExtractorHandler) Handle(ctx context.Context, r slog.Record) error {
    if requestID, ok := ctx.Value(requestIDKey).(string); ok {
        r.AddAttrs(slog.String("request_id", requestID))
    }
    if userID, ok := ctx.Value(userIDKey).(int); ok {
        r.AddAttrs(slog.Int("user_id", userID))
    }
    return h.handler.Handle(ctx, r)
}

func (h *ContextExtractorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &ContextExtractorHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *ContextExtractorHandler) WithGroup(name string) slog.Handler {
    return &ContextExtractorHandler{handler: h.handler.WithGroup(name)}
}

// Setup
func main() {
    baseHandler := slog.NewJSONHandler(os.Stdout, nil)
    contextHandler := NewContextExtractorHandler(baseHandler)
    logger := slog.New(contextHandler)
    slog.SetDefault(logger)

    mux := http.NewServeMux()
    mux.HandleFunc("/", handleRoot)

    handler := RequestIDMiddleware(
        AuthMiddleware(
            LoggingMiddleware(mux),
        ),
    )

    http.ListenAndServe(":8080", handler)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
    // All logs automatically include request_id and user_id
    slog.InfoContext(r.Context(), "handling root request")
    w.Write([]byte("OK"))
}
```

## Database Context Pattern

```go
func QueryWithContext(ctx context.Context, db *sql.DB, query string, args ...any) (*sql.Rows, error) {
    start := time.Now()

    slog.DebugContext(ctx, "executing query",
        "query", query,
    )

    rows, err := db.QueryContext(ctx, query, args...)

    duration := time.Since(start)

    if err != nil {
        slog.ErrorContext(ctx, "query failed",
            "error", err,
            "query", query,
            "duration_ms", duration.Milliseconds(),
        )
        return nil, err
    }

    slog.DebugContext(ctx, "query completed",
        "query", query,
        "duration_ms", duration.Milliseconds(),
    )

    return rows, nil
}
```

## Context Timeout Pattern

```go
func ProcessWithTimeout(ctx context.Context, data string) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    slog.InfoContext(ctx, "processing started", "data_size", len(data))

    // Simulate work
    select {
    case <-time.After(10 * time.Second):
        slog.InfoContext(ctx, "processing completed")
        return nil
    case <-ctx.Done():
        slog.WarnContext(ctx, "processing cancelled",
            "error", ctx.Err(),
            "elapsed_ms", 5000,
        )
        return ctx.Err()
    }
}
```

## Best Practices

### 1. Always Pass Context

```go
// Good: context-aware
func ProcessData(ctx context.Context, data Data) error {
    slog.InfoContext(ctx, "processing")
    // ...
}

// Avoid: no context
func ProcessData(data Data) error {
    slog.Info("processing")
    // ...
}
```

### 2. Use Context for Tracing

```go
// Extract trace information from context
func LogWithTrace(ctx context.Context, level slog.Level, msg string, args ...any) {
    if traceID := getTraceID(ctx); traceID != "" {
        args = append([]any{"trace_id", traceID}, args...)
    }
    slog.Log(ctx, level, msg, args...)
}
```

### 3. Create Context-Aware Loggers

```go
// Create logger with context values once
func NewRequestLogger(ctx context.Context) *slog.Logger {
    attrs := []any{}

    if requestID, ok := ctx.Value("request_id").(string); ok {
        attrs = append(attrs, "request_id", requestID)
    }

    return slog.Default().With(attrs...)
}

// Use throughout request lifecycle
logger := NewRequestLogger(ctx)
logger.Info("step 1")
logger.Info("step 2")
```

### 4. Handle Context Cancellation

```go
func StreamProcessor(ctx context.Context, stream <-chan Data) error {
    for {
        select {
        case <-ctx.Done():
            slog.WarnContext(ctx, "stream processing cancelled", "error", ctx.Err())
            return ctx.Err()
        case data := <-stream:
            slog.DebugContext(ctx, "processing data", "id", data.ID)
            // Process data
        }
    }
}
```

## See Also

- [Best Practices](best-practices.md) - Context usage patterns
- [Custom Handlers](custom-handlers.md) - Context-aware handler implementation
- [Logger Configuration](logger-configuration.md) - Setting up context-aware loggers
