# HTTP Request Handling

The `http.Request` type represents an HTTP request and provides methods to access all request data.

## Request Type

```go
type Request struct {
    Method     string      // HTTP method (GET, POST, etc.)
    URL        *url.URL    // Request URL
    Proto      string      // Protocol version (e.g., "HTTP/1.1")
    Header     Header      // Request headers
    Body       io.ReadCloser // Request body
    Host       string      // Host from request or Host header
    RemoteAddr string      // Client IP address
    RequestURI string      // Unmodified request URI
    TLS        *tls.ConnectionState // TLS connection state (nil for HTTP)
    // Additional fields...
}
```

## Accessing Request Data

### HTTP Method

```go
func handler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        // Handle GET
    case http.MethodPost:
        // Handle POST
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}
```

### URL and Path

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Full URL (relative for server requests)
    url := r.URL.String() // e.g., "/path?query=value"

    // Path only
    path := r.URL.Path // e.g., "/path"

    // Query parameters
    query := r.URL.Query()
    value := query.Get("key") // First value for key
    values := query["key"]    // All values for key
}
```

### Headers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Get single header value
    contentType := r.Header.Get("Content-Type")

    // Get all values for a header
    accepts := r.Header.Values("Accept")

    // Check if header exists
    if r.Header.Get("Authorization") == "" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Common header helpers
    userAgent := r.UserAgent()
    referer := r.Referer()
}
```

### Host Information

```go
func handler(w http.ResponseWriter, r *http.Request) {
    host := r.Host         // "example.com:8080"
    remote := r.RemoteAddr // "192.168.1.1:12345"
}
```

## Reading Request Body

### Read All Body

```go
func handler(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read body", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    // Use body...
}
```

### Decode JSON Body

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    // Use user...
}
```

### Limit Body Size

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Limit to 1MB
    r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
        return
    }
    defer r.Body.Close()
}
```

## Form Data

### URL-Encoded Forms

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Parse form data (query + body)
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Form parsing error", http.StatusBadRequest)
        return
    }

    // Get single value (query or POST body)
    name := r.FormValue("name")

    // Get POST value only (ignores query string)
    email := r.PostFormValue("email")

    // Get all values for a key
    tags := r.Form["tags"]

    // Check if form key exists
    if r.Form.Has("submit") {
        // Form was submitted
    }
}
```

Note: `FormValue` and `PostFormValue` call `ParseForm` automatically.

### Multipart Forms (File Uploads)

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form (max 10MB in memory)
    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "Upload error", http.StatusBadRequest)
        return
    }

    // Get uploaded file
    file, header, err := r.FormFile("upload")
    if err != nil {
        http.Error(w, "File not found", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // File info
    filename := header.Filename
    size := header.Size

    // Save file
    dst, err := os.Create("./uploads/" + filename)
    if err != nil {
        http.Error(w, "Cannot save file", http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    if _, err := io.Copy(dst, file); err != nil {
        http.Error(w, "Cannot save file", http.StatusInternalServerError)
        return
    }

    fmt.Fprintf(w, "Uploaded: %s (%d bytes)", filename, size)
}
```

### Multiple File Upload

```go
func handler(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseMultipartForm(32 << 20); err != nil {
        http.Error(w, "Upload error", http.StatusBadRequest)
        return
    }

    files := r.MultipartForm.File["uploads"]
    for _, fileHeader := range files {
        file, err := fileHeader.Open()
        if err != nil {
            continue
        }
        defer file.Close()

        // Process each file...
        fmt.Printf("File: %s (%d bytes)\n", fileHeader.Filename, fileHeader.Size)
    }
}
```

## Cookies

