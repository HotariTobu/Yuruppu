# HTTP Handlers

Handlers are the core of HTTP request processing in Go. They receive requests and write responses.

## Handler Interface

The fundamental interface for handling HTTP requests:

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

Any type that implements `ServeHTTP` can handle HTTP requests.

## HandlerFunc Type

Convert ordinary functions into handlers:

```go
type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
    f(w, r)
}
```

This adapter allows functions to implement the Handler interface.

## Basic Handler Function

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World!")
}

// Register it
http.HandleFunc("/", myHandler)
```

## Inline Handler Function

```go
http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello!")
})
```

## Stateful Handler Type

Create a handler with internal state:

```go
type CountHandler struct {
    mu    sync.Mutex
    count int
}

func (h *CountHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.count++
    fmt.Fprintf(w, "Request count: %d\n", h.count)
}

// Register it
http.Handle("/count", &CountHandler{})
```

Key points:
- Use mutexes for concurrent access to shared state
- Handlers are called concurrently for multiple requests
- Store configuration or dependencies as fields

## Handler Registration

### Using Handle (for Handler types)

```go
type MyHandler struct{}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Handler response")
}

http.Handle("/path", &MyHandler{})
```

### Using HandleFunc (for functions)

```go
http.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Function response")
})
```

### Using Custom ServeMux

```go
mux := http.NewServeMux()
mux.HandleFunc("/", homeHandler)
mux.Handle("/admin", adminHandler)

server := &http.Server{
    Addr:    ":8080",
    Handler: mux,
}
```

## Built-in Handler Constructors

Go provides several utility functions that return handlers:

### FileServer

Serve static files from a directory:

```go
fs := http.FileServer(http.Dir("./static"))
http.Handle("/static/", http.StripPrefix("/static/", fs))
```

### NotFoundHandler

Returns a handler that responds with 404:

```go
handler := http.NotFoundHandler()
http.Handle("/disabled", handler)
```

### RedirectHandler

Returns a handler that redirects to a URL:

```go
handler := http.RedirectHandler("https://example.com", http.StatusMovedPermanently)
http.Handle("/old-path", handler)
```

### StripPrefix

Strip a prefix from the URL path before passing to another handler:

```go
http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir("./uploads"))))
```

Request to `/files/doc.pdf` serves `./uploads/doc.pdf`.

### TimeoutHandler

Add a timeout to any handler:

```go
handler := http.TimeoutHandler(myHandler, 5*time.Second, "Request timeout")
http.Handle("/slow", handler)
```

If the handler takes longer than 5 seconds, returns the timeout message.

### MaxBytesHandler

Limit request body size:

```go
handler := http.MaxBytesHandler(myHandler, 1<<20) // 1 MB limit
http.Handle("/upload", handler)
```

## Handler Patterns

### Method-Based Routing

```go
func userHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        getUser(w, r)
    case http.MethodPost:
        createUser(w, r)
    case http.MethodPut:
        updateUser(w, r)
    case http.MethodDelete:
        deleteUser(w, r)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}
```

### Path Parameter Extraction

```go
func userHandler(w http.ResponseWriter, r *http.Request) {
    // URL: /users/123
    path := r.URL.Path
    id := strings.TrimPrefix(path, "/users/")

    fmt.Fprintf(w, "User ID: %s", id)
}
```

With Go 1.22+ pattern matching:

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    fmt.Fprintf(w, "User ID: %s", id)
})
```

### Dependency Injection

Pass dependencies to handlers using closures:

```go
type Database struct {
    // DB fields
}

func userHandler(db *Database) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Use db here
        users := db.GetAllUsers()
        json.NewEncoder(w).Encode(users)
    }
}

// Usage
db := &Database{/* init */}
http.HandleFunc("/users", userHandler(db))
```

### Handler Struct with Dependencies

```go
type UserHandler struct {
    db     *Database
    logger *log.Logger
}

func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    h.logger.Printf("Request: %s %s", r.Method, r.URL.Path)
    // Use h.db to fetch users
}

// Usage
handler := &UserHandler{
    db:     db,
    logger: logger,
}
http.Handle("/users", handler)
```

### JSON Response Handler

```go
type JSONHandler struct {
    data interface{}
}

func (h *JSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(h.data); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

// Usage
http.Handle("/api/data", &JSONHandler{data: myData})
```

### Error Handling Pattern

```go
type HandlerFunc func(http.ResponseWriter, *http.Request) error

func (fn HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if err := fn(w, r); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

// Usage
func myHandler(w http.ResponseWriter, r *http.Request) error {
    if err := someOperation(); err != nil {
        return err
    }
    fmt.Fprintf(w, "Success")
    return nil
}

http.Handle("/path", HandlerFunc(myHandler))
```

## Handler Wrapping

Create handlers that wrap other handlers (middleware pattern):

```go
func loggingHandler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

// Usage
handler := loggingHandler(myHandler)
http.Handle("/path", handler)
```

## Concurrent Handler Safety

Handlers are called concurrently. Follow these rules:

### Safe: Read-only access

```go
var config = Config{/* ... */}

func handler(w http.ResponseWriter, r *http.Request) {
    // Safe: config is read-only
    fmt.Fprintf(w, "Port: %d", config.Port)
}
```

### Unsafe: Shared mutable state without sync

```go
var counter int // UNSAFE!

func handler(w http.ResponseWriter, r *http.Request) {
    counter++ // Race condition!
    fmt.Fprintf(w, "Count: %d", counter)
}
```

### Safe: Protected shared state

```go
var (
    mu      sync.Mutex
    counter int
)

func handler(w http.ResponseWriter, r *http.Request) {
    mu.Lock()
    counter++
    count := counter
    mu.Unlock()

    fmt.Fprintf(w, "Count: %d", count)
}
```

### Safe: Request-scoped state

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Each request gets its own variable
    localCounter := 0
    localCounter++
    fmt.Fprintf(w, "Count: %d", localCounter)
}
```

## Handler Testing

Test handlers using httptest:

```go
import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHandler(t *testing.T) {
    req := httptest.NewRequest("GET", "/test", nil)
    w := httptest.NewRecorder()

    myHandler(w, req)

    resp := w.Result()
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected 200, got %d", resp.StatusCode)
    }
}
```

## Common Handler Mistakes

### 1. Not checking HTTP method

```go
// Bad: Handles all methods
func handler(w http.ResponseWriter, r *http.Request) {
    // Process POST data regardless of method
}

// Good: Check method
func handler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // Process POST data
}
```

### 2. Writing headers after body

```go
// Bad: Header set after Write
func handler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello"))
    w.Header().Set("Content-Type", "text/plain") // Too late!
}

// Good: Headers before Write
func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte("Hello"))
}
```

### 3. Not handling errors

```go
// Bad: Ignoring errors
func handler(w http.ResponseWriter, r *http.Request) {
    data, _ := fetchData() // Error ignored
    json.NewEncoder(w).Encode(data)
}

// Good: Handle errors
func handler(w http.ResponseWriter, r *http.Request) {
    data, err := fetchData()
    if err != nil {
        http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(data)
}
```

## Related Documentation

- [Getting Started](getting-started.md): Basic handler setup
- [Middleware](middleware.md): Handler wrapping patterns
- [Request](request.md): Working with request data
- [Response](response.md): Writing responses
