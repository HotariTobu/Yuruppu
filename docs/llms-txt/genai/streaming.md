# Streaming Responses

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

## Stream Error Handling

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
