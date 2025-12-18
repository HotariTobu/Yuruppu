# Troubleshooting and Debugging

> Common issues, diagnostic techniques, and solutions for Cloud Run deployments and runtime errors.

## Overview

Cloud Run troubleshooting covers three main error categories: deployment, serving, and connectivity/security issues. The guide emphasizes using Cloud Logging to look for application errors in stdout or stderr logs and checking Error Reporting for crashes.

## Common Deployment Errors

### Container Startup Failures

The most frequent problem is when containers fail to start.

**Solutions**:
1. Verify local container execution first
2. Ensure the container listens on the correct PORT environment variable
3. Confirm the container listens on `0.0.0.0` rather than `127.0.0.1`
4. Validate 64-bit Linux compilation

**Diagnostic commands**:
```bash
# Test locally
docker build -t myapp .
docker run -p 8080:8080 -e PORT=8080 myapp

# Check logs
gcloud run services logs read SERVICE --limit=50
```

### Service Account Problems

Deployment often fails due to missing permissions.

**Required IAM roles**:
- Cloud Run Invoker role for accessing the service
- Permissions for accessing service accounts
- Secret Manager Secret Accessor (if using secrets)

**Grant permissions**:
```bash
gcloud run services add-iam-policy-binding SERVICE \
  --member="user:EMAIL" \
  --role="roles/run.invoker"
```

### Source Deployment Failures

Python deployments may fail if web servers aren't specified in requirements.txt. Add `gunicorn`, `fastapi`, or `uvicorn` to resolve these issues.

For Go applications, ensure `go.mod` exists:
```bash
go mod init example.com/myapp
go mod tidy
```

## Serving Errors

### HTTP 404 Errors

Requests don't reach containers or ingress restrictions block traffic.

**Diagnostics**:
- Check request URLs match your service endpoint
- Verify ingress settings allow traffic source
- Examine Cloud Logging filters for rejected requests
- Confirm route handlers match request paths

### HTTP 429 (Too Many Requests)

Service is at capacity or rate limits exceeded.

**Solutions**:
- Increase maximum instances
- Implement request queuing in client
- Use exponential backoff with jitter

### HTTP 503 (Service Unavailable)

Scaling failures when instances reach capacity limits or startup times are too long.

**Solutions**:
- Increase maximum instances
- Reduce container startup time
- Implement startup CPU boost
- Configure minimum instances to handle baseline traffic
- Check for cold start issues

**Monitor scaling**:
```bash
gcloud run services describe SERVICE --format='value(spec.template.spec.containers[0].resources.limits)'
```

### HTTP 500 (Internal Server Error)

Application errors or crashes.

**Diagnostics**:
1. Check Cloud Logging for error messages
2. Review Error Reporting for stack traces
3. Verify all dependencies are included in container
4. Check for uncaught exceptions

## Memory Issues

### Memory Exhaustion

Files written to the local file system count towards available memory, including logs outside `/var/log/*`.

**Warning signs**:
- Out of memory errors in logs
- Containers being killed unexpectedly
- Slow performance before crashes

**Solutions**:
1. Delete temporary files promptly
2. Increase memory allocation
3. Reduce concurrency per instance
4. Monitor memory usage metrics

**Monitor memory**:
```bash
# View container memory metrics
gcloud monitoring time-series list \
  --filter='resource.type="cloud_run_revision" AND metric.type="run.googleapis.com/container/memory/utilizations"'
```

### Disk Space Issues

Cloud Run uses an in-memory filesystem. Writing large files can exhaust available memory.

**Best practices**:
- Stream large files instead of loading into memory
- Use Cloud Storage for file operations
- Clean up temporary files immediately after use

## Connectivity Issues

### Authentication Errors (HTTP 401)

**Common causes**:
- Missing or invalid JWT tokens
- Incorrect audience claims
- Wrong authorization header format

**Verification**:
```bash
# Get ID token for testing
gcloud auth print-identity-token

# Test authenticated endpoint
curl -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
  https://SERVICE-URL
```

