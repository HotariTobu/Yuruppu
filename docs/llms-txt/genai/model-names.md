# Model Names

## Gemini API Models

- `gemini-2.5-flash` - Latest fast model
- `gemini-2.5-pro` - Latest advanced model
- `gemini-2.0-flash` - Previous flash model
- `gemini-1.5-pro` - Previous pro model
- `gemini-1.5-flash` - Previous flash model

## Vertex AI Models

Vertex AI uses the same model names as above, plus third-party models:
- `meta/llama-3.2-90b-vision-instruct-maas`

**Format:** Model names are used directly without version prefixes. The SDK handles backend-specific formatting automatically.

For Vertex AI, you can also use fully qualified model names:
- `projects/{project}/locations/{location}/publishers/google/models/{model}`

But the short names (e.g., `gemini-2.5-flash`) are recommended and work across both backends.
