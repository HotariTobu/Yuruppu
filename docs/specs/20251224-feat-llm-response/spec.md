# Feature: LLM Response

> LINE bot responds to user messages using an LLM instead of echo.

## Overview

Replace the current echo behavior with LLM-generated responses. When a user sends a text message, the bot calls an LLM API and replies with the generated response.

## Background & Purpose

The current Yuruppu bot simply echoes back user messages with a "Yuruppu: " prefix. To provide more meaningful interactions, we want to use an LLM to generate contextual responses to user messages.

## Requirements

### Functional Requirements

- [ ] FR-001: Call LLM API when a message is received (text, image, sticker, video, audio, location)
- [ ] FR-002: Reply to the user with the LLM-generated response
- [ ] FR-003: Load LLM API credentials from environment variables; bot fails to start during initialization if credentials are missing
- [ ] FR-004: On LLM API error (timeout, rate limit, network error, invalid response, authentication error), do not reply to the user and log the error
- [ ] FR-005: Use a hardcoded system prompt (defined in code); the specific prompt text will be determined in the design phase
- [ ] FR-006: Remove the "Yuruppu: " prefix from responses (LLM generates the full response)
- [ ] FR-007: Do not maintain conversation history between messages (single-turn only); each message is processed independently
- [ ] FR-008: For non-text messages, send a description in the format "[User sent a {type}]" to the LLM, where {type} is: image, sticker, video, audio, or location; the system prompt will guide how the LLM responds
- [ ] FR-009: Do not implement retry logic for LLM API errors; each request is attempted once

### Non-Functional Requirements

- [ ] NFR-001: LLM API total request timeout (from API call initiation to response completion) should be configurable via environment variable (default: 30 seconds)
- [ ] NFR-002: Log LLM API request (system prompt and user message) and response (generated text) at DEBUG level
- [ ] NFR-003: Log LLM API errors at ERROR level with error type and details

### Technical Requirements

- [ ] TR-001: LLM provider and model selection will be determined in the design phase
- [ ] TR-002: Create an abstraction layer (interface) for LLM providers to allow future provider changes

## Design Decisions (To Be Determined)

The following decisions will be made in the `/design` phase:

1. **LLM Provider**: OpenAI, Google Gemini, Anthropic Claude, or Vertex AI
2. **Model Selection**: Specific model to use (e.g., gpt-4o-mini, gemini-1.5-flash, claude-3-haiku)

## Error Handling

| Error Type | Condition | User Response | Logging |
|------------|-----------|---------------|---------|
| LLMTimeoutError | LLM API call exceeds configured timeout | No reply | ERROR level with request details |
| LLMRateLimitError | LLM API rate limit exceeded (HTTP 429) | No reply | ERROR level with retry-after info if available |
| LLMNetworkError | Network error during API call (connection refused, DNS failure, etc.) | No reply | ERROR level with error details |
| LLMResponseError | Invalid or malformed response from LLM | No reply | ERROR level with response details |
| LLMAuthError | Authentication/authorization error (HTTP 401/403, invalid API key) | No reply | ERROR level with error code |

Note: No retry logic is implemented; all errors result in a single failed attempt.

## Acceptance Criteria

### AC-001: LLM response to text message [Linked to FR-001, FR-002]

- **Given**: LINE bot is running with valid LLM API credentials
- **When**: User sends a text message "Hello"
- **Then**:
  - Bot calls LLM API with the user's message as input
  - Bot replies with a non-empty text message (the LLM-generated response)
  - Reply does not contain "Yuruppu: " prefix

### AC-002: LLM API timeout handling [Linked to FR-004]

- **Given**: LINE bot is running with LLM API timeout set to 1 second
- **When**: LLM API takes longer than 1 second to respond
- **Then**:
  - Bot does not reply to the user
  - Error is logged at ERROR level containing "timeout"

### AC-003: LLM API rate limit handling [Linked to FR-004]

- **Given**: LINE bot is running
- **When**: LLM API returns HTTP 429 (rate limit)
- **Then**:
  - Bot does not reply to the user
  - Error is logged at ERROR level containing "rate limit"