### Connection Problems

"Connection reset by peer" and timeout errors are often infrastructure issues.

**Solutions**:
- Implement connection validation
- Add retry logic with exponential backoff
- Disable long-lived connection reuse
- Increase request timeout settings

### HTTP Proxy Configuration

When routing egress traffic through proxies, add exceptions for metadata servers and Google APIs.

**Required exceptions**:
- `127.0.0.1`
- `169.254.*`
- `*.googleapis.com`

## Performance Issues

### Slow Cold Starts

**Optimization strategies**:
1. Minimize container image size
2. Enable startup CPU boost
3. Configure minimum instances
4. Use lighter base images (Alpine, distroless)
5. Lazy-load dependencies

### High Latency

**Diagnostics**:
1. Check Cloud Monitoring for latency metrics
2. Review request logs for slow operations
3. Profile application code
4. Check database connection times

**Solutions**:
- Implement connection pooling
- Cache frequently accessed data
- Use global variables for reusable resources
- Optimize database queries

## Diagnostic Tools

### Cloud Logging
Primary tool for examining application errors and system behavior.

```bash
# Filter for errors
gcloud logging read "resource.type=cloud_run_revision AND severity>=ERROR" --limit=50

# Search for specific text
gcloud logging read "resource.type=cloud_run_revision AND textPayload:timeout" --limit=20
```

### Cloud Monitoring
Tracks container instance counts and CPU utilization metrics.

Key metrics:
- `run.googleapis.com/container/cpu/utilizations`
- `run.googleapis.com/container/memory/utilizations`
- `run.googleapis.com/request_count`
- `run.googleapis.com/request_latencies`

### Error Reporting
Captures crash information automatically at:
https://console.cloud.google.com/errors

### Cloud Trace
Distributed tracing for request flow analysis.

### Local Testing

Test containers locally before deploying:

```bash
# Build and test
docker build -t myapp .
docker run -p 8080:8080 -e PORT=8080 myapp

# Test with curl
curl http://localhost:8080
```

## Best Practices

### 1. Implement Health Checks

Use liveness probes to terminate unhealthy instances:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
```

### 2. Configure Appropriate Timeouts

Set timeouts based on actual application needs rather than defaults:

```bash
gcloud run services update SERVICE --timeout=300
```

### 3. Handle Signals Gracefully

Respond to `SIGTERM` for graceful shutdown:

```go
func main() {
    srv := &http.Server{Addr: ":8080"}

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    <-quit

    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Shutdown error: %v", err)
    }
}
```

### 4. Use Structured Logging

Makes troubleshooting easier with queryable log fields.

### 5. Monitor Key Metrics

Set up alerts for:
- Error rates
- High latency
- Memory usage
- Instance scaling events

### 6. Test Error Scenarios

Deliberately trigger failures in staging:
- Simulate OOM conditions
- Test with invalid configurations
- Verify graceful degradation
- Check retry behavior

## Go-Specific Debugging

### Common Go Issues

**Port binding errors**:
```go
// Wrong - listens on localhost only
http.ListenAndServe("127.0.0.1:8080", nil)

// Correct - listens on all interfaces
http.ListenAndServe(":8080", nil)
```

**Missing dependencies**:
```bash
# Ensure all dependencies are in go.mod
go mod tidy

# Verify vendor directory if using vendoring
go mod vendor
```

**Panic recovery**:
```go
func handler(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if err := recover(); err != nil {
            log.Printf("Panic recovered: %v", err)
            http.Error(w, "Internal server error", http.StatusInternalServerError)
        }
    }()
    // Handler logic
}
```

### Enable pprof for Profiling

```go
import _ "net/http/pprof"

func main() {
    // pprof automatically registers handlers
    http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
```

Access profiles at:
- `https://SERVICE-URL/debug/pprof/`
- `https://SERVICE-URL/debug/pprof/heap`
- `https://SERVICE-URL/debug/pprof/goroutine`

**Security note**: Only enable pprof in development or protect with authentication.
