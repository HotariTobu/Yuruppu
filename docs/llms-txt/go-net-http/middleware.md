# Middleware Patterns

Middleware wraps HTTP handlers to add cross-cutting functionality like logging, authentication, or error handling.

## What is Middleware?

Middleware is a function that wraps a handler and returns a new handler:

```go
func middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Before handler
        next.ServeHTTP(w, r)
        // After handler
    })
}
```

## Basic Middleware Pattern

### Simple Logging Middleware

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        log.Printf("Started %s %s", r.Method, r.URL.Path)

        next.ServeHTTP(w, r)

        log.Printf("Completed in %v", time.Since(start))
    })
}

// Usage
handler := loggingMiddleware(myHandler)
http.Handle("/", handler)
```

### Recovery Middleware

Recover from panics and return 500 error:

```go
func recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Panic: %v", err)
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()

        next.ServeHTTP(w, r)
    })
}
```

## Authentication Middleware

### Basic Auth

```go
func basicAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        username, password, ok := r.BasicAuth()
        if !ok {
            w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        if !validateCredentials(username, password) {
            http.Error(w, "Invalid credentials", http.StatusUnauthorized)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Token Authentication

```go
func tokenAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Missing token", http.StatusUnauthorized)
            return
        }

        // Validate token
        if !isValidToken(token) {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### JWT Middleware with Context

```go
type contextKey string

const userContextKey contextKey = "user"

func jwtMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            http.Error(w, "Invalid authorization", http.StatusUnauthorized)
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        user, err := validateJWT(token)
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        // Add user to context
        ctx := context.WithValue(r.Context(), userContextKey, user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Access user in handler
func handler(w http.ResponseWriter, r *http.Request) {
    user := r.Context().Value(userContextKey).(*User)
    fmt.Fprintf(w, "Hello, %s", user.Name)
}
```

## Request/Response Modification

### Content-Type Middleware

```go
func contentTypeMiddleware(contentType string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Content-Type", contentType)
            next.ServeHTTP(w, r)
        })
    }
}

// Usage
handler := contentTypeMiddleware("application/json")(myHandler)
```

### CORS Middleware

```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Request ID Middleware

```go
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = generateRequestID()
        }

        w.Header().Set("X-Request-ID", requestID)

        ctx := context.WithValue(r.Context(), "request_id", requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Rate Limiting

### Simple Rate Limiter

```go
import "golang.org/x/time/rate"

func rateLimitMiddleware(limiter *rate.Limiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// Usage
limiter := rate.NewLimiter(rate.Limit(10), 20) // 10 req/sec, burst 20
handler := rateLimitMiddleware(limiter)(myHandler)
```

### Per-IP Rate Limiter

```go
type IPRateLimiter struct {
    ips map[string]*rate.Limiter
    mu  *sync.RWMutex
    r   rate.Limit
    b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
    return &IPRateLimiter{
        ips: make(map[string]*rate.Limiter),
        mu:  &sync.RWMutex{},
        r:   r,
        b:   b,
    }
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
    i.mu.Lock()
    defer i.mu.Unlock()

    limiter, exists := i.ips[ip]
    if !exists {
        limiter = rate.NewLimiter(i.r, i.b)
        i.ips[ip] = limiter
    }

    return limiter
}

func (i *IPRateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := getIP(r)
        limiter := i.GetLimiter(ip)

        if !limiter.Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func getIP(r *http.Request) string {
    ip := r.Header.Get("X-Forwarded-For")
    if ip == "" {
        ip = r.Header.Get("X-Real-IP")
    }
    if ip == "" {
        ip = r.RemoteAddr
    }
    return ip
}
```

## Timeout Middleware

```go
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx, cancel := context.WithTimeout(r.Context(), timeout)
            defer cancel()

            r = r.WithContext(ctx)

            done := make(chan struct{})
            go func() {
                next.ServeHTTP(w, r)
                close(done)
            }()

            select {
            case <-done:
                return
            case <-ctx.Done():
                http.Error(w, "Request Timeout", http.StatusRequestTimeout)
            }
        })
    }
}

// Usage with built-in TimeoutHandler (simpler)
handler := http.TimeoutHandler(myHandler, 5*time.Second, "Request timeout")
```

## Chaining Middleware

### Manual Chaining

```go
handler := loggingMiddleware(
    recoveryMiddleware(
        authMiddleware(
            myHandler,
        ),
    ),
)
```

### Chain Helper Function

```go
func chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
    for i := len(middlewares) - 1; i >= 0; i-- {
        handler = middlewares[i](handler)
    }
    return handler
}

// Usage
handler := chain(
    myHandler,
    loggingMiddleware,
    recoveryMiddleware,
    authMiddleware,
)
```

### Middleware Stack Type

```go
type Middleware func(http.Handler) http.Handler

