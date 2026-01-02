# ADR: Reply Tool for LLM Response Control

> Date: 2026-01-02
> Status: **Adopted**

## Context

The current implementation has the LLM return text directly, which is then sent as a LINE reply. For the optional LLM reply feature (spec: 20260102-feat-optional-llm-reply), we need a mechanism for the LLM to decide whether to send a reply.

## Decision Drivers

- LLM must be able to explicitly decide whether to reply
- Decision mechanism must be unambiguous (no false positives/negatives)
- Must integrate with existing tool-based architecture

## Options Considered

- **Option 1:** Empty response - LLM returns empty text to skip reply
- **Option 2:** Special marker text - LLM returns text like "[NO_REPLY]"
- **Option 3:** Thought-only response - LLM returns only internal reasoning, no visible text
- **Option 4:** Reply tool - LLM calls a dedicated tool to send a reply

## Evaluation

| Criterion | Empty Response | Special Marker | Thought-only | Reply Tool |
|-----------|----------------|----------------|--------------|------------|
| Explicitness | Low - ambiguous with errors | Medium - requires parsing | Medium - depends on model support | High - explicit tool call |
| Reliability | Low - empty could be error state | Medium - marker could leak | Medium - model behavior varies | High - tool call or not |
| Consistency | Low | Medium | Low | High - follows existing patterns |
| Implementation complexity | Low | Low | Medium | Medium |

## Decision

Adopt **Reply tool**.

## Rationale

The reply tool approach provides the clearest and most explicit mechanism for the LLM to indicate its intent. When the LLM wants to reply, it calls the `reply` tool with the message content. When it does not want to reply, it simply does not call the tool.

This approach:
- Aligns with the existing tool-based architecture
- Eliminates ambiguity (tool call present = reply, absent = no reply)
- Gives the LLM explicit control over the reply decision
- Makes the intent clear in logs and debugging

## Consequences

**Positive:**
- Clear, unambiguous mechanism for reply decisions
- Consistent with existing tool patterns in the codebase
- Easy to debug and trace

**Negative:**
- Requires refactoring current response handling
- All existing prompts must be updated to use the tool

**Risks:**
- LLM might not call the tool when it should; mitigation: clear system prompt instructions

## Related Decisions

- [20241214-line-bot-architecture.md](./20241214-line-bot-architecture.md)
