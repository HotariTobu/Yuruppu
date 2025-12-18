# Logger Configuration

This guide covers various configuration patterns for setting up slog loggers.

## Setting the Default Logger

Replace the package-level default logger:

```go
func SetDefault(l *Logger)
```

Example:

```go
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})
logger := slog.New(handler)
slog.SetDefault(logger)

// Now all top-level functions use this logger
slog.Debug("this will be logged") // Previously filtered
slog.Info("using JSON format")
```

## Dynamic Log Levels

Change log level at runtime without recreating loggers:

```go
type LevelVar struct { /* ... */ }

func (v *LevelVar) Level() Level
func (v *LevelVar) Set(l Level)
```

Example:

```go
// Create a level variable
var programLevel = new(slog.LevelVar) // Defaults to Info

// Use it in handler options
opts := &slog.HandlerOptions{
    Level: programLevel,
}
handler := slog.NewJSONHandler(os.Stderr, opts)
logger := slog.New(handler)
slog.SetDefault(logger)

// Initial state: Info level
slog.Debug("not visible")
slog.Info("visible")

// Change level dynamically
programLevel.Set(slog.LevelDebug)

// Now debug logs appear
slog.Debug("now visible")
```

### Use Cases for Dynamic Levels

1. **Debug mode toggle**: Enable debug logging in production for troubleshooting
2. **HTTP endpoint**: Change log level via API request
3. **Signal handling**: Increase verbosity with SIGUSR1

Example with HTTP endpoint:

```go
var logLevel = new(slog.LevelVar)

func init() {
    handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
    slog.SetDefault(slog.New(handler))
}

func handleSetLogLevel(w http.ResponseWriter, r *http.Request) {
    levelStr := r.URL.Query().Get("level")
    var level slog.Level
    if err := level.UnmarshalText([]byte(levelStr)); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    logLevel.Set(level)
    fmt.Fprintf(w, "Log level set to %s\n", level)
}

// GET /set-log-level?level=DEBUG
```

## Environment-Based Configuration

Configure logging based on environment:

```go
func setupLogger() *slog.Logger {
    var handler slog.Handler

    env := os.Getenv("ENV")

    switch env {
    case "production":
        // JSON format, Info level, with source
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level:     slog.LevelInfo,
            AddSource: true,
        })
    case "development":
        // Text format, Debug level
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelDebug,
        })
    default:
        // Default: JSON with Info
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelInfo,
        })
    }

    return slog.New(handler)
}

func main() {
    logger := setupLogger()
    slog.SetDefault(logger)

    slog.Info("application started", "env", os.Getenv("ENV"))
}
```

## Logger with Context Attributes

Create loggers with pre-configured attributes:

```go
func NewAppLogger(service string, version string) *slog.Logger {
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    })

    return slog.New(handler).With(
        "service", service,
        "version", version,
        "hostname", os.Hostname(),
        "pid", os.Getpid(),
    )
}

// Usage
logger := NewAppLogger("api-server", "1.2.3")
logger.Info("started") // Includes service, version, hostname, pid
```

## Per-Component Loggers

Create specialized loggers for different components:

```go
type Loggers struct {
    App      *slog.Logger
    Database *slog.Logger
    Cache    *slog.Logger
    HTTP     *slog.Logger
}

func NewLoggers() *Loggers {
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    })
    base := slog.New(handler)

    return &Loggers{
        App:      base.With("component", "app"),
        Database: base.With("component", "database"),
        Cache:    base.With("component", "cache"),
        HTTP:     base.With("component", "http"),
    }
}

// Usage
loggers := NewLoggers()
loggers.Database.Info("connected", "host", "localhost")
loggers.HTTP.Info("request", "method", "GET", "path", "/api/users")
```

## Structured Configuration with Groups

```go
func NewStructuredLogger() *slog.Logger {
    handler := slog.NewJSONHandler(os.Stdout, nil)

    return slog.New(handler).With(
        slog.Group("app",
            "name", "myapp",
            "version", "1.0.0",
            "env", os.Getenv("ENV"),
        ),
        slog.Group("system",
            "hostname", os.Hostname(),
            "pid", os.Getpid(),
        ),
    )
}

// All logs include:
// {
//   "app": {"name": "myapp", "version": "1.0.0", "env": "production"},
//   "system": {"hostname": "server-01", "pid": 12345},
//   ...
// }
```

## Custom Log Levels

Define custom log levels:

```go
const (
    LevelTrace   = slog.LevelDebug - 4  // -8
    LevelNotice  = slog.LevelInfo + 2   // 2
    LevelFatal   = slog.LevelError + 4  // 12
)

func Trace(msg string, args ...any) {
    slog.Log(context.Background(), LevelTrace, msg, args...)
}

func Fatal(msg string, args ...any) {
    slog.Log(context.Background(), LevelFatal, msg, args...)
    os.Exit(1)
}

// Usage
Trace("detailed trace information", "data", data)
Fatal("critical error", "error", err)
```

