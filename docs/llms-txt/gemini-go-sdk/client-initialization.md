# Client Initialization

## Default Client (Environment Variables)

The simplest way to create a client is to use environment variables:

```go
client, err := genai.NewClient(ctx, nil)
if err != nil {
    log.Fatal(err)
}
```

This reads configuration from:
- `GOOGLE_API_KEY` or `GEMINI_API_KEY` - API key for Gemini API
- `GOOGLE_GENAI_USE_VERTEXAI` - Set to `true` to use Vertex AI
- `GOOGLE_CLOUD_PROJECT` - GCP project ID (Vertex AI)
- `GOOGLE_CLOUD_LOCATION` - GCP region (Vertex AI)

## Gemini API Client

Explicit configuration for Gemini Developer API:

```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  "your-api-key",
    Backend: genai.BackendGeminiAPI,
})
if err != nil {
    log.Fatal(err)
}
```

## Vertex AI Client

Configuration for Google Cloud Vertex AI:

```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    Project:  "your-project-id",
    Location: "us-central1",
    Backend:  genai.BackendVertexAI,
})
if err != nil {
    log.Fatal(err)
}
```

## Custom HTTP Client

For advanced scenarios like custom timeouts or proxy configuration:

```go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
}

client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:     apiKey,
    Backend:    genai.BackendGeminiAPI,
    HTTPClient: httpClient,
})
```

## HTTP Options

Customize API endpoint and headers:

```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  apiKey,
    Backend: genai.BackendGeminiAPI,
    HTTPOptions: genai.HTTPOptions{
        BaseURL:    "https://custom-endpoint.com",
        APIVersion: "v1beta",
        Headers: http.Header{
            "Custom-Header": []string{"value"},
        },
        Timeout: 60 * time.Second,
    },
})
```

## ClientConfig Structure

```go
type ClientConfig struct {
    // API Key for Gemini API
    APIKey string

    // Backend selection
    Backend Backend // BackendGeminiAPI or BackendVertexAI

    // GCP Project ID (Vertex AI only)
    Project string

    // GCP Location/Region (Vertex AI only)
    Location string

    // Google credentials
    Credentials *auth.Credentials

    // Custom HTTP client
    HTTPClient *http.Client

    // HTTP options override
    HTTPOptions HTTPOptions
}
```

## Backend Constants

```go
const (
    BackendGeminiAPI Backend = iota  // Gemini Developer API
    BackendVertexAI                   // Google Cloud Vertex AI
)
```

## Retrieving Configuration

Get the current client configuration:

```go
config := client.ClientConfig()
fmt.Printf("Backend: %v\n", config.Backend)
fmt.Printf("Project: %s\n", config.Project)
```

## Best Practices

1. **Use environment variables** for API keys in production
2. **Never hardcode** API keys in source code
3. **Choose the right backend**: Use Gemini API for simple projects, Vertex AI for enterprise
4. **Set appropriate timeouts** based on expected response times
5. **Reuse clients** instead of creating new ones for each request
