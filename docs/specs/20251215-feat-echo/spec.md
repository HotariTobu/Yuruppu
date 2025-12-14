# Feature: Echo Messages

> LINE bot echoes back user messages with a character prefix.

## Overview

Yuruppu bot receives text messages from LINE users and responds by echoing the message back with a "Yuruppu: " prefix.

## Background & Purpose

This is the foundational feature for the Yuruppu LINE bot. It establishes the basic message handling flow and provides a simple way to verify that the bot is working correctly.

## Requirements

### Functional Requirements

- [ ] FR-001: Receive text messages from LINE webhook
- [ ] FR-002: Reply with the same message prefixed with "Yuruppu: "
- [ ] FR-003: Handle LINE webhook signature verification

### Non-Functional Requirements

- [ ] NFR-001: Respond within 1 second to avoid LINE timeout
- [ ] NFR-002: Log all incoming messages for debugging

## API Design

### Functions/Methods

```go
// HandleWebhook processes incoming LINE webhook requests.
// w is the HTTP response writer.
// r is the HTTP request containing the webhook payload.
func HandleWebhook(w http.ResponseWriter, r *http.Request)

// HandleTextMessage processes a text message event and sends an echo reply.
// bot is the LINE bot client.
// event is the message event from LINE.
// Returns any error encountered during reply.
func HandleTextMessage(bot *linebot.Client, event *linebot.Event) error

// FormatEchoMessage formats a message with the Yuruppu prefix.
// message is the original user message.
// Returns the formatted echo message.
func FormatEchoMessage(message string) string
```

### Type Definitions

```go
// No custom types required for this feature.
// Uses LINE SDK types directly.
```

## Usage Examples

```go
// Example webhook handler setup
http.HandleFunc("/webhook", HandleWebhook)

// Example message formatting
formatted := FormatEchoMessage("Hello")
// Result: "Yuruppu: Hello"
```

## Error Handling

| Error Type | Condition | Message |
|------------|-----------|---------|
| SignatureError | Invalid webhook signature | "Invalid signature" |
| ReplyError | Failed to send reply to LINE | "Failed to send reply: ..." |

## Acceptance Criteria

### AC-001: Echo text message [Linked to FR-001, FR-002]

- **Given**: LINE bot is running and connected
- **When**: User sends a text message "Hello"
- **Then**:
  - Bot receives the message via webhook
  - Bot replies with "Yuruppu: Hello"

### AC-002: Signature verification [Linked to FR-003]

- **Given**: LINE bot webhook endpoint is exposed
- **When**: Request with invalid signature is received
- **Then**:
  - Request is rejected
  - No reply is sent

### AC-003: Empty message handling [Linked to FR-002]

- **Given**: LINE bot is running
- **When**: User sends an empty text message ""
- **Then**:
  - Bot replies with "Yuruppu: "

## Implementation Notes

- Use the official LINE Messaging API SDK for Go
- Channel secret and access token should be configured via environment variables
- Webhook URL needs to be registered in LINE Developer Console

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-15 | 1.0 | Initial version | - |
