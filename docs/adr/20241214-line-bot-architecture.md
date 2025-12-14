# ADR: LINE Bot Architecture for Yuruppu

> Date: 2024-12-14
> Status: **Adopted**

## Context

We are building "Yuruppu", a LINE bot that responds as a character. The bot needs to:
- Receive messages from LINE users via webhook
- Generate character-appropriate responses using AI
- Reply through the LINE Messaging API

Key technical decisions needed:
1. Programming language and SDK
2. AI/LLM provider for response generation
3. Hosting platform

## Decision Drivers

- **Character consistency**: AI must generate responses matching Yuruppu's personality
- **Cost efficiency**: Minimize operational costs for a personal/hobby project
- **Development speed**: Quick iteration and deployment
- **Reliability**: Webhook must respond within LINE's timeout (30 seconds)
- **Maintainability**: Simple architecture, minimal operational overhead

## Options Considered

### Language
- **Option 1:** Go with line-bot-sdk-go v8
- **Option 2:** TypeScript/Node.js with @line/bot-sdk
- **Option 3:** Python with line-bot-sdk

### AI Provider
- **Option 1:** Gemini Flash Lite
- **Option 2:** Claude API (Haiku)
- **Option 3:** OpenAI GPT-4o-mini

### Hosting
- **Option 1:** Google Cloud Run
- **Option 2:** AWS Lambda
- **Option 3:** Cloudflare Workers
- **Option 4:** Self-hosted VPS

## Evaluation

### Language Evaluation

| Criterion | Weight | Go | Node.js | Python |
|-----------|--------|-----|---------|--------|
| Performance | 25% | 5 (1.25) | 3 (0.75) | 3 (0.75) |
| SDK Quality | 25% | 5 (1.25) | 5 (1.25) | 4 (1.00) |
| AI Integration | 20% | 4 (0.80) | 5 (1.00) | 5 (1.00) |
| Deployment | 15% | 5 (0.75) | 4 (0.60) | 4 (0.60) |
| Developer Familiarity | 15% | 4 (0.60) | 4 (0.60) | 4 (0.60) |
| **Total** | 100% | **4.65** | **4.20** | **3.95** |

### AI Provider Evaluation

| Criterion | Weight | Gemini Flash Lite | Claude Haiku | GPT-4o-mini |
|-----------|--------|-------------------|--------------|-------------|
| Cost | 30% | 5 (1.50) | 4 (1.20) | 4 (1.20) |
| Response Quality | 30% | 4 (1.20) | 5 (1.50) | 4 (1.20) |
| Latency | 20% | 5 (1.00) | 4 (0.80) | 4 (0.80) |
| Free Tier | 20% | 5 (1.00) | 3 (0.60) | 3 (0.60) |
| **Total** | 100% | **4.70** | **4.10** | **3.80** |

### Hosting Evaluation

| Criterion | Weight | Cloud Run | Lambda | CF Workers | VPS |
|-----------|--------|-----------|--------|------------|-----|
| Go Support | 25% | 5 (1.25) | 4 (1.00) | 2 (0.50) | 5 (1.25) |
| Cost | 25% | 5 (1.25) | 4 (1.00) | 5 (1.25) | 2 (0.50) |
| Scale to Zero | 20% | 5 (1.00) | 5 (1.00) | 5 (1.00) | 1 (0.20) |
| Cold Start | 15% | 4 (0.60) | 4 (0.60) | 5 (0.75) | 5 (0.75) |
| Gemini Integration | 15% | 5 (0.75) | 3 (0.45) | 3 (0.45) | 4 (0.60) |
| **Total** | 100% | **4.85** | **4.05** | **3.95** | **3.30** |

## Decision

Adopt the following architecture:

| Component | Choice |
|-----------|--------|
| **Language** | Go with line-bot-sdk-go v8 |
| **AI Provider** | Gemini Flash Lite |
| **Hosting** | Google Cloud Run |

## Rationale

### Go + line-bot-sdk-go
- Official SDK with type-safe interfaces and comprehensive API coverage
- Excellent performance with low memory footprint
- Single binary deployment simplifies container builds
- Strong ecosystem synergy with Google Cloud

### Gemini Flash Lite
- Best cost efficiency for character chat use case
- Very low latency suitable for real-time chat
- Generous free tier (up to 1500 requests/day)
- Native integration with Google Cloud

### Google Cloud Run
- First-class Go support (Go is Google's language)
- Scales to zero when not in use, minimizing costs
- Same ecosystem as Gemini API (simpler auth, lower latency)
- 2 million requests/month free tier
- Easy deployment with just a Dockerfile

## Consequences

**Positive:**
- All components from Google ecosystem = simpler integration and auth
- Cost-efficient for low-to-medium traffic
- Fast cold starts with Go (< 500ms)
- Type-safe development experience

**Negative:**
- Vendor lock-in to Google Cloud ecosystem
- Gemini Flash Lite may produce less nuanced responses than larger models
- Need to learn Go if unfamiliar

**Risks:**
- Gemini API rate limits during traffic spikes
  - Mitigation: Implement retry with exponential backoff
- Cloud Run cold start affecting webhook response time
  - Mitigation: Go cold starts are fast; enable min-instances if needed

## Confirmation

Success criteria:
- Webhook responds within 5 seconds under normal load
- Monthly cost stays under $10 for expected traffic
- Character responses feel natural and consistent

## Architecture Overview

```
┌─────────────┐     Webhook      ┌─────────────────┐
│    LINE     │ ───────────────→ │  Cloud Run      │
│   Platform  │                  │  (Go)           │
│             │ ←─────────────── │                 │
└─────────────┘   Reply Message  │  - line-bot-sdk │
                                 │  - webhook処理   │
                                 └────────┬────────┘
                                          │
                                          │ API Call
                                          ▼
                                 ┌─────────────────┐
                                 │  Gemini API     │
                                 │  (Flash Lite)   │
                                 │                 │
                                 │  キャラクター応答  │
                                 └─────────────────┘
```

## Related Decisions

- None (first ADR)

## Resources

| Component | Documentation | Repository |
|-----------|---------------|------------|
| line-bot-sdk-go | [docs](https://developers.line.biz/en/docs/messaging-api/) | [repo](https://github.com/line/line-bot-sdk-go) |
| Gemini API | [docs](https://ai.google.dev/docs) | - |
| Cloud Run | [docs](https://cloud.google.com/run/docs) | - |

## Sources

- LINE Messaging API Overview: https://developers.line.biz/en/docs/messaging-api/overview/
- Google Cloud Run Pricing: https://cloud.google.com/run/pricing
- Gemini API Pricing: https://ai.google.dev/pricing
