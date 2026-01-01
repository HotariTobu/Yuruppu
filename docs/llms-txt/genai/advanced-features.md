# Advanced Features

## Token Counting

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

## Code Execution Tool

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

## File Upload

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

## JSON Structured Output

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
