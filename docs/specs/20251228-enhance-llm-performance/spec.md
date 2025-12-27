# Enhancement: LLM Performance Optimization

## Overview

Enhance LLM API calls by enabling context caching for system prompts and disabling thinking mode to reduce latency and cost.

## Background & Purpose

Currently, the system prompt is sent with every LLM API request, which is inefficient because:
1. The system prompt is static and identical across all requests
2. Each request incurs full token processing cost for the system prompt
3. Gemini 2.5 models support thinking mode by default, which adds latency for simple conversational responses

By implementing context caching and disabling thinking mode:
- System prompt tokens are cached and reused (up to 90% cost reduction for system prompt tokens only)
- Response latency is reduced by eliminating reasoning overhead
- Simple character responses don't require complex reasoning

**Non-goals**:
- This enhancement does not optimize user message caching
- This enhancement does not change model selection logic

## Current Behavior

1. `GenerateText()` sends the full system prompt with every request
2. No caching mechanism is used for repeated content
3. Thinking mode is enabled by default on Gemini 2.5 models

## Proposed Changes

- [ ] CH-001: Add context caching for system prompt
- [ ] CH-002: Disable thinking mode
- [ ] CH-003: Add Provider lifecycle management (Close method)

## Acceptance Criteria

### AC-001: Context Cache Creation [Linked to CH-001]

- **Given**: VertexAI client initialization with a system prompt
- **When**: Client is created
- **Then**:
  - The system prompt is cached for reuse across requests
  - Cache has a reasonable TTL (e.g., 60 minutes)

### AC-002: Context Cache Usage [Linked to CH-001]

- **Given**: A cached system prompt exists
- **When**: `GenerateText()` is called
- **Then**:
  - The cached system prompt is used instead of sending it with each request
  - The API returns a successful response

### AC-003: Thinking Mode Disabled [Linked to CH-002]

- **Given**: VertexAI client is configured
- **When**: `GenerateText()` is called
- **Then**:
  - Thinking/reasoning mode is disabled
  - No thinking tokens are consumed

### AC-004: Provider Close Method [Linked to CH-003]

- **Given**: A Provider instance is created
- **When**: `Close(ctx)` is called
- **Then**:
  - Cached resources are cleaned up
  - `Close()` is idempotent (safe to call multiple times)
  - After `Close()`, subsequent `GenerateText()` calls return an error

### AC-005: Backward Compatibility [Linked to CH-001, CH-002, CH-003]

- **Given**: Existing code using the Provider interface
- **When**: The code calls `GenerateText()`
- **Then**:
  - Existing behavior is preserved
  - The enhancement is transparent to callers

### AC-006: Cache Expiration Handling [Linked to CH-001]

- **Given**: Cached content has expired or been deleted
- **When**: `GenerateText()` is called
- **Then**:
  - The cache is recreated automatically
  - The request completes successfully
  - A warning log is recorded

### AC-007: Fallback for Insufficient Token Count [Linked to CH-001]

- **Given**: System prompt is below the minimum token requirement for caching
- **When**: Client is created or cache creation fails
- **Then**:
  - Caching is skipped gracefully
  - `GenerateText()` works without caching (current behavior)
  - An info log is recorded

## References

- [Vertex AI Context Caching Overview](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/context-cache/context-cache-overview)
- [Vertex AI Thinking Configuration](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/thinking)
- [Google GenAI Go SDK](https://pkg.go.dev/google.golang.org/genai)

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-28 | 1.0 | Initial version | - |
| 2025-12-28 | 1.1 | Remove implementation details, focus on observable behavior | - |
