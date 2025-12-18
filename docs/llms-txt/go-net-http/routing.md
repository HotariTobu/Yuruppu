# Routing with ServeMux

The `http.ServeMux` type is Go's built-in HTTP request multiplexer (router) that matches URLs to handlers.

## ServeMux Basics

### Creating a ServeMux

```go
mux := http.NewServeMux()
```

### Default ServeMux

Go provides a default ServeMux accessible via package-level functions:

```go
// Uses DefaultServeMux
http.HandleFunc("/", handler)
http.ListenAndServe(":8080", nil)

// Equivalent to:
mux := http.NewServeMux()
mux.HandleFunc("/", handler)
http.ListenAndServe(":8080", mux)
```

## Registering Routes

### Using HandleFunc

```go
mux := http.NewServeMux()

mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Home")
})

mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "About")
})
```

### Using Handle

```go
type AboutHandler struct{}

func (h *AboutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "About")
}

mux.Handle("/about", &AboutHandler{})
```

## Pattern Matching

### Exact Match

```go
mux.HandleFunc("/hello", handler)
```

Matches only `/hello`, not `/hello/` or `/hello/world`.

### Subtree Match

Patterns ending with `/` match all paths with that prefix:

```go
mux.HandleFunc("/images/", handler)
```

Matches:
- `/images/`
- `/images/photo.jpg`
- `/images/dir/photo.jpg`

Does not match:
- `/images` (no trailing slash)

### Root Pattern

```go
mux.HandleFunc("/", handler)
```

Matches all paths not matched by other patterns (catch-all).

## Pattern Precedence

More specific patterns take precedence:

```go
mux.HandleFunc("/", homeHandler)           // Least specific
mux.HandleFunc("/api/", apiHandler)        // More specific
mux.HandleFunc("/api/users", usersHandler) // Most specific
```

Requests:
- `/` → homeHandler
- `/about` → homeHandler (catch-all)
- `/api/` → apiHandler
- `/api/status` → apiHandler (subtree)
- `/api/users` → usersHandler (exact match)

## Go 1.22+ Enhanced Routing

Go 1.22 introduced enhanced pattern matching with methods and wildcards.

### Method-Based Routing

```go
mux := http.NewServeMux()

mux.HandleFunc("GET /users", listUsers)
mux.HandleFunc("POST /users", createUser)
mux.HandleFunc("GET /users/{id}", getUser)
mux.HandleFunc("PUT /users/{id}", updateUser)
mux.HandleFunc("DELETE /users/{id}", deleteUser)
```

### Path Parameters

Extract variables from URL paths:

```go
mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    fmt.Fprintf(w, "User ID: %s", id)
})

mux.HandleFunc("GET /posts/{year}/{month}/{slug}", func(w http.ResponseWriter, r *http.Request) {
    year := r.PathValue("year")
    month := r.PathValue("month")
    slug := r.PathValue("slug")

    fmt.Fprintf(w, "Post: %s/%s/%s", year, month, slug)
})
```

### Wildcard Patterns

Match remaining path segments:

```go
mux.HandleFunc("/files/{path...}", func(w http.ResponseWriter, r *http.Request) {
    path := r.PathValue("path")
    // path can contain slashes: "dir/subdir/file.txt"
    fmt.Fprintf(w, "File: %s", path)
})
```

## Multiple ServeMux

Use multiple routers for different parts of your application:

```go
// API routes
apiMux := http.NewServeMux()
apiMux.HandleFunc("/users", usersHandler)
apiMux.HandleFunc("/posts", postsHandler)

// Admin routes
adminMux := http.NewServeMux()
adminMux.HandleFunc("/dashboard", dashboardHandler)
adminMux.HandleFunc("/settings", settingsHandler)

// Main router
mainMux := http.NewServeMux()
mainMux.HandleFunc("/", homeHandler)
mainMux.Handle("/api/", http.StripPrefix("/api", apiMux))
mainMux.Handle("/admin/", http.StripPrefix("/admin", adminMux))

http.ListenAndServe(":8080", mainMux)
```

## Host-Based Routing

Route based on Host header (Go 1.22+):

```go
mux := http.NewServeMux()

mux.HandleFunc("example.com/", exampleHandler)
mux.HandleFunc("api.example.com/", apiHandler)
mux.HandleFunc("admin.example.com/", adminHandler)
```

## Cleaning URL Paths

ServeMux automatically cleans paths:

```go
// Redirects:
// /foo/../bar → /bar
// /foo//bar → /foo/bar
// /foo/ with /foo handler → /foo (redirect)
```

To disable redirects, set `Handler` directly without trailing slash matching.

## Not Found Handler

Handle 404 errors:

```go
mux := http.NewServeMux()
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }
    fmt.Fprintf(w, "Home")
})
```

Or use a custom 404 handler:

```go
mux := http.NewServeMux()
mux.HandleFunc("/", homeHandler)
mux.HandleFunc("/about", aboutHandler)

handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if strings.HasPrefix(r.URL.Path, "/api/") {
        mux.ServeHTTP(w, r)
        return
    }

    // Custom 404
    w.WriteHeader(http.StatusNotFound)
    fmt.Fprintf(w, "Page not found: %s", r.URL.Path)
})

http.ListenAndServe(":8080", handler)
```

## RESTful Routing Example

