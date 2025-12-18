# Attributes and Values

The `Attr` and `Value` types provide type-safe, efficient structured logging without allocations for common types.

## Attr Type

An `Attr` represents a key-value pair:

```go
type Attr struct {
    Key   string
    Value Value
}
```

## Creating Attrs

### Type-Specific Constructors

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

Examples:

```go
attrs := []slog.Attr{
    slog.String("name", "alice"),
    slog.Int("age", 30),
    slog.Bool("active", true),
    slog.Duration("elapsed", 5*time.Second),
    slog.Time("created_at", time.Now()),
}
```

### Group Attrs

Group related attributes:

```go
func Group(key string, args ...any) Attr
func GroupAttrs(key string, attrs ...Attr) Attr
```

Examples:

```go
// Using alternating key-value pairs
attr := slog.Group("user",
    "id", 123,
    "name", "alice",
)

// Using Attr slice
attr := slog.GroupAttrs("user",
    slog.Int("id", 123),
    slog.String("name", "alice"),
)
```

## Value Type

The `Value` type represents any Go value without allocation for common types:

```go
type Value struct { /* ... */ }
```

### Value Kinds

```go
type Kind int

const (
    KindAny       Kind = iota  // Any Go value (uses reflection)
    KindBool                   // bool
    KindDuration               // time.Duration
    KindFloat64                // float64
    KindInt64                  // int64
    KindString                 // string
    KindTime                   // time.Time
    KindUint64                 // uint64
    KindGroup                  // []Attr
    KindLogValuer              // LogValuer interface
)
```

## Creating Values

### Type-Specific Value Constructors

```go
func StringValue(value string) Value
func IntValue(v int) Value
func Int64Value(v int64) Value
func Uint64Value(v uint64) Value
func Float64Value(v float64) Value
func BoolValue(v bool) Value
func TimeValue(v time.Time) Value
func DurationValue(v time.Duration) Value
func GroupValue(as ...Attr) Value
func AnyValue(v any) Value
```

Examples:

```go
v1 := slog.StringValue("hello")
v2 := slog.IntValue(42)
v3 := slog.BoolValue(true)
v4 := slog.TimeValue(time.Now())
```

## Value Methods

### Kind

Get the value's kind:

```go
func (v Value) Kind() Kind
```

Example:

```go
v := slog.IntValue(42)
if v.Kind() == slog.KindInt64 {
    fmt.Println("Integer value:", v.Int64())
}
```

### Type Conversion Methods

Extract the underlying value:

```go
func (v Value) String() string
func (v Value) Int64() int64
func (v Value) Uint64() uint64
func (v Value) Float64() float64
func (v Value) Bool() bool
func (v Value) Time() time.Time
func (v Value) Duration() time.Duration
func (v Value) Group() []Attr
func (v Value) Any() any
func (v Value) LogValuer() LogValuer
```

Example:

```go
v := slog.IntValue(42)
n := v.Int64() // 42

v2 := slog.StringValue("hello")
s := v2.String() // "hello"
```

### Resolve

Recursively resolve `LogValuer` values:

```go
func (v Value) Resolve() Value
```

Example:

```go
type Token string

func (t Token) LogValue() slog.Value {
    return slog.StringValue("REDACTED")
}

v := slog.AnyValue(Token("secret"))
resolved := v.Resolve()
fmt.Println(resolved.String()) // "REDACTED"
```

### Equal

Compare two values:

```go
func (v Value) Equal(w Value) bool
```

Example:

```go
v1 := slog.IntValue(42)
v2 := slog.IntValue(42)
v3 := slog.IntValue(99)

fmt.Println(v1.Equal(v2)) // true
fmt.Println(v1.Equal(v3)) // false
```

## Using Attrs for Performance

### LogAttrs vs Regular Logging

```go
// Slower: uses reflection for type conversion
logger.Info("message", "count", 42, "active", true)

// Faster: no reflection, zero allocations for basic types
logger.LogAttrs(context.Background(), slog.LevelInfo, "message",
    slog.Int("count", 42),
    slog.Bool("active", true),
)
```

### Method Signatures

```go
func Log(ctx context.Context, level Level, msg string, args ...any)
func LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr)

func (l *Logger) Log(ctx context.Context, level Level, msg string, args ...any)
func (l *Logger) LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr)
```

## LogValuer Interface

Implement custom logging behavior for types:

```go
type LogValuer interface {
    LogValue() Value
}
```

### Example: Redacting Sensitive Data

```go
type CreditCard string

func (cc CreditCard) LogValue() slog.Value {
    if len(cc) < 4 {
        return slog.StringValue("INVALID")
    }
    // Show last 4 digits only
    last4 := cc[len(cc)-4:]
    return slog.StringValue("****" + string(last4))
}

card := CreditCard("1234567812345678")
slog.Info("payment processed", "card", card)
// Output: ... card=****5678
```

### Example: Custom Formatting

