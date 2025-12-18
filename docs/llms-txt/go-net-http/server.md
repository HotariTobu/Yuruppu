# HTTP Server

The `http.Server` type provides complete control over HTTP server configuration and lifecycle.

## Server Type Definition

```go
type Server struct {
    Addr           string        // TCP address to listen on (e.g., ":8080")
    Handler        Handler       // Handler to invoke (nil = DefaultServeMux)
    TLSConfig      *tls.Config   // Optional TLS configuration
    ReadTimeout    time.Duration // Max duration for reading request
    WriteTimeout   time.Duration // Max duration for writing response
    IdleTimeout    time.Duration // Max keep-alive idle time
    MaxHeaderBytes int           // Max request header bytes
    ErrorLog       *log.Logger   // Optional error logger
    BaseContext    func(net.Listener) context.Context
    ConnContext    func(context.Context, net.Conn) context.Context
    ConnState      func(net.Conn, ConnState)
}
```

## Basic Server Creation

```go
server := &http.Server{
    Addr:    ":8080",
    Handler: myHandler,
}
```

## Server Methods

### ListenAndServe

Start HTTP server on the configured address:

```go
func (s *Server) ListenAndServe() error
```

Example:
```go
server := &http.Server{Addr: ":8080"}
log.Fatal(server.ListenAndServe())
```

### ListenAndServeTLS

Start HTTPS server with TLS certificates:

```go
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error
```

Example:
```go
server := &http.Server{Addr: ":8443"}
log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
```

### Serve

Serve connections from an existing listener:

```go
func (s *Server) Serve(l net.Listener) error
```

Example:
```go
listener, err := net.Listen("tcp", ":8080")
if err != nil {
    log.Fatal(err)
}
server := &http.Server{Handler: myHandler}
log.Fatal(server.Serve(listener))
```

### Shutdown

Gracefully shut down the server without interrupting active connections:

```go
func (s *Server) Shutdown(ctx context.Context) error
```

Example:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := server.Shutdown(ctx); err != nil {
    log.Printf("Shutdown error: %v", err)
}
```

### Close

Immediately close all active listeners and connections:

```go
func (s *Server) Close() error
```

Example:
```go
if err := server.Close(); err != nil {
    log.Printf("Close error: %v", err)
}
```

## Timeout Configuration

Configure timeouts to prevent resource exhaustion:

```go
server := &http.Server{
    Addr:         ":8080",
    Handler:      myHandler,
    ReadTimeout:  5 * time.Second,   // Time to read request
    WriteTimeout: 10 * time.Second,  // Time to write response
    IdleTimeout:  120 * time.Second, // Keep-alive timeout
}
```

### ReadTimeout

Maximum duration for reading the entire request, including the body. Prevents slow clients from holding connections.

### WriteTimeout

Maximum duration before timing out writes of the response. Covers the entire response time (from end of request read to end of response write).

### IdleTimeout

Maximum duration to wait for the next request when keep-alives are enabled. If zero, ReadTimeout is used. If both are zero, no timeout.

## Header Size Limits

```go
server := &http.Server{
    Addr:           ":8080",
    MaxHeaderBytes: 1 << 20, // 1 MB
}
```

Limits the maximum number of bytes the server will read parsing request headers. Does not affect request body size.

## Custom Error Logger

```go
import "log"

logger := log.New(os.Stderr, "HTTP Server: ", log.LstdFlags)

server := &http.Server{
    Addr:     ":8080",
    ErrorLog: logger,
}
```

Logs errors from accepting connections, unexpected handler behavior, and FileSystem errors.

## Connection State Tracking

Monitor connection lifecycle:

```go
server := &http.Server{
    Addr: ":8080",
    ConnState: func(conn net.Conn, state http.ConnState) {
        log.Printf("Connection %s: %v", conn.RemoteAddr(), state)
    },
}
```

Connection states:
- `StateNew`: New connection accepted
- `StateActive`: Connection has read 1+ bytes
- `StateIdle`: Connection waiting for new request (keep-alive)
- `StateHijacked`: Connection hijacked by handler
- `StateClosed`: Connection closed

## Base Context Customization

Provide a base context for all requests:

```go
server := &http.Server{
    Addr: ":8080",
    BaseContext: func(l net.Listener) context.Context {
        ctx := context.Background()
        ctx = context.WithValue(ctx, "server", "my-server")
        return ctx
    },
}
```

## Connection Context Customization

Modify context for each connection:

```go
server := &http.Server{
    Addr: ":8080",
    ConnContext: func(ctx context.Context, c net.Conn) context.Context {
        return context.WithValue(ctx, "remote_addr", c.RemoteAddr())
    },
}
```

## Shutdown Hooks

Register functions to call on shutdown:

```go
server.RegisterOnShutdown(func() {
    log.Println("Cleaning up resources...")
    // Cleanup logic
})
```

## Keep-Alive Control

Enable or disable keep-alive connections:

```go
server.SetKeepAlivesEnabled(false) // Disable keep-alives
```

## TLS Configuration

### Basic TLS Setup

```go
server := &http.Server{
    Addr: ":8443",
}
log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
```

### Advanced TLS Configuration

```go
import "crypto/tls"

tlsConfig := &tls.Config{
    MinVersion:               tls.VersionTLS12,
    CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
    PreferServerCipherSuites: true,
}

server := &http.Server{
    Addr:      ":8443",
    TLSConfig: tlsConfig,
}
log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
```

## Complete Production Example

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    // Create handler
    mux := http.NewServeMux()
    mux.HandleFunc("/", homeHandler)

    // Configure server
    server := &http.Server{
        Addr:           ":8080",
        Handler:        mux,
        ReadTimeout:    10 * time.Second,
        WriteTimeout:   10 * time.Second,
        IdleTimeout:    120 * time.Second,
        MaxHeaderBytes: 1 << 20, // 1 MB
        ErrorLog:       log.New(os.Stderr, "HTTP: ", log.LstdFlags),
    }

    // Start server in goroutine
    go func() {
        log.Printf("Server starting on %s", server.Addr)
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("Server failed: %v", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Server shutting down...")

    // Graceful shutdown with 30s timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Server shutdown failed: %v", err)
    }

    log.Println("Server stopped gracefully")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello, World!"))
}
```

## HTTP/2 Support

HTTP/2 is automatically enabled for HTTPS servers. To disable:

```go
import "net/http"

server := &http.Server{
    Addr:         ":8443",
    TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
}
```

Or use environment variable:
```bash
GODEBUG=http2server=0 ./myserver
```

## Key Considerations

1. **Always set timeouts** in production to prevent resource exhaustion
2. **Use graceful shutdown** to handle in-flight requests properly
3. **Configure MaxHeaderBytes** to prevent header-based DoS attacks
4. **Use custom ErrorLog** for better error tracking
5. **Enable HTTP/2** for better performance (default for TLS)
6. **Monitor connection states** for debugging connection issues

## Related Documentation

- [Getting Started](getting-started.md): Basic server setup
- [Handlers](handlers.md): Request handler patterns
- [Middleware](middleware.md): Adding cross-cutting concerns
