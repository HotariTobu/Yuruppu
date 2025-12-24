# Chat Sessions

## Overview

The Chats service enables multi-turn conversations with automatic history management. Each chat session maintains conversation context across multiple messages.

## Creating a Chat Session

```go
chat, err := client.Chats.Create(
    ctx,
    "gemini-2.5-flash",
    nil,  // Optional history
    nil,  // Optional config
)
if err != nil {
    log.Fatal(err)
}
```

## Sending Messages

```go
// First message
result, err := chat.SendMessage(ctx, genai.Part{Text: "What is 1 + 1?"})
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Text()) // "2"

// Follow-up message (context maintained)
result, err = chat.SendMessage(ctx, genai.Part{Text: "How about 2 + 2?"})
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Text()) // "4"
```

## With Configuration

```go
config := &genai.GenerateContentConfig{
    Temperature:     genai.Ptr[float32](0.7),
    MaxOutputTokens: 512,
    SystemInstruction: &genai.Content{
        Parts: []*genai.Part{
            {Text: "You are a helpful math tutor."},
        },
    },
}

chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, config)
```

## With Initial History

```go
history := []*genai.Content{
    {
        Role: genai.RoleUser,
        Parts: []*genai.Part{
            {Text: "Hello!"},
        },
    },
    {
        Role: genai.RoleModel,
        Parts: []*genai.Part{
            {Text: "Hi! How can I help you today?"},
        },
    },
}

chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", history, nil)
```

## Streaming Chat

```go
chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)
if err != nil {
    log.Fatal(err)
}

for result, err := range chat.SendMessageStream(ctx, genai.Part{Text: "Tell me a story"}) {
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(result.Text())
}
fmt.Println()
```

## Accessing History

```go
history := chat.History()
for _, content := range history {
    fmt.Printf("%s: ", content.Role)
    for _, part := range content.Parts {
        fmt.Println(part.Text)
    }
}
```

## Complete Example: Interactive Chat

```go
func runChatSession(ctx context.Context, client *genai.Client) error {
    config := &genai.GenerateContentConfig{
        Temperature:     genai.Ptr[float32](0.7),
        MaxOutputTokens: 512,
    }

    chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, config)
    if err != nil {
        return err
    }

    scanner := bufio.NewScanner(os.Stdin)
    fmt.Println("Chat started. Type 'exit' to quit.")

    for {
        fmt.Print("You: ")
        if !scanner.Scan() {
            break
        }

        userInput := scanner.Text()
        if userInput == "exit" {
            break
        }

        result, err := chat.SendMessage(ctx, genai.Part{Text: userInput})
        if err != nil {
            return err
        }

        fmt.Printf("Bot: %s\n\n", result.Text())
    }

    return nil
}
```

## Multimodal Chat

```go
// Send text + image
imageData, _ := os.ReadFile("photo.jpg")

result, err := chat.SendMessage(ctx,
    genai.Part{Text: "What's in this image?"},
    genai.Part{InlineData: &genai.Blob{
        Data:     imageData,
        MIMEType: "image/jpeg",
    }},
)

// Continue conversation
result, err = chat.SendMessage(ctx, genai.Part{Text: "Tell me more about it"})
```

## LINE Bot Pattern (Not Recommended)

Chat sessions are **not recommended** for LINE bots because:
- LINE users expect stateless interactions
- Session management adds complexity
- History grows unbounded
- Increases token costs

**Instead, use single-turn generation:**

```go
func handleLINEMessage(ctx context.Context, client *genai.Client, message string) string {
    result, err := client.Models.GenerateContent(
        ctx,
        "gemini-2.5-flash",
        genai.Text(message),
        nil,
    )
    if err != nil {
        log.Printf("Error: %v", err)
        return "Sorry, I couldn't process that."
    }
    return result.Text()
}
```

**If you need context for LINE bots:**

Store recent messages manually and include in prompt:

```go
type ConversationHistory struct {
    mu       sync.Mutex
    messages map[string][]string // userID -> messages
}

func (h *ConversationHistory) Add(userID, message string) {
    h.mu.Lock()
    defer h.mu.Unlock()

    if len(h.messages[userID]) >= 5 {
        h.messages[userID] = h.messages[userID][1:] // Keep last 5
    }
    h.messages[userID] = append(h.messages[userID], message)
}

func (h *ConversationHistory) GetContext(userID string) string {
    h.mu.Lock()
    defer h.mu.Unlock()

    messages := h.messages[userID]
    if len(messages) == 0 {
        return ""
    }
    return "Recent conversation:\n" + strings.Join(messages, "\n")
}

func handleWithContext(ctx context.Context, client *genai.Client, userID, message string, history *ConversationHistory) string {
    context := history.GetContext(userID)
    fullPrompt := context + "\nUser: " + message

    result, _ := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(fullPrompt), nil)

    history.Add(userID, message)
    history.Add(userID, "Bot: "+result.Text())

    return result.Text()
}
```

## Error Handling

```go
result, err := chat.SendMessage(ctx, genai.Part{Text: "Hello"})
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        return "Request timed out"
    }

    if apiErr, ok := err.(genai.APIError); ok {
        log.Printf("API Error: %s", apiErr.Message)
        return "Sorry, something went wrong"
    }

    return "Error processing message"
}
```

## Managing History Size

```go
// Manually limit history
history := chat.History()
if len(history) > 20 {
    // Keep system message + last 20 messages
    trimmedHistory := history[len(history)-20:]

    // Create new chat with trimmed history
    chat, _ = client.Chats.Create(ctx, "gemini-2.5-flash", trimmedHistory, config)
}
```

## Token Management

```go
// Count tokens in conversation
contents := chat.History()
response, err := client.Models.CountTokens(ctx, "gemini-2.5-flash", contents, nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Conversation tokens: %d\n", response.TotalTokens)

// Reset if too large
if response.TotalTokens > 100000 {
    chat, _ = client.Chats.Create(ctx, "gemini-2.5-flash", nil, config)
}
```

## Best Practices

1. **Use for interactive sessions**: CLI tools, long conversations
2. **Not for LINE bots**: Use single-turn generation instead
3. **Monitor history size**: Trim or reset when too large
4. **Track token usage**: History accumulates costs
5. **Set max output tokens**: Prevent runaway responses
6. **Include system instructions**: Define behavior consistently
7. **Handle errors gracefully**: Provide fallback responses
8. **Consider context limits**: Models have maximum context windows
9. **Reset periodically**: Start fresh when context becomes stale
10. **Use timeout contexts**: Prevent hanging on slow responses

## When to Use Chat vs Single-Turn

**Use Chat for:**
- Interactive CLI applications
- Desktop applications with persistent sessions
- Development and testing
- Applications with explicit "new conversation" actions

**Use Single-Turn for:**
- LINE bots and messaging platforms
- Stateless APIs
- Simple Q&A
- Cost-sensitive applications
- High-volume requests
