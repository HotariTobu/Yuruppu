# Best Practices for Cloud Run

> Performance optimization, cost management, security recommendations, and operational best practices for production Cloud Run services.

## Performance Optimization

### Container Startup

Start containers quickly since instances are scaled as needed, making startup time critical for request latency.

**Optimization techniques**:

1. **Enable startup CPU boost**: Temporarily increases CPU allocation during initialization
   ```bash
   gcloud run services update SERVICE --cpu-boost
   ```

2. **Configure minimum instances**: Reduces cold-start delays
   ```bash
   gcloud run services update SERVICE --min-instances=1
   ```

3. **Minimize dependencies**: Reduce the number and size of dependencies
   - Use vendoring for Go applications
   - Remove unused dependencies
   - Use smaller base images

4. **Multi-stage Docker builds**: Reduce final image size
   ```dockerfile
   # Build stage
   FROM golang:1.21 AS builder
   WORKDIR /app
   COPY . .
   RUN go build -o server

   # Runtime stage
   FROM gcr.io/distroless/base-debian11
   COPY --from=builder /app/server /server
   ENTRYPOINT ["/server"]
   ```

### Global Variables and Caching

Use global scope variables to preserve state between requests on reused instances.

```go
var (
    db     *sql.DB
    client *http.Client
    cache  map[string]interface{}
)

func init() {
    // Initialize expensive resources once
    db = initDatabase()
    client = &http.Client{Timeout: 10 * time.Second}
    cache = make(map[string]interface{})
}

func handler(w http.ResponseWriter, r *http.Request) {
    // Reuse global resources across requests
    rows, err := db.Query("SELECT * FROM users")
    // ...
}
```

**Benefits**:
- Cache expensive-to-recreate objects in memory
- Reuse database connections
- Preserve computed data

**Lazy initialization** for infrequently used objects:

```go
var (
    expensiveResource     *Resource
    expensiveResourceOnce sync.Once
)

func getExpensiveResource() *Resource {
    expensiveResourceOnce.Do(func() {
        expensiveResource = initExpensiveResource()
    })
    return expensiveResource
}
```

### Concurrency Management

Default maximum concurrency of 80 works well for many services, but should be tuned to your workload.

**Considerations**:
- Match memory allocation to concurrency levels
- Each request requires additional memory
- Use load testing to identify maximum stable concurrency

```bash
# Set concurrency limit
gcloud run services update SERVICE --concurrency=50
```

**Load testing**:
```bash
# Using Apache Bench
ab -n 1000 -c 50 https://SERVICE-URL/

# Using hey
hey -n 1000 -c 50 https://SERVICE-URL/
```

## Cost Management

### Billing Models

**Instance-based billing**: Background activities consume billable instance time.

**Request-based billing**: When the Cloud Run service finishes handling a request, the instance's access to CPU will be disabled or severely limited.

Choose based on your workload:
- **Request-based**: Stateless services with no background processing
- **Instance-based**: Services with background tasks, WebSockets, streaming

### Optimize for Cost

1. **Lower concurrency can reduce costs** if latency improvements offset increased instance count
2. **Monitor the `container/billable_instance_time` metric** when testing different configurations
3. **Set appropriate maximum instances** to prevent runaway costs
4. **Use minimum instances sparingly** as they incur continuous charges

```bash
# Configure billing and scaling
gcloud run services update SERVICE \
  --cpu-throttling \
  --min-instances=0 \
  --max-instances=10 \
  --concurrency=80
```

### Resource Right-Sizing

```bash
# Monitor actual usage
gcloud monitoring time-series list \
  --filter='resource.type="cloud_run_revision" AND metric.type="run.googleapis.com/container/memory/utilizations"'

# Adjust based on usage
gcloud run services update SERVICE \
  --memory=512Mi \
  --cpu=1
```

## Security Recommendations

### Container Security

**Use actively maintained, secure base images**:

