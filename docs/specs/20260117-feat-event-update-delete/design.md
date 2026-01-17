# Design: Event Update and Delete Tools

## Overview

Add `update_event` and `delete_event` tools to complete the event management lifecycle. Both tools operate on the current chat room's event and require creator authorization.

## File Structure

| File | Purpose |
|------|---------|
| `internal/event/event.go` | Add `Update()` and `Delete()` methods to Service |
| `internal/event/event_test.go` | Add tests for Update/Delete service methods |
| `internal/toolset/event/update/update.go` | update_event tool implementation |
| `internal/toolset/event/update/update_test.go` | update_event tool tests |
| `internal/toolset/event/update/parameters.json` | Input schema for update_event |
| `internal/toolset/event/update/response.json` | Output schema for update_event |
| `internal/toolset/event/remove/remove.go` | delete_event tool implementation |
| `internal/toolset/event/remove/remove_test.go` | delete_event tool tests |
| `internal/toolset/event/remove/parameters.json` | Input schema for delete_event |
| `internal/toolset/event/remove/response.json` | Output schema for delete_event |
| `internal/toolset/event/event.go` | Extend EventService interface and NewTools() |
| `internal/toolset/event/event_test.go` | Update mock and test expectations |

## Interfaces

### Event Service Methods

```go
// Update updates the description of an existing event.
// Returns error if the event is not found or if storage operations fail.
func (s *Service) Update(ctx context.Context, chatRoomID string, description string) error

// Delete removes an event from storage.
// Returns error if the event is not found or if storage operations fail.
func (s *Service) Delete(ctx context.Context, chatRoomID string) error
```

### EventService Interface (toolset)

```go
type EventService interface {
    Create(ctx context.Context, ev *event.Event) error
    Get(ctx context.Context, chatRoomID string) (*event.Event, error)
    List(ctx context.Context, opts event.ListOptions) ([]*event.Event, error)
    Update(ctx context.Context, chatRoomID string, description string) error
    Delete(ctx context.Context, chatRoomID string) error
}
```

### update_event Tool

```go
type Tool struct {
    eventService EventService
    logger       *slog.Logger
}

func New(eventService EventService, logger *slog.Logger) (*Tool, error)
func (t *Tool) Name() string                    // "update_event"
func (t *Tool) Description() string             // LLM description
func (t *Tool) ParametersJsonSchema() []byte    // embedded parameters.json
func (t *Tool) ResponseJsonSchema() []byte      // embedded response.json
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error)
```

### delete_event Tool

```go
type Tool struct {
    eventService EventService
    logger       *slog.Logger
}

func New(eventService EventService, logger *slog.Logger) (*Tool, error)
func (t *Tool) Name() string                    // "delete_event"
func (t *Tool) Description() string             // LLM description
func (t *Tool) ParametersJsonSchema() []byte    // embedded parameters.json
func (t *Tool) ResponseJsonSchema() []byte      // embedded response.json
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error)
```

## JSON Schemas

### update_event parameters.json

```json
{
  "type": "object",
  "properties": {
    "description": {
      "type": "string",
      "description": "New description for the event",
      "minLength": 1,
      "maxLength": 2000
    }
  },
  "required": ["description"],
  "additionalProperties": false
}
```

### update_event response.json

```json
{
  "type": "object",
  "properties": {
    "chat_room_id": {
      "type": "string",
      "description": "ID of the chat room where the event was updated"
    }
  },
  "required": ["chat_room_id"],
  "additionalProperties": false
}
```

### delete_event parameters.json

```json
{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}
```

### delete_event response.json

```json
{
  "type": "object",
  "properties": {
    "chat_room_id": {
      "type": "string",
      "description": "ID of the chat room where the event was deleted"
    }
  },
  "required": ["chat_room_id"],
  "additionalProperties": false
}
```

## Data Flow

### update_event

```
1. Input: context (sourceID, userID), args {description}
2. Extract sourceID from context (current chat room)
3. Extract userID from context (requesting user)
4. Call eventService.Get(sourceID) to fetch event
5. Authorization: Check event.CreatorID == userID
   - If not: Return error "only the event creator can update the event"
6. Call eventService.Update(sourceID, description)
   - Service reads events with generation
   - Service updates description field
   - Service writes back with generation check (NFR-001: atomicity)
7. Output: {chat_room_id: sourceID}
```

### delete_event

```
1. Input: context (sourceID, userID), args {} (empty)
2. Extract sourceID from context (current chat room)
3. Extract userID from context (requesting user)
4. Call eventService.Get(sourceID) to fetch event
   - If not found: Return error "event not found"
5. Authorization: Check event.CreatorID == userID
   - If not: Return error "only the event creator can delete the event"
6. Call eventService.Delete(sourceID)
   - Service reads events with generation
   - Service filters out the event (hard delete)
   - Service writes back with generation check (NFR-001: atomicity)
7. Output: {chat_room_id: sourceID}
```

## Error Messages

| Scenario | Error Message |
|----------|---------------|
| Context missing sourceID/userID | "internal error" |
| Event not found in chat room | "event not found" |
| User is not event creator | "only the event creator can update/delete the event" |
| Service operation fails | "failed to update/delete event" |

## Tool Descriptions (for LLM)

- **update_event**: "Use this tool to update the event description in the current group chat. Only the event creator can update the event."
- **delete_event**: "Use this tool to delete (cancel) the event in the current group chat. Only the event creator can delete the event."

## Requirements Coverage

| Requirement | How Addressed |
|-------------|---------------|
| FR-001 | Tools registered with descriptive names and descriptions for LLM |
| FR-002 | Update tool accepts description parameter |
| FR-003 | Only description field is updated in service layer |
| FR-004 | Tool checks CreatorID == userID before update |
| FR-005 | Tool uses sourceID from context (current chat room) |
| FR-006 | Service returns error if event not found |
| FR-007 | Delete tool removes event |
| FR-008 | Tool checks CreatorID == userID before delete |
| FR-009 | Tool uses sourceID from context (current chat room) |
| FR-010 | Service returns error if event not found |
| FR-011 | Service physically removes event from storage |
| NFR-001 | Service uses optimistic locking (read-modify-write with generation) |
