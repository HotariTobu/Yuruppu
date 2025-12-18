# Feature: Echo Messages

> LINE bot echoes back user messages with a character prefix.

## Overview

Yuruppu bot receives text messages from LINE users and responds by echoing the message back with a "Yuruppu: " prefix.

## Background & Purpose

This is the foundational feature for the Yuruppu LINE bot. It establishes the basic message handling flow and provides a simple way to verify that the bot is working correctly.

## Requirements

### Functional Requirements

- [x] FR-001: Receive text messages from LINE webhook
- [x] FR-002: Reply with the same message prefixed with "Yuruppu: "
- [x] FR-003: Handle LINE webhook signature verification
- [x] FR-004: Ignore non-text messages (images, stickers, audio, video, location)
- [x] FR-005: Load LINE channel secret and access token from environment variables `LINE_CHANNEL_SECRET` and `LINE_CHANNEL_ACCESS_TOKEN`

### Non-Functional Requirements

- [x] NFR-001: Respond within 1 second to avoid LINE timeout
- [x] NFR-002: Log all incoming messages at INFO level including: timestamp, user ID, message type, and message text (for text messages)
- [x] NFR-003: Handle concurrent webhook requests safely

## API Design

### Functions/Methods

```go
// NewBot creates a new LINE bot client with the given credentials.
// channelSecret is the LINE channel secret for signature verification.
// channelAccessToken is the LINE channel access token for API calls.
// Returns the bot client or an error if initialization fails.
func NewBot(channelSecret, channelAccessToken string) (*linebot.Client, error)

// HandleWebhook processes incoming LINE webhook requests.
// w is the HTTP response writer.
// r is the HTTP request containing the webhook payload.
// Returns HTTP 200 on success, 400 on invalid payload, 401 on invalid signature.
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
// Initialize the bot
bot, err := NewBot(os.Getenv("LINE_CHANNEL_SECRET"), os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"))
if err != nil {
    log.Fatal(err)
}

// Register webhook handler
http.HandleFunc("/webhook", HandleWebhook)

// Start server
log.Fatal(http.ListenAndServe(":8080", nil))
```

```go
// Message formatting example
formatted := FormatEchoMessage("Hello")
// Result: "Yuruppu: Hello"

formatted := FormatEchoMessage("  ")
// Result: "Yuruppu:   "
```

## Error Handling

| Error Type | Condition | HTTP Status | Message |
|------------|-----------|-------------|---------|
| ConfigError | Missing or empty environment variables | N/A (startup) | "Missing required environment variable: ..." |
| SignatureError | Invalid webhook signature | 401 | "Invalid signature" |
| PayloadError | Malformed webhook payload | 400 | "Invalid payload" |
| ReplyError | Failed to send reply to LINE | 200 (logged) | "Failed to send reply: ..." |

Note: ReplyError returns HTTP 200 to prevent LINE from retrying, but the error is logged.

## Acceptance Criteria

### AC-001: Echo text message [Linked to FR-001, FR-002]

- **Given**: LINE bot is running and connected
- **When**: User sends a text message "Hello"
- **Then**:
  - Bot receives the message via webhook
  - Bot replies with "Yuruppu: Hello"

### AC-002: Invalid signature rejection [Linked to FR-003]

- **Given**: LINE bot webhook endpoint is exposed
- **When**: Request with invalid signature is received
- **Then**:
  - Request is rejected with HTTP 401
  - No reply is sent

### AC-003: Valid signature acceptance [Linked to FR-003]

- **Given**: LINE bot webhook endpoint is exposed
- **When**: Request with valid signature is received
- **Then**:
  - Request is accepted with HTTP 200
  - Message is processed

### AC-004: Non-text message handling [Linked to FR-004]

- **Given**: LINE bot is running
- **When**: User sends a non-text message (image, sticker, etc.)
- **Then**:
  - Bot does not reply
  - No error is raised

### AC-005: Whitespace-only message handling [Linked to FR-002]

- **Given**: LINE bot is running
- **When**: User sends a whitespace-only text message "   "
- **Then**:
  - Bot replies with "Yuruppu:    "

### AC-006: Long message handling [Linked to FR-002]

- **Given**: LINE bot is running
- **When**: User sends a text message with 5000 characters
- **Then**:
  - Bot replies with the full message prefixed with "Yuruppu: "

### AC-007: Response time [Linked to NFR-001]

- **Given**: LINE bot is running under normal load
- **When**: User sends a text message
- **Then**:
  - Bot replies within 1 second (measured from webhook receipt to LINE API call completion)

### AC-008: Environment configuration [Linked to FR-005]

- **Given**: `LINE_CHANNEL_SECRET` and `LINE_CHANNEL_ACCESS_TOKEN` are set
- **When**: Bot starts
- **Then**:
  - Bot initializes successfully

### AC-009: Missing configuration [Linked to FR-005]

- **Given**: `LINE_CHANNEL_SECRET` or `LINE_CHANNEL_ACCESS_TOKEN` is not set
- **When**: Bot attempts to start
- **Then**:
  - Bot fails to start with ConfigError
  - Error message indicates which variable is missing

## Implementation Notes

- Use the official LINE Messaging API SDK for Go (`github.com/line/line-bot-sdk-go/v8`)
- Minimum Go version: 1.21
- Channel secret and access token must be configured via environment variables
- Webhook URL needs to be registered in LINE Developer Console
- The handler is safe for concurrent use

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-15 | 1.0 | Initial version | - |
| 2025-12-16 | 1.1 | Added FR-004, FR-005, NFR-003; completed error handling; added AC-003 to AC-009; improved API design with NewBot and HTTP status codes | - |
