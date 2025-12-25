# Refactor: LLM Integration Code Cleanup

> Remove redundant code patterns and hardcoded values in the LLM integration.

## Overview

Clean up redundant code patterns, unused fields, and hardcoded values introduced during the LLM response feature implementation.

## Background & Purpose

The LLM response feature (`20251224-feat-llm-response`) introduced several code patterns that can be improved:
- Hardcoded region that should be derived from Cloud Run metadata
- Duplicate type checking patterns (pointer/non-pointer)
- Unused struct fields
- Repeated logic across multiple functions

## Current Structure

- `internal/llm/vertexai.go`: Contains hardcoded `defaultRegion = "asia-northeast1"`
- `internal/llm/provider.go`: Contains `LLMRateLimitError.RetryAfter` field that is never populated
- `internal/llm/errors.go`: Creates `LLMRateLimitError` without setting `RetryAfter`
- `internal/bot/bot.go`: Contains duplicate pointer/non-pointer type checks for all message types in two separate switch statements

## Proposed Structure

- `internal/llm/vertexai.go`: Read region from Cloud Run metadata server with fallback to `GCP_REGION` environment variable, then hardcoded default
- `internal/llm/provider.go`: Remove unused `RetryAfter` field from `LLMRateLimitError`
- `internal/bot/bot.go`:
  - Remove redundant non-pointer type cases (LINE SDK returns pointers)
  - Extract message type detection into a helper function to eliminate duplication

## Scope

- [ ] SC-001: Remove hardcoded `defaultRegion` and use Cloud Run metadata
- [ ] SC-002: Remove redundant pointer/non-pointer type checks in `logIncomingMessage`
- [ ] SC-003: Remove redundant pointer/non-pointer type checks in `HandleWebhook`
- [ ] SC-004: Extract message type and content extraction into a helper function
- [ ] SC-005: Remove unused `RetryAfter` field from `LLMRateLimitError`

## Breaking Changes

None - all changes are internal implementation details.

## Acceptance Criteria

### AC-001: Region derived from Cloud Run metadata [Linked to SC-001]

- **Given**: Application is running on Cloud Run
- **When**: Vertex AI client is created
- **Then**:
  - Region is read from Cloud Run metadata server (`http://metadata.google.internal/computeMetadata/v1/instance/region`)
  - Metadata server request has a 2-second timeout
  - If metadata server is unavailable or times out, fallback to `GCP_REGION` environment variable
  - If metadata response format is unexpected (not `projects/*/regions/*`), fallback to `GCP_REGION` environment variable
  - If environment variable is not set, fallback to `asia-northeast1`
  - All existing tests pass

### AC-002: Single type case per message type in logIncomingMessage [Linked to SC-002]

- **Given**: Refactoring is complete
- **When**: `logIncomingMessage` switch statement is reviewed
- **Then**:
  - Each message type has only one case (pointer type)
  - Total cases reduced from 12 to 6
  - All existing tests pass

### AC-003: Single type case per message type in HandleWebhook [Linked to SC-003]

- **Given**: Refactoring is complete
- **When**: `HandleWebhook` switch statement is reviewed
- **Then**:
  - Each message type has only one case (pointer type)
  - Total cases reduced from 12 to 6
  - All existing tests pass

### AC-004: Message type and content extraction helper [Linked to SC-004]

- **Given**: Refactoring is complete
- **When**: Code is reviewed
- **Then**:
  - A helper function exists with signature: `func extractMessageInfo(message webhook.MessageContentInterface) (messageType string, userMessage string)`
  - `messageType` returns one of: "text", "image", "sticker", "video", "audio", "location", or "unknown"
  - `userMessage` returns the text content for text messages, or formatted string like "[User sent an image]" for non-text messages
  - Both `logIncomingMessage` and `HandleWebhook` use this helper
  - Code duplication eliminated
  - All existing tests pass

### AC-005: RetryAfter field removed [Linked to SC-005]

- **Given**: Refactoring is complete
- **When**: `LLMRateLimitError` struct is reviewed
- **Then**:
  - `RetryAfter` field is removed from the struct
  - All references to `RetryAfter` are removed
  - All existing tests pass

## Implementation Notes

- Cloud Run metadata endpoint: `http://metadata.google.internal/computeMetadata/v1/instance/region`
- Response format: `projects/PROJECT-NUMBER/regions/REGION` - need to extract just the region
- Metadata request requires header: `Metadata-Flavor: Google`
- LINE SDK webhook types are always pointers (e.g., `*webhook.TextMessageContent`)

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-25 | 1.0 | Initial version | - |
| 2025-12-25 | 1.1 | Added metadata timeout (2s), error handling for malformed response; clarified helper function signature | - |
