# Available Models

## Overview

Google offers multiple Gemini models optimized for different use cases. Choose based on your requirements for speed, quality, cost, and capabilities.

## Current Generation Models

### Gemini 2.5 Flash

**Model ID:** `gemini-2.5-flash`

**Best for:** Price-performance balance, most common use cases

**Capabilities:**
- Text generation
- Multimodal input (text, image, video, audio, PDF)
- Thinking mode for complex reasoning
- Fast response times
- Large-scale processing

**Limits:**
- Input: 1M tokens
- Output: 8K tokens

**Use cases:**
- LINE bots and chatbots
- Content summarization
- Question answering
- Code generation
- Agentic applications

**Recommended for:** Most applications, especially when cost and speed matter.

### Gemini 2.5 Flash-Lite

**Model ID:** `gemini-2.5-flash-lite`

**Best for:** Ultra-fast, cost-efficient, high-volume tasks

**Capabilities:**
- Optimized for throughput
- Maintains broad capability support
- Reduced latency

**Limits:**
- Input: 1M tokens
- Output: 8K tokens

**Use cases:**
- High-volume API calls
- Real-time chat responses
- Cost-sensitive applications

**Recommended for:** High-traffic LINE bots with simple queries.

### Gemini 3 Flash

**Model ID:** `gemini-3-flash`

**Best for:** Better quality with superior search and grounding

**Capabilities:**
- Frontier intelligence
- Enhanced search grounding
- Full multimodal support
- Thinking enabled by default

**Limits:**
- Input: 1M tokens
- Output: 65K tokens

**Use cases:**
- High-quality responses
- Research and analysis
- Complex reasoning tasks

**Important:** Keep temperature at 1.0 (default). Lower values may cause issues.

**Recommended for:** Applications requiring higher quality than Flash 2.5.

### Gemini 3 Pro

**Model ID:** `gemini-3-pro`

**Best for:** Flagship model with best multimodal understanding

**Capabilities:**
- Best-in-class multimodal understanding
- Most powerful for agentic work
- Full tool support (code execution, search)
- Structured outputs

**Limits:**
- Input: 1M tokens
- Output: 65K tokens

**Use cases:**
- Complex analysis
- Large document processing
- Advanced reasoning
- Production-grade applications

**Important:** Keep temperature at 1.0 (default).

**Recommended for:** Premium applications with complex requirements.

### Gemini 2.5 Pro

**Model ID:** `gemini-2.5-pro`

**Best for:** Advanced reasoning over complex problems

**Capabilities:**
- Specialized for code, math, and STEM
- Excels at analyzing large codebases
- Best for sophisticated analytical tasks

**Limits:**
- Input: 1M tokens
- Output: 8K tokens

**Use cases:**
- Code analysis and generation
- Mathematical reasoning
- Scientific analysis
- Technical documentation

**Recommended for:** Technical and analytical applications.

## Previous Generation

### Gemini 2.0 Flash

**Model ID:** `gemini-2.0-flash`

**Status:** Stable alternative to 2.5 Flash

**Capabilities:**
- 1M token context window
- Multimodal support
- Proven reliability

**Recommended for:** Applications requiring stability over latest features.

## Model Selection Guide

### For LINE Bots

**Simple responses:** `gemini-2.5-flash-lite`
- Fastest response time
- Lowest cost
- Good for greetings, simple Q&A

**General use:** `gemini-2.5-flash`
- Best balance of speed, quality, and cost
- Handles most user queries well
- Recommended starting point

**High quality:** `gemini-3-flash`
- Better understanding
- More nuanced responses
- Worth the extra cost for premium bots

### By Response Time

**Fastest:** `gemini-2.5-flash-lite`

**Fast:** `gemini-2.5-flash`

**Moderate:** `gemini-3-flash`, `gemini-2.5-pro`

**Slower:** `gemini-3-pro` (with thinking enabled)

### By Cost (Lowest to Highest)

1. `gemini-2.5-flash-lite`
2. `gemini-2.5-flash`
3. `gemini-3-flash`
4. `gemini-2.5-pro`
5. `gemini-3-pro`

### By Quality (Lower to Higher)

1. `gemini-2.5-flash-lite`
2. `gemini-2.5-flash`
3. `gemini-2.5-pro`
4. `gemini-3-flash`
5. `gemini-3-pro`

## Usage Examples

### Using Flash for LINE Bot

```go
result, err := client.Models.GenerateContent(
    ctx,
    "gemini-2.5-flash",
    genai.Text(userMessage),
    &genai.GenerateContentConfig{
        Temperature:     genai.Ptr[float32](0.7),
        MaxOutputTokens: 512,
        SystemInstruction: &genai.Content{
            Parts: []*genai.Part{
                {Text: "You are Yuruppu, a friendly helper."},
            },
        },
    },
)
```

### Using Pro for Complex Analysis

```go
result, err := client.Models.GenerateContent(
    ctx,
    "gemini-3-pro",
    genai.Text("Analyze this complex dataset..."),
    &genai.GenerateContentConfig{
        Temperature:     genai.Ptr[float32](1.0), // Keep at 1.0 for Gemini 3
        MaxOutputTokens: 2048,
    },
)
```

### Using Flash-Lite for High Volume

```go
result, err := client.Models.GenerateContent(
    ctx,
    "gemini-2.5-flash-lite",
    genai.Text(simpleQuery),
    &genai.GenerateContentConfig{
        MaxOutputTokens: 256,
    },
)
```

## Thinking Mode

Gemini 2.5 Flash and 3.x models include thinking mode:

**Enabled by default** for Flash and Pro 3.x models
- Improves quality
- Increases response time
- Uses more tokens

**Disable thinking** for 2.5 Flash (if needed):
```go
config := &genai.GenerateContentConfig{
    ThinkingBudget: genai.Ptr[int32](0),
}
```

## Best Practices

1. **Start with gemini-2.5-flash**: Best default choice
2. **Use flash-lite for volume**: When handling many requests
3. **Use 3.x for quality**: When response quality is critical
4. **Keep temperature at 1.0**: For Gemini 3 models
5. **Monitor costs**: Track token usage per model
6. **Test different models**: Find the best fit for your use case
7. **Don't over-engineer**: Flash is good enough for most cases
8. **Consider latency**: Flash-lite for time-sensitive responses
