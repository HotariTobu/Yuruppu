# Response Structure

## GenerateContentResponse

The main response type for content generation:

```go
type GenerateContentResponse struct {
    Candidates      []*Candidate
    CreateTime      time.Time
    ModelVersion    string
    PromptFeedback  *GenerateContentResponsePromptFeedback
    ResponseID      string
    UsageMetadata   *GenerateContentResponseUsageMetadata
    SDKHTTPResponse *HTTPResponse
}
```

## Extracting Text

### Simple Method

```go
result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
if err != nil {
    log.Fatal(err)
}

text := result.Text()
fmt.Println(text)
```

The `Text()` method concatenates all text parts from the first candidate.

### Manual Extraction

```go
if len(result.Candidates) > 0 {
    candidate := result.Candidates[0]
    for _, part := range candidate.Content.Parts {
        if part.Text != "" {
            fmt.Println(part.Text)
        }
    }
}
```

## Candidate Structure

```go
type Candidate struct {
    Content           *Content
    CitationMetadata  *CitationMetadata
    FinishMessage     string
    TokenCount        int32
    FinishReason      FinishReason
    AvgLogprobs       float64
    GroundingMetadata *GroundingMetadata
    Index             int32
    LogprobsResult    *LogprobsResult
    SafetyRatings     []*SafetyRating
    URLContextMetadata *URLContextMetadata
}
```

### Accessing Candidates

```go
result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
if err != nil {
    log.Fatal(err)
}

for i, candidate := range result.Candidates {
    fmt.Printf("Candidate %d:\n", i+1)
    fmt.Printf("  Text: %s\n", candidate.Content.Parts[0].Text)
    fmt.Printf("  Finish Reason: %v\n", candidate.FinishReason)
    fmt.Printf("  Token Count: %d\n", candidate.TokenCount)
}
```

## Finish Reasons

Indicates why generation stopped:

```go
const (
    FinishReasonUnspecified     FinishReason = "FINISH_REASON_UNSPECIFIED"
    FinishReasonStop            FinishReason = "STOP"            // Natural stop
    FinishReasonMaxTokens       FinishReason = "MAX_TOKENS"      // Hit token limit
    FinishReasonSafety          FinishReason = "SAFETY"          // Safety filter
    FinishReasonRecitation      FinishReason = "RECITATION"      // Recitation filter
    FinishReasonOther           FinishReason = "OTHER"           // Other reason
    FinishReasonBlocklist       FinishReason = "BLOCKLIST"       // Blocked content
    FinishReasonProhibitedContent FinishReason = "PROHIBITED_CONTENT"
    FinishReasonSpii            FinishReason = "SPII"            // Sensitive info
)
```

### Checking Finish Reason

```go
candidate := result.Candidates[0]
switch candidate.FinishReason {
case genai.FinishReasonStop:
    fmt.Println("Completed normally")
case genai.FinishReasonMaxTokens:
    fmt.Println("Truncated - hit max tokens")
case genai.FinishReasonSafety:
    fmt.Println("Blocked by safety filters")
default:
    fmt.Printf("Stopped for reason: %v\n", candidate.FinishReason)
}
```

## Usage Metadata

Token usage information:

```go
type GenerateContentResponseUsageMetadata struct {
    PromptTokenCount        int32
    CachedContentTokenCount int32
    CandidatesTokenCount    int32
    TotalTokenCount         int32
    PromptTokensDetails     []*ModalityTokenCount
    CandidatesTokensDetails []*ModalityTokenCount
    TrafficType             TrafficType
}
```

### Accessing Usage Data

```go
usage := result.UsageMetadata
fmt.Printf("Prompt tokens: %d\n", usage.PromptTokenCount)
fmt.Printf("Response tokens: %d\n", usage.CandidatesTokenCount)
fmt.Printf("Total tokens: %d\n", usage.TotalTokenCount)
fmt.Printf("Cached tokens: %d\n", usage.CachedContentTokenCount)
```

