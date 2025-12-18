# Attr and Value Types API Reference

Complete API reference for `Attr` and `Value` types used for structured logging.

## Attr Type

```go
type Attr struct {
    Key   string
    Value Value
}
```

A key-value pair representing a log attribute.

## Attr Constructors

### String

```go
func String(key, value string) Attr
```

Create a string attribute.

Example:
```go
attr := slog.String("username", "alice")
```

### Int

```go
func Int(key string, value int) Attr
```

Create an int attribute (stored as int64).

Example:
```go
attr := slog.Int("count", 42)
```

### Int64

```go
func Int64(key string, value int64) Attr
```

Create an int64 attribute.

Example:
```go
attr := slog.Int64("timestamp", time.Now().Unix())
```

### Uint64

```go
func Uint64(key string, v uint64) Attr
```

Create a uint64 attribute.

Example:
```go
attr := slog.Uint64("size", uint64(1024))
```

### Float64

```go
func Float64(key string, v float64) Attr
```

Create a float64 attribute.

Example:
```go
attr := slog.Float64("cpu_usage", 0.85)
```

### Bool

```go
func Bool(key string, v bool) Attr
```

Create a boolean attribute.

Example:
```go
attr := slog.Bool("active", true)
```

### Time

```go
func Time(key string, v time.Time) Attr
```

Create a time.Time attribute.

Example:
```go
attr := slog.Time("created_at", time.Now())
```

### Duration

```go
func Duration(key string, v time.Duration) Attr
```

Create a time.Duration attribute.

Example:
```go
attr := slog.Duration("elapsed", 5*time.Second)
```

### Any

```go
func Any(key string, value any) Attr
```

Create an attribute from any value. Uses reflection and may allocate.

Example:
```go
attr := slog.Any("data", complexStruct)
```

### Group

```go
func Group(key string, args ...any) Attr
```

Create a group attribute with alternating key-value pairs.

Example:
```go
attr := slog.Group("user",
    "id", 123,
    "name", "alice",
)
```

### GroupAttrs

```go
func GroupAttrs(key string, attrs ...Attr) Attr
```

Create a group attribute from a slice of Attrs.

Example:
```go
attr := slog.GroupAttrs("user",
    slog.Int("id", 123),
    slog.String("name", "alice"),
)
```

## Attr Methods

### Equal

```go
func (a Attr) Equal(b Attr) bool
```

Report whether two attributes are equal.

Example:
```go
a1 := slog.Int("count", 42)
a2 := slog.Int("count", 42)
fmt.Println(a1.Equal(a2)) // true
```

### String

```go
func (a Attr) String() string
```

Return a string representation of the attribute.

Example:
```go
attr := slog.Int("count", 42)
fmt.Println(attr.String()) // "count=42"
```

## Value Type

```go
type Value struct {
    // contains filtered or unexported fields
}
```

A type-safe representation of any Go value, optimized to avoid allocations for common types.

## Value Constructors

### StringValue

```go
func StringValue(value string) Value
```

Create a string Value.

Example:
```go
v := slog.StringValue("hello")
```

### IntValue

```go
func IntValue(v int) Value
```

Create an int Value (stored as int64).

Example:
```go
v := slog.IntValue(42)
```

### Int64Value

```go
func Int64Value(v int64) Value
```

Create an int64 Value.

Example:
```go
v := slog.Int64Value(123456789)
```

### Uint64Value

```go
func Uint64Value(v uint64) Value
```

Create a uint64 Value.

Example:
```go
v := slog.Uint64Value(uint64(1024))
```

### Float64Value

```go
func Float64Value(v float64) Value
```

Create a float64 Value.

Example:
```go
v := slog.Float64Value(3.14159)
```

### BoolValue

```go
func BoolValue(v bool) Value
```

Create a boolean Value.

Example:
```go
v := slog.BoolValue(true)
```

### TimeValue

