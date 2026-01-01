# Migration Guide

## Migrating from Old SDK

If migrating from `cloud.google.com/go/vertexai/genai`:

1. Update import: `google.golang.org/genai`
2. Use `genai.NewClient()` instead of `vertexai.NewClient()`
3. Specify backend: `Backend: genai.BackendVertexAI`
4. Model names remain the same
5. Most API surface is similar, but check updated method signatures
