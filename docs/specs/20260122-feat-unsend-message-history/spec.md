# Feature: Unsend Message History Cleanup

> Handle LINE unsend events to remove recalled messages from conversation history.

## Overview

When a user unsends (recalls) a message in LINE, the bot should detect this event and remove the corresponding message from the stored conversation history.

## Background & Purpose

Currently, when a user unsends a message in LINE, the bot does not handle this event. The recalled message remains in the conversation history, which may cause inconsistency between what the user sees in LINE and what the bot uses for context in AI conversations.

By handling unsend events, we ensure:
- Conversation history accurately reflects the actual chat state
- The AI does not reference messages the user has intentionally removed
- User privacy is respected when they choose to recall a message

## Out of Scope

- Notifying the user that their message was removed from history
- Handling unsend events for bot's own messages (the bot cannot unsend)
- Archiving or logging unsent messages for audit purposes
- Handling bulk unsend operations

## Requirements

### Functional Requirements

- [ ] FR-001: System receives and processes LINE unsend webhook events
- [ ] FR-002: System identifies the target message in history using the message ID from the unsend event
- [ ] FR-003: System removes the identified message from conversation history
- [ ] FR-004: System persists the updated history to storage

### Non-Functional Requirements

- [ ] NFR-001: History updates must prevent data corruption from concurrent modifications

## Acceptance Criteria

### AC-001: Unsend event triggers message removal [FR-001, FR-002, FR-003, FR-004]

- **Given**: A user has previously sent a text message that is stored in history
- **When**: The user unsends that message in LINE
- **Then**:
  - The unsend webhook event is received
  - The corresponding message is identified by its message ID
  - The message is removed from the history
  - The updated history is saved to storage

### AC-002: Unsend for non-existent message [FR-002, Error]

- **Given**: History does not contain a message with the specified message ID
- **When**: An unsend event is received for that message ID
- **Then**:
  - The system logs a warning but does not fail
  - No changes are made to the history

### AC-003: Unsend in group chat [FR-001, FR-003]

- **Given**: A message exists in a group chat history
- **When**: A user unsends that message
- **Then**:
  - The message is removed from the group's history
  - Other messages in the group history remain unaffected

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-22 | 1.0 | Initial version | - |
