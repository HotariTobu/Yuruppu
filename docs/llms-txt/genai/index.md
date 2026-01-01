# Google GenAI Go SDK

> The new official Google Generative AI SDK for Go (google.golang.org/genai) that supports both Gemini API and Vertex AI. Provides unified access to Google's generative AI models with streaming, chat, and multimodal capabilities.

This SDK replaces the older `cloud.google.com/go/vertexai/genai` package and offers a unified interface for both the Gemini Developer API and Vertex AI.

## Getting Started

- [Quick Start](quick-start.md): Client creation and environment setup for both Gemini API and Vertex AI

## Core Concepts

- [Content Generation](content-generation.md): Text generation, system prompts, generation parameters, and multimodal input
- [Streaming](streaming.md): Real-time streaming responses for content generation
- [Chat Sessions](chat-sessions.md): Multi-turn conversations with streaming support
- [Model Names](model-names.md): Available models for Gemini API and Vertex AI
- [Error Handling](error-handling.md): APIError type, error codes, and best practices

## API Reference

- [Client Structure](client-structure.md): Overview of Client services and capabilities
- [Advanced Features](advanced-features.md): Token counting, code execution, file upload, and JSON structured output
- [Helper Functions](helper-functions.md): Utility functions for creating content and parts

## Guides

- [Backend Differences](backend-differences.md): Key differences between Gemini API and Vertex AI
- [Best Practices](best-practices.md): Production recommendations and optimization tips
- [Migration Guide](migration-guide.md): How to migrate from the old Vertex AI SDK

## Optional

- [Live API](https://pkg.go.dev/google.golang.org/genai#Live): Real-time WebSocket connections for bidirectional streaming
- [Caches API](https://pkg.go.dev/google.golang.org/genai#Caches): Cache model responses to reduce latency and costs
- [File Search Stores](https://pkg.go.dev/google.golang.org/genai#FileSearchStores): Retrieval-augmented generation with document stores
- [Batches API](https://pkg.go.dev/google.golang.org/genai#Batches): Process multiple requests in batch jobs
- [Tunings API](https://pkg.go.dev/google.golang.org/genai#Tunings): Fine-tune models with custom datasets
- [Operations API](https://pkg.go.dev/google.golang.org/genai#Operations): Monitor long-running operations
- [Safety Settings](https://pkg.go.dev/google.golang.org/genai#SafetySetting): Configure content filtering and safety thresholds
- [Function Calling](https://pkg.go.dev/google.golang.org/genai#Tool): Define custom functions for the model to call
- [Official Documentation](https://pkg.go.dev/google.golang.org/genai): Complete package reference
- [GitHub Repository](https://github.com/googleapis/google-cloud-go/tree/main/genai): Source code and examples
- [Google AI Studio](https://aistudio.google.com/): Web-based model playground
- [Vertex AI Console](https://console.cloud.google.com/vertex-ai): GCP console for Vertex AI