type Stack struct {
    middlewares []Middleware
}

func NewStack(middlewares ...Middleware) *Stack {
    return &Stack{middlewares: middlewares}
}

func (s *Stack) Then(handler http.Handler) http.Handler {
    for i := len(s.middlewares) - 1; i >= 0; i-- {
        handler = s.middlewares[i](handler)
    }
    return handler
}

// Usage
stack := NewStack(
    loggingMiddleware,
    recoveryMiddleware,
    authMiddleware,
)

http.Handle("/", stack.Then(myHandler))
http.Handle("/api", stack.Then(apiHandler))
```

## Response Capture Middleware

### Capture Status Code

```go
type statusRecorder struct {
    http.ResponseWriter
    statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
    r.statusCode = code
    r.ResponseWriter.WriteHeader(code)
}

func statusLoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        recorder := &statusRecorder{
            ResponseWriter: w,
            statusCode:     200, // Default
        }

        next.ServeHTTP(recorder, r)

        log.Printf("%s %s - %d", r.Method, r.URL.Path, recorder.statusCode)
    })
}
```

### Capture Response Body

```go
type responseRecorder struct {
    http.ResponseWriter
    statusCode int
    body       *bytes.Buffer
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
    return &responseRecorder{
        ResponseWriter: w,
        statusCode:     200,
        body:           new(bytes.Buffer),
    }
}

func (r *responseRecorder) WriteHeader(code int) {
    r.statusCode = code
    r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
    r.body.Write(b)
    return r.ResponseWriter.Write(b)
}

func responseLoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        recorder := newResponseRecorder(w)
        next.ServeHTTP(recorder, r)

        log.Printf("Response: %d (%d bytes)", recorder.statusCode, recorder.body.Len())
    })
}
```

## Conditional Middleware

### Apply Middleware to Specific Paths

```go
func conditionalMiddleware(pattern string, middleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if strings.HasPrefix(r.URL.Path, pattern) {
                middleware(next).ServeHTTP(w, r)
            } else {
                next.ServeHTTP(w, r)
            }
        })
    }
}

// Usage
handler := conditionalMiddleware("/api/", authMiddleware)(myHandler)
```

### Apply Middleware to Specific Methods

```go
func methodMiddleware(method string, middleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.Method == method {
                middleware(next).ServeHTTP(w, r)
            } else {
                next.ServeHTTP(w, r)
            }
        })
    }
}

// Usage
handler := methodMiddleware("POST", csrfMiddleware)(myHandler)
```

## Complete Example

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"
)

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        log.Printf("[%s] %s %s", start.Format(time.RFC3339), r.Method, r.URL.Path)

        next.ServeHTTP(w, r)

        log.Printf("Completed in %v", time.Since(start))
    })
}

// Recovery middleware
func recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Panic recovered: %v", err)
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()

        next.ServeHTTP(w, r)
    })
}

// Request ID middleware
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := generateRequestID()
        ctx := context.WithValue(r.Context(), "request_id", requestID)
        w.Header().Set("X-Request-ID", requestID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Chain helper
func chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
    for i := len(middlewares) - 1; i >= 0; i-- {
        handler = middlewares[i](handler)
    }
    return handler
}

func main() {
    mux := http.NewServeMux()

    // Apply middleware to all routes
    mux.HandleFunc("/", homeHandler)
    mux.HandleFunc("/api/users", usersHandler)

    handler := chain(
        mux,
        loggingMiddleware,
        recoveryMiddleware,
        requestIDMiddleware,
    )

    log.Fatal(http.ListenAndServe(":8080", handler))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    requestID := r.Context().Value("request_id").(string)
    w.Write([]byte("Home - Request ID: " + requestID))
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Users"))
}

func generateRequestID() string {
    return time.Now().Format("20060102150405")
}
```

## Middleware Best Practices

1. **Order matters**: Apply middleware in the correct order (logging → recovery → auth → handler)
2. **Keep it simple**: Each middleware should do one thing
3. **Use context for values**: Pass data between middleware using context
4. **Handle errors gracefully**: Always recover from panics in middleware
5. **Be efficient**: Middleware runs for every request
6. **Make it reusable**: Write middleware that can be applied to different handlers
7. **Document behavior**: Make clear what each middleware does

## Common Middleware Order

```go
handler := chain(
    mux,
    recoveryMiddleware,      // 1. Catch panics first
    loggingMiddleware,       // 2. Log all requests
    corsMiddleware,          // 3. Set CORS headers
    requestIDMiddleware,     // 4. Add request ID
    authMiddleware,          // 5. Authenticate
    rateLimitMiddleware,     // 6. Rate limit
    timeoutMiddleware,       // 7. Set timeout
)
```

## Related Documentation

- [Handlers](handlers.md): Handler patterns and types
- [Request](request.md): Accessing request data
- [Response](response.md): Writing responses