```go
mux := http.NewServeMux()

// Users resource (Go 1.22+)
mux.HandleFunc("GET /users", listUsers)
mux.HandleFunc("POST /users", createUser)
mux.HandleFunc("GET /users/{id}", getUser)
mux.HandleFunc("PUT /users/{id}", updateUser)
mux.HandleFunc("DELETE /users/{id}", deleteUser)

// Posts resource
mux.HandleFunc("GET /posts", listPosts)
mux.HandleFunc("POST /posts", createPost)
mux.HandleFunc("GET /posts/{id}", getPost)
mux.HandleFunc("PUT /posts/{id}", updatePost)
mux.HandleFunc("DELETE /posts/{id}", deletePost)

// Nested resources
mux.HandleFunc("GET /users/{userId}/posts", getUserPosts)
mux.HandleFunc("POST /users/{userId}/posts", createUserPost)

http.ListenAndServe(":8080", mux)
```

Handler implementations:

```go
func listUsers(w http.ResponseWriter, r *http.Request) {
    // GET /users
}

func getUser(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    // GET /users/{id}
}

func getUserPosts(w http.ResponseWriter, r *http.Request) {
    userID := r.PathValue("userId")
    // GET /users/{userId}/posts
}
```

## Pre-1.22 Method Routing

For Go < 1.22, manually check methods:

```go
mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        listUsers(w, r)
    case http.MethodPost:
        createUser(w, r)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
})

mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
    // Extract ID from path
    id := strings.TrimPrefix(r.URL.Path, "/users/")

    switch r.Method {
    case http.MethodGet:
        getUser(w, r, id)
    case http.MethodPut:
        updateUser(w, r, id)
    case http.MethodDelete:
        deleteUser(w, r, id)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
})
```

## Static File Serving

Serve static files from a directory:

```go
mux := http.NewServeMux()

// Serve files from ./static directory
fs := http.FileServer(http.Dir("./static"))
mux.Handle("/static/", http.StripPrefix("/static/", fs))

// Home page
mux.HandleFunc("/", homeHandler)

http.ListenAndServe(":8080", mux)
```

Request `/static/css/style.css` serves `./static/css/style.css`.

## Query Parameters

Query parameters are accessed via `r.URL.Query()`:

```go
mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()

    q := query.Get("q")           // First value
    page := query.Get("page")     // First value
    tags := query["tag"]          // All values

    fmt.Fprintf(w, "Search: %s, Page: %s, Tags: %v", q, page, tags)
})

// Example: /search?q=golang&page=2&tag=web&tag=http
```

## Route Groups with Middleware

Apply middleware to route groups:

```go
func withAuth(handler http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !isAuthenticated(r) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        handler.ServeHTTP(w, r)
    })
}

mux := http.NewServeMux()

// Public routes
mux.HandleFunc("/", homeHandler)
mux.HandleFunc("/login", loginHandler)

// Protected routes
adminMux := http.NewServeMux()
adminMux.HandleFunc("/dashboard", dashboardHandler)
adminMux.HandleFunc("/settings", settingsHandler)

mux.Handle("/admin/", withAuth(http.StripPrefix("/admin", adminMux)))

http.ListenAndServe(":8080", mux)
```

## Debugging Routes

List registered patterns (not available via API, use logging):

```go
mux := http.NewServeMux()

registerRoute := func(pattern string, handler http.HandlerFunc) {
    log.Printf("Route: %s", pattern)
    mux.HandleFunc(pattern, handler)
}

registerRoute("/", homeHandler)
registerRoute("/about", aboutHandler)
registerRoute("/contact", contactHandler)
```

## Testing Routes

```go
import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestRouting(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello"))
    })

    req := httptest.NewRequest("GET", "/hello", nil)
    w := httptest.NewRecorder()

    mux.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", w.Code)
    }

    if w.Body.String() != "Hello" {
        t.Errorf("Expected 'Hello', got %s", w.Body.String())
    }
}
```

## Third-Party Routers

For advanced routing features, consider third-party routers:

- **gorilla/mux**: RESTful routing with regex support
- **chi**: Lightweight router with middleware support
- **httprouter**: Fast HTTP router with zero allocations
- **gin**: Full web framework with routing

Example with gorilla/mux:

```go
import "github.com/gorilla/mux"

r := mux.NewRouter()

// Path parameters
r.HandleFunc("/users/{id:[0-9]+}", getUser).Methods("GET")

// Query parameters
r.HandleFunc("/search", search).Queries("q", "{query}")

// Host routing
r.Host("api.example.com").Handler(apiHandler)

// Subrouters
api := r.PathPrefix("/api").Subrouter()
api.HandleFunc("/users", listUsers).Methods("GET")
```

## Best Practices

1. **Use ServeMux for each logical group**: Separate API, admin, and public routes
2. **Leverage Go 1.22+ features**: Use method and path parameter patterns
3. **Handle 404 explicitly**: Provide custom not found handlers
4. **Strip prefixes carefully**: Use `http.StripPrefix` when delegating to sub-routers
5. **Test routes thoroughly**: Use httptest to verify routing behavior
6. **Consider third-party routers**: For complex routing needs beyond ServeMux

## Related Documentation

- [Handlers](handlers.md): Handler patterns and implementation
- [Middleware](middleware.md): Route-specific middleware
- [Getting Started](getting-started.md): Basic server setup
