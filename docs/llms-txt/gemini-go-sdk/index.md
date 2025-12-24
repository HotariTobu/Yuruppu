# Google Gemini Go SDK

> Official Go SDK for Google's Gemini generative AI models. Provides access to Gemini API and Vertex AI with support for text generation, multimodal input, streaming, and chat functionality.

This documentation focuses on basic text generation patterns, single-turn conversations, context handling, timeout management, and error handling patterns essential for LINE bot development.

## Getting Started

- [Quick Start](quick-start.md): Installation and basic setup
- [Client Initialization](client-initialization.md): Creating and configuring clients

## Core Concepts

- [Text Generation](text-generation.md): Single-turn text generation patterns
- [Context and Timeout Handling](context-timeout.md): Managing request lifecycle with Go contexts
- [Error Handling](error-handling.md): Proper error detection and recovery patterns
- [Configuration](configuration.md): Generation parameters and model settings

## API Reference

- [Client Type](api-client.md): Client structure and methods
- [Response Structure](api-response.md): Understanding GenerateContentResponse
- [Available Models](models.md): Gemini model capabilities and selection

## Optional

- [Streaming Responses](streaming.md): Real-time content generation
- [File Management](files.md): Uploading and managing files
- [Context Caching](caching.md): Improving performance with cached contexts
- [Chat Sessions](chat.md): Multi-turn conversations