### AC-004: LLM API network error handling [Linked to FR-004]

- **Given**: LINE bot is running
- **When**: Network error occurs during LLM API call (connection refused, DNS failure)
- **Then**:
  - Bot does not reply to the user
  - Error is logged at ERROR level containing error details

### AC-005: LLM API authentication error handling [Linked to FR-004]

- **Given**: LINE bot is running with invalid LLM API credentials
- **When**: LLM API returns HTTP 401 or 403
- **Then**:
  - Bot does not reply to the user
  - Error is logged at ERROR level containing "auth" or status code

### AC-006: Hardcoded system prompt [Linked to FR-005]

- **Given**: LINE bot is running
- **When**: Bot calls LLM API
- **Then**:
  - LLM request contains a non-empty system prompt field
  - System prompt value matches the hardcoded constant in the code
  - DEBUG log shows the system prompt in the request

### AC-007: Image message handling [Linked to FR-001, FR-008]

- **Given**: LINE bot is running
- **When**: User sends an image message
- **Then**:
  - Bot calls LLM API with "[User sent an image]" as user message
  - Bot replies with a non-empty text message

### AC-008: Sticker message handling [Linked to FR-001, FR-008]

- **Given**: LINE bot is running
- **When**: User sends a sticker message
- **Then**:
  - Bot calls LLM API with "[User sent a sticker]" as user message
  - Bot replies with a non-empty text message

### AC-009: Video message handling [Linked to FR-001, FR-008]

- **Given**: LINE bot is running
- **When**: User sends a video message
- **Then**:
  - Bot calls LLM API with "[User sent a video]" as user message
  - Bot replies with a non-empty text message

### AC-010: Audio message handling [Linked to FR-001, FR-008]

- **Given**: LINE bot is running
- **When**: User sends an audio message
- **Then**:
  - Bot calls LLM API with "[User sent an audio]" as user message
  - Bot replies with a non-empty text message

### AC-011: Location message handling [Linked to FR-001, FR-008]

- **Given**: LINE bot is running
- **When**: User sends a location message
- **Then**:
  - Bot calls LLM API with "[User sent a location]" as user message
  - Bot replies with a non-empty text message

### AC-012: LLM credentials configuration [Linked to FR-003]

- **Given**: LLM API credentials environment variable is set with valid value
- **When**: Bot starts
- **Then**:
  - Bot initializes LLM client successfully
  - Bot starts without error

### AC-013: Missing LLM credentials [Linked to FR-003]

- **Given**: LLM API credentials environment variable is not set
- **When**: Bot attempts to start
- **Then**:
  - Bot fails to start during initialization (before accepting requests)
  - Error message indicates which variable is missing

### AC-014: Single-turn conversation [Linked to FR-007]

- **Given**: LINE bot is running
- **When**: User sends message "What did I just say?" after sending "Hello"
- **Then**:
  - Bot calls LLM API with only "What did I just say?" as user message
  - DEBUG log shows only one user message in the LLM request (no message history)
  - No previous messages are included in the request

## Implementation Notes

- This feature replaces the echo behavior from the `20251215-feat-echo` spec
- The `FormatEchoMessage` function will no longer be used for user-facing responses
- LLM provider interface should be designed for testability (mock injection)
- Consider using context with timeout for LLM API calls

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-24 | 1.0 | Initial version | - |
| 2025-12-24 | 1.1 | Added FR-007 (single-turn), AC-005 (auth error), AC-011 (single-turn); clarified FR-003, FR-004, FR-005, NFR-001, NFR-002, NFR-003; improved all acceptance criteria with testable conditions | - |
| 2025-12-24 | 1.2 | Changed FR-005 to hardcoded system prompt; added FR-008 for non-text message handling; updated FR-001 to support all message types; added AC-007 to AC-011 for non-text messages | - |
| 2025-12-24 | 1.3 | Added FR-009 (no retry); clarified FR-008 format; improved AC-006, AC-007-011, AC-014 testability; clarified NFR-002 | - |
| 2025-12-24 | 1.4 | Removed system prompt content and env var names from Design Decisions (implementation details) | - |
