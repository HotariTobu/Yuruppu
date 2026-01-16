# Feature: CLI Group Chat Mode

## Overview

Enable group chat simulation in the CLI using `-group-id` flag, mimicking real LINE group behavior with user membership and invitation via slash commands. Users can switch between group members using Ctrl+U.

## Background & Purpose

Currently, the CLI only supports 1-on-1 chat with a single user ID. To test and develop group chat features (where multiple users interact with Yuruppu in the same conversation), developers need a way to simulate a LINE group environment. This feature introduces group management (creation, membership) and allows switching between member identities during a REPL session.

## Out of Scope

- GUI or visual interface changes
- Concurrent message handling (messages are still sent one at a time)
- Real multi-process/multi-client simulation
- Leaving groups or removing members
- Group name/metadata management

## Requirements

### Functional Requirements

#### Group Startup

- [ ] FR-001: CLI accepts optional `-group-id` flag to specify a group
- [ ] FR-002: When `-group-id` is specified and group does not exist, create new group with `-user-id` as the first member
- [ ] FR-003: When `-group-id` is specified and group exists, allow chat only if `-user-id` is already a member
- [ ] FR-004: When `-group-id` is specified and user is not a member of existing group, return error and exit
- [ ] FR-005: When `-group-id` is not specified, behavior remains unchanged (1-on-1 chat)

#### Group Chat REPL

- [ ] FR-006: In group mode, chat type is set to "group" with source ID = group ID
- [ ] FR-007: REPL prompt displays current active user as `DisplayName(user-id)> ` (e.g., `Alice(alice)> `). If user has no profile, display `(user-id)> ` (e.g., `(alice)> `)
- [ ] FR-008: `/switch <user-id>` switches to a specific group member
- [ ] FR-009: `/users` lists all group members

#### User Invitation

- [ ] FR-010: `/invite <user-id>` adds a new user to the group
- [ ] FR-011: If invited user has no profile, they are treated as not having followed the bot (no profile creation triggered)
- [ ] FR-012: Inviting an already-member user shows an error message to stderr

#### Bot Invitation

- [ ] FR-014: Bot is not a member of newly created groups by default
- [ ] FR-015: `/invite-bot` adds the bot to the group and triggers `HandleJoin` event
- [ ] FR-016: When bot is not in the group, messages are not sent to the LLM (no response)
- [ ] FR-017: When bot is in the group, messages are processed by the LLM as normal

#### Event Simulation

- [ ] FR-018: `/invite <user-id>` triggers `HandleMemberJoined` event when bot is already in the group
- [ ] FR-019: `HandleMemberJoined` is called with the invited user's ID (always included, regardless of profile existence)
- [ ] FR-020: `HandleJoin` and `HandleMemberJoined` handlers are added to bot.Handler interface (implementation: log only)

#### Persistence

- [ ] FR-013: Group membership and bot invitation status are persisted to storage (survives CLI restarts)

### Non-Functional Requirements

- [ ] NFR-001: User switching must be instant (no network calls)
- [ ] NFR-002: Existing single-user CLI behavior must not break

## Acceptance Criteria

### AC-001: Create new group [FR-001, FR-002]

- **Given**: CLI is invoked with `-user-id alice -group-id mygroup`
- **And**: Group "mygroup" does not exist
- **When**: The CLI starts
- **Then**:
  - Group "mygroup" is created
  - "alice" is added as the first member
  - REPL starts in group chat mode

### AC-002: Join existing group [FR-001, FR-003]

- **Given**: Group "mygroup" exists with members ["alice", "bob"]
- **When**: CLI is invoked with `-user-id alice -group-id mygroup`
- **Then**:
  - REPL starts in group chat mode
  - "alice" is the active user

### AC-003: Reject non-member [FR-004]

- **Given**: Group "mygroup" exists with members ["alice", "bob"]
- **When**: CLI is invoked with `-user-id charlie -group-id mygroup`
- **Then**:
  - Error message to stderr: "user 'charlie' is not a member of group 'mygroup'"
  - CLI exits with non-zero status

### AC-004: No group-id means 1-on-1 [FR-005]

- **Given**: CLI is invoked with `-user-id alice` (no `-group-id`)
- **When**: User sends a message
- **Then**:
  - Chat type is "1-on-1"
  - Source ID equals user ID ("alice")

### AC-005: Group chat context [FR-006]

- **Given**: CLI is in group chat mode with group ID "mygroup"
- **When**: Any user sends a message
- **Then**:
  - Chat type in context is "group"
  - Source ID is "mygroup"
  - User ID is the current active user

