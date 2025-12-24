# ADR: Gemini Model Selection

> Date: 2025-12-25
> Status: **Adopted**

## Context

With Vertex AI selected as the LLM provider (ADR 20251224-llm-provider.md), we need to choose which Gemini model to use for the Yuruppu LINE bot. The model affects response quality, latency, and cost.

## Decision Drivers

- Low latency for real-time LINE messaging
- Cost-effective for potentially high message volume
- Good conversational quality for character-based responses
- Support for system prompts

## Options Considered

- **Option 1:** Gemini 2.5 Flash
- **Option 2:** Gemini 2.5 Flash-Lite
- **Option 3:** Gemini 3 Flash

## Evaluation

| Criterion | Weight | Flash | Flash-Lite | 3 Flash |
|-----------|--------|-------|------------|---------|
| Response Quality | 30% | 4 (1.20) | 3 (0.90) | 3 (0.90) |
| Latency | 30% | 3.5 (1.05) | 5 (1.50) | 4 (1.20) |
| Cost Efficiency | 25% | 4.5 (1.13) | 5 (1.25) | 3 (0.75) |
| Stability | 15% | 4.5 (0.68) | 3.5 (0.53) | 2 (0.30) |
| **Total** | 100% | **4.06** | **4.18** | **3.15** |

Note: All Gemini 2.5 models have a known truncation bug (P2 priority, unresolved). Gemini 3 Flash has a 91% hallucination rate and is in preview status.

## Decision

Adopt **Gemini 2.5 Flash-Lite**.

## Rationale

1. **Fastest latency**: Flash-Lite offers the lowest time-to-first-token among the options, ideal for real-time LINE messaging where users expect quick responses.

2. **Lowest cost**: Significantly cheaper than Flash and 3 Flash, suitable for potentially high message volume.

3. **Acceptable quality trade-off**: While response quality is lower than Flash, it is adequate for a character chatbot with simple interactions.

4. **Known issues accepted**: The truncation bug affects all Gemini 2.5 models equally, so choosing Flash-Lite doesn't introduce additional risk compared to Flash.

## Consequences

**Positive:**
- Fastest response times for LINE users
- Lowest operating cost
- Same SDK and integration as other Gemini models

**Negative:**
- Lower response quality than Flash
- Truncation bug may cause incomplete responses
- May need to upgrade to Flash if quality is insufficient

**Mitigations:**
- Handle truncation bug at implementation level
- Prepare fallback to Gemini 2.5 Flash if quality is insufficient

## Related Decisions

- [20251224-llm-provider.md](./20251224-llm-provider.md) - Vertex AI provider selection
