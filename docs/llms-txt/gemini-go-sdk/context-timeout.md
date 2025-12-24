# Context and Timeout Handling

## Overview

All SDK methods accept a `context.Context` parameter, enabling proper request lifecycle management, cancellation, and timeout handling.

## Basic Timeout Pattern

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := client.Models.GenerateContent(ctx, model, prompt, config)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Request timed out")
        return
    }
    log.Fatal(err)
}
```

## Context Types

### Background Context

For simple cases without cancellation:

```go
ctx := context.Background()
result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
```

### Timeout Context

Set a deadline for the request:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
```

**Always defer cancel()** to release resources even if the request completes early.

### Cancellable Context

Allow manual cancellation:

```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    time.Sleep(5 * time.Second)
    cancel() // Cancel after 5 seconds
}()

result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
if errors.Is(err, context.Canceled) {
    log.Println("Request was cancelled")
}
```

### Deadline Context

Set an absolute deadline:

```go
deadline := time.Now().Add(1 * time.Minute)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()

result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
```

## Checking Context Errors

```go
result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        log.Println("Request timed out")
    case errors.Is(err, context.Canceled):
        log.Println("Request was cancelled")
    default:
        log.Printf("Request failed: %v", err)
    }
    return
}
```

## LINE Bot Context Pattern

For handling LINE webhook requests with timeout:

```go
func handleMessage(w http.ResponseWriter, r *http.Request, client *genai.Client) {
    // Create timeout context (LINE webhooks expect response within seconds)
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    // Extract user message
    message := extractMessage(r)

    // Generate response with timeout
    result, err := client.Models.GenerateContent(
        ctx,
        "gemini-2.5-flash",
        genai.Text(message),
        nil,
    )

    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            // Respond with fallback message
            respondToLINE(w, "Sorry, I'm taking too long to respond. Please try again.")
            return
        }
        log.Printf("Error: %v", err)
        respondToLINE(w, "Sorry, something went wrong.")
        return
    }

    respondToLINE(w, result.Text())
}
```

## Recommended Timeouts

### Gemini 2.5 Flash
- **Simple requests**: 10-15 seconds
- **Complex requests**: 20-30 seconds

### Gemini 3 Flash (with thinking)
- **Simple requests**: 15-20 seconds
- **Complex requests**: 30-60 seconds

### Gemini 3 Pro
- **Simple requests**: 20-30 seconds
- **Complex requests**: 60-120 seconds

## HTTP Client Timeout vs Context Timeout

You can set timeouts at two levels:

**HTTP Client level (applies to all requests):**
```go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
}

client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:     apiKey,
    HTTPClient: httpClient,
})
```

**Context level (per-request):**
```go
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
```

**Best practice**: Use context timeouts for fine-grained control per request.

## Propagating Context

When calling functions, propagate the context:

```go
func processRequest(ctx context.Context, client *genai.Client, prompt string) (string, error) {
    // Context is propagated to nested calls
    result, err := generateWithRetry(ctx, client, prompt)
    if err != nil {
        return "", err
    }
    return result, nil
}

func generateWithRetry(ctx context.Context, client *genai.Client, prompt string) (string, error) {
    // Check if context is already cancelled
    if ctx.Err() != nil {
        return "", ctx.Err()
    }

    result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(prompt), nil)
    if err != nil {
        return "", err
    }
    return result.Text(), nil
}
```

## Best Practices

1. **Always use context**: Never pass `nil` for context
2. **Set appropriate timeouts**: Based on model and expected response time
3. **Always defer cancel()**: Prevents resource leaks
4. **Check context errors**: Distinguish between timeout, cancellation, and other errors
5. **Propagate context**: Pass context through function calls
6. **Use request context**: For HTTP handlers, derive from `r.Context()`
7. **Don't ignore context.Err()**: Check before making expensive operations
8. **Shorter timeouts for webhooks**: LINE webhooks expect quick responses