```go
func TimeValue(v time.Time) Value
```

Create a time.Time Value.

Example:
```go
v := slog.TimeValue(time.Now())
```

### DurationValue

```go
func DurationValue(v time.Duration) Value
```

Create a time.Duration Value.

Example:
```go
v := slog.DurationValue(5 * time.Second)
```

### GroupValue

```go
func GroupValue(as ...Attr) Value
```

Create a group Value from attributes.

Example:
```go
v := slog.GroupValue(
    slog.String("method", "GET"),
    slog.String("path", "/api/users"),
)
```

### AnyValue

```go
func AnyValue(v any) Value
```

Create a Value from any Go value. Uses reflection.

Example:
```go
v := slog.AnyValue(myStruct)
```

## Kind Type

```go
type Kind int
```

The kind of a Value.

```go
const (
    KindAny       Kind = iota  // Any Go value
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

## Value Methods

### Kind

```go
func (v Value) Kind() Kind
```

Return the kind of the value.

Example:
```go
v := slog.IntValue(42)
if v.Kind() == slog.KindInt64 {
    fmt.Println("Integer:", v.Int64())
}
```

### String

```go
func (v Value) String() string
```

Return the value as a string. Panics if Kind is not KindString.

Example:
```go
v := slog.StringValue("hello")
s := v.String() // "hello"
```

### Int64

```go
func (v Value) Int64() int64
```

Return the value as int64. Panics if Kind is not KindInt64.

Example:
```go
v := slog.IntValue(42)
n := v.Int64() // 42
```

### Uint64

```go
func (v Value) Uint64() uint64
```

Return the value as uint64. Panics if Kind is not KindUint64.

Example:
```go
v := slog.Uint64Value(1024)
n := v.Uint64() // 1024
```

### Float64

```go
func (v Value) Float64() float64
```

Return the value as float64. Panics if Kind is not KindFloat64.

Example:
```go
v := slog.Float64Value(3.14)
f := v.Float64() // 3.14
```

### Bool

```go
func (v Value) Bool() bool
```

Return the value as bool. Panics if Kind is not KindBool.

Example:
```go
v := slog.BoolValue(true)
b := v.Bool() // true
```

### Time

```go
func (v Value) Time() time.Time
```

Return the value as time.Time. Panics if Kind is not KindTime.

Example:
```go
now := time.Now()
v := slog.TimeValue(now)
t := v.Time() // now
```

### Duration

```go
func (v Value) Duration() time.Duration
```

Return the value as time.Duration. Panics if Kind is not KindDuration.

Example:
```go
v := slog.DurationValue(5 * time.Second)
d := v.Duration() // 5s
```

### Group

```go
func (v Value) Group() []Attr
```

Return the value as []Attr. Panics if Kind is not KindGroup.

Example:
```go
v := slog.GroupValue(
    slog.String("a", "b"),
    slog.Int("c", 1),
)
attrs := v.Group() // []Attr with 2 elements
```

### Any

```go
func (v Value) Any() any
```

Return the value as any.

Example:
```go
v := slog.IntValue(42)
a := v.Any() // interface{} containing 42
```

### LogValuer

```go
func (v Value) LogValuer() LogValuer
```

Return the value as LogValuer. Panics if Kind is not KindLogValuer.

### Resolve

```go
func (v Value) Resolve() Value
```

Recursively resolve LogValuer values. If the value is not KindLogValuer, returns the value unchanged.

Example:
```go
type Token string

func (Token) LogValue() slog.Value {
    return slog.StringValue("REDACTED")
}

v := slog.AnyValue(Token("secret"))
resolved := v.Resolve()
fmt.Println(resolved.String()) // "REDACTED"
```

### Equal

```go
func (v Value) Equal(w Value) bool
```

Report whether two values are equal.

Example:
```go
v1 := slog.IntValue(42)
v2 := slog.IntValue(42)
v3 := slog.IntValue(99)

