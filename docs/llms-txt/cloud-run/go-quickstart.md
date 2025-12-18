# Go Quickstart

> Quick guide to building and deploying Go applications to Cloud Run with code examples and best practices.

## Minimal Go Application

### Sample Code

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func main() {
    log.Print("starting server...")
    http.HandleFunc("/", handler)

    // Determine port for HTTP service
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
        log.Printf("defaulting to port %s", port)
    }

    // Start HTTP server
    log.Printf("listening on port %s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}

func handler(w http.ResponseWriter, r *http.Request) {
    name := os.Getenv("NAME")
    if name == "" {
        name = "World"
    }
    fmt.Fprintf(w, "Hello %s!\n", name)
}
```

### Key Configuration Requirements

**Port Configuration**: The application must listen on the port specified by the `PORT` environment variable, defaulting to 8080 if not set.

**Module Setup**: Initialize a Go module before deployment:

```bash
go mod init example.com/myapp
go mod tidy
```

## Deployment Methods

### Deploy from Source

Deploy directly from source using:

```bash
gcloud run deploy --source .
```

This command automatically builds a container image and deploys it without requiring a pre-written Dockerfile.

**Prerequisites**:
- Go module initialized (`go.mod` exists)
- Valid Go project structure

### Deploy with Dockerfile

Create a `Dockerfile`:

```dockerfile
FROM golang:1.21 AS builder

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -v -o server

FROM gcr.io/distroless/static-debian11
COPY --from=builder /app/server /server
CMD ["/server"]
```

Build and deploy:

```bash
# Build locally
docker build -t gcr.io/PROJECT_ID/myapp .

# Push to registry
docker push gcr.io/PROJECT_ID/myapp

# Deploy
gcloud run deploy myapp --image gcr.io/PROJECT_ID/myapp
```

## Project Structure

### Basic Structure

```
myapp/
├── go.mod
├── go.sum
├── main.go
└── Dockerfile (optional)
```

### Recommended Structure

```
myapp/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── handlers/
│   │   └── http.go
│   └── middleware/
│       └── logging.go
├── go.mod
├── go.sum
└── Dockerfile
```

## Complete Example with Best Practices

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

// LogEntry represents a structured log entry
type LogEntry struct {
    Severity string `json:"severity"`
    Message  string `json:"message"`
    Trace    string `json:"logging.googleapis.com/trace,omitempty"`
}

func (e LogEntry) Write() {
    jsonBytes, _ := json.Marshal(e)
    fmt.Println(string(jsonBytes))
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

    // Create HTTP server
    mux := http.NewServeMux()
    mux.HandleFunc("/", homeHandler)
    mux.HandleFunc("/health", healthHandler)

    srv := &http.Server{
        Addr:         ":" + port,
        Handler:      loggingMiddleware(mux),
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    // Start server in goroutine
    go func() {
        LogEntry{
            Severity: "INFO",
            Message:  "Server started successfully",
        }.Write()

        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            LogEntry{
                Severity: "CRITICAL",
                Message:  fmt.Sprintf("Server failed: %v", err),
            }.Write()
            log.Fatal(err)
        }
    }()

    // Wait for interrupt signal for graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    <-quit

    LogEntry{
        Severity: "INFO",
        Message:  "Shutting down server...",
    }.Write()

    // Cloud Run gives 10 seconds for graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        LogEntry{
            Severity: "ERROR",
            Message:  fmt.Sprintf("Server forced to shutdown: %v", err),
        }.Write()
    }

    LogEntry{
        Severity: "INFO",
        Message:  "Server exited",
    }.Write()
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    name := os.Getenv("NAME")
    if name == "" {
        name = "World"
    }
    fmt.Fprintf(w, "Hello %s!\n", name)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, "OK")
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Get trace context for log correlation
        trace := getTraceContext(r)

        LogEntry{
            Severity: "INFO",
            Message:  fmt.Sprintf("%s %s", r.Method, r.URL.Path),
            Trace:    trace,
        }.Write()

        next.ServeHTTP(w, r)

        duration := time.Since(start).Milliseconds()
        LogEntry{
            Severity: "INFO",
            Message:  fmt.Sprintf("Request completed in %dms", duration),
            Trace:    trace,
        }.Write()
    })
}

func getTraceContext(r *http.Request) string {
    traceHeader := r.Header.Get("X-Cloud-Trace-Context")
    projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")

    if traceHeader != "" && projectID != "" {
        // Extract trace ID (format: TRACE_ID/SPAN_ID;o=TRACE_TRUE)
        if len(traceHeader) > 0 {
            traceID := traceHeader
            if idx := len(traceHeader); idx > 0 {
                for i, c := range traceHeader {
                    if c == '/' || c == ';' {
                        traceID = traceHeader[:i]
                        break
                    }
                }
            }
            return fmt.Sprintf("projects/%s/traces/%s", projectID, traceID)
        }
    }
    return ""
}
```

## Dockerfile Best Practices

### Multi-Stage Build

```dockerfile
# Build stage
FROM golang:1.21 AS builder

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o server \
    ./cmd/server

# Runtime stage - distroless for security
FROM gcr.io/distroless/static-debian11

COPY --from=builder /app/server /server

# Run as non-root
USER nonroot:nonroot

ENTRYPOINT ["/server"]
```

### With Alpine

```dockerfile
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o server

# Runtime
FROM alpine:3.19

RUN apk --no-cache add ca-certificates && \
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

COPY --from=builder /app/server /server

USER appuser

ENTRYPOINT ["/server"]
```

## Environment Configuration

### Using Environment Variables

```go
type Config struct {
    Port        string
    Environment string
    LogLevel    string
    DatabaseURL string
}

func loadConfig() Config {
    return Config{
        Port:        getEnv("PORT", "8080"),
        Environment: getEnv("ENVIRONMENT", "production"),
        LogLevel:    getEnv("LOG_LEVEL", "info"),
        DatabaseURL: os.Getenv("DATABASE_URL"),
    }
}

func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}
```

### Deploy with Environment Variables

```bash
gcloud run deploy myapp \
  --source . \
  --set-env-vars NAME=Cloud \
  --set-env-vars LOG_LEVEL=debug
```

## Common Patterns

### Database Connection

```go
var db *sql.DB

func init() {
    var err error
    db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    // Verify connection
    if err = db.Ping(); err != nil {
        log.Fatalf("Failed to ping database: %v", err)
    }
}
```

### HTTP Client with Timeout

```go
var httpClient *http.Client

func init() {
    httpClient = &http.Client{
        Timeout: 10 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
        },
    }
}

func callExternalAPI(ctx context.Context, url string) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    return httpClient.Do(req)
}
```

### Secrets from Secret Manager

```go
import (
    secretmanager "cloud.google.com/go/secretmanager/apiv1"
    "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

func getSecret(ctx context.Context, name string) (string, error) {
    client, err := secretmanager.NewClient(ctx)
    if err != nil {
        return "", err
    }
    defer client.Close()

    req := &secretmanagerpb.AccessSecretVersionRequest{
        Name: name, // projects/PROJECT_ID/secrets/SECRET_NAME/versions/latest
    }

    result, err := client.AccessSecretVersion(ctx, req)
    if err != nil {
        return "", err
    }

    return string(result.Payload.Data), nil
}
```

## Testing Locally

### Run with Docker

```bash
# Build
docker build -t myapp .

# Run with environment variables
docker run -p 8080:8080 \
  -e PORT=8080 \
  -e NAME=Local \
  myapp

# Test
curl http://localhost:8080
```

### Run without Docker

```bash
# Set environment variables
export PORT=8080
export NAME=Local

# Run
go run main.go

# Or build and run
go build -o server
./server
```

## Deployment Commands

### Basic Deployment

```bash
# Deploy from source
gcloud run deploy myapp --source .

# Deploy with options
gcloud run deploy myapp \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --memory 512Mi \
  --cpu 1 \
  --max-instances 10
```

### Update Existing Service

```bash
# Update environment variables
gcloud run services update myapp \
  --update-env-vars NAME=Production

# Update resources
gcloud run services update myapp \
  --memory 1Gi \
  --cpu 2

# Update scaling
gcloud run services update myapp \
  --min-instances 1 \
  --max-instances 100 \
  --concurrency 80
```

## Troubleshooting

### Common Issues

**Container won't start**:
```bash
# Check logs
gcloud run services logs read myapp --limit=50

# Test locally
docker build -t myapp . && docker run -p 8080:8080 -e PORT=8080 myapp
```

**Port binding errors**:
```go
// Wrong - won't work on Cloud Run
http.ListenAndServe("127.0.0.1:8080", nil)

// Correct - listens on all interfaces
http.ListenAndServe(":8080", nil)
```

**Missing dependencies**:
```bash
# Update go.mod
go mod tidy

# Verify build
go build -o server
```
