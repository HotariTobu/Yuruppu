# Advanced Features

This document covers advanced HTTP server features including HTTP/2, connection hijacking, streaming, and server push.

## HTTP/2

### Automatic HTTP/2 Support

HTTP/2 is automatically enabled for HTTPS servers:

```go
server := &http.Server{
    Addr: ":8443",
}

// HTTP/2 is automatically enabled
log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
```

No configuration needed. HTTP/2 support is transparent.

### Disable HTTP/2

To disable HTTP/2:

```go
// Server-side
server := &http.Server{
    Addr:         ":8443",
    TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
}

log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
```

Or via environment variable:

```bash
GODEBUG=http2server=0 ./myserver
```

### Check HTTP Version

```go
func handler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Protocol: %s", r.Proto)

    if r.ProtoMajor == 2 {
        log.Println("Using HTTP/2")
    }
}
```

### HTTP/2 Server Push

Push resources to client before they request them:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    pusher, ok := w.(http.Pusher)
    if !ok {
        // HTTP/2 push not supported
        http.ServeFile(w, r, "index.html")
        return
    }

    // Push CSS before serving HTML
    if err := pusher.Push("/style.css", nil); err != nil {
        log.Printf("Push error: %v", err)
    }

    // Push JavaScript
    if err := pusher.Push("/script.js", nil); err != nil {
        log.Printf("Push error: %v", err)
    }

    // Serve main content
    http.ServeFile(w, r, "index.html")
}
```

With options:

```go
options := &http.PushOptions{
    Header: http.Header{
        "Content-Type": []string{"text/css"},
    },
}

if err := pusher.Push("/style.css", options); err != nil {
    log.Printf("Push error: %v", err)
}
```

## Connection Hijacking

Take over the underlying TCP connection:

```go
func wsHandler(w http.ResponseWriter, r *http.Request) {
    hijacker, ok := w.(http.Hijacker)
    if !ok {
        http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
        return
    }

    conn, buf, err := hijacker.Hijack()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer conn.Close()

    // Now you control the raw connection
    // Example: WebSocket upgrade
    fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\n")
    fmt.Fprintf(conn, "Upgrade: websocket\r\n")
    fmt.Fprintf(conn, "Connection: Upgrade\r\n\r\n")

    // Handle WebSocket frames...
}
```

After hijacking:
- The HTTP server no longer manages the connection
- You're responsible for reading/writing raw data
- Connection won't be closed automatically

## Streaming Responses

### Server-Sent Events (SSE)

Stream events to clients:

```go
func sseHandler(w http.ResponseWriter, r *http.Request) {
    // Set headers for SSE
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    // Check if flushing is supported
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }

    // Stream events
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-r.Context().Done():
            // Client disconnected
            return
        case t := <-ticker.C:
            // Send event
            fmt.Fprintf(w, "data: %s\n\n", t.Format(time.RFC3339))
            flusher.Flush()
        }
    }
}
```

Client-side JavaScript:

```javascript
const eventSource = new EventSource('/events');

eventSource.onmessage = (event) => {
    console.log('Event:', event.data);
};

eventSource.onerror = (error) => {
    console.error('Error:', error);
    eventSource.close();
};
```

### Chunked Transfer Encoding

Stream data in chunks:

```go
func streamHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain")
    w.Header().Set("Transfer-Encoding", "chunked")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }

    for i := 0; i < 10; i++ {
        fmt.Fprintf(w, "Chunk %d\n", i)
        flusher.Flush() // Send immediately

        time.Sleep(time.Second)

        // Check if client disconnected
        select {
        case <-r.Context().Done():
            log.Println("Client disconnected")
            return
        default:
        }
    }
}
```

### Progressive Image Loading

```go
func progressiveImageHandler(w http.ResponseWriter, r *http.Request) {
    file, err := os.Open("large-image.jpg")
    if err != nil {
        http.Error(w, "Image not found", http.StatusNotFound)
        return
    }
    defer file.Close()

    w.Header().Set("Content-Type", "image/jpeg")

    flusher, _ := w.(http.Flusher)

    // Stream image in chunks
    buffer := make([]byte, 4096)
    for {
        n, err := file.Read(buffer)
        if err != nil && err != io.EOF {
            return
        }
        if n == 0 {
            break
        }

        w.Write(buffer[:n])
        if flusher != nil {
            flusher.Flush()
        }
    }
}
```

## Flusher Interface

Flush buffered data immediately:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Flushing not supported", http.StatusInternalServerError)
        return
    }

    // Write and flush immediately
    fmt.Fprintf(w, "First chunk\n")
    flusher.Flush()

    time.Sleep(time.Second)

    fmt.Fprintf(w, "Second chunk\n")
    flusher.Flush()
}
```

## Request Cancellation

Handle client disconnections gracefully:

```go
func longHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Simulate long operation
    for i := 0; i < 10; i++ {
        select {
        case <-ctx.Done():
            // Client disconnected or request cancelled
            log.Println("Request cancelled:", ctx.Err())
            return
        case <-time.After(time.Second):
            fmt.Fprintf(w, "Progress: %d%%\n", (i+1)*10)
            if f, ok := w.(http.Flusher); ok {
                f.Flush()
            }
        }
    }
}
```

## Full-Duplex Communication

Enable simultaneous reading and writing (Go 1.21+):

```go
func fullDuplexHandler(w http.ResponseWriter, r *http.Request) {
    rc := http.NewResponseController(w)

    if err := rc.EnableFullDuplex(); err != nil {
        http.Error(w, "Full duplex not supported", http.StatusInternalServerError)
        return
    }

    // Read from request body while writing response
    go func() {
        scanner := bufio.NewScanner(r.Body)
        for scanner.Scan() {
            line := scanner.Text()
            log.Printf("Received: %s", line)
        }
    }()

    // Write response
    for i := 0; i < 10; i++ {
        fmt.Fprintf(w, "Line %d\n", i)
        time.Sleep(time.Second)
    }
}
```

