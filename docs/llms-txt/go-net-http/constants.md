# HTTP Constants

This document lists common HTTP constants provided by the net/http package.

## HTTP Methods

```go
const (
    MethodGet     = "GET"
    MethodHead    = "HEAD"
    MethodPost    = "POST"
    MethodPut     = "PUT"
    MethodPatch   = "PATCH"
    MethodDelete  = "DELETE"
    MethodConnect = "CONNECT"
    MethodOptions = "OPTIONS"
    MethodTrace   = "TRACE"
)
```

Usage:
```go
if r.Method == http.MethodPost {
    // Handle POST request
}
```

## Status Codes

### 1xx Informational

```go
const (
    StatusContinue           = 100
    StatusSwitchingProtocols = 101
    StatusProcessing         = 102
    StatusEarlyHints         = 103
)
```

### 2xx Success

```go
const (
    StatusOK                   = 200
    StatusCreated              = 201
    StatusAccepted             = 202
    StatusNonAuthoritativeInfo = 203
    StatusNoContent            = 204
    StatusResetContent         = 205
    StatusPartialContent       = 206
    StatusMultiStatus          = 207
    StatusAlreadyReported      = 208
    StatusIMUsed               = 226
)
```

Common usage:
```go
w.WriteHeader(http.StatusOK)                // 200
w.WriteHeader(http.StatusCreated)           // 201
w.WriteHeader(http.StatusNoContent)         // 204
```

### 3xx Redirection

```go
const (
    StatusMultipleChoices   = 300
    StatusMovedPermanently  = 301
    StatusFound             = 302
    StatusSeeOther          = 303
    StatusNotModified       = 304
    StatusUseProxy          = 305
    StatusTemporaryRedirect = 307
    StatusPermanentRedirect = 308
)
```

Common usage:
```go
http.Redirect(w, r, "/new-url", http.StatusMovedPermanently)  // 301
http.Redirect(w, r, "/new-url", http.StatusFound)             // 302
http.Redirect(w, r, "/success", http.StatusSeeOther)          // 303
http.Redirect(w, r, "/new-url", http.StatusTemporaryRedirect) // 307
```

### 4xx Client Errors

```go
const (
    StatusBadRequest                   = 400
    StatusUnauthorized                 = 401
    StatusPaymentRequired              = 402
    StatusForbidden                    = 403
    StatusNotFound                     = 404
    StatusMethodNotAllowed             = 405
    StatusNotAcceptable                = 406
    StatusProxyAuthRequired            = 407
    StatusRequestTimeout               = 408
    StatusConflict                     = 409
    StatusGone                         = 410
    StatusLengthRequired               = 411
    StatusPreconditionFailed           = 412
    StatusRequestEntityTooLarge        = 413
    StatusRequestURITooLong            = 414
    StatusUnsupportedMediaType         = 415
    StatusRequestedRangeNotSatisfiable = 416
    StatusExpectationFailed            = 417
    StatusTeapot                       = 418
    StatusMisdirectedRequest           = 421
    StatusUnprocessableEntity          = 422
    StatusLocked                       = 423
    StatusFailedDependency             = 424
    StatusTooEarly                     = 425
    StatusUpgradeRequired              = 426
    StatusPreconditionRequired         = 428
    StatusTooManyRequests              = 429
    StatusRequestHeaderFieldsTooLarge  = 431
    StatusUnavailableForLegalReasons   = 451
)
```

Common usage:
```go
http.Error(w, "Bad request", http.StatusBadRequest)            // 400
http.Error(w, "Unauthorized", http.StatusUnauthorized)         // 401
http.Error(w, "Forbidden", http.StatusForbidden)               // 403
http.Error(w, "Not found", http.StatusNotFound)                // 404
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed) // 405
http.Error(w, "Too many requests", http.StatusTooManyRequests) // 429
```

### 5xx Server Errors

```go
const (
    StatusInternalServerError           = 500
    StatusNotImplemented                = 501
    StatusBadGateway                    = 502
    StatusServiceUnavailable            = 503
    StatusGatewayTimeout                = 504
    StatusHTTPVersionNotSupported       = 505
    StatusVariantAlsoNegotiates         = 506
    StatusInsufficientStorage           = 507
    StatusLoopDetected                  = 508
    StatusNotExtended                   = 510
    StatusNetworkAuthenticationRequired = 511
)
```

Common usage:
```go
http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
http.Error(w, "Not implemented", http.StatusNotImplemented)             // 501
http.Error(w, "Service unavailable", http.StatusServiceUnavailable)     // 503
```

## Status Text

Get human-readable status text:

```go
text := http.StatusText(200)  // "OK"
text := http.StatusText(404)  // "Not Found"
text := http.StatusText(500)  // "Internal Server Error"
```

Usage in error responses:
```go
code := http.StatusBadRequest
http.Error(w, http.StatusText(code), code)
```

## Common Headers

### Request Headers

```go
// Authentication
r.Header.Get("Authorization")

// Content negotiation
r.Header.Get("Accept")
r.Header.Get("Accept-Language")
r.Header.Get("Accept-Encoding")

// Client info
r.Header.Get("User-Agent")
r.Header.Get("Referer")

// Content info
r.Header.Get("Content-Type")
r.Header.Get("Content-Length")

// Proxy headers
r.Header.Get("X-Forwarded-For")
r.Header.Get("X-Real-IP")

// CORS
r.Header.Get("Origin")

// Cache
r.Header.Get("If-None-Match")
r.Header.Get("If-Modified-Since")
```

