# ADR: Chat History Storage Backend

> Date: 2025-12-28
> Status: **Adopted**

## Context

The chat-history feature requires storing conversation history per SourceID (user/group/room) and retrieving it for LLM context. Storage operations should not significantly impact message processing latency (+100ms guideline per NFR-001).

## Decision Drivers

- Simple integration with existing GCP Cloud Run deployment
- Low operational overhead (no VPC, no provisioned infrastructure)
- Cost-effective for small-scale LINE bot
- Sufficient performance for chat history retrieval

## Options Considered

- **Option 1:** Google Cloud Storage (GCS)
- **Option 2:** Memorystore Redis
- **Option 3:** Firestore
- **Option 4:** Upstash
- **Option 5:** Bigtable

## Evaluation

See `evaluation-criteria.md` for criteria definitions.

| Criterion | Weight | GCS | Memorystore | Firestore | Upstash | Bigtable |
|-----------|--------|-----|-------------|-----------|---------|----------|
| Functional Fit | 25% | 2 (0.50) | 5 (1.25) | 2 (0.50) | 5 (1.25) | 5 (1.25) |
| Go Compatibility | 20% | 5 (1.00) | 5 (1.00) | 5 (1.00) | 3 (0.60) | 5 (1.00) |
| Lightweight | 15% | 3 (0.45) | 3 (0.45) | 2 (0.30) | 2 (0.30) | 2 (0.30) |
| Security | 15% | 5 (0.75) | 5 (0.75) | 5 (0.75) | 5 (0.75) | 5 (0.75) |
| Documentation | 15% | 5 (0.75) | 4 (0.60) | 4 (0.60) | 3 (0.45) | 4 (0.60) |
| Ecosystem | 10% | 5 (0.50) | 5 (0.50) | 5 (0.50) | 4 (0.40) | 5 (0.50) |
| **Total** | 100% | **3.95** | **4.55** | **3.65** | **3.75** | **4.40** |

## Decision

Adopt **Google Cloud Storage**.

## Rationale

- **Simplicity**: No VPC configuration, no provisioned infrastructure, zero operational overhead
- **Cost**: Extremely low cost (~$0.02/GB/month + minimal operation fees) vs Memorystore ($33+/month) or Bigtable ($476+/month)
- **Sufficient performance**: 60-120ms p99 latency is acceptable given LLM response takes seconds
- **Native GCP integration**: Automatic IAM authentication from Cloud Run, no credential management
- **Strong consistency**: Generation-based preconditions prevent race conditions in read-modify-write

Trade-off accepted: Objects are immutable, requiring read-modify-write pattern for appends. This is acceptable for the expected message volume.

## Consequences

**Positive:**
- Zero infrastructure management
- Minimal cost for small-scale usage
- Strong consistency guarantees with preconditions
- Automatic encryption at rest and in transit

**Negative:**
- Read-modify-write pattern adds complexity vs native append
- Higher latency than in-memory solutions (acceptable trade-off)
- Must handle 412 Precondition Failed for concurrent writes

**Risks:**
- If message volume grows significantly, may need to revisit (migrate to Redis)
- Mitigation: Design storage interface to allow backend swap

## Related Decisions

- [20251224-llm-provider.md](./20251224-llm-provider.md)

## Resources

| Option | Documentation | Repository |
|--------|---------------|------------|
| Cloud Storage | [docs](https://cloud.google.com/storage/docs) | [google-cloud-go](https://github.com/googleapis/google-cloud-go) |
| Memorystore | [docs](https://cloud.google.com/memorystore/docs/redis) | [go-redis](https://github.com/redis/go-redis) |

## Sources

- [Cloud Storage Go SDK](https://pkg.go.dev/cloud.google.com/go/storage)
- [Cloud Storage Consistency](https://cloud.google.com/storage/docs/consistency)
- [Request Preconditions](https://cloud.google.com/storage/docs/request-preconditions)