```go
type User struct {
    ID       int
    Username string
    Email    string
}

func (u User) LogValue() slog.Value {
    return slog.GroupValue(
        slog.Int("id", u.ID),
        slog.String("username", u.Username),
        // Omit email for privacy
    )
}

user := User{ID: 123, Username: "alice", Email: "alice@example.com"}
slog.Info("user action", "user", user)
// Output: ... user.id=123 user.username=alice
```

### Example: Dynamic Values

```go
type LazyQuery struct {
    SQL  string
    Args []any
}

func (q LazyQuery) LogValue() slog.Value {
    // Only format the query if actually logged
    formatted := fmt.Sprintf(q.SQL, q.Args...)
    return slog.StringValue(formatted)
}

query := LazyQuery{SQL: "SELECT * FROM users WHERE id = %d", Args: []any{123}}
slog.Debug("executing query", "query", query)
// Query is only formatted if Debug level is enabled
```

## Empty and Zero Values

### Creating Empty Attrs

Return an empty `Attr` to omit it:

```go
opts := &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == "password" {
            return slog.Attr{} // Remove this attribute
        }
        return a
    },
}
```

### Zero Value

An `Attr` with an empty key is ignored:

```go
attr := slog.Attr{Key: "", Value: slog.IntValue(42)} // Ignored
```

## Working with Groups

### Creating Groups

```go
// Method 1: Group with alternating args
attr := slog.Group("request",
    "method", "GET",
    "path", "/api/users",
    "ip", "192.168.1.1",
)

// Method 2: GroupAttrs with Attr slice
attr := slog.GroupAttrs("request",
    slog.String("method", "GET"),
    slog.String("path", "/api/users"),
    slog.String("ip", "192.168.1.1"),
)

// Method 3: GroupValue for Value type
value := slog.GroupValue(
    slog.String("method", "GET"),
    slog.String("path", "/api/users"),
)
```

### Extracting Group Contents

```go
attr := slog.Group("user", "id", 123, "name", "alice")
value := attr.Value

if value.Kind() == slog.KindGroup {
    attrs := value.Group()
    for _, a := range attrs {
        fmt.Printf("%s: %v\n", a.Key, a.Value)
    }
}
```

## Performance Considerations

### Zero-Allocation Types

These types don't allocate when creating Values:

- `string`
- `int`, `int64`, `uint64`
- `float64`
- `bool`
- `time.Time`
- `time.Duration`

```go
// No allocations
v1 := slog.IntValue(42)
v2 := slog.StringValue("hello")
v3 := slog.BoolValue(true)
```

### Types That May Allocate

- `any` (uses reflection)
- Complex types (structs, slices, maps without LogValuer)

```go
// May allocate
v := slog.AnyValue(complexStruct)

// Better: implement LogValuer
func (c ComplexStruct) LogValue() slog.Value {
    return slog.GroupValue(
        slog.String("field1", c.Field1),
        slog.Int("field2", c.Field2),
    )
}
```

## Attr and Value Best Practices

1. **Prefer specific constructors**: Use `Int()`, `String()`, etc. over `Any()`
2. **Use LogAttrs for hot paths**: Avoid reflection overhead
3. **Implement LogValuer**: For custom types logged frequently
4. **Use Groups for structure**: Group related attributes
5. **Check Enabled before expensive ops**: Avoid work if level is disabled

Example:

```go
if logger.Enabled(ctx, slog.LevelDebug) {
    // Only compute expensive value if debug is enabled
    debugData := computeExpensiveDebugData()
    logger.LogAttrs(ctx, slog.LevelDebug, "debug info",
        slog.Any("data", debugData),
    )
}
```

## Attr Methods

### Equal

Compare attributes:

```go
func (a Attr) Equal(b Attr) bool
```

Example:

```go
a1 := slog.Int("count", 42)
a2 := slog.Int("count", 42)
a3 := slog.Int("count", 99)

fmt.Println(a1.Equal(a2)) // true
fmt.Println(a1.Equal(a3)) // false
```

### String

Get string representation:

```go
func (a Attr) String() string
```

Example:

```go
attr := slog.Int("count", 42)
fmt.Println(attr.String()) // "count=42"
```

## Complete Example

```go
type Request struct {
    Method string
    Path   string
    IP     string
    UserID int
}

func (r Request) LogValue() slog.Value {
    return slog.GroupValue(
        slog.String("method", r.Method),
        slog.String("path", r.Path),
        slog.String("ip", r.IP),
        slog.Int("user_id", r.UserID),
    )
}

func handleRequest(ctx context.Context, req Request) {
    start := time.Now()

    logger := slog.Default()

    // Fast path: use LogAttrs
    logger.LogAttrs(ctx, slog.LevelInfo, "request started",
        slog.Any("request", req),
    )

    // ... process request ...

    logger.LogAttrs(ctx, slog.LevelInfo, "request completed",
        slog.Any("request", req),
        slog.Duration("duration", time.Since(start)),
        slog.Int("status", 200),
    )
}
```

## Next Steps

- Learn performance optimization in [Best Practices](best-practices.md)
- See complete API in [Logger Type](api-logger.md)
- Explore handler implementation in [Handlers](handlers.md)
