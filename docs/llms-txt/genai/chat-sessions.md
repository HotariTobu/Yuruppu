# Chat Sessions

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
