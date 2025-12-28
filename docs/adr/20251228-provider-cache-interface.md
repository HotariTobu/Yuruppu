# ADR: Provider Interface Design for Cache Support

> Date: 2025-12-28
> Status: **Adopted**

## Context

The LLM Provider/Agent separation (spec: 20251228-refact-llm-agent-separation) requires Agent to manage cache lifecycle while Provider handles API calls. Agent needs a way to use cached content through the Provider interface.

## Decision Drivers

- Provider interface should remain a pure API abstraction layer
- Agent needs to call Provider with either systemPrompt or cacheName
- API intent should be clear and unambiguous
- No parameters that are conditionally ignored

## Options Considered

- **Option A:** Add cacheName parameter to existing GenerateText
- **Option B:** Use Options struct for flexible configuration
- **Option C:** Separate methods for cached and non-cached calls
- **Option D:** Agent uses concrete type directly (bypass interface)

## Evaluation

| Criterion | Option A | Option B | Option C | Option D |
|-----------|----------|----------|----------|----------|
| API Clarity | Poor (param ignored) | Medium (verbose) | Excellent | Poor (breaks abstraction) |
| Simplicity | Medium | Poor | Good | Medium |
| Type Safety | Good | Good | Excellent | Poor |
| Extensibility | Poor | Excellent | Good | Poor |

## Decision

Adopt **Option C: Separate methods**.

The Provider interface will have five methods:
- `GenerateText` for non-cached calls (accepts system prompt and user input)
- `GenerateTextCached` for cached calls (accepts cache reference and user input)
- `CreateCache` to create a cache from system prompt content
- `DeleteCache` to delete a cache
- `Close` for resource cleanup

## Rationale

1. **Clear intent**: Each method has a single purpose. No ambiguity about which parameters are used.

2. **Type safety**: Cannot accidentally pass wrong combination of parameters.

3. **Simple Agent logic**: Agent checks if cacheName exists and calls the appropriate method. No conditional parameter handling.

4. **Pure abstraction**: Provider interface defines what operations are available, not implementation details.

## Consequences

**Positive:**
- Explicit API contract
- Easy to understand and use correctly
- Each method is independently testable

**Negative:**
- Two code paths in Provider implementation
- Slightly larger interface surface

**Risks:**
- None significant. Two methods are simpler than conditional logic.

## Related Decisions

- [20251224-llm-provider.md](./20251224-llm-provider.md) - Vertex AI selection