```dockerfile
# Good - official minimal images
FROM golang:1.21-alpine AS builder
FROM gcr.io/distroless/base-debian11

# Better - Google's distroless images
FROM gcr.io/distroless/static-debian11

# Avoid - large, outdated images
FROM ubuntu:latest
```

**Include only necessary components**:
```dockerfile
# Multi-stage builds to minimize attack surface
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o server

FROM scratch
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]
```

**Run as non-root user**:

```dockerfile
FROM gcr.io/distroless/base-debian11

# Add non-root user
USER nonroot:nonroot

COPY server /server
ENTRYPOINT ["/server"]
```

Or with Alpine:

```dockerfile
FROM alpine:3.19

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

USER appuser

COPY server /server
ENTRYPOINT ["/server"]
```

### Image Optimization

**Build minimal container images**:

1. Use lean base images (Alpine, distroless, scratch)
2. Employ multi-stage builds
3. Remove build tools from runtime image
4. Copy only necessary files

```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

FROM scratch
COPY --from=builder /app/server /server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/server"]
```

**Enable vulnerability scanning**:

```bash
# Enable in Artifact Registry
gcloud artifacts repositories update REPOSITORY \
  --location=LOCATION \
  --enable-security-scanning
```

### Secrets Management

**Never hardcode secrets**:

```go
// Bad
const apiKey = "secret-key-12345"

// Good
apiKey := os.Getenv("API_KEY")

// Better
apiKey := getSecretFromSecretManager("api-key")
```

**Use Secret Manager**:

```bash
gcloud run services update SERVICE \
  --update-secrets=API_KEY=my-secret:latest
```

### Network Security

**Restrict ingress**:

```bash
# Internal only
gcloud run services update SERVICE --ingress=internal

# Internal and Cloud Load Balancing
gcloud run services update SERVICE --ingress=internal-and-cloud-load-balancing
```

**Use VPC connectors** for private resources:

```bash
gcloud run services update SERVICE \
  --vpc-connector=CONNECTOR_NAME \
  --vpc-egress=private-ranges-only
```

## Operational Best Practices

### Error Handling

Handle all exceptions; crashes trigger slow container restarts while traffic queues.

```go
func handler(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if err := recover(); err != nil {
            log.Printf("Panic recovered: %v", err)
            http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        }
    }()

    // Handler logic
    if err := processRequest(r); err != nil {
        log.Printf("Error processing request: %v", err)
        http.Error(w, "Error processing request", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

### Resource Management

**Delete temporary files promptly**: Disk storage uses in-memory filesystem consuming available memory.

```go
func processFile(data []byte) error {
    tmpFile, err := os.CreateTemp("/tmp", "upload-*")
    if err != nil {
        return err
    }
    defer os.Remove(tmpFile.Name()) // Clean up immediately

    if _, err := tmpFile.Write(data); err != nil {
        return err
    }

    // Process file
    return processFileContents(tmpFile.Name())
}
```

**Stream large files** instead of loading into memory:

```go
func handleUpload(w http.ResponseWriter, r *http.Request) {
    // Bad - loads entire file into memory
    body, _ := io.ReadAll(r.Body)
    processData(body)

    // Good - streams data
    reader := bufio.NewReader(r.Body)
    for {
        line, err := reader.ReadBytes('\n')
        if err == io.EOF {
            break
        }
        processLine(line)
    }
}
```

### Background Activities

Under request-based billing, avoid background threads that run outside request handlers. Running background threads with request-based billing enabled can result in unexpected behavior.

```go
// Bad with request-based billing
func main() {
    // Background worker will be throttled
    go backgroundWorker()

    http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}

