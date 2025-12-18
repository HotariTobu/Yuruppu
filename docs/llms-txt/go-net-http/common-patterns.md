# Common Patterns

This document covers common HTTP server patterns and best practices.

## File Serving

### Serve Static Files

```go
// Serve files from ./static directory
fs := http.FileServer(http.Dir("./static"))
http.Handle("/static/", http.StripPrefix("/static/", fs))
```

Request `/static/css/style.css` serves `./static/css/style.css`.

### Serve Single Directory

```go
http.Handle("/", http.FileServer(http.Dir("./public")))
```

### Custom File Server

```go
func fileHandler(w http.ResponseWriter, r *http.Request) {
    // Security: prevent directory traversal
    path := filepath.Clean(r.URL.Path)
    if strings.Contains(path, "..") {
        http.Error(w, "Invalid path", http.StatusBadRequest)
        return
    }

    fullPath := filepath.Join("./uploads", path)

    // Check if file exists
    info, err := os.Stat(fullPath)
    if err != nil {
        http.NotFound(w, r)
        return
    }

    if info.IsDir() {
        http.Error(w, "Directory listing denied", http.StatusForbidden)
        return
    }

    http.ServeFile(w, r, fullPath)
}
```

### Force Download

```go
func downloadHandler(w http.ResponseWriter, r *http.Request) {
    filename := "document.pdf"
    filepath := "./files/" + filename

    w.Header().Set("Content-Disposition", "attachment; filename="+filename)
    w.Header().Set("Content-Type", "application/pdf")

    http.ServeFile(w, r, filepath)
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

### HTTPS Redirect

```go
func httpsRedirect(w http.ResponseWriter, r *http.Request) {
    target := "https://" + r.Host + r.URL.Path
    if r.URL.RawQuery != "" {
        target += "?" + r.URL.RawQuery
    }
    http.Redirect(w, r, target, http.StatusMovedPermanently)
}

// Usage
go http.ListenAndServe(":80", http.HandlerFunc(httpsRedirect))
http.ListenAndServeTLS(":443", "cert.pem", "key.pem", mainHandler)
```

### Trailing Slash Redirect

```go
func handler(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/users" {
        http.Redirect(w, r, "/users/", http.StatusMovedPermanently)
        return
    }
    // Handle request
}
```

## Error Handling

### Standard Error Response

```go
func handler(w http.ResponseWriter, r *http.Request) {
    if err := someOperation(); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}
```

### JSON Error Response

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message"`
    Code    int    `json:"code"`
}

func sendJSONError(w http.ResponseWriter, message string, code int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)

    json.NewEncoder(w).Encode(ErrorResponse{
        Error:   http.StatusText(code),
        Message: message,
        Code:    code,
    })
}

// Usage
func handler(w http.ResponseWriter, r *http.Request) {
    if err := validate(r); err != nil {
        sendJSONError(w, "Validation failed", http.StatusBadRequest)
        return
    }
}
```

### Custom Error Page

```go
func errorPage(w http.ResponseWriter, code int, message string) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(code)

    tmpl := `
    <!DOCTYPE html>
    <html>
    <head><title>Error {{.Code}}</title></head>
    <body>
        <h1>{{.Title}}</h1>
        <p>{{.Message}}</p>
    </body>
    </html>
    `

    data := struct {
        Code    int
        Title   string
        Message string
    }{
        Code:    code,
        Title:   http.StatusText(code),
        Message: message,
    }

    t := template.Must(template.New("error").Parse(tmpl))
    t.Execute(w, data)
}
```

## JSON API

### JSON Request/Response

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type CreateUserResponse struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // Validate
    if req.Name == "" || req.Email == "" {
        sendJSONError(w, "Missing required fields", http.StatusBadRequest)
        return
    }

    // Create user
    user := createUser(req.Name, req.Email)

    // Send response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(CreateUserResponse{
        ID:    user.ID,
        Name:  user.Name,
        Email: user.Email,
    })
}
```

### JSON Helper Functions

```go
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func readJSON(r *http.Request, v interface{}) error {
    return json.NewDecoder(r.Body).Decode(v)
}

// Usage
func handler(w http.ResponseWriter, r *http.Request) {
    var req RequestType
    if err := readJSON(r, &req); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
        return
    }

    writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
```

## Form Handling

### HTML Form

```go
func showFormHandler(w http.ResponseWriter, r *http.Request) {
    html := `
    <!DOCTYPE html>
    <html>
    <body>
        <form method="POST" action="/submit">
            <input type="text" name="name" placeholder="Name" required>
            <input type="email" name="email" placeholder="Email" required>
            <button type="submit">Submit</button>
        </form>
    </body>
    </html>
    `
    w.Header().Set("Content-Type", "text/html")
    fmt.Fprint(w, html)
}

func submitFormHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if err := r.ParseForm(); err != nil {
        http.Error(w, "Form error", http.StatusBadRequest)
        return
    }

    name := r.FormValue("name")
    email := r.FormValue("email")

    // Process form data
    fmt.Fprintf(w, "Received: %s (%s)", name, email)
}
```

### File Upload

