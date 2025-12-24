# Client Type

## Overview

The `Client` type is the main entry point for the SDK, providing access to all services.

## Structure

```go
type Client struct {
    Models           *Models
    Live             *Live
    Caches           *Caches
    Chats            *Chats
    Files            *Files
    Operations       *Operations
    FileSearchStores *FileSearchStores
    Batches          *Batches
    Tunings          *Tunings
    AuthTokens       *Tokens
}
```

## Creating a Client

```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  apiKey,
    Backend: genai.BackendGeminiAPI,
})
if err != nil {
    log.Fatal(err)
}
```

See [Client Initialization](client-initialization.md) for details.

## Models Service

The primary service for text generation:

```go
// Generate content
result, err := client.Models.GenerateContent(
    ctx,
    "gemini-2.5-flash",
    genai.Text("Your prompt"),
    nil,
)

// Stream content
for result, err := range client.Models.GenerateContentStream(
    ctx,
    "gemini-2.5-flash",
    genai.Text("Your prompt"),
    nil,
) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(result.Candidates[0].Content.Parts[0].Text)
}

// Count tokens
response, err := client.Models.CountTokens(
    ctx,
    "gemini-2.5-flash",
    []*genai.Content{...},
    nil,
)
```

## Chats Service

For multi-turn conversations:

```go
// Create chat session
chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)
if err != nil {
    log.Fatal(err)
}

// Send message
result, err := chat.SendMessage(ctx, genai.Part{Text: "Hello"})

// Stream chat response
for result, err := range chat.SendMessageStream(ctx, genai.Part{Text: "Continue"}) {
    if err != nil {
        log.Fatal(err)
    }
    // Process result
}
```

## Files Service

Upload and manage files:

```go
// Upload file
file, err := client.Files.UploadFromPath(
    ctx,
    "/path/to/file.pdf",
    &genai.UploadFileConfig{
        MIMEType: "application/pdf",
    },
)

// List files
page, err := client.Files.List(ctx, nil)
for _, f := range page.Items {
    fmt.Println(f.Name)
}

// Get file
file, err := client.Files.Get(ctx, fileName, nil)

// Delete file
_, err := client.Files.Delete(ctx, fileName, nil)
```

## Caches Service

Context caching for performance:

```go
// Create cache
cache, err := client.Caches.Create(
    ctx,
    "gemini-2.5-flash",
    &genai.CreateCachedContentConfig{
        TTL:      1 * time.Hour,
        Contents: contents,
    },
)

// Use cached content
config := &genai.GenerateContentConfig{
    CachedContent: cache.Name,
}
result, err := client.Models.GenerateContent(ctx, model, contents, config)

// Update cache
updated, err := client.Caches.Update(
    ctx,
    cache.Name,
    &genai.UpdateCachedContentConfig{
        TTL: 2 * time.Hour,
    },
)

// Delete cache
_, err := client.Caches.Delete(ctx, cache.Name, nil)
```

## Lifecycle

### No Explicit Cleanup Required

The SDK handles resource cleanup automatically. You don't need to call a `Close()` method.

### Context Management

Always pass a context for proper lifecycle control:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
```

### Client Reuse

Reuse the client for multiple requests:

```go
// Create once
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey: apiKey,
})
if err != nil {
    log.Fatal(err)
}

// Use many times
for _, prompt := range prompts {
    result, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), nil)
    if err != nil {
        log.Printf("Error: %v", err)
        continue
    }
    fmt.Println(result.Text())
}
```

## Getting Configuration

Retrieve the current client configuration:

```go
config := client.ClientConfig()
fmt.Printf("Backend: %v\n", config.Backend)
fmt.Printf("Project: %s\n", config.Project)
fmt.Printf("Location: %s\n", config.Location)
```

## Best Practices

1. **Create client once**: Reuse across multiple requests
2. **Pass context always**: Enable timeout and cancellation
3. **Use Models service**: For single-turn text generation
4. **Use Chats service**: For multi-turn conversations
5. **Check errors**: Every method can return errors
6. **Don't create per-request**: Client creation is expensive
7. **Store globally**: In a singleton or dependency injection container
8. **No cleanup needed**: SDK handles resource management
