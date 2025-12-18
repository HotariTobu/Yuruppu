# Helper Functions

The net/http package provides many utility functions for common HTTP operations.

## Server Functions

### ListenAndServe

Start an HTTP server:

```go
func ListenAndServe(addr string, handler Handler) error
```

Example:
```go
http.ListenAndServe(":8080", nil)
http.ListenAndServe(":8080", myHandler)
```

### ListenAndServeTLS

Start an HTTPS server:

```go
func ListenAndServeTLS(addr, certFile, keyFile string, handler Handler) error
```

Example:
```go
http.ListenAndServeTLS(":8443", "cert.pem", "key.pem", nil)
```

### Serve

Serve connections from a listener:

```go
func Serve(l net.Listener, handler Handler) error
```

Example:
```go
listener, _ := net.Listen("tcp", ":8080")
http.Serve(listener, myHandler)
```

### ServeTLS

Serve TLS connections from a listener:

```go
func ServeTLS(l net.Listener, handler Handler, certFile, keyFile string) error
```

## Handler Registration

### Handle

Register a handler for a pattern:

```go
func Handle(pattern string, handler Handler)
```

Example:
```go
http.Handle("/api", apiHandler)
```

### HandleFunc

Register a handler function for a pattern:

```go
func HandleFunc(pattern string, handler func(ResponseWriter, *Request))
```

Example:
```go
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello")
})
```

## Response Helpers

### Error

Send an error response:

```go
func Error(w ResponseWriter, error string, code int)
```

Example:
```go
http.Error(w, "Bad request", http.StatusBadRequest)
http.Error(w, "Not found", http.StatusNotFound)
http.Error(w, "Internal error", http.StatusInternalServerError)
```

Sets Content-Type to "text/plain; charset=utf-8" and adds a newline.

### NotFound

Send a 404 Not Found response:

```go
func NotFound(w ResponseWriter, r *Request)
```

Example:
```go
http.NotFound(w, r)
```

### NotFoundHandler

Create a handler that returns 404:

```go
func NotFoundHandler() Handler
```

Example:
```go
handler := http.NotFoundHandler()
http.Handle("/disabled", handler)
```

### Redirect

Send a redirect response:

```go
func Redirect(w ResponseWriter, r *Request, url string, code int)
```

Example:
```go
http.Redirect(w, r, "/new-path", http.StatusFound)
http.Redirect(w, r, "/new-path", http.StatusMovedPermanently)
http.Redirect(w, r, "/success", http.StatusSeeOther)
```

### RedirectHandler

Create a handler that redirects:

```go
func RedirectHandler(url string, code int) Handler
```

Example:
```go
handler := http.RedirectHandler("https://example.com", http.StatusMovedPermanently)
http.Handle("/old-path", handler)
```

## File Serving

### ServeFile

Serve a single file:

```go
func ServeFile(w ResponseWriter, r *Request, name string)
```

Example:
```go
http.ServeFile(w, r, "static/index.html")
http.ServeFile(w, r, "/path/to/file.pdf")
```

Automatically handles:
- Content-Type detection
- Range requests
- ETag generation
- If-Modified-Since checks

### ServeContent

Serve content with metadata:

```go
func ServeContent(w ResponseWriter, req *Request, name string, modtime time.Time, content io.ReadSeeker)
```

Example:
```go
data := []byte("File contents")
reader := bytes.NewReader(data)
modtime := time.Now()

http.ServeContent(w, r, "file.txt", modtime, reader)
```

Handles range requests, ETags, and conditional requests.

### FileServer

Create a handler that serves files from a directory:

```go
func FileServer(root FileSystem) Handler
```

Example:
```go
fs := http.FileServer(http.Dir("./static"))
http.Handle("/static/", http.StripPrefix("/static/", fs))
```

### Dir

Create a FileSystem from a directory path:

```go
func Dir(dir string) FileSystem
```

Example:
```go
fs := http.Dir("./public")
handler := http.FileServer(fs)
```

### StripPrefix

Strip a prefix from request URLs before passing to a handler:

```go
func StripPrefix(prefix string, h Handler) Handler
```

Example:
```go
fs := http.FileServer(http.Dir("./uploads"))
http.Handle("/files/", http.StripPrefix("/files/", fs))
```

Request to `/files/doc.pdf` serves `./uploads/doc.pdf`.

## Cookie Helpers

### SetCookie

Set a cookie in the response:

```go
func SetCookie(w ResponseWriter, cookie *Cookie)
```

Example:
```go
cookie := &http.Cookie{
    Name:     "session",
    Value:    "abc123",
    Path:     "/",
    MaxAge:   3600,
    HttpOnly: true,
}
http.SetCookie(w, cookie)
```

## Content Detection

### DetectContentType

Detect MIME type from data:

```go
func DetectContentType(data []byte) string
```

Example:
```go
data := []byte{0xFF, 0xD8, 0xFF} // JPEG signature
contentType := http.DetectContentType(data) // "image/jpeg"

w.Header().Set("Content-Type", contentType)
w.Write(data)
```

Examines up to 512 bytes. Returns "application/octet-stream" if unknown.

## Status Helpers

### StatusText

Get the text description of a status code:

```go
func StatusText(code int) string
```

Example:
```go
text := http.StatusText(200)  // "OK"
text := http.StatusText(404)  // "Not Found"
text := http.StatusText(500)  // "Internal Server Error"
```

Returns empty string for unknown codes.

## Handler Wrappers

### TimeoutHandler

Add a timeout to a handler:

```go
func TimeoutHandler(h Handler, dt time.Duration, msg string) Handler
```

Example:
```go
handler := http.TimeoutHandler(myHandler, 5*time.Second, "Request timeout")
http.Handle("/slow", handler)
```