## Log Level from String

Parse log level from configuration:

```go
func parseLevelFromString(s string) (slog.Level, error) {
    var level slog.Level
    err := level.UnmarshalText([]byte(strings.ToUpper(s)))
    return level, err
}

// Usage
levelStr := os.Getenv("LOG_LEVEL") // "DEBUG", "INFO", "WARN", "ERROR"
if levelStr == "" {
    levelStr = "INFO"
}

level, err := parseLevelFromString(levelStr)
if err != nil {
    level = slog.LevelInfo
}

handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: level,
})
```

## Testing Configuration

For tests, use a buffer to capture output:

```go
func TestLogging(t *testing.T) {
    var buf bytes.Buffer
    handler := slog.NewJSONHandler(&buf, nil)
    logger := slog.New(handler)

    logger.Info("test message", "key", "value")

    output := buf.String()
    if !strings.Contains(output, "test message") {
        t.Errorf("expected log message, got %s", output)
    }
}
```

Or use `DiscardHandler` for performance tests:

```go
func BenchmarkLogging(b *testing.B) {
    logger := slog.New(slog.DiscardHandler)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        logger.Info("benchmark", "iteration", i)
    }
}
```

## Output to Multiple Destinations

### Separate Files by Level

```go
func setupMultipleOutputs() *slog.Logger {
    // All logs to main file
    mainFile, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

    // Errors to separate file
    errorFile, _ := os.OpenFile("error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

    // Use io.MultiWriter for main handler
    mainHandler := slog.NewJSONHandler(mainFile, nil)
    errorHandler := slog.NewJSONHandler(errorFile, &slog.HandlerOptions{
        Level: slog.LevelError,
    })

    // Implement MultiHandler (see handlers.md)
    multiHandler := NewMultiHandler(mainHandler, errorHandler)

    return slog.New(multiHandler)
}
```

### Console and File Output

```go
func setupConsoleAndFile() *slog.Logger {
    file, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

    // Console: text format
    consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    })

    // File: JSON format with debug
    fileHandler := slog.NewJSONHandler(file, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    })

    multiHandler := NewMultiHandler(consoleHandler, fileHandler)
    return slog.New(multiHandler)
}
```

## Standard Library Integration

### Replace standard log package

```go
func SetLogLoggerLevel(level Level) (oldLevel Level)
```

Configure the standard library's `log` package to use slog:

```go
handler := slog.NewJSONHandler(os.Stderr, nil)
logger := slog.NewLogLogger(handler, slog.LevelInfo)
log.SetOutput(io.Discard)
log.SetFlags(0)
// Now use logger instead of log
```

## Configuration Best Practices

1. **Single initialization**: Configure logging once at application startup
2. **Environment-aware**: Use different configurations for dev/staging/prod
3. **Dynamic levels**: Use `LevelVar` for runtime adjustments
4. **Persistent attributes**: Add service/version/hostname to root logger
5. **Component loggers**: Create specialized loggers with `With()` or `WithGroup()`
6. **Structured output**: Use JSON in production for parsing
7. **Human-readable dev**: Use TextHandler in development
8. **Centralize configuration**: Create a single `setupLogger()` function

## Example: Complete Configuration

```go
type LogConfig struct {
    Level      string
    Format     string // "json" or "text"
    AddSource  bool
    Output     string // "stdout", "stderr", or file path
}

func SetupLogging(cfg LogConfig) (*slog.Logger, error) {
    // Parse level
    var level slog.Level
    if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
        return nil, fmt.Errorf("invalid log level: %w", err)
    }

    // Select output
    var output io.Writer
    switch cfg.Output {
    case "stdout":
        output = os.Stdout
    case "stderr":
        output = os.Stderr
    default:
        f, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            return nil, fmt.Errorf("failed to open log file: %w", err)
        }
        output = f
    }

    // Handler options
    opts := &slog.HandlerOptions{
        Level:     level,
        AddSource: cfg.AddSource,
    }

    // Create handler
    var handler slog.Handler
    switch cfg.Format {
    case "json":
        handler = slog.NewJSONHandler(output, opts)
    case "text":
        handler = slog.NewTextHandler(output, opts)
    default:
        return nil, fmt.Errorf("invalid format: %s", cfg.Format)
    }

    // Create logger with metadata
    logger := slog.New(handler).With(
        "service", "myapp",
        "version", "1.0.0",
    )

    return logger, nil
}

// Usage
logger, err := SetupLogging(LogConfig{
    Level:     "INFO",
    Format:    "json",
    AddSource: true,
    Output:    "stdout",
})
if err != nil {
    panic(err)
}
slog.SetDefault(logger)
```

## Next Steps

- Learn about performance optimization in [Best Practices](best-practices.md)
- Explore handler implementation in [Handlers](handlers.md)
- See context integration in [Context Support](context-support.md)
