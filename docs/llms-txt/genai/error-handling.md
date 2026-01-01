# Error Handling

## Basic Error Handling

```go
result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", contents, nil)
if err != nil {
    log.Fatal(err)
}
```

## APIError Type

The SDK returns `APIError` for API-related errors:

```go
type APIError struct {
    Code    int                 // HTTP status code
    Message string              // Error message
    Status  string              // Error status
    Details []map[string]any    // Additional error details
}
```

## Detailed Error Handling

```go
result, err := client.Models.GenerateContent(ctx, model, contents, config)
if err != nil {
    // Type assert to get detailed error information
    if apiErr, ok := err.(genai.APIError); ok {
        log.Printf("API Error: Code=%d, Message=%s, Status=%s",
            apiErr.Code, apiErr.Message, apiErr.Status)

        // Check specific error codes
        if apiErr.Code == 429 {
            // Rate limit exceeded
            log.Println("Rate limit exceeded, retry later")
        } else if apiErr.Code == 400 {
            // Invalid request
            log.Println("Invalid request:", apiErr.Message)
        }
    } else {
        // Other error types
        log.Fatal(err)
    }
    return
}
```

## Common Error Scenarios

1. **Authentication errors** (401) - Invalid API key or credentials
2. **Rate limiting** (429) - Too many requests, implement backoff
3. **Invalid request** (400) - Check input format and parameters
4. **Model not found** (404) - Verify model name and availability
5. **Service errors** (500+) - Retry with exponential backoff