// Good - use Cloud Scheduler or Cloud Tasks instead
func handler(w http.ResponseWriter, r *http.Request) {
    // Trigger background work via Cloud Tasks
    createTask("background-work", payload)
    w.WriteHeader(http.StatusOK)
}
```

### Graceful Shutdown

Handle SIGTERM for clean shutdown:

```go
func main() {
    srv := &http.Server{
        Addr:    ":" + os.Getenv("PORT"),
        Handler: router,
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed: %v", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    <-quit

    log.Println("Shutting down server...")

    // 10 second grace period (Cloud Run allows 10 seconds)
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }

    log.Println("Server exited")
}
```

### Monitoring and Alerting

**Set up alerts for key metrics**:

1. Error rate
2. Latency (p50, p95, p99)
3. Memory utilization
4. Instance count
5. Cold start frequency

```bash
# Create alert policy
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="High Error Rate" \
  --condition-display-name="Error rate > 5%" \
  --condition-expression='
    resource.type="cloud_run_revision" AND
    metric.type="run.googleapis.com/request_count" AND
    metric.label.response_code_class="5xx"
  '
```

### Connection Pooling

Reuse connections for external services:

```go
var (
    httpClient *http.Client
    dbPool     *sql.DB
)

func init() {
    // Configure HTTP client with connection pooling
    httpClient = &http.Client{
        Timeout: 10 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
        },
    }

    // Configure database connection pool
    dbPool, _ = sql.Open("postgres", os.Getenv("DATABASE_URL"))
    dbPool.SetMaxOpenConns(25)
    dbPool.SetMaxIdleConns(5)
    dbPool.SetConnMaxLifetime(5 * time.Minute)
}
```

### Structured Logging

Use structured logging for better observability:

```go
type LogEntry struct {
    Severity  string                 `json:"severity"`
    Message   string                 `json:"message"`
    Component string                 `json:"component,omitempty"`
    Duration  int64                  `json:"duration_ms,omitempty"`
    Labels    map[string]string      `json:"logging.googleapis.com/labels,omitempty"`
}

func logInfo(msg string, labels map[string]string) {
    entry := LogEntry{
        Severity: "INFO",
        Message:  msg,
        Labels:   labels,
    }
    jsonBytes, _ := json.Marshal(entry)
    fmt.Println(string(jsonBytes))
}
```

## Go-Specific Best Practices

### Optimize Build

```dockerfile
# Use build cache effectively
FROM golang:1.21 AS builder
WORKDIR /app

# Copy go.mod first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Then copy source
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o server .

FROM gcr.io/distroless/static-debian11
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]
```

### Use Context for Timeouts

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    // Pass context to database queries
    row := db.QueryRowContext(ctx, "SELECT * FROM users WHERE id = $1", id)

    // Pass context to HTTP requests
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := httpClient.Do(req)
    if err != nil {
        http.Error(w, "Request timeout", http.StatusGatewayTimeout)
        return
    }
    defer resp.Body.Close()
}
```

### Avoid Global State Mutations

```go
// Bad - concurrent map access causes panics
var cache = make(map[string]interface{})

func handler(w http.ResponseWriter, r *http.Request) {
    cache[r.URL.Path] = time.Now() // Race condition!
}

// Good - use sync.Map or mutex
var cache sync.Map

func handler(w http.ResponseWriter, r *http.Request) {
    cache.Store(r.URL.Path, time.Now())
}

// Or with explicit locking
var (
    cache   = make(map[string]interface{})
    cacheMu sync.RWMutex
)

func handler(w http.ResponseWriter, r *http.Request) {
    cacheMu.Lock()
    cache[r.URL.Path] = time.Now()
    cacheMu.Unlock()
}
```

### Profile Performance

```go
import (
    _ "net/http/pprof"
)

func main() {
    // pprof handlers automatically registered
    // Access at /debug/pprof/

    port := os.Getenv("PORT")
    http.ListenAndServe(":"+port, nil)
}
```

## Testing Before Deployment

```bash
# Test locally
docker build -t myapp .
docker run -p 8080:8080 -e PORT=8080 myapp

# Load test
hey -n 10000 -c 100 http://localhost:8080

# Test with production-like config
docker run -p 8080:8080 \
  -e PORT=8080 \
  -e DATABASE_URL=... \
  -e LOG_LEVEL=info \
  --memory=512m \
  --cpus=1 \
  myapp
```
