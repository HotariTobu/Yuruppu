# Feature: Pass User Profile to LLM

## Overview

Retrieve LINE user profile information and pass it to the LLM so Yuruppu can personalize responses and understand who is sending each message.

## Background & Purpose

Currently, user messages are sent to the LLM with only the LINE user ID (e.g., "U1234567890abcdef"), which is an opaque identifier. Yuruppu cannot address users by name or personalize conversations. By fetching and passing user profile information, Yuruppu can:

- Address users by their display name (e.g., "〇〇くん、こんにちは〜")
- Understand who is speaking in multi-message conversations
- Provide a more natural and personalized chat experience

## Out of Scope

- Using profile information for analytics or tracking
- Modifying system prompt based on user profile
- Group/room profile information (group name, icon)
- User language preference usage
- Profile refresh (updating stored profiles when user changes their LINE profile)

## Requirements

### Functional Requirements

- [ ] FR-001: Fetch user profile (display name, avatar URL, status message) from LINE when processing a message
- [ ] FR-002: Include fetched profile fields in user messages sent to LLM
- [ ] FR-003: Cache user profiles to avoid redundant API calls during bot runtime
- [ ] FR-004: Persist user profiles so that profiles survive bot restarts
- [ ] FR-005: Support profile retrieval in 1:1 chats, group chats, and room chats

### Non-Functional Requirements

- [ ] NFR-001: Profile fetch failures must not block message processing

## Acceptance Criteria

### AC-001: Profile included in LLM request [FR-001, FR-002]

- **Given**: A user with display name, avatar URL, and status message sends a message
- **When**: The message is processed and sent to the LLM
- **Then**: The LLM request includes the user's display name, avatar URL, and status message

### AC-002: Profile cached [FR-003]

- **Given**: A user sends multiple messages during bot runtime
- **When**: Each message is processed
- **Then**: LINE profile API is called only once for the first message

### AC-003: Profile persisted [FR-004]

- **Given**: A user has sent a message before (profile already fetched)
- **When**: Bot restarts and the user sends another message
- **Then**: The stored profile is used without calling LINE API again

### AC-004: All chat types supported [FR-005]

- **Given**: A user sends a message in a 1:1 chat, group chat, or room chat
- **When**: The message is processed
- **Then**: The user's profile is retrieved regardless of chat type

### AC-005: Profile fetch failure handled gracefully [NFR-001]

- **Given**: LINE API returns an error when fetching profile
- **When**: The message is processed
- **Then**:
  - Message processing continues normally
  - Bot sends a response to the user (without personalization)

### AC-006: Missing profile fields handled [FR-002]

- **Given**: A user's LINE profile has missing optional fields (e.g., no status message)
- **When**: The profile is fetched
- **Then**: Missing fields are treated as empty, no error occurs

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-03 | 1.0 | Initial version | - |
| 2026-01-03 | 1.1 | Consolidate requirements, remove implementation details, add persistent storage | - |
