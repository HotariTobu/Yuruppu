# Google GenAI Go SDK

> The new official Google Generative AI SDK for Go (google.golang.org/genai) that supports both Gemini API and Vertex AI. Provides unified access to Google's generative AI models with streaming, chat, and multimodal capabilities.

This SDK replaces the older `cloud.google.com/go/vertexai/genai` package and offers a unified interface for both the Gemini Developer API and Vertex AI.

## Quick Start

### Creating a Client

**Gemini API (with API key):**
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

**Vertex AI (with GCP project):**
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

**Auto-configure from environment:**
```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{})
```

The SDK automatically reads from environment variables to determine backend and credentials.

## Content Generation

### Simple Text Generation

```go
result, err := client.Models.GenerateContent(ctx,
    "gemini-2.5-flash",
    genai.Text("Tell me about New York?"),
    nil)
if err != nil {
    log.Fatal(err)
}

text := result.Candidates[0].Content.Parts[0].Text
```

### With System Prompt

System instructions help steer the model toward specific behavior:

```go
config := &genai.GenerateContentConfig{
    SystemInstruction: &genai.Content{
        Parts: []*genai.Part{{
            Text: "You are a helpful assistant that responds concisely. Always answer in a professional tone.",
        }},
    },
    Temperature:     genai.Ptr[float32](0.7),
    MaxOutputTokens: 1024,
}

result, err := client.Models.GenerateContent(ctx,
    "gemini-2.5-flash",
    genai.Text("What is Go?"),
    config)
```

### Generation Parameters

```go
config := &genai.GenerateContentConfig{
    // System instructions
    SystemInstruction: &genai.Content{
        Parts: []*genai.Part{{Text: "Your system prompt here"}},
    },

    // Sampling parameters
    Temperature:      genai.Ptr[float32](0.7),  // 0.0-2.0, lower = less random
    TopP:             genai.Ptr[float32](0.9),  // Nucleus sampling
    TopK:             genai.Ptr[float32](40.0), // Top-k sampling

    // Output control
    MaxOutputTokens:  1024,
    CandidateCount:   1,
    StopSequences:    []string{"\n\n"},

    // Penalties
    PresencePenalty:  genai.Ptr[float32](0.0),  // Penalize tokens already used
    FrequencyPenalty: genai.Ptr[float32](0.0),  // Penalize repeated tokens

    // Structured output
    ResponseMIMEType: "application/json",
}
```

### Multimodal Input (Text + Image)

```go
parts := []*genai.Part{
    {Text: "What's this image about?"},
    {InlineData: &genai.Blob{
        Data:     imageBytes,
        MIMEType: "image/jpeg",
    }},
}

result, err := client.Models.GenerateContent(ctx,
    "gemini-2.5-flash",
    []*genai.Content{{Parts: parts}},
    nil)
```

### Streaming Responses

For real-time content generation:

```go
for result, err := range client.Models.GenerateContentStream(
    ctx,
    "gemini-2.5-flash",
    genai.Text("Give me top 3 ideas for a blog post."),
    nil,
) {
    if err != nil {
        log.Fatal(err)
    }
    text := result.Candidates[0].Content.Parts[0].Text
    fmt.Print(text)
}
```

## Chat Sessions

Multi-turn conversations:

```go
chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)
if err != nil {
    log.Fatal(err)
}

// First message
result, err := chat.SendMessage(ctx, genai.Part{Text: "What's the weather in New York?"})
if err != nil {
    log.Fatal(err)
}

// Continue conversation
result, err = chat.SendMessage(ctx, genai.Part{Text: "How about San Francisco?"})

// Streaming chat
for result, err := range chat.SendMessageStream(ctx, genai.Part{Text: "Tell me a story"}) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(result.Candidates[0].Content.Parts[0].Text)
}
```

## Model Names

### Gemini API Models
- `gemini-2.5-flash` - Latest fast model
- `gemini-2.5-pro` - Latest advanced model
- `gemini-2.0-flash` - Previous flash model
- `gemini-1.5-pro` - Previous pro model
- `gemini-1.5-flash` - Previous flash model

### Vertex AI Models
Vertex AI uses the same model names as above, plus third-party models:
- `meta/llama-3.2-90b-vision-instruct-maas`

**Format:** Model names are used directly without version prefixes. The SDK handles backend-specific formatting automatically.

For Vertex AI, you can also use fully qualified model names:
- `projects/{project}/locations/{location}/publishers/google/models/{model}`

But the short names (e.g., `gemini-2.5-flash`) are recommended and work across both backends.

## Error Handling

### Basic Error Handling

```go
result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", contents, nil)
if err != nil {
    log.Fatal(err)
}
```

### APIError Type

The SDK returns `APIError` for API-related errors:

```go
type APIError struct {
    Code    int                 // HTTP status code
    Message string              // Error message
    Status  string              // Error status
    Details []map[string]any    // Additional error details
}
```

### Detailed Error Handling

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

### Stream Error Handling

```go
for result, err := range client.Models.GenerateContentStream(ctx, model, contents, nil) {
    if err != nil {
        // Handle error in stream
        log.Printf("Stream error: %v", err)
        break
    }
    // Process result
}
```

### Common Error Scenarios