### AC-006: Prompt shows current user with profile [FR-007]

- **Given**: CLI is in group chat mode
- **And**: Current user is "alice" with display name "Alice"
- **When**: REPL prompt is displayed
- **Then**:
  - Prompt shows `Alice(alice)> `

### AC-006b: Prompt shows current user without profile [FR-007]

- **Given**: CLI is in group chat mode
- **And**: Current user is "bob" with no profile
- **When**: REPL prompt is displayed
- **Then**:
  - Prompt shows `(bob)> `

### AC-007: /switch command [FR-008]

- **Given**: Group has members ["alice", "bob", "charlie"] with display names "Alice", "Bob", "Charlie"
- **And**: Current user is "alice"
- **When**: User types `/switch charlie`
- **Then**:
  - Current user changes to "charlie"
  - Prompt updates to `Charlie(charlie)> `

### AC-008: /switch with invalid user [FR-008, Error]

- **Given**: Group has members ["alice", "bob"]
- **When**: User types `/switch unknown`
- **Then**:
  - Error message to stderr: "'unknown' is not a member of this group"
  - Current user remains unchanged

### AC-009: /users command [FR-009]

- **Given**: Group has members ["alice", "bob", "charlie"] with display names "Alice", "Bob", "Charlie"
- **When**: User types `/users`
- **Then**:
  - Output: `Alice(alice), Bob(bob), Charlie(charlie)`

### AC-010: /invite new user [FR-010]

- **Given**: Group has members ["alice"], current is "alice"
- **When**: User types `/invite bob`
- **Then**:
  - "bob" is added to group members
  - Message: "bob has been invited to the group"

### AC-011: /invite user without profile [FR-010, FR-011]

- **Given**: Group has members ["alice"]
- **And**: User "newuser" has no profile (never followed the bot)
- **When**: User types `/invite newuser`
- **Then**:
  - "newuser" is added to group members
  - No profile creation is triggered
  - When "newuser" sends a message, bot treats them as not having followed

### AC-012: /invite existing member [FR-012]

- **Given**: Group has members ["alice", "bob"]
- **When**: User types `/invite bob`
- **Then**:
  - Error message to stderr: "bob is already a member of this group"
  - Membership unchanged

### AC-013: Bot not in group by default [FR-014, FR-016]

- **Given**: New group "mygroup" is created with user "alice"
- **And**: Bot has not been invited
- **When**: User sends a message
- **Then**:
  - Message is not sent to the LLM
  - No bot response is displayed

### AC-014: Invite bot to group [FR-015, FR-017]

- **Given**: Group "mygroup" exists, bot is not a member
- **When**: User types `/invite-bot`
- **Then**:
  - Bot is added to the group
  - `HandleJoin` is called with group context (source.type="group", source.groupId="mygroup")
  - Message: "Bot has been invited to the group"
- **When**: User sends a message after bot is invited
- **Then**:
  - Message is sent to the LLM
  - Bot response is displayed

### AC-015: Invite user triggers HandleMemberJoined [FR-018, FR-019]

- **Given**: Group "mygroup" exists with bot as member
- **When**: User types `/invite bob`
- **Then**:
  - "bob" is added to group members
  - `HandleMemberJoined` is called with:
    - source.type="group", source.groupId="mygroup"
    - joined.members=[{type: "user", userId: "bob"}]
  - Message: "bob has been invited to the group"

### AC-016: Invite user without bot does not trigger event [FR-018]

- **Given**: Group "mygroup" exists, bot is NOT a member
- **When**: User types `/invite bob`
- **Then**:
  - "bob" is added to group members
  - `HandleMemberJoined` is NOT called
  - Message: "bob has been invited to the group"

### AC-017: Group persists across restarts [FR-013]

- **Given**: Group "mygroup" was created with members ["alice", "bob"]
- **And**: CLI was restarted
- **When**: CLI is invoked with `-user-id alice -group-id mygroup`
- **Then**:
  - Group membership is ["alice", "bob"] (not reset)
  - REPL starts normally

### AC-018: Single-turn mode with group [FR-001, FR-006]

- **Given**: Group "mygroup" exists with members ["alice", "bob"] and bot
- **When**: CLI is invoked with `-user-id alice -group-id mygroup -message "Hello"`
- **Then**:
  - Message is processed as "alice" speaking in group "mygroup"
  - Chat type is "group", source ID is "mygroup", user ID is "alice"
  - Bot response is displayed
  - CLI exits (no REPL)

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-16 | 1.0 | Initial version | - |
| 2026-01-16 | 1.1 | Remove Ctrl+U requirement (FR-008), use /switch only | - |
