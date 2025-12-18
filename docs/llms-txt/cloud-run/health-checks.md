# Health Checks

> Configure startup, liveness, and readiness probes to ensure container instances run correctly and serve traffic reliably.

## Overview

Cloud Run provides three types of health check probes to ensure container instances run correctly.

## Probe Types

### Startup Probes

Determine whether the container has started and is ready to accept traffic. They prevent liveness checks from interfering with slow-starting containers by disabling other checks until startup succeeds.

**Use cases**:
- Applications with long initialization
- Loading large datasets at startup
- Establishing database connections
- Warming caches

**Key point**: Liveness and readiness probes are disabled until startup probe succeeds.

### Liveness Probes

Determine whether to restart a container. These check if running instances can recover, restarting those experiencing unrecoverable failures like deadlocks.

When a service experiences repeated probe failures, Cloud Run limits instance restarts to prevent uncontrolled crash loops.

**Use cases**:
- Detecting deadlocks
- Identifying hung processes
- Recovering from memory leaks
- Restarting after unrecoverable errors

**Behavior**: Failed instances are terminated and replaced with new ones.

### Readiness Probes (Preview)

Determine when an instance in your Cloud Run service should serve traffic. Unlike liveness probes, readiness checks stop sending new traffic to failing instances without terminating them, resuming when the instance recovers.

**Use cases**:
- Temporary overload conditions
- Warming up after idle periods
- Dependency failures (database, cache)
- Graceful degradation scenarios

**Behavior**: Failed instances remain running but don't receive new requests.

## Implementation Types

All probes support three implementation types:

### HTTP Probes

Require HTTP/1 endpoints; expect 2XX or 3XX responses.

