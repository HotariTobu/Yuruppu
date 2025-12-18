# Best Practices

Guidelines for effective and performant use of the log/slog package.

## Performance Optimization

### 1. Use LogAttrs for Hot Paths

Avoid reflection overhead by using `LogAttrs` with type-specific constructors:

```go
// Slower: uses reflection
logger.Info("request", "user_id", userID, "duration", duration, "status", status)

// Faster: no reflection, zero allocations
logger.LogAttrs(ctx, slog.LevelInfo, "request",
    slog.Int("user_id", userID),
    slog.Duration("duration", duration),
    slog.Int("status", status),
)
```

### 2. Check Enabled Before Expensive Operations

Avoid computing expensive values when logging is disabled:

```go
// Bad: always computes expensive value
logger.Debug("data", "payload", computeExpensiveDebugData())

// Good: only compute if debug is enabled
if logger.Enabled(ctx, slog.LevelDebug) {
    logger.Debug("data", "payload", computeExpensiveDebugData())
}
```

### 3. Implement LogValuer for Custom Types

Avoid repeated formatting by implementing `LogValuer`:

```go
type User struct {
    ID       int
    Username string
    Email    string
}

// Efficient: called only when logged
func (u User) LogValue() slog.Value {
    return slog.GroupValue(
        slog.Int("id", u.ID),
        slog.String("username", u.Username),
    )
}

// Now this is efficient
logger.Info("user action", "user", user)
```

### 4. Reuse Loggers with Pre-configured Attributes

Create loggers once with common attributes:

```go
// Bad: repeating attributes on every call
logger.Info("event", "service", "api", "version", "1.0")
logger.Info("event", "service", "api", "version", "1.0")

// Good: pre-configure once
apiLogger := logger.With("service", "api", "version", "1.0")
apiLogger.Info("event")
apiLogger.Info("event")
```

### 5. Use Specific Attr Constructors

Prefer specific constructors over `Any()`:

```go
// Slower: may allocate and use reflection
slog.Any("count", 42)
slog.Any("name", "alice")

// Faster: type-specific, no allocation
slog.Int("count", 42)
slog.String("name", "alice")
```

## Structured Logging Patterns

### 1. Consistent Key Naming

Use consistent, predictable key names:

```go
// Good: consistent snake_case
logger.Info("event", "user_id", 123, "request_id", "abc")

// Avoid: inconsistent naming
logger.Info("event", "userID", 123, "request-id", "abc")
```

### 2. Use Groups for Related Data

Group related attributes together:

```go
// Good: grouped structure
logger.Info("http request",
    slog.Group("request",
        "method", r.Method,
        "path", r.URL.Path,
        "remote_addr", r.RemoteAddr,
    ),
    slog.Group("response",
        "status", statusCode,
        "bytes", bytesWritten,
        "duration_ms", duration.Milliseconds(),
    ),
)

// Avoid: flat structure with prefixes
logger.Info("http request",
    "request_method", r.Method,
    "request_path", r.URL.Path,
    "response_status", statusCode,
    "response_bytes", bytesWritten,
)
```

### 3. Add Context at Function Entry

Create function-scoped loggers with context:

```go
func ProcessOrder(ctx context.Context, orderID int) error {
    logger := slog.Default().With("order_id", orderID)

    logger.Info("processing started")

    // ... business logic ...

    logger.Info("processing completed", "status", "success")
    return nil
}
```

### 4. Log Errors with Context

Always include relevant context when logging errors:

```go
// Bad: minimal context
logger.Error("failed", "error", err)

// Good: full context
logger.Error("database query failed",
    "error", err,
    "operation", "insert",
    "table", "users",
    "user_id", userID,
    "retry_count", retryCount,
)
```

## Log Levels

### When to Use Each Level

#### Debug
- Detailed diagnostic information
- Variable values and execution flow
- Usually disabled in production

```go
logger.Debug("parsing request body",
    "content_type", contentType,
    "size_bytes", len(body),
)
```

#### Info
- Normal application events
- Startup/shutdown messages
- State changes

```go
logger.Info("server started", "port", 8080, "env", "production")
logger.Info("user logged in", "user_id", 123)
```

#### Warn
- Unexpected but handled conditions
- Deprecated feature usage
- Performance issues

```go
logger.Warn("slow query detected",
    "duration_ms", duration.Milliseconds(),
    "threshold_ms", 1000,
    "query", query,
)
```

#### Error
- Error conditions requiring attention
- Operation failures
- Exception conditions

```go
logger.Error("payment processing failed",
    "error", err,
    "order_id", orderID,
    "amount", amount,
)
```

## Security

### 1. Redact Sensitive Information

Implement `LogValuer` for sensitive types:

```go
type Password string

func (Password) LogValue() slog.Value {
    return slog.StringValue("REDACTED")
}

type APIKey string

func (APIKey) LogValue() slog.Value {
    return slog.StringValue("REDACTED")
}

// Usage
logger.Info("authentication", "password", Password("secret123"))
// Output: ... password=REDACTED
```

### 2. Use ReplaceAttr for Sensitive Keys

Filter sensitive keys globally:

```go
opts := &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        sensitiveKeys := map[string]bool{
            "password": true,
            "token":    true,
            "api_key":  true,
            "secret":   true,
        }

        if sensitiveKeys[a.Key] {
            return slog.String(a.Key, "REDACTED")
        }
        return a
    },
}
```

### 3. Sanitize User Input

Be careful with user-provided data:

```go
// Bad: raw user input
logger.Info("search", "query", userQuery)

// Good: sanitized or limited
logger.Info("search", "query", sanitize(userQuery))
```

## Error Handling

### 1. Include Error Details

Always log the full error with context:

```go
if err != nil {
    logger.Error("operation failed",
        "error", err,
        "operation", "database_insert",
        "context", additionalContext,
    )
    return err
}
```

### 2. Log Once Per Error

Avoid logging the same error multiple times:

```go
// Bad: error logged at every level
func A() error {
    if err := B(); err != nil {
        logger.Error("B failed", "error", err)
        return err
    }
}

func B() error {
    if err := C(); err != nil {
        logger.Error("C failed", "error", err)
        return err
    }
}

// Good: error logged once at top level
func A() error {
    if err := B(); err != nil {
        logger.Error("operation failed", "error", err, "operation", "B")
        return err
    }
}

func B() error {
    if err := C(); err != nil {
        return fmt.Errorf("B: %w", err) // Wrap, don't log
    }
}
```

### 3. Use Error Wrapping

Wrap errors to preserve context:

```go
result, err := db.Query(ctx, query)
if err != nil {
    logger.Error("query failed",
        "error", err, // Original error preserved
        "query", query,
        "table", table,
    )
    return fmt.Errorf("database query failed: %w", err)
}
```

## Testing

### 1. Test with Buffer

Capture logs for testing:

```go
func TestHandler(t *testing.T) {
    var buf bytes.Buffer
    handler := slog.NewJSONHandler(&buf, nil)
    logger := slog.New(handler)

    logger.Info("test message", "key", "value")

    var logEntry map[string]any
    if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
        t.Fatal(err)
    }

    if logEntry["msg"] != "test message" {
        t.Errorf("unexpected message: %v", logEntry["msg"])
    }
}
```

### 2. Use DiscardHandler for Benchmarks

Measure logging overhead:

```go
func BenchmarkLogging(b *testing.B) {
    logger := slog.New(slog.DiscardHandler)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        logger.Info("message", "iteration", i)
    }
}
```

### 3. Test Custom Handlers

Test handler behavior:

```go
type TestHandler struct {
    records []slog.Record
}

func (h *TestHandler) Handle(ctx context.Context, r slog.Record) error {
    h.records = append(h.records, r)
    return nil
}

func (h *TestHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return true
}

func (h *TestHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return h
}

func (h *TestHandler) WithGroup(name string) slog.Handler {
    return h
}

func TestCustomHandler(t *testing.T) {
    handler := &TestHandler{}
    logger := slog.New(handler)

    logger.Info("test")

    if len(handler.records) != 1 {
        t.Errorf("expected 1 record, got %d", len(handler.records))
    }
}
```

## Context Integration

### 1. Pass Context to Logging Functions

Use context-aware methods:

```go
// Good: preserves context
logger.InfoContext(ctx, "processing", "step", 1)

// Acceptable but misses context benefits
logger.Info("processing", "step", 1)
```

### 2. Extract Trace IDs from Context

```go
func logWithTrace(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
    traceID, ok := ctx.Value("trace_id").(string)
    if ok {
        args = append(args, "trace_id", traceID)
    }
    logger.InfoContext(ctx, msg, args...)
}
```

### 3. Create Context-Aware Loggers

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
func Handler(ctx context.Context) {
    logger := LoggerFromContext(ctx)
    logger.Info("handling request")
}
```

## Configuration

### 1. Environment-Based Setup

```go
func NewLogger() *slog.Logger {
    env := os.Getenv("ENV")
    level := slog.LevelInfo
    if env == "development" {
        level = slog.LevelDebug
    }

    var handler slog.Handler
    if env == "production" {
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level:     level,
            AddSource: false,
        })
    } else {
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level:     level,
            AddSource: true,
        })
    }

    return slog.New(handler).With(
        "env", env,
        "service", "myapp",
    )
}
```

### 2. Dynamic Level Changes

```go
var logLevel = new(slog.LevelVar)

func init() {
    handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: logLevel,
    })
    slog.SetDefault(slog.New(handler))
}

func SetLogLevel(level slog.Level) {
    logLevel.Set(level)
    slog.Info("log level changed", "level", level.String())
}
```

## Common Anti-Patterns

### 1. Avoid String Formatting

```go
// Bad: string formatting before logging
logger.Info(fmt.Sprintf("user %d logged in", userID))

// Good: structured
logger.Info("user logged in", "user_id", userID)
```

### 2. Don't Log in Loops (Usually)

```go
// Bad: logs every iteration
for i := 0; i < 1000000; i++ {
    logger.Debug("processing", "index", i)
}

// Good: log milestones
for i := 0; i < 1000000; i++ {
    if i%10000 == 0 {
        logger.Debug("processing", "index", i, "progress_pct", i/10000)
    }
}
```

### 3. Avoid Side Effects in LogValuer

```go
// Bad: side effects
func (u *User) LogValue() slog.Value {
    u.lastLogged = time.Now() // Side effect!
    return slog.StringValue(u.Name)
}

// Good: pure function
func (u User) LogValue() slog.Value {
    return slog.StringValue(u.Name)
}
```

## Summary Checklist

- [ ] Use `LogAttrs` for performance-critical paths
- [ ] Check `Enabled()` before expensive operations
- [ ] Implement `LogValuer` for frequently logged custom types
- [ ] Use consistent key naming conventions
- [ ] Group related attributes
- [ ] Redact sensitive information
- [ ] Log errors with full context
- [ ] Avoid logging the same error multiple times
- [ ] Use appropriate log levels
- [ ] Create function-scoped loggers for context
- [ ] Pass context to logging functions
- [ ] Configure based on environment
- [ ] Test logging behavior
- [ ] Avoid string formatting in log messages
- [ ] Use JSON format in production
- [ ] Use text format in development

## Next Steps

- Review [Structured Logging](structured-logging.md) patterns
- Explore [Context Support](context-support.md) for distributed tracing
- See [Custom Handlers](custom-handlers.md) for advanced use cases