### Response Headers

```go
// Content
w.Header().Set("Content-Type", "application/json")
w.Header().Set("Content-Length", "1234")
w.Header().Set("Content-Encoding", "gzip")

// Cache control
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
w.Header().Set("ETag", "\"abc123\"")

// CORS
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

// Security
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Strict-Transport-Security", "max-age=31536000")

// Location (redirects)
w.Header().Set("Location", "https://example.com/new-path")
```

## Content Types

Common MIME types:

```go
// Text
"text/plain"
"text/html"
"text/css"
"text/javascript"
"text/csv"

// Application
"application/json"
"application/xml"
"application/pdf"
"application/zip"
"application/x-www-form-urlencoded"
"application/octet-stream"

// Multipart
"multipart/form-data"
"multipart/byteranges"

// Images
"image/jpeg"
"image/png"
"image/gif"
"image/svg+xml"
"image/webp"

// Video
"video/mp4"
"video/webm"

// Audio
"audio/mpeg"
"audio/wav"
```

Usage:
```go
w.Header().Set("Content-Type", "application/json; charset=utf-8")
w.Header().Set("Content-Type", "text/html; charset=utf-8")
w.Header().Set("Content-Type", "image/png")
```

## HTTP Versions

Protocol versions:

```go
r.Proto       // "HTTP/1.1"
r.ProtoMajor  // 1
r.ProtoMinor  // 1

// Check version
if r.ProtoAtLeast(1, 1) {
    // HTTP/1.1 or later
}

if r.ProtoAtLeast(2, 0) {
    // HTTP/2 or later
}
```

## Time Formats

HTTP date format (RFC1123):

```go
const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// Format time for HTTP headers
t := time.Now()
httpDate := t.UTC().Format(http.TimeFormat)

w.Header().Set("Date", httpDate)
w.Header().Set("Last-Modified", httpDate)
w.Header().Set("Expires", httpDate)

// Parse HTTP date
date := r.Header.Get("If-Modified-Since")
t, err := time.Parse(http.TimeFormat, date)
```

## Default Values

### Default Ports

```go
const (
    DefaultMaxHeaderBytes = 1 << 20 // 1 MB
    DefaultMaxIdleConnsPerHost = 2
)
```

### Default Server

Package-level functions use DefaultServeMux:

```go
http.HandleFunc("/", handler)
// Equivalent to:
http.DefaultServeMux.HandleFunc("/", handler)
```

## Connection States

```go
const (
    StateNew        // New connection
    StateActive     // Connection with 1+ bytes read
    StateIdle       // Connection waiting for request (keep-alive)
    StateHijacked   // Connection hijacked
    StateClosed     // Connection closed
)
```

Usage:
```go
server := &http.Server{
    ConnState: func(conn net.Conn, state http.ConnState) {
        switch state {
        case http.StateNew:
            log.Println("New connection")
        case http.StateClosed:
            log.Println("Connection closed")
        }
    },
}
```

## SameSite Cookie Values

```go
const (
    SameSiteDefaultMode // Browser default
    SameSiteLaxMode     // Lax mode
    SameSiteStrictMode  // Strict mode
    SameSiteNoneMode    // None mode (requires Secure)
)
```

Usage:
```go
cookie := &http.Cookie{
    Name:     "session",
    Value:    "abc123",
    SameSite: http.SameSiteStrictMode,
}
```

## Error Constants

```go
var (
    ErrBodyNotAllowed      // Request method/status doesn't allow body
    ErrHijacked           // Connection has been hijacked
    ErrContentLength      // Content-Length mismatch
    ErrWriteAfterFlush    // Write after Flush
    ErrBodyReadAfterClose // Read after Close
    ErrHandlerTimeout     // Handler timeout
    ErrLineTooLong        // Header line too long
    ErrMissingFile        // No file for FormFile
    ErrNoCookie           // Cookie not found
    ErrNoLocation         // No Location header
    ErrServerClosed       // Server closed
)
```

Usage:
```go
cookie, err := r.Cookie("session")
if err == http.ErrNoCookie {
    // Cookie not found
}

if err := server.Shutdown(ctx); err != http.ErrServerClosed {
    log.Fatal(err)
}
```

## Quick Reference

### Most Common Status Codes

```go
200 OK                      // Success
201 Created                 // Resource created
204 No Content              // Success with no body
301 Moved Permanently       // Permanent redirect
302 Found                   // Temporary redirect
304 Not Modified            // Cached response valid
400 Bad Request             // Invalid client request
401 Unauthorized            // Authentication required
403 Forbidden               // Access denied
404 Not Found               // Resource not found
405 Method Not Allowed      // Wrong HTTP method
429 Too Many Requests       // Rate limit exceeded
500 Internal Server Error   // Server error
503 Service Unavailable     // Server overloaded/down
```

### Most Common Content Types

```go
application/json            // JSON data
text/html                   // HTML pages
text/plain                  // Plain text
application/x-www-form-urlencoded // Form data
multipart/form-data         // File uploads
application/octet-stream    // Binary data
```

## Related Documentation

- [Response](response.md): Using status codes and headers
- [Request](request.md): Reading headers and methods
- [Common Patterns](common-patterns.md): Practical usage examples
