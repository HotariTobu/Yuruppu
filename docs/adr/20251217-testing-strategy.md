# ADR: Testing Strategy

> Date: 2025-12-17
> Status: **Adopted**

## Context

Need to test the LINE bot without calling real LINE API. The spec has 9 acceptance criteria including verifying reply content and ensuring non-text messages are ignored.

## Decision Drivers

- Need to verify ReplyMessage is called with correct content (AC-001)
- Need to verify ReplyMessage is NOT called for non-text (AC-004)
- Minimal dependencies preferred
- LINE Bot SDK doesn't provide test utilities

## Options Considered

- **Option 1:** Pure functions + httptest only
- **Option 2:** Interface + manual mock (no mock library)
- **Option 3:** Interface + mock library (gomock, testify/mock)

## Decision

Adopt **Option 2: Interface + manual mock**.

## Rationale

- Can verify ReplyMessage calls (content and count)
- No external dependencies
- Small overhead for simple bot

### Implementation

```go
// Interface for LINE message sending
type MessageSender interface {
    ReplyMessage(*messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error)
}

// Production: use real messaging_api.MessagingApiAPI
// Tests: use mock implementation

type mockSender struct {
    calls   []*messaging_api.ReplyMessageRequest
}

func (m *mockSender) ReplyMessage(req *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error) {
    m.calls = append(m.calls, req)
    return &messaging_api.ReplyMessageResponse{}, nil
}
```

### Test Coverage

| Function | Test Method |
|----------|-------------|
| FormatEchoMessage | Direct unit test |
| HandleWebhook | httptest + mock sender |
| HandleTextMessage | Mock sender |
| NewBot | Integration test only |

## Consequences

**Positive:**
- Full test coverage possible
- No external dependencies
- Clear separation of concerns

**Negative:**
- Requires interface abstraction
- Mock needs manual maintenance

## Related Decisions

- [20251217-project-structure.md](./20251217-project-structure.md)