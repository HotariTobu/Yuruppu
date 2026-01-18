# Feature: Track Group Member Count

## Overview

Track the number of members in LINE groups and pass this information to the LLM as conversation context, enabling Yuruppu to understand group size when responding.

## Background & Purpose

Knowing the group size provides valuable context for Yuruppu's responses. A small private group (3-5 people) has different conversation dynamics than a large community group (50+ people). By tracking member counts, Yuruppu can:

- Adjust response tone based on audience size
- Understand the scale of the group when making suggestions
- Provide more contextually appropriate responses

## Out of Scope

- Tracking individual member identities or profiles
- Member activity analytics
- Notifying users when members join/leave
- Historical member count tracking (trend data)
- Handling member count going below 0 (cannot happen)

## Requirements

### Functional Requirements

- [ ] FR-001: Retrieve group member count when the bot joins a group
- [x] FR-002: Increment member count by the number of members who joined
- [ ] FR-003: Decrement member count by the number of members who left
- [ ] FR-004: Persist member count to storage along with group information
- [ ] FR-005: Include group member count in the context passed to LLM for group messages

## Acceptance Criteria

### AC-001: Member count retrieved on join [FR-001, FR-004]

- **Given**: The bot is invited to a LINE group
- **When**: The bot receives a join event
- **Then**:
  - The member count is retrieved from LINE API
  - The count is saved to storage with the group information

### AC-002: Member count incremented on member join [FR-002, FR-004]

- **Given**: The bot is already in a group with a stored member count
- **When**: Members join the group (member joined event)
- **Then**:
  - The stored member count is incremented by the number of members who joined
  - The updated count is persisted

### AC-003: Member count decremented on member leave [FR-003, FR-004]

- **Given**: The bot is already in a group with a stored member count
- **When**: Members leave the group (member left event)
- **Then**:
  - The stored member count is decremented by the number of members who left
  - The updated count is persisted

### AC-004: Member count passed to LLM [FR-005]

- **Given**: A user sends a message in a group chat
- **When**: The message is sent to the LLM
- **Then**: The group member count is included in the context

### AC-005: Handle missing member count gracefully [FR-005]

- **Given**: A group message is received but member count is not stored (e.g., bot joined before this feature)
- **When**: The message is processed
- **Then**:
  - Message processing continues normally
  - LLM request is sent without member count (or with null/unknown value)

### AC-006: Handle API failure on join [FR-001]

- **Given**: The bot is invited to a group
- **When**: The LINE API call to get member count fails
- **Then**:
  - The error is logged
  - Join event handling completes without crashing
  - Group is saved without member count

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-18 | 1.0 | Initial version | - |
