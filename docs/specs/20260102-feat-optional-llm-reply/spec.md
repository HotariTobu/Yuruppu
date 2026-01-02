# Feature: Optional LLM Reply

## Overview

Allow the LLM to decide whether to send a reply to a LINE message, rather than always responding.

## Background & Purpose

Currently, Yuruppu always sends a reply when a user sends a message. However, not all messages require a response. For example:
- Simple acknowledgments like "OK" or "got it"
- Messages that are clearly not directed at the bot in group chats
- Sticker-only messages that don't need verbal acknowledgment

By making replies optional, the bot can behave more naturally and avoid unnecessary responses.

## Out of Scope

- Per-user or per-group configuration settings
- Alternative actions (reactions, read receipts)
- Analytics or logging of skipped replies

## Requirements

### Functional Requirements

- [ ] FR-001: LLM can decide not to reply, and the bot silently sends no response
- [ ] FR-002: User message is saved to conversation history even when no reply is sent

## Acceptance Criteria

### AC-001: LLM skips reply [FR-001]

- **Given**: A message is received that does not require a reply
- **When**: LLM generates a response indicating no reply needed
- **Then**:
  - No reply is sent to the user
  - No error is raised
  - Handler completes successfully

### AC-002: User message saved without reply [FR-002]

- **Given**: User sent message A and LLM decided not to reply
- **When**: User sends message B
- **Then**:
  - Message A is included in conversation history
  - No assistant message exists between A and B
  - LLM receives both A and B as context

### AC-003: Normal reply behavior unchanged [FR-001]

- **Given**: A message that requires a reply
- **When**: LLM generates a normal response
- **Then**:
  - Reply is sent to the user as before
  - Existing behavior is preserved

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-02 | 1.0 | Initial version | - |
