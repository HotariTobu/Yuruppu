# Feature: LLM Context Summarization

> Reduce LLM context consumption and costs by summarizing old conversation history.

## Overview

Summarize conversation history older than 1 week into a single summary, reducing the amount of context passed to the LLM while preserving conversational continuity.

## Background & Purpose

- Current implementation loads and passes the entire conversation history to the LLM for each request
- As conversations grow, context consumption and API costs increase
- Most relevant context is in recent messages; older messages can be summarized without significant quality loss

## Out of Scope

- Automatic deletion of old history (history is preserved, just summarized for LLM context)
- User-facing summary display
- Configurable time window (fixed at 1 week for initial implementation)

## Requirements

### Functional Requirements

- [ ] FR-001: Store summaries in a separate bucket (`summary`) from conversation history
- [ ] FR-002: When generating a response, use summary + messages from the past 1 week as context
- [ ] FR-003: Generate summary asynchronously after returning the response to user
- [ ] FR-004: Load summary and history in parallel to minimize latency

### Non-Functional Requirements

- [ ] NFR-001: Response latency should not increase due to summary generation
- [ ] NFR-002: Summary consistency is best-effort (eventual consistency acceptable)

## Acceptance Criteria

### AC-001: Summary storage [FR-001]

- **Given**: A conversation with history older than 1 week
- **When**: Summary is generated
- **Then**:
  - Summary is stored in the `summary` bucket
  - Summary is keyed by sourceID (same as history)

### AC-002: Context building with summary [FR-002]

- **Given**: A conversation with existing summary and recent messages
- **When**: LLM context is built for response generation
- **Then**:
  - Summary is included as initial context
  - Only messages from the past 1 week are included
  - Messages older than 1 week are excluded from context

### AC-003: Async summary generation [FR-003]

- **Given**: A conversation with messages older than 1 week and no existing summary
- **When**: Response is generated and returned to user
- **Then**:
  - Response is returned without waiting for summary generation
  - Summary generation happens in background goroutine
  - Generated summary is saved to summary bucket

### AC-004: Parallel loading [FR-004]

- **Given**: A request requiring both summary and history
- **When**: Context is loaded
- **Then**:
  - Summary and history are fetched concurrently
  - Neither fetch blocks the other

### AC-005: New conversation without summary [FR-002]

- **Given**: A new conversation with no history older than 1 week
- **When**: LLM context is built
- **Then**:
  - All messages are included in context
  - No summary is used (empty/missing summary is acceptable)

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-03 | 0.1 | Initial draft | - |
