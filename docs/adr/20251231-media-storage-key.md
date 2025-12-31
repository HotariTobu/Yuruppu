# ADR: Media Storage Key Naming Convention

> Date: 2025-12-31
> Status: **Adopted**

## Context

When users send images via LINE, the bot needs to store them persistently in cloud storage (GCS) for later retrieval when building AI agent context. A consistent naming convention for storage keys is needed.

## Decision Drivers

- Keys should be unique and collision-free
- Keys should be sortable chronologically within a conversation
- Keys should allow grouping by conversation (sourceID)
- Keys should be simple and not redundant (MIME type stored separately)

## Options Considered

- **Option 1:** `/{sourceID}/{messageID}` - Groups by conversation, uses LINE message ID
- **Option 2:** `/{messageID}` - Flat structure, relies on LINE message ID uniqueness
- **Option 3:** `/{sourceID}/{uuidv7}` - Groups by conversation, uses time-ordered UUID (no extension)

## Evaluation

| Criterion | Option 1 | Option 2 | Option 3 |
|-----------|----------|----------|----------|
| Uniqueness | Yes (LINE guarantees) | Yes | Yes (cryptographic) |
| Chronological sorting | No | No | Yes |
| Grouping by conversation | Yes | No | Yes |
| Independence from LINE | No | No | Yes |

## Decision

Adopt **Option 3**: `/{sourceID}/{uuidv7}`

Format: `/{sourceID}/{uuidv7}`

- `sourceID`: LINE user ID or group ID
- `uuidv7`: Time-ordered UUID (RFC 9562)
- MIME type is stored separately in conversation history, not in the key

## Rationale

UUIDv7 provides time-ordered uniqueness without depending on LINE's message ID format. This enables:
- Chronological listing of media within a conversation
- Future extensibility if media comes from sources other than LINE
- Standard UUID format familiar to developers

UUIDv7 is a standard format with broad language/library support.

## Consequences

**Positive:**
- Media files are naturally sorted by upload time
- Easy to list/cleanup media by conversation
- No dependency on external ID format

**Negative:**
- Need to track mapping between messageID and storage key if needed later

## Related Decisions

- [20251228-chat-history-storage.md](./20251228-chat-history-storage.md)
