# HTTP Client

The `http.Client` type provides HTTP client functionality with connection pooling, redirects, and cookie handling.

## Client Type

```go
type Client struct {
    Transport     RoundTripper
    CheckRedirect func(req *Request, via []*Request) error
    Jar           CookieJar
    Timeout       time.Duration
}
```

Key characteristics:
- Zero value (`DefaultClient`) is usable with `DefaultTransport`
- Safe for concurrent use by multiple goroutines
- Should be reused instead of created per-request (maintains cached TCP connections)
- Handles cookies and redirects automatically

## Basic Usage

### Using DefaultClient

```go
// Simple GET
resp, err := http.Get("https://example.com")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
```

### Custom Client

```go
client := &http.Client{
    Timeout: 30 * time.Second,
}

resp, err := client.Get("https://example.com")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
```

## Client Methods

### Get

```go
resp, err := client.Get("https://example.com")
```

Issues GET request, follows redirects (301, 302, 303, 307, 308).

### Post

```go
resp, err := client.Post(
    "https://example.com/upload",
    "application/json",
    bytes.NewBuffer(jsonData),
)
```

### PostForm

```go
resp, err := client.PostForm("https://example.com/form", url.Values{
    "username": {"alice"},
    "password": {"secret"},
})
```

Automatically sets Content-Type to `application/x-www-form-urlencoded`.

### Head

```go
resp, err := client.Head("https://example.com")
```

Issues HEAD request (no response body).

### Do

```go
req, err := http.NewRequest("PUT", "https://example.com/resource", body)
if err != nil {
    log.Fatal(err)
}
req.Header.Set("Authorization", "Bearer token")

resp, err := client.Do(req)
```

Sends custom request with full control over method, headers, and body.

## Request Construction

### NewRequest

```go
req, err := http.NewRequest("GET", "https://example.com", nil)
if err != nil {
    log.Fatal(err)
}
req.Header.Add("Accept", "application/json")
req.Header.Add("User-Agent", "MyApp/1.0")

resp, err := client.Do(req)
```

### NewRequestWithContext

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
if err != nil {
    log.Fatal(err)
}

resp, err := client.Do(req)
```

Use context for cancellation and per-request timeouts.

## Timeout Configuration

### Client Timeout

```go
client := &http.Client{
    Timeout: 30 * time.Second,
}
```

Includes connection time, redirects, and response body reading.

### Context Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := client.Do(req)
```

Per-request timeout using context.

### Transport Timeouts

```go
transport := &http.Transport{
    DialContext: (&net.Dialer{
        Timeout:   5 * time.Second,
        KeepAlive: 30 * time.Second,
    }).DialContext,
    TLSHandshakeTimeout:   5 * time.Second,
    ResponseHeaderTimeout: 10 * time.Second,
    IdleConnTimeout:       90 * time.Second,
}

client := &http.Client{Transport: transport}
```

Fine-grained timeout control at transport level.

## Transport and Connection Pooling

### Custom Transport

```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
    DisableCompression:  false,
}

client := &http.Client{Transport: transport}
```

### Key Transport Fields

```go
type Transport struct {
    MaxIdleConns        int           // Max idle connections (all hosts)
    MaxIdleConnsPerHost int           // Max idle connections per host (default: 2)
    MaxConnsPerHost     int           // Max total connections per host
    IdleConnTimeout     time.Duration // How long idle connections stay open
    DisableCompression  bool          // Disable gzip compression
    DisableKeepAlives   bool          // Disable HTTP keep-alives
}
```

### Close Idle Connections

```go
client.CloseIdleConnections()
```

Closes any idle keep-alive connections.

## Redirect Handling

### Default Behavior

By default, Client follows up to 10 consecutive redirects.

### Custom Redirect Policy

```go
client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        if len(via) >= 5 {
            return errors.New("stopped after 5 redirects")
        }
        return nil
    },
}
```

### Disable Redirects

```go
client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse
    },
}
```

Returns the redirect response without following it.

## Cookie Handling

### Using CookieJar

```go
import "net/http/cookiejar"

jar, err := cookiejar.New(nil)
if err != nil {
    log.Fatal(err)
}

client := &http.Client{
    Jar: jar,
}
```

Cookies are automatically stored and sent on subsequent requests.

### Manual Cookie Setting

```go
req, _ := http.NewRequest("GET", "https://example.com", nil)
req.AddCookie(&http.Cookie{
    Name:  "session",
    Value: "abc123",
})
resp, err := client.Do(req)
```

## Error Handling

### Check Error Type

```go
resp, err := client.Get(url)
if err != nil {
    if urlErr, ok := err.(*url.Error); ok {
        if urlErr.Timeout() {
            log.Println("Request timed out")
        }
        log.Printf("URL Error: %v", urlErr)
    }
    return
}
```

### Response Status Check

```go
resp, err := client.Get(url)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
    log.Printf("Unexpected status: %s", resp.Status)
}
```

Note: Non-2xx status codes do NOT cause errors.

## Best Practices

### Reuse Client

```go
// Good: single client for all requests
var client = &http.Client{Timeout: 30 * time.Second}

func makeRequest() {
    resp, err := client.Get(url)
    // ...
}
```

### Always Close Response Body

```go
resp, err := client.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()  // Always close!

// Read body...
```

### Drain Body Before Closing

```go
resp, err := client.Get(url)
if err != nil {
    return err
}
defer func() {
    io.Copy(io.Discard, resp.Body)  // Drain remaining data
    resp.Body.Close()
}()
```

Ensures connection can be reused for keep-alive.

### Use Context for Cancellation

```go
func fetchWithCancel(ctx context.Context, url string) ([]byte, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    return io.ReadAll(resp.Body)
}
```

## Complete Example

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type APIClient struct {
    client  *http.Client
    baseURL string
}

func NewAPIClient(baseURL string) *APIClient {
    return &APIClient{
        client: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        10,
                MaxIdleConnsPerHost: 5,
                IdleConnTimeout:     90 * time.Second,
            },
        },
        baseURL: baseURL,
    }
}

func (c *APIClient) Get(ctx context.Context, path string, result interface{}) error {
    req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Accept", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
    }

    return json.NewDecoder(resp.Body).Decode(result)
}

func main() {
    client := NewAPIClient("https://api.example.com")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var data map[string]interface{}
    if err := client.Get(ctx, "/users/1", &data); err != nil {
        fmt.Println("Error:", err)
        return
    }

    fmt.Printf("Data: %+v\n", data)
}
```
