# Design: event-management

## Overview

Add event management tools (create, get, list) to the bot. All events are stored in a single JSONL file (`events.jsonl`) in GCS. Optimistic locking via `expectedGeneration` ensures safe concurrent updates.

## Environment Variables

| Name | Description | Default |
|------|-------------|---------|
| `EVENT_LIST_MAX_PERIOD_DAYS` | Max period in days for list_events when both start and end specified | `365` |
| `EVENT_LIST_LIMIT` | Max items for list_events when period not fully specified | `5` |

## File Structure

| File | Purpose |
|------|---------|
| `internal/event/event.go` | Event struct and Service |
| `internal/event/event_test.go` | Service unit tests |
| `internal/toolset/event/create/create.go` | Event creation tool |
| `internal/toolset/event/create/create_test.go` | Creation tool tests |
| `internal/toolset/event/create/parameters.json` | Creation parameters schema |
| `internal/toolset/event/create/response.json` | Creation response schema |
| `internal/toolset/event/get/get.go` | Event detail retrieval tool |
| `internal/toolset/event/get/get_test.go` | Get tool tests |
| `internal/toolset/event/get/parameters.json` | Get parameters schema |
| `internal/toolset/event/get/response.json` | Get response schema |
| `internal/toolset/event/list/list.go` | Event list retrieval tool |
| `internal/toolset/event/list/list_test.go` | List tool tests |
| `internal/toolset/event/list/parameters.json` | List parameters schema |
| `internal/toolset/event/list/response.json` | List response schema |
| `internal/toolset/event/event.go` | Factory to create all event tools |
| `main.go` | Register event tools with agent |
| `infra/variables.tf` | Add event variables (at end) |
| `infra/main.tf` | Pass event env vars to Cloud Run (at end of env) |

## Interfaces

### Event (event.go)

```go
type Event struct {
    ChatRoomID  string    `json:"chatRoomId"`
    CreatorID   string    `json:"creatorId"`
    Title       string    `json:"title"`
    StartTime   time.Time `json:"startTime"`
    EndTime     time.Time `json:"endTime"`
    Fee         string    `json:"fee"`
    Capacity    int       `json:"capacity"`
    Description string    `json:"description"`
    ShowCreator bool      `json:"showCreator"`
}

type ListOptions struct {
    CreatorID *string    // Filter by creator (nil = no filter)
    Start     *time.Time // Filter events with StartTime >= this time
    End       *time.Time // Filter events with StartTime <= this time
    Limit     int        // Max items to return (0 = no limit)
}

type Service struct {
    storage storage.Storage
}

func NewService(s storage.Storage) (*Service, error)
func (s *Service) Create(ctx context.Context, ev *Event) error
func (s *Service) List(ctx context.Context, opts ListOptions) ([]*Event, error)
func (s *Service) Get(ctx context.Context, chatRoomID string) (*Event, error)

// List behavior:
// - Sort: Start only or Start+End → ascending by StartTime
//         End only → descending by StartTime
// - Limit applied after sort
```

### Tool Interfaces

```go
// internal/toolset/event/create/create.go
type EventService interface {
    Create(ctx context.Context, ev *event.Event) error
}
func New(eventService EventService) (*Tool, error)

// internal/toolset/event/get/get.go
type EventService interface {
    Get(ctx context.Context, chatRoomID string) (*event.Event, error)
}
type ProfileService interface {
    GetDisplayName(ctx context.Context, userID string) (string, error)
}
func New(eventService EventService, profileService ProfileService) (*Tool, error)

// internal/toolset/event/list/list.go
type EventService interface {
    List(ctx context.Context, opts event.ListOptions) ([]*event.Event, error)
}
func New(eventService EventService, maxPeriodDays, limit int) (*Tool, error)

// internal/toolset/event/event.go
func NewTools(eventService *event.Service, profileService ProfileService, listMaxPeriodDays, listLimit int) ([]*Tool, error)
```

### LLM Response Structs

```go
// Response for get_event tool
type GetEventResponse struct {
    CreatorName *string
    Title       string
    StartTime   string    // JST RFC3339
    EndTime     string    // JST RFC3339
    Fee         string
    Capacity    int
    Description string
}

// Response for list_events tool (limited fields per FR-016)
type ListEventResponse struct {
    ChatRoomID string
    Title      string
    StartTime  string    // JST RFC3339
    EndTime    string    // JST RFC3339
    Fee        string
}
```

## Data Flow

### create_event

```
1. Receive tool callback
   args: title, start_time, end_time, capacity, fee, description, show_creator

2. Get sourceID, userID from context
   sourceID == userID → error (group chat only)

3. Validation
   now < start_time, start_time < end_time

4. Create Event struct
   CreatorID = userID

5. service.Create(ctx, event)
   - Read events.jsonl → get all events + generation
   - Check if ChatRoomID already exists → error if exists (FR-004)
   - Append new event line
   - Write with expectedGeneration (optimistic lock)

6. Return response
   success: true/false, chat_room_id or error
```

### get_event

```
1. Receive tool callback
   args: chat_room_id (optional)

2. If chat_room_id not specified → use sourceID from context

3. service.Get(ctx, chatRoomID)
   - Read events.jsonl
   - Find event by ChatRoomID

4. Resolve CreatorName
   show_creator=true → get displayName from profile
   show_creator=false → nil

5. Build response
   {creator_name?, title, start_time, end_time, capacity, fee, description}
   start_time, end_time: JST (+09:00) RFC3339 format
```

### list_events

```
1. Receive tool callback
   args: created_by_me, start, end
   start/end: RFC3339 or "today"

2. Get userID from context

3. Resolve "today" to time.Time (tool side only)
   "today" → current date 00:00:00 JST

4. Validation
   start, end both specified → end - start ≤ maxPeriodDays (from constructor)

5. Build ListOptions
   CreatorID: created_by_me=true → &userID, else nil
   Start: resolved start time
   End: resolved end time
   Limit: start+end → 0, otherwise → limit (from constructor)

6. service.List(ctx, opts)
   - Read events.jsonl
   - Filter, sort, limit per List behavior

7. Build response
   Limited fields only (FR-016)
   start_time, end_time: JST (+09:00) RFC3339 format
```