## Content Structure

```go
type Content struct {
    Role  string
    Parts []*Part
}
```

### Accessing Parts

```go
content := result.Candidates[0].Content
for _, part := range content.Parts {
    if part.Text != "" {
        fmt.Println(part.Text)
    }
}
```

## Part Types

```go
type Part struct {
    Text              string
    InlineData        *Blob
    FileData          *FileData
    FunctionCall      *FunctionCall
    FunctionResponse  *FunctionResponse
    ExecutableCode    *ExecutableCode
    CodeExecutionResult *CodeExecutionResult
}
```

## Helper Methods

### Function Calls

```go
functionCalls := result.FunctionCalls()
for _, fc := range functionCalls {
    fmt.Printf("Function: %s\n", fc.Name)
    fmt.Printf("Args: %v\n", fc.Args)
}
```

### Executable Code

```go
code := result.ExecutableCode()
if code != "" {
    fmt.Println("Generated code:", code)
}
```

### Code Execution Result

```go
execResult := result.CodeExecutionResult()
if execResult != "" {
    fmt.Println("Execution result:", execResult)
}
```

## Safety Ratings

```go
type SafetyRating struct {
    Category         HarmCategory
    Probability      HarmProbability
    ProbabilityScore float64
    Severity         HarmSeverity
    SeverityScore    float64
    Blocked          bool
}
```

### Checking Safety

```go
candidate := result.Candidates[0]
for _, rating := range candidate.SafetyRatings {
    fmt.Printf("Category: %v\n", rating.Category)
    fmt.Printf("Probability: %v\n", rating.Probability)
    fmt.Printf("Blocked: %v\n", rating.Blocked)
}
```

## Citation Metadata

```go
type CitationMetadata struct {
    Citations []*Citation
}

type Citation struct {
    StartIndex  int32
    EndIndex    int32
    URI         string
    Title       string
    License     string
    PublicationDate *Date
}
```

### Accessing Citations

```go
if candidate.CitationMetadata != nil {
    for _, citation := range candidate.CitationMetadata.Citations {
        fmt.Printf("Source: %s\n", citation.URI)
        fmt.Printf("Title: %s\n", citation.Title)
    }
}
```

## Prompt Feedback

```go
type GenerateContentResponsePromptFeedback struct {
    BlockReason       BlockReason
    BlockReasonMessage string
    SafetyRatings     []*SafetyRating
}
```

### Checking Prompt Block

```go
if result.PromptFeedback != nil && result.PromptFeedback.BlockReason != "" {
    fmt.Printf("Prompt blocked: %s\n", result.PromptFeedback.BlockReasonMessage)
}
```

## Complete Example

```go
result, err := client.Models.GenerateContent(ctx, model, prompt, nil)
if err != nil {
    log.Fatal(err)
}

// Check if response was generated
if len(result.Candidates) == 0 {
    log.Println("No candidates returned")
    return
}

candidate := result.Candidates[0]

// Check finish reason
if candidate.FinishReason != genai.FinishReasonStop {
    log.Printf("Generation stopped: %v\n", candidate.FinishReason)
}

// Get text
text := result.Text()
fmt.Println("Response:", text)

// Check usage
if result.UsageMetadata != nil {
    fmt.Printf("Tokens used: %d\n", result.UsageMetadata.TotalTokenCount)
}

// Check safety
for _, rating := range candidate.SafetyRatings {
    if rating.Blocked {
        fmt.Printf("Content blocked for: %v\n", rating.Category)
    }
}
```

## Best Practices

1. **Use `result.Text()`**: Simplest way to get response text
2. **Check candidates length**: Ensure response was generated
3. **Verify finish reason**: Detect truncation or blocks
4. **Monitor token usage**: Track costs and limits
5. **Handle safety ratings**: Detect filtered content
6. **Check prompt feedback**: Detect blocked prompts early
7. **Access metadata**: Use for debugging and monitoring
