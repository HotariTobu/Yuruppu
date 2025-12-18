# Level and Record Types API Reference

Complete API reference for log levels, records, and related types.

## Level Type

```go
type Level int
```

The severity level of a log message.

## Standard Levels

```go
const (
    LevelDebug Level = -4
    LevelInfo  Level = 0
    LevelWarn  Level = 4
    LevelError Level = 8
)
```

Levels are integers to allow custom levels between standard values:

```go
const (
    LevelTrace  = slog.LevelDebug - 4  // -8
    LevelNotice = slog.LevelInfo + 2   // 2
    LevelFatal  = slog.LevelError + 4  // 12
)
```

## Level Methods

### Level

```go
func (l Level) Level() Level
```

Return the level (implements Leveler interface).

### String

```go
func (l Level) String() string
```

Return string representation of the level.

Example:
```go
level := slog.LevelInfo
fmt.Println(level.String()) // "INFO"
```

Standard level strings:
- `LevelDebug` → "DEBUG"
- `LevelInfo` → "INFO"
- `LevelWarn` → "WARN"
- `LevelError` → "ERROR"

### MarshalText

```go
func (l Level) MarshalText() ([]byte, error)
```

Marshal the level as text.

Example:
```go
level := slog.LevelInfo
text, _ := level.MarshalText()
fmt.Println(string(text)) // "INFO"
```

### UnmarshalText

```go
func (l *Level) UnmarshalText(data []byte) error
```

Unmarshal level from text.

Example:
```go
var level slog.Level
err := level.UnmarshalText([]byte("WARN"))
fmt.Println(level) // LevelWarn (4)
```

### MarshalJSON

```go
func (l Level) MarshalJSON() ([]byte, error)
```

Marshal level as JSON string.

### UnmarshalJSON

```go
func (l *Level) UnmarshalJSON(data []byte) error
```

Unmarshal level from JSON.

### AppendText

```go
func (l Level) AppendText(b []byte) ([]byte, error)
```

Append the text representation to a byte slice.

## LevelVar Type

```go
type LevelVar struct {
    // contains filtered or unexported fields
}
```

A dynamic Level that can be changed at runtime.

### Creating LevelVar

```go
var programLevel = new(slog.LevelVar) // Defaults to LevelInfo
```

### LevelVar Methods

#### Level

```go
func (v *LevelVar) Level() Level
```

Return the current level.

Example:
```go
level := programLevel.Level()
fmt.Println(level) // 0 (LevelInfo)
```

#### Set

```go
func (v *LevelVar) Set(l Level)
```

Set the level dynamically.

Example:
```go
var programLevel = new(slog.LevelVar)

opts := &slog.HandlerOptions{
    Level: programLevel, // Use LevelVar
}
handler := slog.NewJSONHandler(os.Stderr, opts)
slog.SetDefault(slog.New(handler))

// Change level at runtime
programLevel.Set(slog.LevelDebug)
```

#### String

```go
func (v *LevelVar) String() string
```

Return string representation of the current level.

#### MarshalText

```go
func (v *LevelVar) MarshalText() ([]byte, error)
```

#### UnmarshalText

```go
func (v *LevelVar) UnmarshalText(data []byte) error
```

#### AppendText

```go
func (v *LevelVar) AppendText(b []byte) ([]byte, error)
```

## Leveler Interface

```go
type Leveler interface {
    Level() Level
}
```

Implemented by both `Level` and `LevelVar` to allow flexible level configuration.

## Record Type

```go
type Record struct {
    Time    time.Time
    Message string
    Level   Level
    PC      uintptr
    // contains filtered or unexported fields
}
```

A log record representing a single log event.

### NewRecord

```go
func NewRecord(t time.Time, level Level, msg string, pc uintptr) Record
```

Create a new Record. The `pc` parameter is a program counter from `runtime.Callers`.

Example:
```go
var pcs [1]uintptr
runtime.Callers(1, pcs[:])

record := slog.NewRecord(
    time.Now(),
    slog.LevelInfo,
    "message",
    pcs[0],
)
```

## Record Methods

### Add

```go
func (r *Record) Add(args ...any)
```

Add attributes from alternating key-value pairs.

Example:
```go
record.Add("key1", "value1", "key2", 42)
```

### AddAttrs

```go
func (r *Record) AddAttrs(attrs ...Attr)
```

Add pre-constructed Attr values.

Example:
```go
record.AddAttrs(
    slog.String("key1", "value1"),
    slog.Int("key2", 42),
)
```

### Attrs

```go
func (r Record) Attrs(f func(Attr) bool)
```

Iterate over the record's attributes. The function is called for each attribute; if it returns false, iteration stops.

Example:
```go
record.Attrs(func(a slog.Attr) bool {
    fmt.Printf("%s: %v\n", a.Key, a.Value)
    return true // Continue iteration
})
```

### NumAttrs

```go
func (r Record) NumAttrs() int
```

Return the number of attributes in the record.

