# Quick Start

## Installation

Install the SDK using Go modules:

```bash
go get google.golang.org/genai
```

## Environment Setup

Set your API key as an environment variable:

```bash
export GOOGLE_API_KEY='your-api-key'
# or
export GEMINI_API_KEY='your-api-key'
```

## Basic Usage

Here's a minimal example for text generation:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "google.golang.org/genai"
)

func main() {
    ctx := context.Background()

    // Create client (reads API key from environment)
    client, err := genai.NewClient(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Generate content
    result, err := client.Models.GenerateContent(
        ctx,
        "gemini-2.5-flash",
        genai.Text("Explain how AI works in a few words"),
        nil,
    )
    if err != nil {
        log.Fatal(err)
    }

    // Print response
    fmt.Println(result.Text())
}
```

## Key Points

- The client automatically reads the API key from `GOOGLE_API_KEY` or `GEMINI_API_KEY` environment variables
- Use `gemini-2.5-flash` for cost-effective, fast responses
- Use `gemini-3-flash` for better quality with multimodal support
- The `Text()` method extracts the generated text from the response
- Always pass a `context.Context` for proper lifecycle management

## Next Steps

- Learn about [Client Initialization](client-initialization.md) options
- Understand [Text Generation](text-generation.md) patterns
- Implement [Error Handling](error-handling.md) properly
