# Feature: Mock External Services CLI

> CLI tool for local conversation testing with mocked external services (except LLM).

## Overview

A command-line interface that enables local testing of Yuruppu conversations by mocking external services (LINE API, GCS, Weather API) while using the real LLM. Users can input messages via stdin and see bot responses in the terminal.

## Background & Purpose

- **Local development**: Developers need to test conversation flows without deploying to Cloud Run or setting up all external services
- **Faster iteration**: Testing with real LINE webhooks requires ngrok/deployment, which slows development
- **LLM behavior verification**: The primary goal is to verify how the LLM responds to various inputs, so the LLM must remain real while other services are mocked

## Out of Scope

- Mocking the LLM (Gemini) - must use real LLM for meaningful conversation testing
- GUI or web interface - CLI only
- Multi-user simulation - single user conversation only
- Media message handling (images, videos, audio) - text messages only for initial version
- Persistent storage between sessions - in-memory only

## Requirements

### Functional Requirements

- [ ] FR-001: CLI accepts text input from stdin and displays bot responses to stdout
- [ ] FR-002: LINE API is mocked - replies are printed to stdout instead of sent via API
- [ ] FR-003: User profile is mocked with configurable display name
- [ ] FR-004: GCS storage is mocked with in-memory storage for history and profiles
- [ ] FR-005: Weather tool returns mock weather data (configurable or fixed response)
- [ ] FR-006: Conversation history is maintained within a session (in-memory)
- [ ] FR-007: CLI can be exited gracefully with Ctrl+C or "exit" command

### Non-Functional Requirements

- [ ] NFR-001: CLI requires only LLM-related environment variables (GCP_PROJECT_ID, GCP_REGION, LLM_MODEL)
- [ ] NFR-002: No network calls to LINE API, GCS, or weather API during operation

### Technical Requirements

- [ ] TR-001: Reuse existing interfaces (Handler, Storage, Agent, Tool) with mock implementations
- [ ] TR-002: CLI entry point is separate from the webhook server (e.g., `cmd/cli/main.go`)

## Acceptance Criteria

### AC-001: Basic conversation flow [FR-001, FR-002]

- **Given**: CLI is started with valid LLM configuration
- **When**: User types "Hello" and presses Enter
- **Then**:
  - Bot processes the message using real LLM
  - Bot response is displayed in stdout
  - No HTTP calls to LINE API are made

### AC-002: Mock user profile [FR-003]

- **Given**: CLI is started with `--user-name "TestUser"` flag
- **When**: Bot accesses user profile
- **Then**:
  - Profile contains display name "TestUser"
  - No calls to LINE API GetProfile are made

### AC-003: Conversation history within session [FR-004, FR-006]

- **Given**: User has sent multiple messages in the session
- **When**: User asks "What did I say earlier?"
- **Then**:
  - LLM can access previous messages from in-memory history
  - Response reflects conversation context

### AC-004: Weather tool mock [FR-005]

- **Given**: CLI is running
- **When**: User asks about weather and bot invokes weather tool
- **Then**:
  - Mock weather data is returned (e.g., "Sunny, 20°C")
  - No HTTP calls to wttr.in are made

### AC-005: Graceful exit [FR-007]

- **Given**: CLI is running
- **When**: User types "exit" or presses Ctrl+C
- **Then**:
  - CLI exits cleanly without errors
  - Any cleanup (LLM client close) is performed

### AC-006: Minimal configuration [NFR-001]

- **Given**: Only LLM environment variables are set (GCP_PROJECT_ID, GCP_REGION, LLM_MODEL)
- **When**: CLI is started
- **Then**:
  - CLI starts successfully without LINE or GCS credentials
  - Error messages do not reference missing LINE/GCS configuration

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-04 | 1.0 | Initial version | - |
