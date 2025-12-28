# Refactor: Separate Provider and Agent Responsibilities

## Overview

Separate LLM Provider (pure API calls) from Agent (system prompt + caching management) to achieve single responsibility principle.

## Background & Purpose

The current `vertexAIClient` violates single responsibility principle by mixing:
- LLM API calls
- System prompt storage
- Cache creation/management/recreation
- Cache expiration handling

Provider should only handle API calls. A new Agent component should manage system prompt and caching.

## Current Structure

- `internal/llm/provider.go`: Defines Provider interface with `GenerateText(ctx, systemPrompt, userMessage)` and `Close(ctx)`
- `internal/llm/vertexai.go`: Implements Provider but also contains:
  - `systemPrompt` field for caching
  - `cacheName` field for cache management
  - `createCache()`, `isCacheError()`, `handleCacheErrorAndRetry()` methods
  - `NewVertexAIClientWithCache()` constructor that mixes caching concern
- `internal/yuruppu/handler.go`: Handler uses LLMProvider interface, passes SystemPrompt on every call

## Proposed Structure

### Provider (Pure API Layer)
- `internal/llm/provider.go`: Provider interface extended (ADR: 20251228-provider-cache-interface)
  - `GenerateText(ctx, systemPrompt, userMessage) (string, error)` - non-cached calls
  - `GenerateTextCached(ctx, cacheName, userMessage) (string, error)` - cached calls
  - `CreateCache(ctx, systemPrompt) (cacheName, error)` - cache creation
  - `DeleteCache(ctx, cacheName) error` - cache deletion
  - `Close(ctx) error`
- `internal/llm/vertexai.go`: Pure API implementation only
  - Remove `systemPrompt`, `cacheName`, `cacheRecreate` fields
  - Remove `NewVertexAIClientWithCache()`
  - Implement new cache methods (moved from current caching logic)

### Agent (System Prompt + Caching)
- `internal/llm/agent.go`: New file with Agent struct and interface
  - `Agent` interface: `GenerateText(ctx, userMessage) (string, error)`, `Close(ctx) error`
  - `NewAgent(provider Provider, systemPrompt string, logger) Agent`
  - Stores Provider reference (dependency injection)
  - Stores systemPrompt and cacheName
  - Manages cache lifecycle via Provider's cache methods
  - Calls `provider.GenerateTextCached()` when cache available, otherwise `provider.GenerateText()`

### Handler Integration
- `internal/yuruppu/handler.go`: Change LLMProvider interface
  - New signature: `GenerateText(ctx, userMessage) (string, error)`
  - Handler no longer passes SystemPrompt (Agent manages it)

## Resource Ownership

```
main.go creates:
  1. Provider (NewVertexAIClient)
  2. Agent (NewAgent with Provider)
  3. Handler (NewHandler with Agent)

On shutdown:
  1. Agent.Close() - cleans up cache only
  2. Provider.Close() - cleans up provider resources
```

Agent does NOT close Provider. Caller (main.go) is responsible for closing both.

## Cache Lifecycle

1. **Initialization**: Cache created in `NewAgent()`. If creation fails, Agent continues in fallback mode.
2. **During GenerateText**: If cache error detected, attempt recreation. If recreation fails, fall back to non-cached mode.
3. **Close**: Cache deleted in `Agent.Close()`.

Thread-safety: Concurrent cache recreation attempts are prevented using mutex.

## Scope

- [x] SC-001: Remove caching and system prompt logic from `internal/llm/vertexai.go`
- [x] SC-002: Create new `internal/llm/agent.go` with Agent interface and implementation
- [x] SC-003: Update `internal/yuruppu/handler.go` LLMProvider interface
- [x] SC-004: Update `main.go` to create Provider, Agent, and pass Agent to Handler
- [x] SC-005: Update tests for new structure

## Breaking Changes

- `NewVertexAIClientWithCache()` removed
- `yuruppu.LLMProvider` interface signature changes from `GenerateText(ctx, systemPrompt, userMessage)` to `GenerateText(ctx, userMessage)`
- Handler no longer accepts raw Provider; requires Agent

**Migration**: This is a breaking API change that must be deployed atomically. All changes (Provider refactor, Agent creation, Handler update) must be deployed together.

## Acceptance Criteria

### AC-001: Provider is Pure API Layer [Linked to SC-001]

- **Given**: Refactored Provider implementation
- **When**: Code is reviewed
- **Then**:
  - `vertexAIClient` has no `systemPrompt`, `cacheName`, `cacheRecreate` fields (no state for caching)
  - No `isCacheError()`, `handleCacheErrorAndRetry()` methods exist (no cache management logic)
  - `GenerateText(ctx, systemPrompt, userMessage)` passes system prompt directly to API each call
  - `GenerateTextCached(ctx, cacheName, userMessage)` uses provided cacheName directly
  - `CreateCache(ctx, systemPrompt)` creates cache and returns cacheName (no internal storage)
  - `DeleteCache(ctx, cacheName)` deletes specified cache (no internal state update)

