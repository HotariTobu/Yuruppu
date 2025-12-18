# Structured Logging

Structured logging uses key-value pairs instead of format strings, making logs machine-parseable and easier to query.

## Key-Value Pairs

### Basic Syntax

Arguments alternate between keys (strings) and values (any type):

```go
slog.Info("user action",
    "action", "login",
    "user_id", 123,
    "ip_address", "192.168.1.1",
)
```

Output (text format):
```
time=2024-01-15T10:30:00.000Z level=INFO msg="user action" action=login user_id=123 ip_address=192.168.1.1
```

Output (JSON format):
```json
{"time":"2024-01-15T10:30:00.000Z","level":"INFO","msg":"user action","action":"login","user_id":123,"ip_address":"192.168.1.1"}
```

## Using Attrs for Type Safety

For better performance and type safety, use `Attr` constructors:

### Common Attr Types

```go
func String(key, value string) Attr
func Int(key string, value int) Attr
func Int64(key string, value int64) Attr
func Uint64(key string, v uint64) Attr
func Float64(key string, v float64) Attr
func Bool(key string, v bool) Attr
func Time(key string, v time.Time) Attr
func Duration(key string, v time.Duration) Attr
func Any(key string, value any) Attr
```

### Using Attrs with LogAttrs

For maximum performance, use `LogAttrs` instead of `Info`/`Error`:

```go
logger.LogAttrs(context.Background(), slog.LevelInfo, "user action",
    slog.String("action", "login"),
    slog.Int("user_id", 123),
    slog.String("ip_address", "192.168.1.1"),
)
```

**Benefits:**
- Zero allocations for most value types
- Type checking at compile time
- Better performance for high-throughput logging

## Groups

Group related attributes together:

```go
slog.Info("request processed",
    "method", "GET",
    "path", "/api/users",
    slog.Group("response",
        "status", 200,
        "bytes", 1024,
    ),
    slog.Group("timing",
        "duration_ms", 45,
    ),
)
```

Text output:
```
level=INFO msg="request processed" method=GET path=/api/users response.status=200 response.bytes=1024 timing.duration_ms=45
```

JSON output:
```json
{
  "level": "INFO",
  "msg": "request processed",
  "method": "GET",
  "path": "/api/users",
  "response": {
    "status": 200,
    "bytes": 1024
  },
  "timing": {
    "duration_ms": 45
  }
}
```

## Logger with Persistent Attributes

Use `With()` to create a logger with pre-configured attributes:

```go
// Base logger
logger := slog.Default()

// Request-scoped logger with request ID
requestLogger := logger.With(
    "request_id", "abc-123",
    "user_id", 456,
)

// All logs from this logger include the attributes
requestLogger.Info("processing started")
requestLogger.Info("validation complete")
requestLogger.Info("processing complete")
```

All three logs will include `request_id` and `user_id`.

### Method Signature

```go
func (l *Logger) With(args ...any) *Logger
```

## Logger with Groups

Use `WithGroup()` to namespace all subsequent attributes:

```go
logger := slog.Default()
dbLogger := logger.WithGroup("database")

dbLogger.Info("query executed",
    "query", "SELECT * FROM users",
    "duration_ms", 23,
)
```

Output:
```
level=INFO msg="query executed" database.query="SELECT * FROM users" database.duration_ms=23
```

### Method Signature

```go
func (l *Logger) WithGroup(name string) *Logger
```

## Combining With and WithGroup

```go
logger := slog.Default()

// Add persistent attributes
appLogger := logger.With("app", "myapp", "version", "1.2.3")

// Create component logger with group
dbLogger := appLogger.WithGroup("database")
cacheLogger := appLogger.WithGroup("cache")

dbLogger.Info("connected", "host", "localhost")
// Output includes: app=myapp version=1.2.3 database.host=localhost

cacheLogger.Info("cache hit", "key", "user:123")
// Output includes: app=myapp version=1.2.3 cache.key=user:123
```

## LogValue Interface for Custom Types

Implement `LogValuer` to control how types are logged:

```go
type LogValuer interface {
    LogValue() Value
}
```

### Example: Redacting Sensitive Data

```go
type Password string

func (Password) LogValue() slog.Value {
    return slog.StringValue("REDACTED")
}

type User struct {
    Username string
    Password Password
}

func (u User) LogValue() slog.Value {
    return slog.GroupValue(
        slog.String("username", u.Username),
        slog.Any("password", u.Password), // Will be redacted
    )
}

user := User{Username: "alice", Password: "secret123"}
slog.Info("user created", "user", user)
// Output: level=INFO msg="user created" user.username=alice user.password=REDACTED
```

### Example: Formatting Complex Types

```go
type Request struct {
    Method string
    URL    *url.URL
    Headers http.Header
}

func (r Request) LogValue() slog.Value {
    return slog.GroupValue(
        slog.String("method", r.Method),
        slog.String("url", r.URL.String()),
        slog.Int("header_count", len(r.Headers)),
    )
}
```

## Best Practices

### 1. Use Consistent Key Names

```go
// Good: consistent naming
slog.Info("event", "user_id", 123)
slog.Info("event", "user_id", 456)

// Avoid: inconsistent naming
slog.Info("event", "user_id", 123)
slog.Info("event", "userId", 456) // Different key name
```

### 2. Prefer Attrs for Performance-Critical Code

```go
// Slower: uses reflection
logger.Info("message", "count", count, "user", user)

// Faster: no reflection
logger.LogAttrs(ctx, slog.LevelInfo, "message",
    slog.Int("count", count),
    slog.Any("user", user),
)
```

### 3. Use Groups for Related Data

```go
// Good: grouped related attributes
slog.Info("http request",
    slog.Group("request",
        "method", "GET",
        "path", "/api/users",
        "remote_addr", "192.168.1.1",
    ),
    slog.Group("response",
        "status", 200,
        "bytes", 1024,
    ),
)

// Avoid: flat structure for related data
slog.Info("http request",
    "request_method", "GET",
    "request_path", "/api/users",
    "request_remote_addr", "192.168.1.1",
    "response_status", 200,
    "response_bytes", 1024,
)
```

### 4. Create Scoped Loggers

```go
// Create loggers for different components
func NewDatabaseLogger() *slog.Logger {
    return slog.Default().WithGroup("database")
}

func NewCacheLogger() *slog.Logger {
    return slog.Default().WithGroup("cache")
}

// Use in your code
dbLogger := NewDatabaseLogger()
dbLogger.Info("query executed", "duration_ms", 45)
```

## Next Steps

- Learn about handler configuration in [Handlers](handlers.md)
- Explore the Attr API in [Attributes and Values](attributes-values.md)
- See advanced patterns in [Best Practices](best-practices.md)
