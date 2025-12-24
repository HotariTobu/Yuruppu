# Configuration

## GenerateContentConfig

The main configuration type for text generation:

```go
type GenerateContentConfig struct {
    Temperature       *float32
    TopP              *float32
    TopK              *float32
    MaxOutputTokens   int32
    StopSequences     []string
    CandidateCount    int32
    Seed              *int32
    ResponseMIMEType  string
    SystemInstruction *Content
    Tools             []*Tool
    ToolConfig        *ToolConfig
    SafetySettings    []*SafetySetting
}
```

## Temperature

Controls randomness of output (0.0 - 2.0):

```go
config := &genai.GenerateContentConfig{
    Temperature: genai.Ptr[float32](0.7),
}
```

**Values:**
- `0.0` - Deterministic, focused output
- `0.7` - Balanced creativity and coherence (recommended)
- `1.0` - Default (use for Gemini 3 models)
- `2.0` - Maximum randomness

**Important**: For Gemini 3 models, keep temperature at 1.0. Lower values may cause looping or degraded performance.

## Max Output Tokens

Limit the length of generated responses:

```go
config := &genai.GenerateContentConfig{
    MaxOutputTokens: 512,  // ~400 words
}
```

**Common values:**
- `256` - Short responses (1-2 paragraphs)
- `512` - Medium responses (2-3 paragraphs)
- `1024` - Long responses (multiple paragraphs)
- `2048` - Very long responses

## Top-P (Nucleus Sampling)

Controls diversity via cumulative probability (0.0 - 1.0):

```go
config := &genai.GenerateContentConfig{
    TopP: genai.Ptr[float32](0.9),
}
```

**Values:**
- `0.1` - Very focused
- `0.9` - Recommended for most cases
- `1.0` - Consider all possibilities

## Top-K Sampling

Limits vocabulary to top K tokens:

```go
config := &genai.GenerateContentConfig{
    TopK: genai.Ptr[float32](40),
}
```

**Common values:**
- `1` - Greedy decoding (most likely token)
- `40` - Balanced (default)
- `80` - More diverse

## Stop Sequences

Stop generation when patterns appear:

```go
config := &genai.GenerateContentConfig{
    StopSequences: []string{"\n\n", "END", "###"},
}
```

Useful for:
- Preventing run-on responses
- Structured output format
- Chat turn boundaries

## System Instructions

Set the assistant's behavior or role:

```go
config := &genai.GenerateContentConfig{
    SystemInstruction: &genai.Content{
        Parts: []*genai.Part{
            {Text: "You are Yuruppu, a friendly and helpful character. Respond warmly and concisely."},
        },
    },
}
```

**Best practices:**
- Be specific about tone and style
- Define response format
- Set boundaries (what to avoid)
- Keep it concise

## Response MIME Type

Request structured output:

```go
config := &genai.GenerateContentConfig{
    ResponseMIMEType: "application/json",
}
```

**Supported types:**
- `"text/plain"` - Default text response
- `"application/json"` - JSON output

**JSON example:**
```go
config := &genai.GenerateContentConfig{
    ResponseMIMEType: "application/json",
    SystemInstruction: &genai.Content{
        Parts: []*genai.Part{
            {Text: `Return JSON with fields: "summary", "sentiment", "topics"`},
        },
    },
}
```

## Seed (Reproducible Output)

Generate consistent results:

```go
config := &genai.GenerateContentConfig{
    Seed: genai.Ptr[int32](42),
}
```

Useful for:
- Testing
- Debugging
- Reproducible experiments

**Note**: Same seed + same input = same output (mostly)

## Candidate Count

Request multiple response variations:

```go
config := &genai.GenerateContentConfig{
    CandidateCount: 3,
}

result, err := client.Models.GenerateContent(ctx, model, prompt, config)
if err != nil {
    log.Fatal(err)
}

// Access all candidates
for i, candidate := range result.Candidates {
    fmt.Printf("Candidate %d: %s\n", i+1, candidate.Content.Parts[0].Text)
}
```

**Warning**: Multiple candidates increase token usage.

## Safety Settings

Control content filtering (optional):

```go
config := &genai.GenerateContentConfig{
    SafetySettings: []*genai.SafetySetting{
        {
            Category:  genai.HarmCategoryHateSpeech,
            Threshold: genai.HarmBlockMediumAndAbove,
        },
    },
}
```

**Categories:**
- `HarmCategoryHateSpeech`
- `HarmCategoryDangerousContent`
- `HarmCategorySexuallyExplicit`
- `HarmCategoryHarassment`

**Thresholds:**
- `HarmBlockNone`
- `HarmBlockLowAndAbove`
- `HarmBlockMediumAndAbove` (default)
- `HarmBlockOnlyHigh`

## Complete Configuration Example

```go
config := &genai.GenerateContentConfig{
    // Core parameters
    Temperature:     genai.Ptr[float32](0.7),
    MaxOutputTokens: 512,
    TopP:            genai.Ptr[float32](0.9),
    TopK:            genai.Ptr[float32](40),

    // Behavior
    SystemInstruction: &genai.Content{
        Parts: []*genai.Part{
            {Text: "You are Yuruppu, a friendly helper. Be concise."},
        },
    },

    // Output control
    StopSequences:    []string{"\n\n"},
    ResponseMIMEType: "text/plain",

    // Reproducibility
    Seed: genai.Ptr[int32](42),
}

result, err := client.Models.GenerateContent(
    ctx,
    "gemini-2.5-flash",
    genai.Text("What is photosynthesis?"),
    config,
)
```

## Helper Function: Ptr

Use `genai.Ptr()` to create pointers for optional fields:

```go
temp := genai.Ptr[float32](0.7)
topP := genai.Ptr[float32](0.9)
seed := genai.Ptr[int32](42)

config := &genai.GenerateContentConfig{
    Temperature: temp,
    TopP:        topP,
    Seed:        seed,
}
```

## Preset Configurations

Create reusable configuration presets:

```go
// Concise responses
var ConciseConfig = &genai.GenerateContentConfig{
    Temperature:     genai.Ptr[float32](0.5),
    MaxOutputTokens: 256,
    StopSequences:   []string{"\n\n"},
}

// Creative responses
var CreativeConfig = &genai.GenerateContentConfig{
    Temperature:     genai.Ptr[float32](1.2),
    MaxOutputTokens: 1024,
    TopP:            genai.Ptr[float32](0.95),
}

// Deterministic responses
var DeterministicConfig = &genai.GenerateContentConfig{
    Temperature:     genai.Ptr[float32](0.0),
    MaxOutputTokens: 512,
    Seed:            genai.Ptr[int32](42),
}

// Usage
result, err := client.Models.GenerateContent(ctx, model, prompt, ConciseConfig)
```

## Best Practices

1. **Start with defaults**: Use `nil` config first, then tune as needed
2. **Temperature 1.0 for Gemini 3**: Don't lower it
3. **Set MaxOutputTokens**: Prevent unexpectedly long responses
4. **Use system instructions**: Consistent behavior across requests
5. **Reuse configurations**: Create preset configs for common patterns
6. **Test with seeds**: Use seed for reproducible testing
7. **Monitor token usage**: Higher values = more cost
8. **StopSequences for structure**: Define clear boundaries
9. **JSON for structured data**: Use ResponseMIMEType when needed
10. **Don't over-configure**: Only set parameters you need