### AC-002: Agent Interface Defined [Linked to SC-002]

- **Given**: New Agent component
- **When**: `internal/llm/agent.go` is created
- **Then**:
  - `Agent` interface defined with `GenerateText(ctx context.Context, userMessage string) (string, error)` and `Close(ctx context.Context) error`
  - `NewAgent(provider Provider, systemPrompt string, logger *slog.Logger) Agent` constructor exists (returns Agent, no error - cache failure is handled internally)
  - Agent stores Provider via dependency injection (does not create Provider)

### AC-003: Agent Manages Cache [Linked to SC-002]

- **Given**: Agent initialized with Provider and systemPrompt
- **When**: Cache operations occur
- **Then**:
  - Cache created during `NewAgent()` via `provider.CreateCache()` with 60-minute TTL
  - If initial cache creation fails, Agent operates in fallback mode (no error returned)
  - Agent calls `provider.GenerateTextCached()` when cacheName is set, otherwise `provider.GenerateText()`
  - Cache errors during `GenerateTextCached()` trigger automatic recreation via `provider.CreateCache()`
  - If recreation fails, falls back to non-cached mode for that call
  - Concurrent recreation attempts prevented by mutex
  - `Close()` deletes cache via `provider.DeleteCache()` (does not close Provider)
  - If cache deletion fails during `Close()`, error is logged but `Close()` completes successfully

### AC-004: Handler Uses Agent [Linked to SC-003]

- **Given**: Updated Handler
- **When**: Code is reviewed
- **Then**:
  - `LLMProvider` interface has signature `GenerateText(ctx context.Context, userMessage string) (string, error)`
  - Handler no longer imports or uses `SystemPrompt` directly
  - Handler calls `llm.GenerateText(ctx, userMessage)` without systemPrompt parameter
  - When Agent is closed, `GenerateText` returns `LLMClosedError`

### AC-005: Application Integration [Linked to SC-004]

- **Given**: Updated main.go
- **When**: Application starts
- **Then**:
  - Provider created with `llm.NewVertexAIClient()`
  - Agent created with `llm.NewAgent(provider, systemPrompt, logger)`
  - Handler created with Agent (not Provider)
  - On shutdown: `Agent.Close()` called, then `Provider.Close()` called

### AC-006: Test Coverage [Linked to SC-005]

- **Given**: Refactoring is complete
- **When**: Tests are run
- **Then**:
  - Provider tests verify pure API behavior (no caching)
  - Agent tests verify:
    - Initialization with/without successful cache creation
    - GenerateText with cache hit
    - GenerateText with cache miss and recreation
    - GenerateText with cache recreation failure (fallback)
    - Close() cleans up cache
  - Handler tests use mock Agent (not mock Provider)
  - Existing test files updated: `provider_test.go`, `context_cache_tdd_test.go`, `handler_test.go`

## Implementation Notes

- Agent's interface is intentionally different from Provider's interface
- SystemPrompt is embedded in `internal/yuruppu/yuruppu.go` via `go:embed`
- Yuruppu wrapper encapsulates Agent and system prompt

## Actual Implementation (Updated)

The implementation uses a 3-layer architecture for better separation of concerns:

```
internal/
  llm/
    provider.go      - LLM API (pure API layer)
  agent/
    agent.go         - Generic Agent (cache management)
  yuruppu/
    yuruppu.go       - Yuruppu character (wraps Agent, embeds prompt)
    handler.go       - Handler created from Yuruppu
```

Dependency flow: `yuruppu -> agent -> llm`

```go
// main.go
provider := llm.NewVertexAIClient(...)
yuruppuAgent := yuruppu.New(provider, logger)
handler := yuruppuAgent.NewHandler(client)
```

This differs from the original spec design but achieves better:
- **Reusability**: Generic `agent` package can be reused for other characters
- **Encapsulation**: System prompt embedded in Yuruppu (not passed from main)
- **Clear Layering**: Character -> behavior -> API separation

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-28 | 1.0 | Initial version | - |
| 2025-12-28 | 1.1 | Add interface design, resource ownership, cache lifecycle details | - |
| 2025-12-28 | 1.2 | Add error handling details, migration path | - |
| 2025-12-28 | 1.3 | Design phase: Extend Provider interface with cache methods (ADR: 20251228-provider-cache-interface) | - |
| 2025-12-28 | 1.4 | Implementation: Agent moved to `internal/agent`, Yuruppu wrapper introduced for 3-layer architecture | - |
