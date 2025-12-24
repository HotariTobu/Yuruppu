# ADR: LLM Provider Selection

> Date: 2025-12-24
> Status: **Adopted**

## Context

The Yuruppu LINE bot needs to replace echo behavior with LLM-generated responses (spec: 20251224-feat-llm-response). This requires selecting an LLM provider and model that integrates well with the existing Go/Cloud Run architecture.

## Decision Drivers

- Official Go SDK with active maintenance
- Single-turn text generation with system prompt support
- Simple authentication that works with Cloud Run
- Cost-effective for chat use case
- Low latency for real-time LINE messaging

## Options Considered

- **Option 1:** Vertex AI (Gemini models)
- **Option 2:** OpenAI (GPT models)
- **Option 3:** Anthropic Claude
- **Option 4:** Gemini AI Studio API

## Evaluation

| Criterion | Weight | Vertex AI | OpenAI | Claude | Gemini API |
|-----------|--------|-----------|--------|--------|------------|
| Functional Fit | 25% | 5 (1.25) | 5 (1.25) | 5 (1.25) | 5 (1.25) |
| Go Compatibility | 20% | 5 (1.00) | 4 (0.80) | 4 (0.80) | 4 (0.80) |
| Lightweight | 15% | 4 (0.60) | 3 (0.45) | 3 (0.45) | 3 (0.45) |
| Security | 15% | 5 (0.75) | 3 (0.45) | 4 (0.60) | 3 (0.45) |
| Documentation | 15% | 4 (0.60) | 5 (0.75) | 4 (0.60) | 4 (0.60) |
| Ecosystem | 10% | 4 (0.40) | 5 (0.50) | 4 (0.40) | 4 (0.40) |
| **Total** | 100% | **4.60** | **4.20** | **4.10** | **3.95** |

## Decision

Adopt **Vertex AI** with cost-optimized Gemini models (Flash tier).

## Rationale

1. **Native GCP integration**: Already deploying on Cloud Run; Vertex AI provides seamless authentication via Application Default Credentials without API key management.

2. **Superior Go SDK**: The official Go SDK scored highest in Go compatibility with idiomatic design, proper context support, and clean error handling.

3. **Security**: No API keys to manage or rotate. IAM-based access control integrates with existing GCP security posture.

4. **Enterprise reliability**: Vertex AI has better SLA guarantees than Gemini AI Studio's free tier (which had 61+ outages in 6 months).

5. **Cost-effective**: Cost-optimized model tier provides good performance at reasonable cost, with context caching for significant cost reduction on system prompts.

## Consequences

**Positive:**
- Zero-config authentication on Cloud Run
- Unified billing and monitoring via GCP console
- Access to latest Gemini models
- Built-in Cloud Monitoring dashboards

**Negative:**
- Vendor lock-in to GCP
- Slightly higher complexity than simple API key auth for local development

**Risks:**
- Model deprecation (mitigated by Google's deprecation policy and migration guides)
- Quota changes (mitigated by using Vertex AI's more stable enterprise quotas vs free tier)

## Related Decisions

- Testing strategy ADR - Mock-based testing applies to LLM client
- LINE bot architecture ADR
