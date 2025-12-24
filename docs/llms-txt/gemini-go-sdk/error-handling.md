# Error Handling

## Basic Error Handling

All SDK methods return an error as the second value:

```go
result, err := client.Models.GenerateContent(ctx, model, prompt, config)
if err != nil {
    log.Fatal(err)
}
```

## APIError Type

The SDK returns `genai.APIError` for API-level errors:

```go
type APIError struct {
    Code    int                // HTTP status code
    Message string             // Error message from server
    Status  string             // Status string (e.g., "INVALID_ARGUMENT")
    Details []map[string]any   // Additional error details
}

func (e APIError) Error() string
```

## Detecting API Errors

```go
result, err := client.Models.GenerateContent(ctx, model, prompt, config)
if err != nil {
    if apiErr, ok := err.(genai.APIError); ok {
        log.Printf("API Error %d: %s", apiErr.Code, apiErr.Message)
        log.Printf("Status: %s", apiErr.Status)
    } else {
        log.Printf("Error: %v", err)
    }
}
```

## Common Error Codes

### 400 - Invalid Argument
Request parameters are invalid:

```go
if apiErr.Code == 400 {
    log.Println("Invalid request parameters")
    // Check prompt, model name, or configuration
}
```

### 401 - Unauthorized
API key is missing or invalid:

```go
if apiErr.Code == 401 {
    log.Println("Invalid or missing API key")
    // Check GOOGLE_API_KEY environment variable
}
```

### 403 - Permission Denied
API key doesn't have access:

```go
if apiErr.Code == 403 {
    log.Println("API key does not have permission")
    // Verify API key permissions in Google Cloud Console
}
```

### 404 - Not Found
Model or resource doesn't exist:

```go
if apiErr.Code == 404 {
    log.Println("Model not found")
    // Check model name spelling
}
```

### 429 - Rate Limit Exceeded
Too many requests:

```go
if apiErr.Code == 429 {
    log.Println("Rate limit exceeded")
    // Implement exponential backoff
}
```

### 500 - Internal Server Error
Server-side error:

```go
if apiErr.Code == 500 {
    log.Println("Server error, retry later")
    // Implement retry logic
}
```

### 503 - Service Unavailable
Service is temporarily unavailable:

```go
if apiErr.Code == 503 {
    log.Println("Service unavailable")
    // Implement retry with backoff
}
```

## Context Errors

Check for context-related errors separately:

```go
result, err := client.Models.GenerateContent(ctx, model, prompt, config)
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        log.Println("Request timed out")
        return
    case errors.Is(err, context.Canceled):
        log.Println("Request was cancelled")
        return
    default:
        // Handle other errors
    }
}
```

## Complete Error Handling Pattern

```go
func generateWithErrorHandling(ctx context.Context, client *genai.Client, prompt string) (string, error) {
    result, err := client.Models.GenerateContent(
        ctx,
        "gemini-2.5-flash",
        genai.Text(prompt),
        nil,
    )

    if err != nil {
        // Check for context errors first
        if errors.Is(err, context.DeadlineExceeded) {
            return "", fmt.Errorf("request timed out: %w", err)
        }
        if errors.Is(err, context.Canceled) {
            return "", fmt.Errorf("request cancelled: %w", err)
        }

        // Check for API errors
        if apiErr, ok := err.(genai.APIError); ok {
            switch apiErr.Code {
            case 400:
                return "", fmt.Errorf("invalid request: %s", apiErr.Message)
            case 401:
                return "", fmt.Errorf("authentication failed: %s", apiErr.Message)
            case 429:
                return "", fmt.Errorf("rate limit exceeded: %s", apiErr.Message)
            case 500, 503:
                return "", fmt.Errorf("server error (retry later): %s", apiErr.Message)
            default:
                return "", fmt.Errorf("API error %d: %s", apiErr.Code, apiErr.Message)
            }
        }

        // Unknown error
        return "", fmt.Errorf("unexpected error: %w", err)
    }

    return result.Text(), nil
}
```

## Retry Logic with Exponential Backoff

```go
func generateWithRetry(ctx context.Context, client *genai.Client, prompt string) (string, error) {
    maxRetries := 3
    baseDelay := 1 * time.Second

    for attempt := 0; attempt < maxRetries; attempt++ {
        result, err := client.Models.GenerateContent(
            ctx,
            "gemini-2.5-flash",
            genai.Text(prompt),
            nil,
        )

        if err == nil {
            return result.Text(), nil
        }

        // Check if error is retryable
        if apiErr, ok := err.(genai.APIError); ok {
            if apiErr.Code == 429 || apiErr.Code >= 500 {
                // Retryable error
                if attempt < maxRetries-1 {
                    delay := baseDelay * time.Duration(1<<uint(attempt))
                    log.Printf("Retry %d/%d after %v: %s", attempt+1, maxRetries, delay, apiErr.Message)

                    select {
                    case <-time.After(delay):
                        continue
                    case <-ctx.Done():
                        return "", ctx.Err()
                    }
                }
            }
        }

        // Non-retryable error
        return "", err
    }

    return "", fmt.Errorf("max retries exceeded")
}
```

## LINE Bot Error Handling

For LINE bot responses, always provide user-friendly error messages:

```go
func handleLINEMessage(ctx context.Context, client *genai.Client, message string) string {
    result, err := client.Models.GenerateContent(
        ctx,
        "gemini-2.5-flash",
        genai.Text(message),
        nil,
    )

    if err != nil {
        // Log the actual error
        log.Printf("Error generating response: %v", err)

        // Return user-friendly message
        if errors.Is(err, context.DeadlineExceeded) {
            return "Sorry, I'm taking too long to respond. Please try again."
        }

        if apiErr, ok := err.(genai.APIError); ok {
            if apiErr.Code == 429 {
                return "I'm a bit busy right now. Please try again in a moment."
            }
            if apiErr.Code >= 500 {
                return "I'm having technical difficulties. Please try again later."
            }
        }

        return "Sorry, I couldn't process your message. Please try again."
    }

    return result.Text()
}
```

## Validation Before API Call

Validate inputs to prevent unnecessary API calls:

```go
func validatePrompt(prompt string) error {
    if len(strings.TrimSpace(prompt)) == 0 {
        return fmt.Errorf("prompt cannot be empty")
    }
    if len(prompt) > 1000000 {
        return fmt.Errorf("prompt too long (max 1M characters)")
    }
    return nil
}

func generateSafe(ctx context.Context, client *genai.Client, prompt string) (string, error) {
    if err := validatePrompt(prompt); err != nil {
        return "", err
    }

    result, err := client.Models.GenerateContent(
        ctx,
        "gemini-2.5-flash",
        genai.Text(prompt),
        nil,
    )
    if err != nil {
        return "", err
    }

    return result.Text(), nil
}
```

## Best Practices

1. **Check errors immediately**: Don't ignore the error return value
2. **Distinguish error types**: Handle context errors, API errors, and other errors differently
3. **Provide user-friendly messages**: Don't expose technical errors to end users
4. **Implement retries**: For rate limits (429) and server errors (500, 503)
5. **Use exponential backoff**: Don't retry immediately
6. **Log errors properly**: Include context for debugging
7. **Validate inputs**: Check before making API calls
8. **Set timeouts**: Prevent hanging requests
9. **Wrap errors**: Use `fmt.Errorf("context: %w", err)` for error chains
10. **Handle context errors first**: Check for timeout/cancellation before API errors
