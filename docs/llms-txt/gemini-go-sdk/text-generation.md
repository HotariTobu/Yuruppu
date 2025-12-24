# Text Generation

## Basic Pattern

The core method for single-turn text generation:

```go
result, err := client.Models.GenerateContent(
    ctx,
    "gemini-2.5-flash",
    genai.Text("Your prompt here"),
    nil,
)
if err != nil {
    log.Fatal(err)
}

text := result.Text()
fmt.Println(text)
```

## Method Signature

```go
func (m *Models) GenerateContent(
    ctx context.Context,
    model string,
    contents any,
    config *GenerateContentConfig,
) (*GenerateContentResponse, error)
```

## Parameters

### Model Selection

Common models for text generation:

```go
"gemini-2.5-flash"      // Best price-performance, fast responses
"gemini-2.5-flash-lite" // Ultra-fast, most cost-efficient
"gemini-3-flash"        // Better quality, multimodal support
"gemini-3-pro"          // Flagship model, best understanding
"gemini-2.5-pro"        // Advanced reasoning for complex tasks
```

### Content Input

**Simple text:**
```go
genai.Text("Explain how photosynthesis works")
```

**Custom content:**
```go
contents := []*genai.Content{
    {
        Role: genai.RoleUser,
        Parts: []*genai.Part{
            {Text: "What is the capital of France?"},
        },
    },
}
```

### Configuration

Pass `nil` for defaults, or customize:

```go
config := &genai.GenerateContentConfig{
    Temperature:     genai.Ptr[float32](0.7),
    MaxOutputTokens: 512,
    TopP:            genai.Ptr[float32](0.9),
    TopK:            genai.Ptr[float32](40),
}

result, err := client.Models.GenerateContent(ctx, model, prompt, config)
```

## Extracting Response Text

### Simple Text Extraction

```go
text := result.Text()
```

This concatenates all text parts from the first candidate.

### Accessing Full Response

```go
if len(result.Candidates) > 0 {
    candidate := result.Candidates[0]
    for _, part := range candidate.Content.Parts {
        fmt.Println(part.Text)
    }
}
```

## Configuration Options

### Temperature

Controls randomness (0.0 = deterministic, 2.0 = very random):

```go
Temperature: genai.Ptr[float32](0.7)
```

**Important**: For Gemini 3 models, keep temperature at 1.0 (default). Setting it below 1.0 may cause looping or degraded performance.

### Max Output Tokens

Limit response length:

```go
MaxOutputTokens: 512
```

### Sampling Parameters

```go
TopP: genai.Ptr[float32](0.9)  // Nucleus sampling
TopK: genai.Ptr[float32](40)   // Top-k sampling
```

### Stop Sequences

Stop generation when certain patterns appear:

```go
StopSequences: []string{"\n\n", "END"}
```

### System Instructions

Set behavior or role:

```go
SystemInstruction: &genai.Content{
    Parts: []*genai.Part{
        {Text: "You are a helpful assistant that responds concisely."},
    },
}
```

### Response Format

Request JSON output:

```go
ResponseMIMEType: "application/json"
```

### Seed (Deterministic Output)

For reproducible results:

```go
Seed: genai.Ptr[int32](42)
```

## Complete Example

```go
func generateResponse(ctx context.Context, client *genai.Client, prompt string) (string, error) {
    config := &genai.GenerateContentConfig{
        Temperature:     genai.Ptr[float32](0.7),
        MaxOutputTokens: 256,
        SystemInstruction: &genai.Content{
            Parts: []*genai.Part{
                {Text: "You are a concise assistant."},
            },
        },
    }

    result, err := client.Models.GenerateContent(
        ctx,
        "gemini-2.5-flash",
        genai.Text(prompt),
        config,
    )
    if err != nil {
        return "", err
    }

    return result.Text(), nil
}
```

## Best Practices

1. **Use appropriate models**: `gemini-2.5-flash` for most cases, `gemini-3-flash` for better quality
2. **Keep temperature at 1.0** for Gemini 3 models
3. **Set MaxOutputTokens** to prevent excessive responses
4. **Use system instructions** to guide behavior consistently
5. **Extract text with `result.Text()`** for simple use cases
6. **Check for errors** before accessing response data
7. **Reuse the client** across multiple requests
