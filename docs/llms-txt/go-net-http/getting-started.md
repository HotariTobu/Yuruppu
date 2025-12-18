# Getting Started with net/http

This guide covers the basics of creating HTTP servers in Go using the net/http package.

## Basic HTTP Server

The simplest way to create an HTTP server:

```go
package main

import (
    "fmt"
    "log"
    "net/http"
)

func main() {
    // Register a handler function
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, World!")
    })

    // Start server on port 8080
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Key points:
- `http.HandleFunc()` registers a handler function for a URL pattern
- `http.ListenAndServe()` starts the server with address and handler
- Passing `nil` as handler uses the `DefaultServeMux` (default router)
- The server runs until an error occurs (logged via `log.Fatal`)

## Multiple Route Handlers

```go
func main() {
    http.HandleFunc("/", homeHandler)
    http.HandleFunc("/about", aboutHandler)
    http.HandleFunc("/api/users", usersHandler)

    log.Println("Starting server on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Welcome to the home page!")
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "About page")
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"users": ["alice", "bob"]}`)
}
```

## Custom Server Configuration

For production use, configure timeouts and other server options:

```go
import (
    "log"
    "net/http"
    "time"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", homeHandler)

    server := &http.Server{
        Addr:         ":8080",
        Handler:      mux,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    log.Println("Server starting on", server.Addr)
    log.Fatal(server.ListenAndServe())
}
```

Why configure timeouts:
- `ReadTimeout`: Prevents slow clients from holding connections
- `WriteTimeout`: Ensures responses are sent in reasonable time
- `IdleTimeout`: Closes idle keep-alive connections

## Handler vs HandlerFunc

Two ways to register handlers:

### Using HandlerFunc (for functions)

```go
http.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) {
    // Handler logic
})
```

### Using Handle (for types implementing Handler interface)

```go
type MyHandler struct {
    // Handler state
}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Handler logic
}

http.Handle("/path", &MyHandler{})
```

## Setting Response Headers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Set content type
    w.Header().Set("Content-Type", "application/json")

    // Set custom headers
    w.Header().Set("X-Custom-Header", "value")

    // Write status code (must be before Write)
    w.WriteHeader(http.StatusOK)

    // Write response body
    fmt.Fprintf(w, `{"status": "ok"}`)
}
```

Important: Headers must be set before writing the response body.

## Handling Different HTTP Methods

```go
func handler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        // Handle GET request
        fmt.Fprintf(w, "GET request")
    case http.MethodPost:
        // Handle POST request
        fmt.Fprintf(w, "POST request")
    case http.MethodPut:
        // Handle PUT request
        fmt.Fprintf(w, "PUT request")
    default:
        // Method not allowed
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}
```

## Error Responses

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Simple error response
    http.Error(w, "Something went wrong", http.StatusInternalServerError)

    // Not found
    http.NotFound(w, r)

    // Custom error response
    w.WriteHeader(http.StatusBadRequest)
    fmt.Fprintf(w, `{"error": "Invalid input"}`)
}
```

## Graceful Shutdown

```go
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
    server := &http.Server{
        Addr:    ":8080",
        Handler: http.DefaultServeMux,
    }

    // Start server in goroutine
    go func() {
        log.Println("Server starting on", server.Addr)
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }

    log.Println("Server stopped")
}
```

## HTTPS Server

```go
func main() {
    http.HandleFunc("/", handler)

    // Start HTTPS server
    log.Fatal(http.ListenAndServeTLS(
        ":8443",
        "cert.pem",  // TLS certificate
        "key.pem",   // TLS private key
        nil,
    ))
}
```

## Quick Reference

| Function | Purpose |
|----------|---------|
| `http.HandleFunc(pattern, func)` | Register handler function |
| `http.Handle(pattern, handler)` | Register handler type |
| `http.ListenAndServe(addr, handler)` | Start HTTP server |
| `http.ListenAndServeTLS(addr, cert, key, handler)` | Start HTTPS server |
| `http.Error(w, msg, code)` | Send error response |
| `http.NotFound(w, r)` | Send 404 response |
| `http.Redirect(w, r, url, code)` | Send redirect response |

## Next Steps

- Learn about [Handlers](handlers.md) for advanced handler patterns
- Configure [Server](server.md) options for production
- Implement [Middleware](middleware.md) for cross-cutting concerns
- Handle [Request](request.md) data like forms and cookies