## Connection State Tracking

Monitor connection lifecycle:

```go
var activeConnections int32

func main() {
    server := &http.Server{
        Addr: ":8080",
        ConnState: func(conn net.Conn, state http.ConnState) {
            switch state {
            case http.StateNew:
                atomic.AddInt32(&activeConnections, 1)
                log.Printf("New connection: %s (active: %d)",
                    conn.RemoteAddr(), activeConnections)

            case http.StateClosed:
                atomic.AddInt32(&activeConnections, -1)
                log.Printf("Closed connection: %s (active: %d)",
                    conn.RemoteAddr(), activeConnections)

            case http.StateHijacked:
                atomic.AddInt32(&activeConnections, -1)
                log.Printf("Hijacked connection: %s", conn.RemoteAddr())
            }
        },
    }

    log.Fatal(server.ListenAndServe())
}
```

Connection states:
- `StateNew`: New connection accepted
- `StateActive`: Connection with 1+ bytes read
- `StateIdle`: Waiting for new request (keep-alive)
- `StateHijacked`: Connection hijacked
- `StateClosed`: Connection closed

## Custom Dialer and Transport

Configure low-level connection settings:

```go
import "net"

server := &http.Server{
    Addr: ":8080",
    ConnContext: func(ctx context.Context, c net.Conn) context.Context {
        // Add connection info to context
        return context.WithValue(ctx, "remote_addr", c.RemoteAddr())
    },
}
```

## Response Buffering

Control response buffering:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    rc := http.NewResponseController(w)

    // Disable buffering
    if err := rc.Flush(); err != nil {
        log.Printf("Flush error: %v", err)
    }

    // Write immediately
    fmt.Fprintf(w, "Immediate response\n")
}
```

## Deadlines and Timeouts

Set read/write deadlines:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    rc := http.NewResponseController(w)

    // Set write deadline
    deadline := time.Now().Add(5 * time.Second)
    if err := rc.SetWriteDeadline(deadline); err != nil {
        log.Printf("Deadline error: %v", err)
    }

    // Perform write operation
    data := generateResponse()
    w.Write(data)
}
```

## Trailers

Send headers after the response body (HTTP/1.1+):

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Declare trailers
    w.Header().Set("Trailer", "X-Checksum")

    // Send response body
    data := []byte("Response data")
    w.Write(data)

    // Calculate checksum
    checksum := calculateChecksum(data)

    // Send trailer
    w.Header().Set("Trailer:X-Checksum", checksum)
}
```

## Protocol Switching

Switch to different protocol (e.g., WebSocket):

```go
func upgradeHandler(w http.ResponseWriter, r *http.Request) {
    if r.Header.Get("Upgrade") != "websocket" {
        http.Error(w, "Upgrade required", http.StatusUpgradeRequired)
        return
    }

    hijacker, ok := w.(http.Hijacker)
    if !ok {
        http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
        return
    }

    conn, _, err := hijacker.Hijack()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer conn.Close()

    // Send upgrade response
    fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\n")
    fmt.Fprintf(conn, "Upgrade: websocket\r\n")
    fmt.Fprintf(conn, "Connection: Upgrade\r\n\r\n")

    // Handle WebSocket protocol
    // ...
}
```

## Bandwidth Limiting

Limit response bandwidth:

```go
type rateLimitedReader struct {
    reader  io.Reader
    limiter *rate.Limiter
}

func (r *rateLimitedReader) Read(p []byte) (int, error) {
    n, err := r.reader.Read(p)
    if n > 0 {
        r.limiter.WaitN(context.Background(), n)
    }
    return n, err
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
    file, _ := os.Open("large-file.zip")
    defer file.Close()

    // Limit to 1MB/s
    limiter := rate.NewLimiter(rate.Limit(1<<20), 1<<20)
    limited := &rateLimitedReader{
        reader:  file,
        limiter: limiter,
    }

    w.Header().Set("Content-Type", "application/zip")
    io.Copy(w, limited)
}
```

## Zero-Downtime Restart

Graceful server restart:

```go
import (
    "os"
    "os/signal"
    "syscall"
)

func main() {
    server := &http.Server{Addr: ":8080"}

    // Start server
    go func() {
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    // Wait for signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    <-quit

    log.Println("Shutting down server...")

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatal("Server shutdown error:", err)
    }

    log.Println("Server stopped")
}
```

## Connection Pooling

Configure connection pool (client-side, but useful for proxies):

```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
    DisableKeepAlives:   false,
}

client := &http.Client{
    Transport: transport,
    Timeout:   10 * time.Second,
}
```

## Performance Monitoring

Track handler performance:

```go
func timingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Capture response size
        rec := &responseRecorder{ResponseWriter: w}
        next.ServeHTTP(rec, r)

        duration := time.Since(start)
        log.Printf("%s %s - %d (%s, %d bytes)",
            r.Method, r.URL.Path,
            rec.statusCode,
            duration,
            rec.bytesWritten)
    })
}

type responseRecorder struct {
    http.ResponseWriter
    statusCode   int
    bytesWritten int
}

func (r *responseRecorder) WriteHeader(code int) {
    r.statusCode = code
    r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
    n, err := r.ResponseWriter.Write(b)
    r.bytesWritten += n
    return n, err
}
```

## Related Documentation

- [Server](server.md): Server configuration and lifecycle
- [Handlers](handlers.md): Handler patterns
- [Middleware](middleware.md): Request processing pipelines
- [Response](response.md): Response writing basics