### Read Cookies

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Get single cookie
    cookie, err := r.Cookie("session")
    if err != nil {
        if err == http.ErrNoCookie {
            // Cookie not found
        }
        return
    }

    sessionID := cookie.Value

    // Get all cookies
    cookies := r.Cookies()
    for _, c := range cookies {
        fmt.Printf("%s = %s\n", c.Name, c.Value)
    }
}
```

### Set Cookies

```go
func handler(w http.ResponseWriter, r *http.Request) {
    cookie := &http.Cookie{
        Name:     "session",
        Value:    "abc123",
        Path:     "/",
        MaxAge:   3600,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    }

    http.SetCookie(w, cookie)
}
```

## Context

Every request has an associated context for cancellation and request-scoped values.

### Get Request Context

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Context is cancelled when:
    // - Client disconnects
    // - Request times out
    // - Handler returns

    select {
    case <-time.After(5 * time.Second):
        // Long operation
    case <-ctx.Done():
        // Request was cancelled
        return
    }
}
```

### Store Values in Context

```go
type contextKey string

const userKey contextKey = "user"

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := authenticateUser(r)

        // Add user to context
        ctx := context.WithValue(r.Context(), userKey, user)
        r = r.WithContext(ctx)

        next.ServeHTTP(w, r)
    })
}

func handler(w http.ResponseWriter, r *http.Request) {
    // Get user from context
    user, ok := r.Context().Value(userKey).(*User)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    fmt.Fprintf(w, "Hello, %s", user.Name)
}
```

### Context with Timeout

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    // Use ctx for operations that should timeout
    result, err := fetchDataWithContext(ctx)
    if err != nil {
        if err == context.DeadlineExceeded {
            http.Error(w, "Request timeout", http.StatusRequestTimeout)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    fmt.Fprintf(w, "Result: %v", result)
}
```

## Authentication

### Basic Authentication

```go
func handler(w http.ResponseWriter, r *http.Request) {
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

    // Authenticated
    fmt.Fprintf(w, "Welcome, %s", username)
}
```

### Bearer Token

```go
func handler(w http.ResponseWriter, r *http.Request) {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Extract token
    parts := strings.Split(authHeader, " ")
    if len(parts) != 2 || parts[0] != "Bearer" {
        http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
        return
    }

    token := parts[1]
    if !validateToken(token) {
        http.Error(w, "Invalid token", http.StatusUnauthorized)
        return
    }

    // Authenticated
}
```

## TLS Information

```go
func handler(w http.ResponseWriter, r *http.Request) {
    if r.TLS != nil {
        // HTTPS request
        version := r.TLS.Version
        cipher := r.TLS.CipherSuite

        // Client certificates (if mutual TLS)
        if len(r.TLS.PeerCertificates) > 0 {
            clientCert := r.TLS.PeerCertificates[0]
            fmt.Printf("Client: %s\n", clientCert.Subject)
        }
    } else {
        // HTTP request
    }
}
```

## Request Cloning

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Clone request with new context
    ctx := context.WithValue(r.Context(), "key", "value")
    r2 := r.Clone(ctx)

    // r2 is a deep copy with new context
    // Body is shared and will panic if read from both
}
```

## Complete Example: Form Submission

```go
func formHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        // Display form
        fmt.Fprintf(w, `
            <form method="POST">
                <input name="username" required>
                <input name="email" type="email" required>
                <input type="file" name="avatar">
                <button type="submit">Submit</button>
            </form>
        `)
        return
    }

    if r.Method == http.MethodPost {
        // Parse multipart form
        if err := r.ParseMultipartForm(10 << 20); err != nil {
            http.Error(w, "Form error", http.StatusBadRequest)
            return
        }

        // Get form values
        username := r.FormValue("username")
        email := r.FormValue("email")

        // Get uploaded file (optional)
        file, header, err := r.FormFile("avatar")
        if err == nil {
            defer file.Close()
            fmt.Printf("Uploaded: %s\n", header.Filename)
        }

        // Process form data
        fmt.Fprintf(w, "User: %s, Email: %s", username, email)
        return
    }

    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
```

## Related Documentation

- [Response](response.md): Writing HTTP responses
- [Handlers](handlers.md): Handler patterns
- [Middleware](middleware.md): Request processing pipelines
