# Feature: Mock External Services CLI

> CLI tool for local conversation testing with mocked external services (except LLM and Weather).

## Overview

A command-line interface that enables local testing of Yuruppu conversations by mocking external services (LINE API, GCS) while using the real LLM and Weather API. Users can input messages via stdin and see bot responses in the terminal.

## Background & Purpose

- **Local development**: Developers need to test conversation flows without deploying to Cloud Run or setting up all external services
- **Faster iteration**: Testing with real LINE webhooks requires ngrok/deployment, which slows development
- **LLM behavior verification**: The primary goal is to verify how the LLM responds to various inputs, so the LLM must remain real while other services are mocked

## Out of Scope

- Mocking the LLM (Gemini) - must use real LLM for meaningful conversation testing
- Mocking the Weather API (wttr.in) - public API with no auth required
- GUI or web interface - CLI only
- Multi-user simulation - single user conversation only
- Media message handling (images, videos, audio) - text messages only for initial version

## Requirements

### Functional Requirements

- [ ] FR-001: CLI runs as interactive REPL with `> ` prompt, displaying bot responses immediately after each input
- [ ] FR-002: CLI supports `--message` flag for single-turn mode (send one message, print response, exit)
- [ ] FR-003: LINE API is mocked - replies are printed to stdout instead of sent via API
- [ ] FR-004: User ID is configurable via `--user-id` flag (default: "default"); must match pattern `[0-9a-z_]+`
- [ ] FR-005: For new user IDs, CLI prompts for all profile fields (display name required; picture URL, status message optional). If picture URL is provided, MIME type is fetched automatically
- [ ] FR-006: GCS storage is mocked with local filesystem storage for history and profiles
- [ ] FR-007: Conversation history persists between sessions (stored in local files)
- [ ] FR-008: Storage directory is configurable via `--data-dir` flag (default: `.yuruppu/` in current directory)
- [ ] FR-009: If storage directory does not exist, CLI prompts user for confirmation before creating it
- [ ] FR-010: CLI can be exited with `/quit` command or Ctrl+C twice in a row
- [ ] FR-011: CLI outputs verbose logs (tool calls, LLM processing, storage operations) to stderr

### Non-Functional Requirements

- [ ] NFR-001: CLI requires only LLM-related environment variables (GCP_PROJECT_ID, GCP_REGION, LLM_MODEL)
- [ ] NFR-002: No network calls to LINE API or GCS during operation

### Technical Requirements

- [ ] TR-001: Reuse existing interfaces (Handler, Storage, Agent, Tool) with mock implementations
- [ ] TR-002: CLI entry point is separate from the webhook server (e.g., `cmd/cli/main.go`)
- [ ] TR-003: CLI uses userID as sourceID (single-user mode)
- [ ] TR-004: Storage directory contains subdirectories per bucket (profiles, history, media)

## Acceptance Criteria

### AC-001: Interactive REPL mode [FR-001, FR-003]

- **Given**: CLI is started with valid LLM configuration
- **When**: User types "Hello" at the `> ` prompt and presses Enter
- **Then**:
  - Bot processes the message using real LLM
  - Bot response is displayed in stdout
  - Prompt `> ` appears again for next input

### AC-002: Single-turn message mode [FR-002]

- **Given**: CLI is started with `--message "Hello"`
- **When**: CLI processes the message
- **Then**:
  - Bot response is displayed in stdout
  - CLI exits automatically after response (no REPL)

### AC-003: Existing user profile [FR-004, FR-006]

- **Given**: CLI is started with `--user-id "user123"` and profile for "user123" exists
- **When**: CLI starts
- **Then**:
  - Profile is loaded from local filesystem
  - No name input prompt is shown
  - Conversation starts immediately

### AC-004: New user profile creation [FR-004, FR-005]

- **Given**: CLI is started with `--user-id "newuser"` and no profile exists for "newuser"
- **When**: CLI starts
- **Then**:
  - CLI prompts for display name (required)
  - CLI prompts for picture URL (optional, can skip with Enter)
  - CLI prompts for status message (optional, can skip with Enter)
  - Profile is saved to local filesystem
  - Conversation starts after profile creation

### AC-005: Empty display name rejection [FR-005]

- **Given**: CLI prompts for display name for a new user
- **When**: User presses Enter without typing a name
- **Then**:
  - CLI re-prompts for display name
  - Process repeats until non-empty name is provided

### AC-006: Invalid user ID rejection [FR-004]

- **Given**: CLI is started with `--user-id "User@123"` (contains invalid characters)
- **When**: CLI validates the user ID
- **Then**:
  - CLI displays error message about invalid user ID format
  - CLI exits with non-zero status

### AC-007: Conversation history persistence [FR-006, FR-007]

- **Given**: User had a previous conversation and restarted CLI with same user ID
- **When**: Conversation starts
- **Then**:
  - Previous messages are loaded from local filesystem
  - LLM can access conversation context

### AC-008: Storage directory creation prompt [FR-009]

- **Given**: CLI is started with `--data-dir "/tmp/new-dir"` and directory does not exist
- **When**: CLI starts
- **Then**:
  - CLI prompts "Directory /tmp/new-dir does not exist. Create it? [y/N]"
  - If user enters "y": directory is created and CLI continues
  - If user enters anything else: CLI exits

### AC-009: Graceful exit with /quit [FR-010]

- **Given**: CLI is running in REPL mode
- **When**: User types "/quit"
- **Then**:
  - CLI exits cleanly without errors
  - Any cleanup (LLM client close) is performed

### AC-011: Graceful exit with Ctrl+C twice [FR-010]

- **Given**: CLI is running in REPL mode
- **When**: User presses Ctrl+C twice in a row
- **Then**:
  - First Ctrl+C shows message (e.g., "Press Ctrl+C again to exit")
  - Second Ctrl+C exits CLI cleanly

### AC-010: Minimal configuration [NFR-001]

- **Given**: Only LLM environment variables are set (GCP_PROJECT_ID, GCP_REGION, LLM_MODEL)
- **When**: CLI is started
- **Then**:
  - CLI starts successfully without LINE or GCS credentials
  - Error messages do not reference missing LINE/GCS configuration

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-04 | 1.0 | Initial version | - |
| 2026-01-04 | 1.1 | Remove weather API mocking (use real API) | - |
| 2026-01-04 | 1.2 | Change storage from in-memory to local filesystem | - |
| 2026-01-04 | 1.3 | Add configurable storage directory (--data-dir flag) | - |
| 2026-01-04 | 1.4 | Replace --user-name with --user-id, add new user prompt | - |
| 2026-01-04 | 1.5 | Add REPL style, --message flag, /quit, directory prompt, user ID validation | - |
| 2026-01-04 | 1.6 | Profile prompts all fields, Ctrl+C twice to exit | - |
| 2026-01-04 | 1.7 | Add verbose logging, userID=sourceID, bucket subdirs, MIME type fetch | - |