Example:
```go
count := record.NumAttrs()
fmt.Println("Attribute count:", count)
```

### Clone

```go
func (r Record) Clone() Record
```

Create a copy of the record. The clone contains independent copies of all attributes.

Example:
```go
r2 := record.Clone()
r2.Add("extra", "attribute")
// r2 has the extra attribute, original record does not
```

### Source

```go
func (r Record) Source() *Source
```

Return the source code location from the PC field, or nil if PC is zero or invalid.

Example:
```go
if src := record.Source(); src != nil {
    fmt.Printf("%s:%d in %s\n", src.File, src.Line, src.Function)
}
```

## Source Type

```go
type Source struct {
    Function string `json:"function"`
    File     string `json:"file"`
    Line     int    `json:"line"`
}
```

Source code location information.

Example output:
```json
{
  "function": "main.handleRequest",
  "file": "/path/to/main.go",
  "line": 42
}
```

## Custom Level Example

Define custom levels for specific needs:

```go
const (
    LevelTrace   = slog.LevelDebug - 4  // -8
    LevelNotice  = slog.LevelInfo + 2   // 2
    LevelCritical = slog.LevelError + 4 // 12
    LevelFatal   = slog.LevelError + 8  // 16
)

func Trace(msg string, args ...any) {
    slog.Log(context.Background(), LevelTrace, msg, args...)
}

func Fatal(msg string, args ...any) {
    slog.Log(context.Background(), LevelFatal, msg, args...)
    os.Exit(1)
}
```

## Level Parsing Example

Parse level from string:

```go
func ParseLevel(s string) (slog.Level, error) {
    var level slog.Level
    err := level.UnmarshalText([]byte(strings.ToUpper(s)))
    return level, err
}

// Usage
level, err := ParseLevel("debug")
if err != nil {
    log.Fatal(err)
}

opts := &slog.HandlerOptions{Level: level}
```

## Dynamic Level Configuration Example

```go
type Config struct {
    logLevel *slog.LevelVar
}

func NewConfig() *Config {
    return &Config{
        logLevel: new(slog.LevelVar),
    }
}

func (c *Config) SetupLogger() *slog.Logger {
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: c.logLevel,
    })
    return slog.New(handler)
}

func (c *Config) SetLogLevel(level slog.Level) {
    c.logLevel.Set(level)
    slog.Info("log level changed", "level", level.String())
}

// Usage
config := NewConfig()
logger := config.SetupLogger()
slog.SetDefault(logger)

// Later, change level dynamically
config.SetLogLevel(slog.LevelDebug)
```

## Record Creation Example

Create custom log records for wrapping:

```go
func InfoWithCaller(logger *slog.Logger, skip int, msg string, args ...any) {
    if !logger.Enabled(context.Background(), slog.LevelInfo) {
        return
    }

    var pcs [1]uintptr
    runtime.Callers(skip+2, pcs[:])

    r := slog.NewRecord(time.Now(), slog.LevelInfo, msg, pcs[0])
    r.Add(args...)

    _ = logger.Handler().Handle(context.Background(), r)
}

// Usage
InfoWithCaller(logger, 0, "message", "key", "value")
```

## Record Manipulation Example

```go
type RecordTransformer struct {
    handler slog.Handler
}

func (rt *RecordTransformer) Handle(ctx context.Context, r slog.Record) error {
    // Clone to avoid modifying original
    r2 := r.Clone()

    // Add hostname to all records
    hostname, _ := os.Hostname()
    r2.AddAttrs(slog.String("hostname", hostname))

    // Transform level display
    r2.Level = r.Level

    return rt.handler.Handle(ctx, r2)
}
```

## Level Comparison

Levels can be compared:

```go
if record.Level >= slog.LevelError {
    // Handle error or critical logs specially
    sendAlert(record)
}

if record.Level < slog.LevelWarn {
    // Debug or Info level
    writeToDebugLog(record)
}
```

## Working with Source Information

```go
opts := &slog.HandlerOptions{
    AddSource: true,
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == slog.SourceKey {
            source := a.Value.Any().(*slog.Source)
            // Shorten file path
            source.File = filepath.Base(source.File)
            return slog.Any(slog.SourceKey, source)
        }
        return a
    },
}
```

## Package Constants

```go
const (
    TimeKey    = "time"
    LevelKey   = "level"
    MessageKey = "msg"
    SourceKey  = "source"
)
```

These constants define the default keys used by built-in handlers.

Example of customizing keys:

```go
opts := &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        // Rename "msg" to "message"
        if a.Key == slog.MessageKey {
            a.Key = "message"
        }
        // Rename "level" to "severity"
        if a.Key == slog.LevelKey {
            a.Key = "severity"
        }
        return a
    },
}
```

## See Also

- [Logger Configuration](logger-configuration.md) - Dynamic level configuration
- [Handler Interface](api-handler.md) - Handler API and Record handling
- [Best Practices](best-practices.md) - Level selection guidelines
