# Quick Start

## Creating a Client

### Gemini API (with API key)

```go
import "google.golang.org/genai"

client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  apiKey,
    Backend: genai.BackendGeminiAPI,
})
if err != nil {
    log.Fatal(err)
}
```

Environment variable setup:
```bash
export GOOGLE_API_KEY='your-api-key'
```

### Vertex AI (with GCP project)

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

Environment variable setup:
```bash
export GOOGLE_GENAI_USE_VERTEXAI=true
export GOOGLE_CLOUD_PROJECT='your-project-id'
export GOOGLE_CLOUD_LOCATION='us-central1'
```

### Auto-configure from environment

```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{})
```

The SDK automatically reads from environment variables to determine backend and credentials.
