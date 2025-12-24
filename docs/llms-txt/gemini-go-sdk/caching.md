# Context Caching

## Overview

Context caching allows you to cache frequently used content (like system instructions or large documents) to improve performance and reduce costs for repeated requests.

## Creating a Cache

```go
contents := []*genai.Content{
    {
        Parts: []*genai.Part{
            {Text: "You are Yuruppu, a friendly LINE bot character..."},
        },
    },
}

cache, err := client.Caches.Create(
    ctx,
    "gemini-2.5-flash",
    &genai.CreateCachedContentConfig{
        TTL:      1 * time.Hour,
        Contents: contents,
    },
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Cache created: %s\n", cache.Name)
```

## Using Cached Content

```go
config := &genai.GenerateContentConfig{
    CachedContent: cache.Name,
}

result, err := client.Models.GenerateContent(
    ctx,
    "gemini-2.5-flash",
    genai.Text("What is your name?"),
    config,
)
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Text())
```

## Configuration Options

### Time-to-Live (TTL)

```go
cache, err := client.Caches.Create(ctx, model, &genai.CreateCachedContentConfig{
    TTL:      2 * time.Hour,
    Contents: contents,
})
```

### Expiration Time

```go
expirationTime := time.Now().Add(24 * time.Hour)

cache, err := client.Caches.Create(ctx, model, &genai.CreateCachedContentConfig{
    ExpireTime: &expirationTime,
    Contents:   contents,
})
```

### Display Name

```go
cache, err := client.Caches.Create(ctx, model, &genai.CreateCachedContentConfig{
    DisplayName: "Yuruppu Character Instructions",
    TTL:         1 * time.Hour,
    Contents:    contents,
})
```

## Updating Cache

### Extend TTL

```go
updated, err := client.Caches.Update(
    ctx,
    cache.Name,
    &genai.UpdateCachedContentConfig{
        TTL: 3 * time.Hour,
    },
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Cache extended until: %v\n", updated.ExpireTime)
```

### Update Expiration Time

```go
newExpiration := time.Now().Add(6 * time.Hour)

updated, err := client.Caches.Update(
    ctx,
    cache.Name,
    &genai.UpdateCachedContentConfig{
        ExpireTime: &newExpiration,
    },
)
```

## Listing Caches

```go
page, err := client.Caches.List(ctx, &genai.ListCachedContentsConfig{
    PageSize: 10,
})
if err != nil {
    log.Fatal(err)
}

for _, cache := range page.Items {
    fmt.Printf("Name: %s\n", cache.Name)
    fmt.Printf("Display Name: %s\n", cache.DisplayName)
    fmt.Printf("Expires: %v\n", cache.ExpireTime)
    fmt.Printf("Model: %s\n", cache.Model)
    fmt.Println("---")
}
```

## Getting Cache Details

```go
cache, err := client.Caches.Get(ctx, cacheName, nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Cache: %s\n", cache.Name)
fmt.Printf("Created: %v\n", cache.CreateTime)
fmt.Printf("Expires: %v\n", cache.ExpireTime)
fmt.Printf("Usage: %d tokens\n", cache.UsageMetadata.TotalTokenCount)
```

## Deleting Cache

```go
_, err := client.Caches.Delete(ctx, cache.Name, nil)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Cache deleted")
```

## Complete Example: LINE Bot with Caching

```go
var characterCache *genai.CachedContent

func initializeCache(ctx context.Context, client *genai.Client) error {
    systemPrompt := `You are Yuruppu, a friendly and helpful LINE bot character.

    Personality:
    - Warm and approachable
    - Concise and clear
    - Helpful and supportive

    Response guidelines:
    - Keep responses under 200 characters
    - Use friendly tone
    - Be helpful but brief
    `

    contents := []*genai.Content{
        {
            Parts: []*genai.Part{
                {Text: systemPrompt},
            },
        },
    }

    cache, err := client.Caches.Create(ctx, "gemini-2.5-flash", &genai.CreateCachedContentConfig{
        DisplayName: "Yuruppu Character",
        TTL:         24 * time.Hour,
        Contents:    contents,
    })
    if err != nil {
        return err
    }

    characterCache = cache
    return nil
}

func handleMessage(ctx context.Context, client *genai.Client, message string) (string, error) {
    config := &genai.GenerateContentConfig{
        CachedContent:   characterCache.Name,
        MaxOutputTokens: 200,
    }

    result, err := client.Models.GenerateContent(
        ctx,
        "gemini-2.5-flash",
        genai.Text(message),
        config,
    )
    if err != nil {
        return "", err
    }

    return result.Text(), nil
}

func refreshCache(ctx context.Context, client *genai.Client) error {
    // Extend cache lifetime
    _, err := client.Caches.Update(ctx, characterCache.Name, &genai.UpdateCachedContentConfig{
        TTL: 24 * time.Hour,
    })
    return err
}
```

## When to Use Caching

**Use caching for:**
- System instructions used in every request
- Large documents referenced repeatedly
- Character personality definitions
- Fixed context shared across conversations
- Frequently used prompts

**Don't use caching for:**
- Dynamic content that changes frequently
- One-time requests
- User-specific information
- Small prompts (< 32K tokens)

## Cache Limits

### Token Requirements
- Minimum cached tokens: 32,768 (32K)
- Maximum cached tokens: Depends on model

### TTL Limits
- Minimum TTL: 5 minutes
- Maximum TTL: 24 hours (Gemini API), 7 days (Vertex AI)

## Cost Savings

**Without cache:**
- Every request pays for full system instruction tokens

**With cache:**
- First request: Normal cost
- Subsequent requests: Reduced cost for cached tokens

**Example:**
- System prompt: 10K tokens
- User message: 100 tokens
- Without cache: 10,100 tokens per request
- With cache: ~100 tokens per request (90% savings)

## Cache Lifecycle Management

```go
// Create cache at startup
cache, _ := createCharacterCache(ctx, client)

// Use throughout application lifetime
for _, message := range messages {
    result, _ := generateWithCache(ctx, client, cache.Name, message)
    fmt.Println(result)
}

// Refresh periodically
ticker := time.NewTicker(12 * time.Hour)
go func() {
    for range ticker.C {
        client.Caches.Update(ctx, cache.Name, &genai.UpdateCachedContentConfig{
            TTL: 24 * time.Hour,
        })
    }
}()

// Clean up on shutdown
defer client.Caches.Delete(ctx, cache.Name, nil)
```

## Best Practices

1. **Cache large, static content**: System instructions, character definitions
2. **Set appropriate TTL**: Balance cost vs refresh frequency
3. **Monitor expiration**: Refresh before cache expires
4. **Use display names**: For easier identification
5. **Clean up unused caches**: Delete when no longer needed
6. **Minimum 32K tokens**: Ensure cached content meets minimum
7. **Reuse across requests**: Maximize cache benefits
8. **Update instead of recreate**: Extend TTL rather than creating new cache