If handler takes longer than `dt`, returns 503 with `msg`.

### MaxBytesHandler

Limit request body size:

```go
func MaxBytesHandler(h Handler, n int64) Handler
```

Example:
```go
handler := http.MaxBytesHandler(uploadHandler, 10<<20) // 10 MB
http.Handle("/upload", handler)
```

Returns 413 Request Entity Too Large if exceeded.

### AllowQuerySemicolons

Allow semicolons in query strings (Go 1.17+):

```go
func AllowQuerySemicolons(h Handler) Handler
```

Example:
```go
handler := http.AllowQuerySemicolons(myHandler)
```

By default, semicolons in query strings are rejected for security.

## Request Helpers

### MaxBytesReader

Limit request body reading:

```go
func MaxBytesReader(w ResponseWriter, r io.ReadCloser, n int64) io.ReadCloser
```

Example:
```go
func handler(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

    data, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
        return
    }
}
```

Prevents reading more than `n` bytes from body.

### ParseHTTPVersion

Parse HTTP version string:

```go
func ParseHTTPVersion(vers string) (major, minor int, ok bool)
```

Example:
```go
major, minor, ok := http.ParseHTTPVersion("HTTP/1.1")
// major=1, minor=1, ok=true
```

### ParseTime

Parse HTTP date:

```go
func ParseTime(text string) (t time.Time, err error)
```

Example:
```go
date := r.Header.Get("If-Modified-Since")
t, err := http.ParseTime(date)
if err == nil && modTime.Before(t) {
    w.WriteHeader(http.StatusNotModified)
    return
}
```

Handles RFC1123, RFC850, and ANSI C's asctime formats.

## URL Helpers

### CanonicalHeaderKey

Get canonical header key format:

```go
func CanonicalHeaderKey(s string) string
```

Example:
```go
key := http.CanonicalHeaderKey("content-type")  // "Content-Type"
key := http.CanonicalHeaderKey("x-request-id")  // "X-Request-Id"
```

## Response Controller

### NewResponseController

Create a ResponseController for advanced operations (Go 1.20+):

```go
func NewResponseController(rw ResponseWriter) *ResponseController
```

Example:
```go
func handler(w http.ResponseWriter, r *http.Request) {
    rc := http.NewResponseController(w)

    // Flush buffered data
    rc.Flush()

    // Set deadlines
    rc.SetWriteDeadline(time.Now().Add(5 * time.Second))
    rc.SetReadDeadline(time.Now().Add(5 * time.Second))

    // Enable full-duplex communication
    rc.EnableFullDuplex()
}
```

Methods:
- `Flush() error` - Flush buffered data
- `Hijack() (net.Conn, *bufio.ReadWriter, error)` - Take over connection
- `SetReadDeadline(deadline time.Time) error` - Set read deadline
- `SetWriteDeadline(deadline time.Time) error` - Set write deadline
- `EnableFullDuplex() error` - Enable full-duplex communication

## Complete Examples

### Custom 404 Page

```go
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.ServeFile(w, r, "static/404.html")
        return
    }
    http.ServeFile(w, r, "static/index.html")
})
```

### Conditional File Serving

```go
func handler(w http.ResponseWriter, r *http.Request) {
    path := "./files" + r.URL.Path

    info, err := os.Stat(path)
    if err != nil {
        http.NotFound(w, r)
        return
    }

    if info.IsDir() {
        http.Error(w, "Directory listing denied", http.StatusForbidden)
        return
    }

    // Check If-Modified-Since
    modtime := info.ModTime()
    if t, err := http.ParseTime(r.Header.Get("If-Modified-Since")); err == nil {
        if !modtime.After(t) {
            w.WriteHeader(http.StatusNotModified)
            return
        }
    }

    http.ServeFile(w, r, path)
}
```

### Smart Content Type Detection

```go
func handler(w http.ResponseWriter, r *http.Request) {
    data, err := os.ReadFile("upload.bin")
    if err != nil {
        http.Error(w, "File not found", http.StatusNotFound)
        return
    }

    contentType := http.DetectContentType(data)
    w.Header().Set("Content-Type", contentType)
    w.Write(data)
}
```

### Size-Limited Upload

```go
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    // Limit to 10MB
    r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
        return
    }

    file, header, err := r.FormFile("upload")
    if err != nil {
        http.Error(w, "Upload error", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Process file...
    fmt.Fprintf(w, "Uploaded: %s (%d bytes)", header.Filename, header.Size)
}
```

### Timed Response

```go
func handler(w http.ResponseWriter, r *http.Request) {
    rc := http.NewResponseController(w)

    // Set write timeout
    if err := rc.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
        log.Printf("Deadline error: %v", err)
    }

    // Perform operation
    data := performSlowOperation()

    w.Write(data)
}
```

## Helper Function Summary

| Function | Purpose |
|----------|---------|
| `ListenAndServe` | Start HTTP server |
| `ListenAndServeTLS` | Start HTTPS server |
| `Handle` | Register handler |
| `HandleFunc` | Register handler function |
| `Error` | Send error response |
| `NotFound` | Send 404 response |
| `Redirect` | Send redirect |
| `ServeFile` | Serve single file |
| `FileServer` | Serve directory |
| `StripPrefix` | Strip URL prefix |
| `SetCookie` | Set cookie |
| `DetectContentType` | Detect MIME type |
| `StatusText` | Get status description |
| `TimeoutHandler` | Add timeout to handler |
| `MaxBytesHandler` | Limit request body size |
| `MaxBytesReader` | Limit body reading |

## Related Documentation

- [Getting Started](getting-started.md): Basic server setup
- [Handlers](handlers.md): Handler patterns
- [Response](response.md): Response writing
- [Common Patterns](common-patterns.md): Practical examples
