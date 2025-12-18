# HTTP Response Writing

The `http.ResponseWriter` interface is used to construct HTTP responses in handlers.

## ResponseWriter Interface

```go
type ResponseWriter interface {
    Header() Header              // Returns the header map
    Write([]byte) (int, error)   // Writes the response body
    WriteHeader(statusCode int)  // Sends HTTP status code
}
```

## Basic Response Writing

### Simple Text Response

```go
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World!")
}
```

### Write Bytes

```go
func handler(w http.ResponseWriter, r *http.Request) {
    data := []byte("Response data")
    w.Write(data)
}
```

### Write String

```go
func handler(w http.ResponseWriter, r *http.Request) {
    io.WriteString(w, "Hello, World!")
}
```

## Setting Headers

Headers must be set before writing the response body.

### Set Content-Type

```go
func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"message": "Hello"}`)
}
```

### Set Multiple Headers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("X-Custom-Header", "value")

    fmt.Fprintf(w, "<h1>Hello</h1>")
}
```

### Add Multiple Values

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Add (append) vs Set (replace)
    w.Header().Add("Set-Cookie", "session=abc123")
    w.Header().Add("Set-Cookie", "theme=dark")

    // Alternative using SetCookie
    http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123"})
}
```

### Delete Header

```go
func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Del("X-Powered-By")
}
```

## Status Codes

### Default Status (200 OK)

If you don't call `WriteHeader`, the first `Write` call sends status 200:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Implicitly sends 200 OK
    fmt.Fprintf(w, "Success")
}
```

### Explicit Status Code

```go
func handler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusCreated) // 201
    fmt.Fprintf(w, "Resource created")
}
```

Must call `WriteHeader` before any `Write` calls.

### Common Status Codes

```go
w.WriteHeader(http.StatusOK)                    // 200
w.WriteHeader(http.StatusCreated)               // 201
w.WriteHeader(http.StatusNoContent)             // 204
w.WriteHeader(http.StatusMovedPermanently)      // 301
w.WriteHeader(http.StatusFound)                 // 302
w.WriteHeader(http.StatusBadRequest)            // 400
w.WriteHeader(http.StatusUnauthorized)          // 401
w.WriteHeader(http.StatusForbidden)             // 403
w.WriteHeader(http.StatusNotFound)              // 404
w.WriteHeader(http.StatusMethodNotAllowed)      // 405
w.WriteHeader(http.StatusInternalServerError)   // 500
w.WriteHeader(http.StatusServiceUnavailable)    // 503
```

## JSON Responses

### Encode JSON

```go
type Response struct {
    Message string `json:"message"`
    Status  int    `json:"status"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    resp := Response{
        Message: "Success",
        Status:  200,
    }

    if err := json.NewEncoder(w).Encode(resp); err != nil {
        http.Error(w, "Encoding error", http.StatusInternalServerError)
    }
}
```

### JSON Error Response

```go
func sendJSONError(w http.ResponseWriter, message string, code int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{
        "error": message,
    })
}

func handler(w http.ResponseWriter, r *http.Request) {
    if err := validate(r); err != nil {
        sendJSONError(w, "Validation failed", http.StatusBadRequest)
        return
    }
}
```

## Error Responses

### Using http.Error

```go
func handler(w http.ResponseWriter, r *http.Request) {
    if err := someOperation(); err != nil {
        http.Error(w, "Something went wrong", http.StatusInternalServerError)
        return
    }
}
```

`http.Error` sets Content-Type to "text/plain; charset=utf-8" and adds a newline.

### Using http.NotFound

```go
func handler(w http.ResponseWriter, r *http.Request) {
    resource := findResource(r.URL.Path)
    if resource == nil {
        http.NotFound(w, r)
        return
    }
}
```

### Custom Error Page

```go
func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusNotFound)
    fmt.Fprintf(w, `
        <!DOCTYPE html>
        <html>
        <body>
            <h1>404 - Page Not Found</h1>
            <p>The requested page does not exist.</p>
        </body>
        </html>
    `)
}
```

## Redirects

### Temporary Redirect (302)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/new-location", http.StatusFound)
}
```

### Permanent Redirect (301)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/new-location", http.StatusMovedPermanently)
}
```

### See Other (303)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // After POST, redirect to GET
    http.Redirect(w, r, "/success", http.StatusSeeOther)
}
```

## Cookies

### Set Cookie

```go
func handler(w http.ResponseWriter, r *http.Request) {
    cookie := &http.Cookie{
        Name:     "session",
        Value:    "abc123",
        Path:     "/",
        MaxAge:   3600,           // 1 hour
        HttpOnly: true,           // Not accessible via JavaScript
        Secure:   true,           // HTTPS only
        SameSite: http.SameSiteStrictMode,
    }

    http.SetCookie(w, cookie)
}
```

