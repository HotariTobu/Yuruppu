# Streaming Responses

## Overview

Streaming enables real-time content generation, displaying partial responses as they're generated instead of waiting for the complete response.

## Basic Streaming Pattern

```go
for result, err := range client.Models.GenerateContentStream(
    ctx,
    "gemini-2.5-flash",
    genai.Text("Write a long story about a robot"),
    nil,
) {
    if err != nil {
        log.Fatal(err)
    }

    // Print text as it arrives
    if len(result.Candidates) > 0 {
        for _, part := range result.Candidates[0].Content.Parts {
            fmt.Print(part.Text)
        }
    }
}
fmt.Println() // Newline after complete response
```

## Method Signature

```go
func (m *Models) GenerateContentStream(
    ctx context.Context,
    model string,
    contents any,
    config *GenerateContentConfig,
) iter.Seq2[*GenerateContentResponse, error]
```

Returns a Go 1.23+ iterator that yields response chunks.

## Complete Example

```go
func streamResponse(ctx context.Context, client *genai.Client, prompt string) error {
    config := &genai.GenerateContentConfig{
        Temperature:     genai.Ptr[float32](0.7),
        MaxOutputTokens: 1024,
    }

    for result, err := range client.Models.GenerateContentStream(
        ctx,
        "gemini-2.5-flash",
        genai.Text(prompt),
        config,
    ) {
        if err != nil {
            return fmt.Errorf("stream error: %w", err)
        }

        // Process each chunk
        if len(result.Candidates) > 0 {
            candidate := result.Candidates[0]
            for _, part := range candidate.Content.Parts {
                fmt.Print(part.Text)
            }
        }
    }

    fmt.Println()
    return nil
}
```

## Error Handling

```go
for result, err := range client.Models.GenerateContentStream(ctx, model, prompt, nil) {
    if err != nil {
        // Handle error
        if errors.Is(err, context.DeadlineExceeded) {
            log.Println("Stream timed out")
            return
        }
        log.Printf("Stream error: %v", err)
        return
    }

    // Process result
    fmt.Print(result.Candidates[0].Content.Parts[0].Text)
}
```

## With Context Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

for result, err := range client.Models.GenerateContentStream(ctx, model, prompt, config) {
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            log.Println("Stream exceeded timeout")
            return
        }
        log.Fatal(err)
    }

    fmt.Print(result.Text())
}
```

## Accumulating Full Response

```go
var fullText strings.Builder

for result, err := range client.Models.GenerateContentStream(ctx, model, prompt, nil) {
    if err != nil {
        return "", err
    }

    text := result.Text()
    fullText.WriteString(text)
    fmt.Print(text) // Also print as streaming
}

completeResponse := fullText.String()
```

## Chat Streaming

```go
chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)
if err != nil {
    log.Fatal(err)
}

// Stream chat response
for result, err := range chat.SendMessageStream(ctx, genai.Part{Text: "Tell me a story"}) {
    if err != nil {
        log.Fatal(err)
    }

    fmt.Print(result.Candidates[0].Content.Parts[0].Text)
}
fmt.Println()
```

## LINE Bot Streaming (Not Recommended)

LINE webhooks don't support streaming responses directly. You must:

1. Use non-streaming API for synchronous replies
2. Or use async processing with LINE Push API

**Synchronous (recommended):**
```go
result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
if err != nil {
    // Handle error
}
replyToLINE(result.Text())
```

**Async pattern (advanced):**
```go
// Respond immediately to webhook
respondOK()

// Stream and accumulate in background
go func() {
    var response strings.Builder
    for result, err := range client.Models.GenerateContentStream(ctx, model, prompt, nil) {
        if err != nil {
            log.Printf("Error: %v", err)
            return
        }
        response.WriteString(result.Text())
    }

    // Send via Push API
    pushToLINE(userID, response.String())
}()
```

## Monitoring Stream Progress

```go
var chunkCount int
var totalTokens int32

for result, err := range client.Models.GenerateContentStream(ctx, model, prompt, nil) {
    if err != nil {
        log.Fatal(err)
    }

    chunkCount++
    if result.UsageMetadata != nil {
        totalTokens = result.UsageMetadata.TotalTokenCount
    }

    fmt.Print(result.Text())
}

log.Printf("Received %d chunks, %d total tokens", chunkCount, totalTokens)
```

## Best Practices

1. **Use for long responses**: Streaming improves perceived performance
2. **Handle errors per chunk**: Don't skip error checking
3. **Set appropriate timeouts**: Longer than non-streaming requests
4. **Not for LINE webhooks**: Use standard generation instead
5. **Accumulate if needed**: Build full response from chunks
6. **Monitor context cancellation**: Check for timeout/cancellation
7. **Print progressively**: Improve UX for terminal applications
8. **Flush output**: Use `os.Stdout.Sync()` for immediate display

## When to Use Streaming

**Use streaming when:**
- Generating long-form content (stories, articles)
- Building interactive CLI applications
- Displaying progress to users in real-time
- Response time is critical for UX

**Don't use streaming when:**
- Working with LINE webhooks (synchronous reply needed)
- Response must be validated before sending
- Processing requires complete response first
- Building simple request-response APIs
