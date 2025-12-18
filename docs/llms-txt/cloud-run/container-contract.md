# Container Runtime Contract

> Defines the requirements containers must meet to run on Cloud Run, including port configuration, environment variables, startup behavior, and security constraints.

## Port Configuration

The ingress container must listen on `0.0.0.0` on the designated port. Notably, listening on `127.0.0.1` is unsupported. Cloud Run injects a `PORT` environment variable into the ingress container specifying where to accept requests.

**Default port**: 8080 (configurable)

Example Go implementation:

```go
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}
http.ListenAndServe(":"+port, nil)
```

## Built-in Environment Variables

### For Services

- `PORT`: The HTTP server port (default 8080)
- `K_SERVICE`: Service name
- `K_REVISION`: Revision identifier
- `K_CONFIGURATION`: Configuration name

### For Jobs

- `CLOUD_RUN_JOB`: Job name
- `CLOUD_RUN_EXECUTION`: Execution identifier
- `CLOUD_RUN_TASK_INDEX`: Task sequence number
- `CLOUD_RUN_TASK_ATTEMPT`: Retry count
- `CLOUD_RUN_TASK_COUNT`: Total task count

## Startup Requirements

Services must listen within 4 minutes of starting. The platform sends a `SIGTERM` signal before shutdown, allowing a 10-second grace period before `SIGKILL` termination.

Startup probes can verify container readiness and prevent premature liveness checks.

## Job-Specific Requirements

Containers must exit with code 0 on success or non-zero on failure. They should not listen on ports or run web servers since jobs execute batch tasks, not serve requests.

## Security Constraints

Containers run with restricted privileges:
- No root capabilities
- Disabled privilege escalation
- Limited device access
- Cannot execute `setuid` binaries
- Containers operate under Linux namespaces for isolation

## Multi-Container Support

Services can include up to 10 containers total:
- One ingress container handling HTTPS requests
- Up to nine sidecars communicating via localhost ports for monitoring, proxying, or authentication
