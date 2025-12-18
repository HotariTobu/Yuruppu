# Logging and Monitoring

> Comprehensive guide to writing, viewing, and structuring logs in Cloud Run for effective debugging and monitoring.

## Types of Logs

Cloud Run automatically sends three log types to Cloud Logging:

1. **Request logs** (services only): Automatically created logs of incoming requests
2. **Container logs** (services, jobs, worker pools): Output from your code written to supported locations
3. **System logs** (services, jobs, worker pools): Platform-generated information written to `var/log/system`

## Writing Container Logs

### Supported Output Locations

Most developers are expected to write logs using standard output and standard error.

Your application can write logs to:

- Standard output (`stdout`) or standard error (`stderr`) streams (recommended)
- Files under the `/var/log` directory
- syslog (`/dev/log`)
- Cloud Logging client libraries

### Text vs. Structured Logging

**Text Logging**: Simple string output
```go
log.Println("Server started successfully")
```

**Structured Logging**: Single-line serialized JSON
```go
import "encoding/json"

type LogEntry struct {
    Severity string `json:"severity"`
    Message  string `json:"message"`
    UserID   string `json:"user_id,omitempty"`
}

entry := LogEntry{
    Severity: "INFO",
    Message:  "User login successful",
    UserID:   "user123",
}
jsonBytes, _ := json.Marshal(entry)
fmt.Println(string(jsonBytes))
```

Structured JSON is parsed by Cloud Logging and placed into `jsonPayload`, while text appears in `textPayload`.

## Log Correlation for Services

Container logs can be nested under request logs by including a `logging.googleapis.com/trace` field extracted from the `X-Cloud-Trace-Context` request header. This creates a parent-child relationship visible in the Logs Explorer.

### Go Example with Trace Correlation

```go
func handler(w http.ResponseWriter, r *http.Request) {
    traceHeader := r.Header.Get("X-Cloud-Trace-Context")
    projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")

    var trace string
    if traceHeader != "" {
        // Extract trace ID from header
        traceParts := strings.Split(traceHeader, "/")
        if len(traceParts) > 0 {
            trace = fmt.Sprintf("projects/%s/traces/%s", projectID, traceParts[0])
        }
    }

    logEntry := map[string]interface{}{
        "severity": "INFO",
        "message":  "Processing request",
        "logging.googleapis.com/trace": trace,
    }

    jsonBytes, _ := json.Marshal(logEntry)
    fmt.Println(string(jsonBytes))
}
```

## Special JSON Fields

When providing structured logs as JSON, certain fields are automatically extracted from `jsonPayload` to their corresponding LogEntry fields:

- `severity`: Log level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
- `message`: Main log message
- `logging.googleapis.com/trace`: Trace ID for correlation
- `logging.googleapis.com/spanId`: Span ID for distributed tracing
- `logging.googleapis.com/labels`: Custom labels for filtering

## Severity Levels

Use standard severity levels in structured logs:

- `DEFAULT`: The log entry has no assigned severity level
- `DEBUG`: Debug or trace information
- `INFO`: Routine information, such as ongoing status
- `NOTICE`: Normal but significant events
- `WARNING`: Warning events might cause problems
- `ERROR`: Error events are likely to cause problems
- `CRITICAL`: Critical events cause severe problems
- `ALERT`: A person must take action immediately
- `EMERGENCY`: System is unusable

## Viewing Logs

### Cloud Run Console
The simplest method for viewing logs:
1. Navigate to Cloud Run in Google Cloud Console
2. Select your service
3. Click on "Logs" tab

### gcloud CLI

```bash
# View recent logs
gcloud run services logs read SERVICE

# Stream logs in real-time
gcloud run services logs tail SERVICE

# Filter by severity
gcloud run services logs read SERVICE --log-filter="severity>=ERROR"
```

### Cloud Logging Logs Explorer

Most detailed filtering options available at:
https://console.cloud.google.com/logs

**Example queries**:
```
resource.type="cloud_run_revision"
resource.labels.service_name="my-service"
severity>=ERROR
```

### Programmatic Access
- Logging API
- Client libraries for Go, Python, Node.js, etc.

## Best Practices

### 1. Use Structured Logging

Structured JSON logs enable better querying and analysis:

```go
type LogEntry struct {
    Severity  string                 `json:"severity"`
    Message   string                 `json:"message"`
    Component string                 `json:"component,omitempty"`
    UserID    string                 `json:"user_id,omitempty"`
    Duration  float64                `json:"duration_ms,omitempty"`
    Labels    map[string]string      `json:"logging.googleapis.com/labels,omitempty"`
}
```

### 2. Include Trace Context

Always correlate logs with requests using trace context for easier debugging.

### 3. Use Appropriate Severity Levels

- `INFO`: Normal operations
- `WARNING`: Unexpected but recoverable situations
- `ERROR`: Errors that affect request processing
- `CRITICAL`: System-wide failures

### 4. Add Relevant Context

Include useful debugging information:
- Request IDs
- User IDs
- Operation names
- Duration metrics
- Error details

### 5. Avoid Logging Sensitive Data

Never log:
- Passwords or API keys
- Personal identifiable information (PII)
- Credit card numbers
- Authentication tokens

### 6. Control Log Volume

Excessive logging can:
- Increase costs
- Reduce performance
- Make debugging harder

Consider sampling high-frequency logs or using different verbosity levels for development vs. production.

## Go Logging Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
)

type LogEntry struct {
    Severity string                 `json:"severity"`
    Message  string                 `json:"message"`
    Trace    string                 `json:"logging.googleapis.com/trace,omitempty"`
    Labels   map[string]string      `json:"logging.googleapis.com/labels,omitempty"`
}

func (e LogEntry) Write() {
    jsonBytes, _ := json.Marshal(e)
    fmt.Println(string(jsonBytes))
}

func getTrace(r *http.Request) string {
    traceHeader := r.Header.Get("X-Cloud-Trace-Context")
    projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")

    if traceHeader != "" && projectID != "" {
        traceParts := strings.Split(traceHeader, "/")
        if len(traceParts) > 0 {
            return fmt.Sprintf("projects/%s/traces/%s", projectID, traceParts[0])
        }
    }
    return ""
}

func handler(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    trace := getTrace(r)

    LogEntry{
        Severity: "INFO",
        Message:  "Request received",
        Trace:    trace,
        Labels: map[string]string{
            "method": r.Method,
            "path":   r.URL.Path,
        },
    }.Write()

    // Process request...

    duration := time.Since(start).Milliseconds()
    LogEntry{
        Severity: "INFO",
        Message:  fmt.Sprintf("Request completed in %dms", duration),
        Trace:    trace,
    }.Write()

    fmt.Fprintf(w, "Hello, World!")
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    LogEntry{
        Severity: "INFO",
        Message:  fmt.Sprintf("Starting server on port %s", port),
    }.Write()

    http.HandleFunc("/", handler)

    if err := http.ListenAndServe(":"+port, nil); err != nil {
        LogEntry{
            Severity: "CRITICAL",
            Message:  fmt.Sprintf("Server failed to start: %v", err),
        }.Write()
        log.Fatal(err)
    }
}
```
