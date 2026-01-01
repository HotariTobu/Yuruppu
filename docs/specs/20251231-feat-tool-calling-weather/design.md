# Design: tool-calling-weather

## Overview

Add tool calling capability to GeminiAgent with weather tool as the first implementation. Uses wttr.in API and santhosh-tekuri/jsonschema for validation.

## File Structure

### New Files

| File | Purpose |
|------|---------|
| `internal/agent/tool.go` | Tool interface and validation wrapper |
| `internal/agent/tool_test.go` | Unit tests for tool validation logic |
| `internal/toolset/weather/weather.go` | Weather tool implementation |
| `internal/toolset/weather/weather_integration_test.go` | Integration tests (calls real wttr.in API) |
| `internal/toolset/weather/parameters.json` | Input JSON schema (embedded) |
| `internal/toolset/weather/response.json` | Output JSON schema (embedded) |

### Modified Files

| File | Changes |
|------|---------|
| `internal/agent/gemini.go` | Add toolMap field, tool calling loop, executeTool |
| `internal/agent/gemini_test.go` | Add tests for tool calling flow |
| `main.go` | Create weather tool and pass to GeminiConfig |
| `go.mod` | Add `github.com/santhosh-tekuri/jsonschema/v6` |

## Interfaces

### Tool Interface (`internal/agent/tool.go`)

```go
// Tool defines a provider-agnostic interface for function calling tools.
type Tool interface {
    Name() string
    Description() string
    ParametersJsonSchema() []byte
    ResponseJsonSchema() []byte
    Callback(ctx context.Context, validatedArgs map[string]any) (map[string]any, error)
}

// Validator validates data against a schema.
type Validator interface {
    Validate(data any) error
}

// tool wraps Tool with validators (private).
type tool struct {
    impl                Tool
    parametersValidator Validator
    responseValidator   Validator
}

func newTool(t Tool) (tool, error)
func (t *tool) Use(ctx context.Context, args map[string]any) (map[string]any, error)
func compileSchema(schemaBytes []byte) (*jsonschema.Schema, error)
```

### Weather Tool (`internal/toolset/weather/weather.go`)

```go
type Tool struct {
    httpClient *http.Client
}

func NewTool(timeout time.Duration) *Tool
func (t *Tool) Name() string                    // "get_weather"
func (t *Tool) Description() string
func (t *Tool) ParametersJsonSchema() []byte    // //go:embed parameters.json
func (t *Tool) ResponseJsonSchema() []byte      // //go:embed response.json
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error)
```

### GeminiConfig Addition

```go
type GeminiConfig struct {
    // ... existing fields ...
    Tools []Tool  // Optional tools for function calling
}
```

### GeminiAgent Struct Addition

```go
type GeminiAgent struct {
    // ... existing fields ...
    toolMap map[string]tool  // name -> wrapped tool with validators
}
```

### gemini.go New Functions

```go
// generateWithToolLoop handles multi-turn conversation with tool calling.
// Returns all contents added after initialContents.
func (g *GeminiAgent) generateWithToolLoop(ctx context.Context, model string, initialContents []*genai.Content, config *genai.GenerateContentConfig) ([]*genai.Content, error)

// executeTool executes a tool and returns the function response.
func (g *GeminiAgent) executeTool(ctx context.Context, call *genai.FunctionCall) *genai.FunctionResponse

// toGenaiTools converts Tool[] to []*genai.Tool for Gemini API.
func toGenaiTools(tools []Tool) []*genai.Tool
```

### gemini.go Modified Functions

- `NewGeminiAgent`: Initialize toolMap, convert Tools to genai.Tool, add to cache config
- `Generate`: Call generateWithToolLoop instead of direct GenerateContent

### main.go Changes

```go
import "yuruppu/internal/toolset/weather"

// In main() or agent initialization:
weatherTool := weather.NewTool(3 * time.Second)

agentCfg := agent.GeminiConfig{
    // ... existing fields ...
    Tools: []agent.Tool{weatherTool},
}
```

## Data Flow

```
1. User Message
   └─> "東京の天気は？"

2. GeminiAgent.Generate()
   └─> buildContents() → []*genai.Content

3. generateWithToolLoop()
   ├─> GenerateContent() → response with FunctionCall
   │   └─> FunctionCall{Name: "get_weather", Args: {"location": "東京"}}
   │
   ├─> executeTool()
   │   ├─> tool.Use()
   │   │   ├─> Validate args against parameters.json
   │   │   ├─> weather.Callback() → HTTP GET wttr.in/東京?format=j1
   │   │   │   ├─> Success: {"location": "東京", "current_temp_c": "15", "condition": "Sunny"}
   │   │   │   └─> Error: {"error": "API request failed: timeout"}
   │   │   └─> Validate response against response.json
   │   └─> Return FunctionResponse
   │
   ├─> Append FunctionResponse to contents
   └─> Loop: GenerateContent() again with tool result
       └─> Model generates final text response

4. extractContentsToAssistantMessage()
   └─> AssistantMessage with weather info incorporated
```

## Implementation Notes

- Tool initialization happens in `main.go` (agent is created there)
- Multiple tools execute in parallel using `sync.WaitGroup`
- Timeout (3s) enforced by `http.Client.Timeout` (NFR-001)
- Errors return as `{"error": "..."}` for LLM to handle gracefully (NFR-002)
- JSON schemas embedded via `//go:embed` directive
- Loop continues until model returns no FunctionCalls

## Related ADRs

- [20251231-weather-api.md](../../adr/20251231-weather-api.md) - wttr.in selection
- [20260101-json-schema-validator.md](../../adr/20260101-json-schema-validator.md) - santhosh-tekuri/jsonschema selection