**Example**:
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
```

**Go implementation**:
```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Check critical dependencies
    if !database.Ping() {
        http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

func main() {
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/", mainHandler)
    http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
```

### TCP Probes

Verify port connectivity without HTTP overhead.

**Example**:
```yaml
livenessProbe:
  tcpSocket:
    port: 8080
```

**Use case**: Non-HTTP services or when minimal overhead is required.

### gRPC Probes

Require implementation of the gRPC Health Checking protocol.

**Example**:
```yaml
livenessProbe:
  grpc:
    port: 8080
```

## Configuration Parameters

### Common Parameters

- **initialDelaySeconds** (0-240): Time to wait before first probe
- **periodSeconds**: Frequency of probe checks
- **timeoutSeconds**: Time to wait for probe response
- **failureThreshold**: Number of consecutive failures before action

### Readiness-Specific

- **successThreshold**: Consecutive successes needed to mark instance ready

## Configuration Methods

### Console
Navigate to service settings and configure probes in the Health Check section.

### gcloud CLI

```bash
gcloud run services update SERVICE \
  --cpu-startup-boost \
  --startup-probe-type=http-get \
  --startup-probe-path=/startup \
  --startup-probe-initial-delay=10 \
  --startup-probe-period=5 \
  --startup-probe-failure-threshold=3
```

### YAML

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: myservice
spec:
  template:
    spec:
      containers:
      - image: gcr.io/project/image
        ports:
        - containerPort: 8080
        startupProbe:
          httpGet:
            path: /startup
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
          failureThreshold: 3
          timeoutSeconds: 3
        livenessProbe:
          httpGet:
            path: /liveness
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 10
          failureThreshold: 3
          timeoutSeconds: 3
        readinessProbe:
          httpGet:
            path: /readiness
            port: 8080
          periodSeconds: 5
          failureThreshold: 2
          successThreshold: 1
          timeoutSeconds: 3
```

### Terraform

```hcl
resource "google_cloud_run_v2_service" "default" {
  name     = "service-name"
  location = "us-central1"

  template {
    containers {
      image = "gcr.io/project/image"

      startup_probe {
        http_get {
          path = "/startup"
          port = 8080
        }
        initial_delay_seconds = 10
        period_seconds        = 5
        failure_threshold     = 3
      }

      liveness_probe {
        http_get {
          path = "/liveness"
          port = 8080
        }
        initial_delay_seconds = 15
        period_seconds        = 10
        failure_threshold     = 3
      }
    }
  }
}
```

## Best Practices

### 1. Design Lightweight Checks

Health checks should be fast and consume minimal resources.

**Good**:
```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}
```

**Acceptable**:
```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Quick connectivity check with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        http.Error(w, "unhealthy", http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
}
```

**Avoid**: Complex operations, full system diagnostics, or expensive computations.

### 2. Use Different Endpoints

Separate endpoints allow different logic for each probe type:

- `/startup`: Check critical initialization complete
- `/liveness`: Detect fatal errors requiring restart
- `/readiness`: Check if instance can handle requests

### 3. Configure Appropriate Timeouts

- **Startup probe**: Generous failureThreshold for slow initialization
- **Liveness probe**: Conservative failureThreshold to avoid unnecessary restarts
- **Readiness probe**: Sensitive thresholds for quick traffic shifting

### 4. Consider Startup Time

Set `initialDelaySeconds` based on actual container startup time:

```go
// Log startup progress for tuning
func main() {
    start := time.Now()
    log.Println("Starting initialization...")

    // Initialize dependencies
    initDatabase()
    log.Printf("Database ready after %v", time.Since(start))

    loadCache()
    log.Printf("Cache loaded after %v", time.Since(start))

    log.Printf("Startup complete after %v", time.Since(start))

    // Start server
    http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
```

### 5. Handle Probe Failures Gracefully

Don't crash on health check failures; return appropriate status codes:

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    healthy := true
    var errors []string

    // Check dependencies
    if err := checkDatabase(); err != nil {
        healthy = false
        errors = append(errors, fmt.Sprintf("database: %v", err))
    }

    if err := checkCache(); err != nil {
        // Cache failure might not be critical
        log.Printf("Warning: cache unhealthy: %v", err)
    }

    if !healthy {
        log.Printf("Health check failed: %v", errors)
        http.Error(w, strings.Join(errors, "; "), http.StatusServiceUnavailable)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

### 6. Avoid Cascading Failures

Health checks shouldn't call other services' health endpoints:

```go
// Bad - can cause cascading failures
func healthHandler(w http.ResponseWriter, r *http.Request) {
    resp, err := http.Get("https://dependency-service/health")
    if err != nil || resp.StatusCode != 200 {
        http.Error(w, "unhealthy", http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
}

// Good - check your own ability to reach the service
func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Check if connection pool is available
    if serviceClient == nil {
        http.Error(w, "service client not initialized", http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
}
```

## Complete Go Example

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "net/http"
    "os"
    "sync/atomic"
    "time"

    _ "github.com/lib/pq"
)

var (
    db              *sql.DB
    startupComplete atomic.Bool
    isHealthy       atomic.Bool
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    // Initialize in background
    go initialize()

    // Register health check endpoints
    http.HandleFunc("/startup", startupHandler)
    http.HandleFunc("/liveness", livenessHandler)
    http.HandleFunc("/readiness", readinessHandler)
    http.HandleFunc("/", mainHandler)

    log.Printf("Starting server on port %s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}

func initialize() {
    log.Println("Starting initialization...")

    // Connect to database
    var err error
    db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }

    // Wait for database to be ready
    for i := 0; i < 30; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        err = db.PingContext(ctx)
        cancel()

        if err == nil {
            break
        }
        log.Printf("Database not ready, retrying... (%d/30)", i+1)
        time.Sleep(1 * time.Second)
    }

    if err != nil {
        log.Fatalf("Database never became ready: %v", err)
    }

    log.Println("Database connected")

    // Load initial data, warm caches, etc.
    time.Sleep(2 * time.Second)

    log.Println("Initialization complete")
    startupComplete.Store(true)
    isHealthy.Store(true)
}

func startupHandler(w http.ResponseWriter, r *http.Request) {
    if !startupComplete.Load() {
        http.Error(w, "Startup in progress", http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, "Started")
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
    // Check for fatal errors that require restart
    if !isHealthy.Load() {
        http.Error(w, "Unhealthy", http.StatusServiceUnavailable)
        return
    }

    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, "Alive")
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
    // Check if we can handle requests right now
    if !startupComplete.Load() {
        http.Error(w, "Not ready", http.StatusServiceUnavailable)
        return
    }

    // Quick database connectivity check
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        log.Printf("Database ping failed: %v", err)
        http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
        return
    }

    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, "Ready")
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World!")
}
```

## Important Notes

- Any configuration change creates a new revision
- Probes are not executed on instances that are scaling to zero
- Failed probes appear in Cloud Logging for debugging
- Excessive probe failures trigger automatic instance restart throttling
