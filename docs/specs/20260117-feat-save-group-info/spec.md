# Feature: Save Group Information on Invitation

## Overview

When Yuruppu is invited to a LINE group, the bot retrieves and saves the group information to persistent storage for use as context in conversations.

## Background & Purpose

The bot needs group context (name, etc.) to provide more relevant and contextual responses when interacting in group chats. By saving group information when invited, the bot can reference this data during message handling.

## Out of Scope

- Handling when the bot is removed from a group (data cleanup)
- Syncing group member information
- Updating group information when it changes
- Sending a greeting message when invited

## Requirements

### Functional Requirements

- [ ] FR-001: When the bot receives a join event, retrieve group information from LINE API
- [ ] FR-002: Save the retrieved group information to persistent storage
- [ ] FR-003: Store all available group data: group ID, group name, and picture URL

### Non-Functional Requirements

- [ ] NFR-001: Group information retrieval and storage should not block the join event response

## Acceptance Criteria

### AC-001: Save group info on join [FR-001, FR-002, FR-003]

- **Given**: The bot is invited to a LINE group
- **When**: The bot receives a join event with source type "group"
- **Then**:
  - The bot retrieves group summary from LINE API using the group ID
  - The group information (ID, name, picture URL) is saved to storage
  - The join event handling completes successfully

### AC-002: Handle API failure gracefully [FR-001, Error]

- **Given**: The bot is invited to a LINE group
- **When**: The LINE API call to get group summary fails
- **Then**:
  - The error is logged
  - The join event handling completes without crashing
  - No partial data is saved

### AC-003: Handle missing picture URL [FR-003]

- **Given**: The bot is invited to a LINE group without a picture
- **When**: The group summary returns empty picture URL
- **Then**:
  - The group is saved with an empty picture URL field
  - Other fields (ID, name) are saved normally

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-17 | 1.0 | Initial version | - |