### Delete Cookie

```go
func handler(w http.ResponseWriter, r *http.Request) {
    cookie := &http.Cookie{
        Name:   "session",
        Value:  "",
        Path:   "/",
        MaxAge: -1, // Delete immediately
    }

    http.SetCookie(w, cookie)
}
```

## File Downloads

### Serve File

```go
func handler(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "/path/to/file.pdf")
}
```

Automatically sets Content-Type, handles range requests, and ETag.

### Force Download

```go
func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Disposition", "attachment; filename=document.pdf")
    w.Header().Set("Content-Type", "application/pdf")

    http.ServeFile(w, r, "/path/to/file.pdf")
}
```

### Serve Content

```go
func handler(w http.ResponseWriter, r *http.Request) {
    data := []byte("File contents")
    reader := bytes.NewReader(data)

    http.ServeContent(w, r, "file.txt", time.Now(), reader)
}
```

## Streaming Responses

### Stream Text

```go
func handler(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/plain")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    for i := 0; i < 10; i++ {
        fmt.Fprintf(w, "Message %d\n", i)
        flusher.Flush() // Send immediately
        time.Sleep(time.Second)
    }
}
```

### Server-Sent Events (SSE)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        return
    }

    for {
        select {
        case <-r.Context().Done():
            return
        case <-time.After(time.Second):
            fmt.Fprintf(w, "data: %s\n\n", time.Now().Format(time.RFC3339))
            flusher.Flush()
        }
    }
}
```

## Response Controller

For advanced response control (Go 1.20+):

```go
func handler(w http.ResponseWriter, r *http.Request) {
    rc := http.NewResponseController(w)

    // Set write deadline
    rc.SetWriteDeadline(time.Now().Add(5 * time.Second))

    // Flush buffered data
    if err := rc.Flush(); err != nil {
        log.Printf("Flush error: %v", err)
    }

    // Enable full-duplex communication
    if err := rc.EnableFullDuplex(); err != nil {
        log.Printf("Full duplex error: %v", err)
    }
}
```

## Response Buffering

### Custom Response Writer

Capture response for logging or modification:

```go
type ResponseRecorder struct {
    http.ResponseWriter
    StatusCode int
    Body       *bytes.Buffer
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
    return &ResponseRecorder{
        ResponseWriter: w,
        StatusCode:     200,
        Body:           new(bytes.Buffer),
    }
}

func (r *ResponseRecorder) WriteHeader(code int) {
    r.StatusCode = code
    r.ResponseWriter.WriteHeader(code)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
    r.Body.Write(b) // Capture
    return r.ResponseWriter.Write(b)
}

// Usage in middleware
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rec := NewResponseRecorder(w)
        next.ServeHTTP(rec, r)

        log.Printf("Status: %d, Size: %d", rec.StatusCode, rec.Body.Len())
    })
}
```

## Important Rules

### 1. Headers Before Body

```go
// Correct order
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write(data)

// Wrong: Headers ignored
w.Write(data)
w.Header().Set("Content-Type", "application/json") // Too late!
```

### 2. WriteHeader Called Once

```go
// Wrong: Multiple status codes
w.WriteHeader(http.StatusOK)
w.WriteHeader(http.StatusCreated) // Ignored, warning logged
```

### 3. Check for Interface Support

```go
// Check if ResponseWriter supports Flusher
if flusher, ok := w.(http.Flusher); ok {
    flusher.Flush()
}

// Check if ResponseWriter supports Hijacker
if hijacker, ok := w.(http.Hijacker); ok {
    conn, buf, err := hijacker.Hijack()
    // Use raw connection
}
```

## Content Types

Common MIME types:

```go
// Text
w.Header().Set("Content-Type", "text/plain; charset=utf-8")
w.Header().Set("Content-Type", "text/html; charset=utf-8")
w.Header().Set("Content-Type", "text/css")
w.Header().Set("Content-Type", "text/javascript")

// Application
w.Header().Set("Content-Type", "application/json")
w.Header().Set("Content-Type", "application/xml")
w.Header().Set("Content-Type", "application/pdf")
w.Header().Set("Content-Type", "application/octet-stream")

// Multipart
w.Header().Set("Content-Type", "multipart/form-data")

// Images
w.Header().Set("Content-Type", "image/jpeg")
w.Header().Set("Content-Type", "image/png")
w.Header().Set("Content-Type", "image/gif")
```

## Related Documentation

- [Request](request.md): Reading request data
- [Handlers](handlers.md): Handler patterns
- [Common Patterns](common-patterns.md): File serving, redirects