fmt.Println(v1.Equal(v2)) // true
fmt.Println(v1.Equal(v3)) // false
```

## LogValuer Interface

```go
type LogValuer interface {
    LogValue() Value
}
```

Implement this interface to control how a type is logged.

### Example: Redacting Sensitive Data

```go
type Password string

func (Password) LogValue() slog.Value {
    return slog.StringValue("REDACTED")
}

// Usage
pwd := Password("secret123")
slog.Info("login", "password", pwd)
// Output: ... password=REDACTED
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

// Usage
user := User{ID: 123, Username: "alice", Email: "alice@example.com"}
slog.Info("user action", "user", user)
// Output: ... user.id=123 user.username=alice
```

### Example: Lazy Evaluation

```go
type LazyJSON struct {
    data interface{}
}

func (lj LazyJSON) LogValue() slog.Value {
    // Only marshal to JSON if actually logged
    b, _ := json.Marshal(lj.data)
    return slog.StringValue(string(b))
}

// JSON is only created if the log level is enabled
slog.Debug("data", "json", LazyJSON{data: complexObject})
```

## Performance Characteristics

### Zero-Allocation Types

These types don't allocate when creating Values:

```go
slog.StringValue("text")      // No allocation
slog.IntValue(42)              // No allocation
slog.Int64Value(123)           // No allocation
slog.Uint64Value(456)          // No allocation
slog.Float64Value(3.14)        // No allocation
slog.BoolValue(true)           // No allocation
slog.TimeValue(time.Now())     // No allocation
slog.DurationValue(5*time.Second) // No allocation
```

### May Allocate

```go
slog.AnyValue(complexStruct)   // May allocate, uses reflection
slog.GroupValue(attrs...)      // Allocates for []Attr slice
```

## Empty and Zero Values

### Empty Attr

An Attr with an empty key is ignored:

```go
// This attribute is ignored
attr := slog.Attr{Key: "", Value: slog.IntValue(42)}
```

### Zero Value

The zero Value is the zero value of the any type:

```go
var v slog.Value
fmt.Println(v.Kind()) // KindAny
fmt.Println(v.Any())  // nil
```

## Usage Examples

### Type-Safe Logging

```go
logger.LogAttrs(ctx, slog.LevelInfo, "user action",
    slog.String("action", "login"),
    slog.Int("user_id", 123),
    slog.Bool("success", true),
    slog.Duration("duration", 250*time.Millisecond),
)
```

### Working with Groups

```go
requestAttr := slog.Group("request",
    "method", r.Method,
    "path", r.URL.Path,
    "remote_addr", r.RemoteAddr,
)

responseAttr := slog.Group("response",
    "status", 200,
    "bytes", 1024,
)

logger.LogAttrs(ctx, slog.LevelInfo, "http request",
    requestAttr,
    responseAttr,
)
```

### Custom Types with LogValuer

```go
type CreditCard string

func (cc CreditCard) LogValue() slog.Value {
    if len(cc) < 4 {
        return slog.StringValue("INVALID")
    }
    return slog.StringValue("****" + string(cc[len(cc)-4:]))
}

card := CreditCard("1234567812345678")
logger.Info("payment", "card", card)
// Output: ... card=****5678
```

### Checking Value Kind

```go
func processValue(v slog.Value) {
    switch v.Kind() {
    case slog.KindString:
        fmt.Println("String:", v.String())
    case slog.KindInt64:
        fmt.Println("Int:", v.Int64())
    case slog.KindGroup:
        fmt.Println("Group with", len(v.Group()), "attrs")
    default:
        fmt.Println("Other:", v.Any())
    }
}
```

## See Also

- [Attributes and Values Guide](attributes-values.md) - Detailed usage guide
- [Structured Logging](structured-logging.md) - Structured logging patterns
- [Best Practices](best-practices.md) - Performance and usage tips