```go
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Parse multipart form (10MB max)
    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "Upload error", http.StatusBadRequest)
        return
    }

    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "File not found", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Validate file type
    contentType := header.Header.Get("Content-Type")
    if !strings.HasPrefix(contentType, "image/") {
        http.Error(w, "Only images allowed", http.StatusBadRequest)
        return
    }

    // Save file
    dst, err := os.Create("./uploads/" + header.Filename)
    if err != nil {
        http.Error(w, "Cannot save file", http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    if _, err := io.Copy(dst, file); err != nil {
        http.Error(w, "Cannot save file", http.StatusInternalServerError)
        return
    }

    fmt.Fprintf(w, "Uploaded: %s (%d bytes)", header.Filename, header.Size)
}
```

## Authentication

### Session-Based Auth

```go
var sessions = make(map[string]string) // sessionID -> username
var mu sync.RWMutex

func loginHandler(w http.ResponseWriter, r *http.Request) {
    username := r.FormValue("username")
    password := r.FormValue("password")

    if !validateCredentials(username, password) {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    // Create session
    sessionID := generateSessionID()
    mu.Lock()
    sessions[sessionID] = username
    mu.Unlock()

    // Set cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "session",
        Value:    sessionID,
        Path:     "/",
        MaxAge:   3600,
        HttpOnly: true,
        Secure:   true,
    })

    http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cookie, err := r.Cookie("session")
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        mu.RLock()
        username, ok := sessions[cookie.Value]
        mu.RUnlock()

        if !ok {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        ctx := context.WithValue(r.Context(), "username", username)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie("session")
    if err == nil {
        mu.Lock()
        delete(sessions, cookie.Value)
        mu.Unlock()
    }

    // Delete cookie
    http.SetCookie(w, &http.Cookie{
        Name:   "session",
        Value:  "",
        Path:   "/",
        MaxAge: -1,
    })

    http.Redirect(w, r, "/login", http.StatusSeeOther)
}
```

## Templating

### HTML Template

```go
import "html/template"

func handler(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.New("page").Parse(`
        <!DOCTYPE html>
        <html>
        <head><title>{{.Title}}</title></head>
        <body>
            <h1>{{.Title}}</h1>
            <p>{{.Message}}</p>
            <ul>
            {{range .Items}}
                <li>{{.}}</li>
            {{end}}
            </ul>
        </body>
        </html>
    `))

    data := struct {
        Title   string
        Message string
        Items   []string
    }{
        Title:   "My Page",
        Message: "Welcome!",
        Items:   []string{"Item 1", "Item 2", "Item 3"},
    }

    w.Header().Set("Content-Type", "text/html")
    tmpl.Execute(w, data)
}
```

### Template from File

```go
func handler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFiles("templates/page.html")
    if err != nil {
        http.Error(w, "Template error", http.StatusInternalServerError)
        return
    }

    data := PageData{
        Title: "My Page",
        User:  getCurrentUser(r),
    }

    w.Header().Set("Content-Type", "text/html")
    if err := tmpl.Execute(w, data); err != nil {
        log.Printf("Template execution error: %v", err)
    }
}
```

## Health Checks

### Simple Health Check

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprint(w, "OK")
}

// Register
http.HandleFunc("/health", healthHandler)
```

### Detailed Health Check

```go
type HealthStatus struct {
    Status   string            `json:"status"`
    Checks   map[string]string `json:"checks"`
    Uptime   string            `json:"uptime"`
    Version  string            `json:"version"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    status := HealthStatus{
        Status:  "healthy",
        Checks:  make(map[string]string),
        Uptime:  time.Since(startTime).String(),
        Version: "1.0.0",
    }

    // Check database
    if err := db.Ping(); err != nil {
        status.Status = "unhealthy"
        status.Checks["database"] = "down"
    } else {
        status.Checks["database"] = "up"
    }

    // Check cache
    if err := cache.Ping(); err != nil {
        status.Checks["cache"] = "down"
    } else {
        status.Checks["cache"] = "up"
    }

    code := http.StatusOK
    if status.Status == "unhealthy" {
        code = http.StatusServiceUnavailable
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(status)
}
```

## Pagination

### Query-Based Pagination

```go
func listHandler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()

    page, _ := strconv.Atoi(query.Get("page"))
    if page < 1 {
        page = 1
    }

    perPage, _ := strconv.Atoi(query.Get("per_page"))
    if perPage < 1 || perPage > 100 {
        perPage = 20
    }

    offset := (page - 1) * perPage
    items := getItems(offset, perPage)
    total := getTotalItems()

    response := struct {
        Items   []Item `json:"items"`
        Page    int    `json:"page"`
        PerPage int    `json:"per_page"`
        Total   int    `json:"total"`
    }{
        Items:   items,
        Page:    page,
        PerPage: perPage,
        Total:   total,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

## CORS

### Simple CORS

```go
func corsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }

    // Handle actual request
}
```

### Configurable CORS Middleware

```go
type CORSConfig struct {
    AllowOrigins []string
    AllowMethods []string
    AllowHeaders []string
}

func CORSMiddleware(config CORSConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")

            // Check if origin is allowed
            allowed := false
            for _, o := range config.AllowOrigins {
                if o == "*" || o == origin {
                    allowed = true
                    break
                }
            }

            if allowed {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
                w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
            }

            if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusOK)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

## Related Documentation

- [Handlers](handlers.md): Handler patterns
- [Middleware](middleware.md): Request processing
- [Request](request.md): Reading request data
- [Response](response.md): Writing responses