1. **Authentication errors** (401) - Invalid API key or credentials
2. **Rate limiting** (429) - Too many requests, implement backoff
3. **Invalid request** (400) - Check input format and parameters
4. **Model not found** (404) - Verify model name and availability
5. **Service errors** (500+) - Retry with exponential backoff

## Client Structure

The `Client` provides access to various services:

```go
type Client struct {
    Models           *Models            // Content generation
    Live             *Live              // Real-time WebSocket
    Caches           *Caches            // Model response caching
    Chats            *Chats             // Chat session utilities
    Files            *Files             // File upload/management
    Operations       *Operations        // Long-running operations
    FileSearchStores *FileSearchStores  // RAG file stores
    Batches          *Batches           // Batch processing
    Tunings          *Tunings           // Model fine-tuning
    AuthTokens       *Tokens            // Authentication tokens
}
```

## Advanced Features

### Token Counting

```go
response, err := client.Models.CountTokens(ctx,
    "gemini-2.5-flash",
    []*genai.Content{{Parts: []*genai.Part{{Text: "Your text"}}}},
    nil)
if err != nil {
    log.Fatal(err)
}

totalTokens := response.TotalTokens
```

### Code Execution Tool

```go
config := &genai.GenerateContentConfig{
    Tools: []*genai.Tool{
        {CodeExecution: &genai.ToolCodeExecution{}},
    },
}

result, err := client.Models.GenerateContent(ctx,
    "gemini-2.5-flash",
    genai.Text("Calculate the sum of 1 to 100"),
    config)
```

### File Upload

```go
file, err := client.Files.UploadFromPath(ctx, "path/to/file.pdf",
    &genai.UploadFileConfig{MIMEType: "application/pdf"})
if err != nil {
    log.Fatal(err)
}

// Use in content
parts := []*genai.Part{
    {Text: "What's in this document?"},
    {FileData: &genai.FileData{
        FileURI:  file.URI,
        MIMEType: "application/pdf",
    }},
}
```

### JSON Structured Output

```go
config := &genai.GenerateContentConfig{
    ResponseMIMEType: "application/json",
    ResponseSchema: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "name": {Type: genai.TypeString},
            "age":  {Type: genai.TypeInteger},
        },
        Required: []string{"name", "age"},
    },
}

result, err := client.Models.GenerateContent(ctx,
    "gemini-2.5-flash",
    genai.Text("Generate a person"),
    config)
```

## Helper Functions

```go
// Create pointer for optional fields
genai.Ptr[float32](0.7)
genai.Ptr[int32](42)

// Quick text content creation
genai.Text("Your prompt") // Returns []*genai.Content

// Create Parts
genai.NewPartFromText("text")
genai.NewPartFromBytes(data, "image/jpeg")
genai.NewPartFromURI("gs://bucket/file.pdf", "application/pdf")
```

## Backend Differences

### Gemini API
- Requires API key authentication
- Public access, simpler setup
- Model names: `gemini-2.5-flash`, etc.
- Environment: `GOOGLE_API_KEY`

### Vertex AI
- Requires GCP project and location
- Uses Application Default Credentials or service account
- Supports same model names plus third-party models
- Additional enterprise features (VPC-SC, CMEK)
- Environment: `GOOGLE_CLOUD_PROJECT`, `GOOGLE_CLOUD_LOCATION`

The SDK automatically handles API differences based on the backend configuration.

## Best Practices

1. **Use environment variables** for configuration in production
2. **Enable streaming** for longer responses to improve UX
3. **Implement retry logic** with exponential backoff for transient errors
4. **Count tokens** before requests to avoid exceeding limits
5. **Use system instructions** to improve response quality and consistency
6. **Cache responses** with `client.Caches` for repeated queries
7. **Handle errors gracefully** with proper error type checking
8. **Set appropriate timeouts** in context for long-running operations

## Migration from Old SDK

If migrating from `cloud.google.com/go/vertexai/genai`:

1. Update import: `google.golang.org/genai`
2. Use `genai.NewClient()` instead of `vertexai.NewClient()`
3. Specify backend: `Backend: genai.BackendVertexAI`
4. Model names remain the same
5. Most API surface is similar, but check updated method signatures

## Package Documentation

- Official docs: https://pkg.go.dev/google.golang.org/genai
- GitHub: https://github.com/googleapis/google-cloud-go/tree/main/genai
- Google AI Studio: https://aistudio.google.com/
- Vertex AI Console: https://console.cloud.google.com/vertex-ai

## Optional

- [Live API](https://pkg.go.dev/google.golang.org/genai#Live): Real-time WebSocket connections for bidirectional streaming
- [Caches API](https://pkg.go.dev/google.golang.org/genai#Caches): Cache model responses to reduce latency and costs
- [File Search Stores](https://pkg.go.dev/google.golang.org/genai#FileSearchStores): Retrieval-augmented generation with document stores
- [Batches API](https://pkg.go.dev/google.golang.org/genai#Batches): Process multiple requests in batch jobs
- [Tunings API](https://pkg.go.dev/google.golang.org/genai#Tunings): Fine-tune models with custom datasets
- [Operations API](https://pkg.go.dev/google.golang.org/genai#Operations): Monitor long-running operations
- [Safety Settings](https://pkg.go.dev/google.golang.org/genai#SafetySetting): Configure content filtering and safety thresholds
- [Function Calling](https://pkg.go.dev/google.golang.org/genai#Tool): Define custom functions for the model to call
