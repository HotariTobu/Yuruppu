# Content Generation

## Simple Text Generation

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

## With System Prompt

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

## Generation Parameters

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

## Multimodal Input (Text + Image)

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
