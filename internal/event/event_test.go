package event_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"yuruppu/internal/event"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixed timestamps for deterministic tests
var (
	testTime1 = time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	testTime2 = time.Date(2026, 2, 2, 10, 0, 0, 0, time.UTC)
	testTime3 = time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	testTime4 = time.Date(2026, 2, 4, 10, 0, 0, 0, time.UTC)
	testTime5 = time.Date(2026, 2, 5, 10, 0, 0, 0, time.UTC)
	testTime6 = time.Date(2026, 2, 6, 10, 0, 0, 0, time.UTC)
)

// =============================================================================
// NewService Tests
// =============================================================================

func TestNewService_NilStorage(t *testing.T) {
	t.Run("nil storage returns error", func(t *testing.T) {
		svc, err := event.NewService(nil)

		require.Error(t, err)
		assert.Nil(t, svc)
		assert.Contains(t, err.Error(), "storage cannot be nil")
	})
}

// =============================================================================
// Create Tests (FR-004, FR-008, NFR-003)
// =============================================================================

// AC-001: Event creation with all required attributes
func TestService_Create(t *testing.T) {
	t.Run("successfully creates event with all attributes", func(t *testing.T) {
		// Given: Empty storage (no existing events)
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		ev := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Go Meetup",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    50,
			Description: "Monthly Go meetup",
			ShowCreator: true,
		}

		// When: Create event
		err = svc.Create(context.Background(), ev)

		// Then: Should succeed
		require.NoError(t, err)
		assert.Equal(t, 1, store.writeCallCount)

		// Verify stored data format (JSONL)
		storedData := store.lastWriteData
		lines := strings.Split(strings.TrimSpace(string(storedData)), "\n")
		assert.Len(t, lines, 1)

		var stored event.Event
		err = json.Unmarshal([]byte(lines[0]), &stored)
		require.NoError(t, err)
		assert.Equal(t, "chatroom-001", stored.ChatRoomID)
		assert.Equal(t, "user-123", stored.CreatorID)
		assert.Equal(t, "Go Meetup", stored.Title)
		assert.Equal(t, "Free", stored.Fee)
		assert.Equal(t, 50, stored.Capacity)
	})

	t.Run("appends event to existing events", func(t *testing.T) {
		// Given: Storage with one existing event
		store := newMockStorage()
		existing := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "First Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "First",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existing)
		store.data["events.jsonl"] = existingJSON
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		newEvent := &event.Event{
			ChatRoomID:  "chatroom-002",
			CreatorID:   "user-456",
			Title:       "Second Event",
			StartTime:   testTime3,
			EndTime:     testTime4,
			Fee:         "$10",
			Capacity:    20,
			Description: "Second",
			ShowCreator: false,
		}

		// When: Create second event
		err = svc.Create(context.Background(), newEvent)

		// Then: Should append to existing
		require.NoError(t, err)

		// Verify JSONL format (2 lines)
		storedData := store.lastWriteData
		lines := strings.Split(strings.TrimSpace(string(storedData)), "\n")
		assert.Len(t, lines, 2)

		// Verify first line is preserved
		var first event.Event
		err = json.Unmarshal([]byte(lines[0]), &first)
		require.NoError(t, err)
		assert.Equal(t, "chatroom-001", first.ChatRoomID)

		// Verify second line is new event
		var second event.Event
		err = json.Unmarshal([]byte(lines[1]), &second)
		require.NoError(t, err)
		assert.Equal(t, "chatroom-002", second.ChatRoomID)
	})
}

func TestService_Create_InvalidInput(t *testing.T) {
	t.Run("returns error when event is nil", func(t *testing.T) {
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		err = svc.Create(context.Background(), nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "event cannot be nil")
		assert.Equal(t, 0, store.writeCallCount)
	})

	t.Run("returns error when ChatRoomID is empty", func(t *testing.T) {
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		ev := &event.Event{
			ChatRoomID:  "", // Empty
			CreatorID:   "user-123",
			Title:       "Test Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "Test",
			ShowCreator: true,
		}

		err = svc.Create(context.Background(), ev)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "chatRoomID cannot be empty")
		assert.Equal(t, 0, store.writeCallCount)
	})
}

// AC-003: Cannot create duplicate event in same chat room (FR-004)
func TestService_Create_DuplicateChatRoom(t *testing.T) {
	t.Run("returns error when ChatRoomID already exists", func(t *testing.T) {
		// Given: Storage with existing event in chatroom-001
		store := newMockStorage()
		existing := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Existing Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "Existing",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existing)
		store.data["events.jsonl"] = existingJSON
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		duplicate := &event.Event{
			ChatRoomID:  "chatroom-001", // Same ChatRoomID
			CreatorID:   "user-456",
			Title:       "Duplicate Event",
			StartTime:   testTime3,
			EndTime:     testTime4,
			Fee:         "$5",
			Capacity:    15,
			Description: "Should fail",
			ShowCreator: true,
		}

		// When: Try to create duplicate
		err = svc.Create(context.Background(), duplicate)

		// Then: Should return error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
		assert.Contains(t, err.Error(), "chatroom-001")
	})
}

// NFR-003: Concurrent creates - optimistic locking
func TestService_Create_ConcurrentCreation(t *testing.T) {
	t.Run("concurrent creates - one succeeds, one fails with conflict", func(t *testing.T) {
		// Given: Empty storage
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		event1 := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "First Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "First",
			ShowCreator: true,
		}

		event2 := &event.Event{
			ChatRoomID:  "chatroom-002",
			CreatorID:   "user-456",
			Title:       "Second Event",
			StartTime:   testTime3,
			EndTime:     testTime4,
			Fee:         "$10",
			Capacity:    20,
			Description: "Second",
			ShowCreator: false,
		}

		// Simulate concurrent creates by enabling conflict detection
		store.simulateConcurrentWrite = true

		// When: First create succeeds
		err1 := svc.Create(context.Background(), event1)

		// When: Second create with stale generation fails
		err2 := svc.Create(context.Background(), event2)

		// Then: One should succeed, one should fail
		if err1 == nil {
			require.Error(t, err2)
			assert.Contains(t, err2.Error(), "generation mismatch")
		} else {
			require.NoError(t, err2)
			assert.Contains(t, err1.Error(), "generation mismatch")
		}
	})
}

func TestService_Create_StorageErrors(t *testing.T) {
	t.Run("returns error when storage read fails", func(t *testing.T) {
		store := newMockStorage()
		store.readErr = errors.New("storage read error")
		svc, err := event.NewService(store)
		require.NoError(t, err)

		ev := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Test Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "Test",
			ShowCreator: true,
		}

		err = svc.Create(context.Background(), ev)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})

	t.Run("returns error when storage write fails", func(t *testing.T) {
		store := newMockStorage()
		store.writeErr = errors.New("storage write error")
		svc, err := event.NewService(store)
		require.NoError(t, err)

		ev := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Test Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "Test",
			ShowCreator: true,
		}

		err = svc.Create(context.Background(), ev)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write")
	})
}

// =============================================================================
// Get Tests (FR-009)
// =============================================================================

// AC-004: Get existing event returns event details
func TestService_Get(t *testing.T) {
	t.Run("returns event when it exists", func(t *testing.T) {
		// Given: Storage with existing event
		store := newMockStorage()
		existingEvent := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Go Meetup",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    50,
			Description: "Monthly Go meetup",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existingEvent)
		store.data["events.jsonl"] = existingJSON
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Get event by ChatRoomID
		got, err := svc.Get(context.Background(), "chatroom-001")

		// Then: Should return the event
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "chatroom-001", got.ChatRoomID)
		assert.Equal(t, "user-123", got.CreatorID)
		assert.Equal(t, "Go Meetup", got.Title)
		assert.Equal(t, "Free", got.Fee)
		assert.Equal(t, 50, got.Capacity)
		assert.True(t, got.ShowCreator)
	})

	t.Run("returns event when multiple events exist", func(t *testing.T) {
		// Given: Storage with multiple events
		store := newMockStorage()
		events := []*event.Event{
			{
				ChatRoomID:  "chatroom-001",
				CreatorID:   "user-123",
				Title:       "First Event",
				StartTime:   testTime1,
				EndTime:     testTime2,
				Fee:         "Free",
				Capacity:    10,
				Description: "First",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-002",
				CreatorID:   "user-456",
				Title:       "Second Event",
				StartTime:   testTime3,
				EndTime:     testTime4,
				Fee:         "$10",
				Capacity:    20,
				Description: "Second",
				ShowCreator: false,
			},
		}

		var lines []string
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["events.jsonl"] = []byte(strings.Join(lines, "\n"))
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Get specific event
		got, err := svc.Get(context.Background(), "chatroom-002")

		// Then: Should return the correct event
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "chatroom-002", got.ChatRoomID)
		assert.Equal(t, "Second Event", got.Title)
	})
}

// AC-005: Get non-existing event returns error
func TestService_Get_NotFound(t *testing.T) {
	t.Run("returns error when event does not exist", func(t *testing.T) {
		// Given: Empty storage
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Get non-existing event
		got, err := svc.Get(context.Background(), "non-existent")

		// Then: Should return error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
		assert.Contains(t, err.Error(), "non-existent")
		assert.Nil(t, got)
	})

	t.Run("returns error when chatRoomID not found among existing events", func(t *testing.T) {
		// Given: Storage with some events
		store := newMockStorage()
		existingEvent := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Existing Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "Existing",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existingEvent)
		store.data["events.jsonl"] = existingJSON
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Get different chatRoomID
		got, err := svc.Get(context.Background(), "chatroom-999")

		// Then: Should return error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
		assert.Contains(t, err.Error(), "chatroom-999")
		assert.Nil(t, got)
	})
}

func TestService_Get_InvalidInput(t *testing.T) {
	t.Run("returns error when chatRoomID is empty", func(t *testing.T) {
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		got, err := svc.Get(context.Background(), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "chatRoomID cannot be empty")
		assert.Nil(t, got)
		assert.Equal(t, 0, store.readCallCount)
	})
}

func TestService_Get_StorageError(t *testing.T) {
	t.Run("returns error when storage read fails", func(t *testing.T) {
		store := newMockStorage()
		store.readErr = errors.New("storage read error")
		svc, err := event.NewService(store)
		require.NoError(t, err)

		got, err := svc.Get(context.Background(), "chatroom-001")

		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "failed to read")
	})
}

// =============================================================================
// List Tests (FR-010, FR-011, FR-012, FR-013, FR-014, FR-015)
// =============================================================================

// AC-006: List all events (no filters)
func TestService_List_NoFilters(t *testing.T) {
	t.Run("returns all events sorted by StartTime ascending", func(t *testing.T) {
		// Given: Storage with multiple events in random order
		store := newMockStorage()
		events := []*event.Event{
			{
				ChatRoomID:  "chatroom-003",
				CreatorID:   "user-123",
				Title:       "Event 3",
				StartTime:   testTime3, // Middle
				EndTime:     testTime4,
				Fee:         "$5",
				Capacity:    15,
				Description: "Third",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-001",
				CreatorID:   "user-456",
				Title:       "Event 1",
				StartTime:   testTime1, // Earliest
				EndTime:     testTime2,
				Fee:         "Free",
				Capacity:    10,
				Description: "First",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-002",
				CreatorID:   "user-789",
				Title:       "Event 2",
				StartTime:   testTime5, // Latest
				EndTime:     testTime6,
				Fee:         "$10",
				Capacity:    20,
				Description: "Second",
				ShowCreator: false,
			},
		}

		var lines []string
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["events.jsonl"] = []byte(strings.Join(lines, "\n"))
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: List all events (no filters)
		opts := event.ListOptions{}
		got, err := svc.List(context.Background(), opts)

		// Then: Should return all events sorted by StartTime ascending
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "Event 1", got[0].Title)
		assert.Equal(t, "Event 3", got[1].Title)
		assert.Equal(t, "Event 2", got[2].Title)
	})

	t.Run("returns empty list when no events exist", func(t *testing.T) {
		// Given: Empty storage
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: List events
		opts := event.ListOptions{}
		got, err := svc.List(context.Background(), opts)

		// Then: Should return empty list
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

// AC-007: Filter by CreatorID (FR-011)
func TestService_List_FilterByCreator(t *testing.T) {
	t.Run("returns only events created by specified user", func(t *testing.T) {
		// Given: Storage with events from different creators
		store := newMockStorage()
		events := []*event.Event{
			{
				ChatRoomID:  "chatroom-001",
				CreatorID:   "user-123", // Target creator
				Title:       "User 123 Event 1",
				StartTime:   testTime1,
				EndTime:     testTime2,
				Fee:         "Free",
				Capacity:    10,
				Description: "First",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-002",
				CreatorID:   "user-456", // Different creator
				Title:       "User 456 Event",
				StartTime:   testTime2,
				EndTime:     testTime3,
				Fee:         "$5",
				Capacity:    15,
				Description: "Second",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-003",
				CreatorID:   "user-123", // Target creator
				Title:       "User 123 Event 2",
				StartTime:   testTime4,
				EndTime:     testTime5,
				Fee:         "$10",
				Capacity:    20,
				Description: "Third",
				ShowCreator: false,
			},
		}

		var lines []string
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["events.jsonl"] = []byte(strings.Join(lines, "\n"))
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: List events filtered by CreatorID
		creatorID := "user-123"
		opts := event.ListOptions{
			CreatorID: &creatorID,
		}
		got, err := svc.List(context.Background(), opts)

		// Then: Should return only user-123 events
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "User 123 Event 1", got[0].Title)
		assert.Equal(t, "User 123 Event 2", got[1].Title)
		assert.Equal(t, "user-123", got[0].CreatorID)
		assert.Equal(t, "user-123", got[1].CreatorID)
	})

	t.Run("returns empty list when no events match creator", func(t *testing.T) {
		store := newMockStorage()
		existingEvent := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "Test",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existingEvent)
		store.data["events.jsonl"] = existingJSON
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		creatorID := "user-999"
		opts := event.ListOptions{
			CreatorID: &creatorID,
		}
		got, err := svc.List(context.Background(), opts)

		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

// AC-008: Period filter with Start only - ascending order (FR-012, FR-014)
func TestService_List_FilterByStartOnly(t *testing.T) {
	t.Run("returns events with StartTime >= Start, ascending order", func(t *testing.T) {
		// Given: Storage with events at different times
		store := newMockStorage()
		events := []*event.Event{
			{
				ChatRoomID:  "chatroom-001",
				CreatorID:   "user-123",
				Title:       "Past Event",
				StartTime:   testTime1, // Before filter
				EndTime:     testTime2,
				Fee:         "Free",
				Capacity:    10,
				Description: "Past",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-002",
				CreatorID:   "user-123",
				Title:       "Future Event 1",
				StartTime:   testTime4, // After filter
				EndTime:     testTime5,
				Fee:         "$5",
				Capacity:    15,
				Description: "Future 1",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-003",
				CreatorID:   "user-123",
				Title:       "Future Event 2",
				StartTime:   testTime6, // After filter
				EndTime:     testTime6,
				Fee:         "$10",
				Capacity:    20,
				Description: "Future 2",
				ShowCreator: false,
			},
		}

		var lines []string
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["events.jsonl"] = []byte(strings.Join(lines, "\n"))
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: List events with Start filter
		start := testTime3 // Between testTime1 and testTime4
		opts := event.ListOptions{
			Start: &start,
		}
		got, err := svc.List(context.Background(), opts)

		// Then: Should return events >= testTime3, ascending order
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "Future Event 1", got[0].Title)
		assert.Equal(t, "Future Event 2", got[1].Title)
		assert.True(t, got[0].StartTime.Before(got[1].StartTime))
	})
}

// AC-009: Period filter with End only - descending order (FR-012, FR-014)
func TestService_List_FilterByEndOnly(t *testing.T) {
	t.Run("returns events with StartTime <= End, descending order", func(t *testing.T) {
		// Given: Storage with events at different times
		store := newMockStorage()
		events := []*event.Event{
			{
				ChatRoomID:  "chatroom-001",
				CreatorID:   "user-123",
				Title:       "Past Event 1",
				StartTime:   testTime1, // Before filter
				EndTime:     testTime2,
				Fee:         "Free",
				Capacity:    10,
				Description: "Past 1",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-002",
				CreatorID:   "user-123",
				Title:       "Past Event 2",
				StartTime:   testTime3, // Before filter
				EndTime:     testTime4,
				Fee:         "$5",
				Capacity:    15,
				Description: "Past 2",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-003",
				CreatorID:   "user-123",
				Title:       "Future Event",
				StartTime:   testTime6, // After filter
				EndTime:     testTime6,
				Fee:         "$10",
				Capacity:    20,
				Description: "Future",
				ShowCreator: false,
			},
		}

		var lines []string
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["events.jsonl"] = []byte(strings.Join(lines, "\n"))
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: List events with End filter
		end := testTime5 // Between testTime3 and testTime6
		opts := event.ListOptions{
			End: &end,
		}
		got, err := svc.List(context.Background(), opts)

		// Then: Should return events <= testTime5, descending order
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "Past Event 2", got[0].Title)
		assert.Equal(t, "Past Event 1", got[1].Title)
		assert.True(t, got[0].StartTime.After(got[1].StartTime))
	})
}

// AC-010: Period filter with Start and End - ascending order (FR-012, FR-014)
func TestService_List_FilterByStartAndEnd(t *testing.T) {
	t.Run("returns events within period, ascending order", func(t *testing.T) {
		// Given: Storage with events at different times
		store := newMockStorage()
		events := []*event.Event{
			{
				ChatRoomID:  "chatroom-001",
				CreatorID:   "user-123",
				Title:       "Before Period",
				StartTime:   testTime1, // Before range
				EndTime:     testTime2,
				Fee:         "Free",
				Capacity:    10,
				Description: "Before",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-002",
				CreatorID:   "user-123",
				Title:       "In Period 1",
				StartTime:   testTime3, // In range
				EndTime:     testTime4,
				Fee:         "$5",
				Capacity:    15,
				Description: "In 1",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-003",
				CreatorID:   "user-123",
				Title:       "In Period 2",
				StartTime:   testTime4, // In range
				EndTime:     testTime5,
				Fee:         "$7",
				Capacity:    17,
				Description: "In 2",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-004",
				CreatorID:   "user-123",
				Title:       "After Period",
				StartTime:   testTime6, // After range
				EndTime:     testTime6,
				Fee:         "$10",
				Capacity:    20,
				Description: "After",
				ShowCreator: false,
			},
		}

		var lines []string
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["events.jsonl"] = []byte(strings.Join(lines, "\n"))
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: List events with Start and End filter
		start := testTime2
		end := testTime5
		opts := event.ListOptions{
			Start: &start,
			End:   &end,
		}
		got, err := svc.List(context.Background(), opts)

		// Then: Should return events in range, ascending order
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "In Period 1", got[0].Title)
		assert.Equal(t, "In Period 2", got[1].Title)
		assert.True(t, got[0].StartTime.Before(got[1].StartTime) || got[0].StartTime.Equal(got[1].StartTime))
	})
}

// AC-011: Combined CreatorID + period filter (FR-013)
func TestService_List_CombinedFilters(t *testing.T) {
	t.Run("applies both CreatorID and period filters", func(t *testing.T) {
		// Given: Storage with events from different creators and times
		store := newMockStorage()
		events := []*event.Event{
			{
				ChatRoomID:  "chatroom-001",
				CreatorID:   "user-123",
				Title:       "User 123 Past",
				StartTime:   testTime1, // Before period
				EndTime:     testTime2,
				Fee:         "Free",
				Capacity:    10,
				Description: "Past",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-002",
				CreatorID:   "user-123",
				Title:       "User 123 In Period",
				StartTime:   testTime3, // In period
				EndTime:     testTime4,
				Fee:         "$5",
				Capacity:    15,
				Description: "In",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-003",
				CreatorID:   "user-456",
				Title:       "User 456 In Period",
				StartTime:   testTime4, // In period but different creator
				EndTime:     testTime5,
				Fee:         "$7",
				Capacity:    17,
				Description: "Different user",
				ShowCreator: true,
			},
		}

		var lines []string
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["events.jsonl"] = []byte(strings.Join(lines, "\n"))
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: List events with both filters
		creatorID := "user-123"
		start := testTime2
		opts := event.ListOptions{
			CreatorID: &creatorID,
			Start:     &start,
		}
		got, err := svc.List(context.Background(), opts)

		// Then: Should return only user-123 events in period
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "User 123 In Period", got[0].Title)
		assert.Equal(t, "user-123", got[0].CreatorID)
	})
}

// FR-015: Limit applied when Start or End specified (not both)
func TestService_List_WithLimit(t *testing.T) {
	t.Run("applies limit when only Start specified", func(t *testing.T) {
		// Given: Storage with many events
		store := newMockStorage()
		events := []*event.Event{
			{ChatRoomID: "room-1", CreatorID: "user-1", Title: "Event 1", StartTime: testTime1, EndTime: testTime2, Fee: "Free", Capacity: 10, Description: "1", ShowCreator: true},
			{ChatRoomID: "room-2", CreatorID: "user-1", Title: "Event 2", StartTime: testTime2, EndTime: testTime3, Fee: "Free", Capacity: 10, Description: "2", ShowCreator: true},
			{ChatRoomID: "room-3", CreatorID: "user-1", Title: "Event 3", StartTime: testTime3, EndTime: testTime4, Fee: "Free", Capacity: 10, Description: "3", ShowCreator: true},
			{ChatRoomID: "room-4", CreatorID: "user-1", Title: "Event 4", StartTime: testTime4, EndTime: testTime5, Fee: "Free", Capacity: 10, Description: "4", ShowCreator: true},
			{ChatRoomID: "room-5", CreatorID: "user-1", Title: "Event 5", StartTime: testTime5, EndTime: testTime6, Fee: "Free", Capacity: 10, Description: "5", ShowCreator: true},
		}

		var lines []string
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["events.jsonl"] = []byte(strings.Join(lines, "\n"))
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: List with Start and Limit
		start := testTime1
		opts := event.ListOptions{
			Start: &start,
			Limit: 3,
		}
		got, err := svc.List(context.Background(), opts)

		// Then: Should return at most 3 events
		require.NoError(t, err)
		assert.LessOrEqual(t, len(got), 3)
	})

	t.Run("applies limit when only End specified", func(t *testing.T) {
		// Given: Storage with many events
		store := newMockStorage()
		events := []*event.Event{
			{ChatRoomID: "room-1", CreatorID: "user-1", Title: "Event 1", StartTime: testTime1, EndTime: testTime2, Fee: "Free", Capacity: 10, Description: "1", ShowCreator: true},
			{ChatRoomID: "room-2", CreatorID: "user-1", Title: "Event 2", StartTime: testTime2, EndTime: testTime3, Fee: "Free", Capacity: 10, Description: "2", ShowCreator: true},
			{ChatRoomID: "room-3", CreatorID: "user-1", Title: "Event 3", StartTime: testTime3, EndTime: testTime4, Fee: "Free", Capacity: 10, Description: "3", ShowCreator: true},
			{ChatRoomID: "room-4", CreatorID: "user-1", Title: "Event 4", StartTime: testTime4, EndTime: testTime5, Fee: "Free", Capacity: 10, Description: "4", ShowCreator: true},
			{ChatRoomID: "room-5", CreatorID: "user-1", Title: "Event 5", StartTime: testTime5, EndTime: testTime6, Fee: "Free", Capacity: 10, Description: "5", ShowCreator: true},
		}

		var lines []string
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["events.jsonl"] = []byte(strings.Join(lines, "\n"))
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: List with End and Limit
		end := testTime6
		opts := event.ListOptions{
			End:   &end,
			Limit: 3,
		}
		got, err := svc.List(context.Background(), opts)

		// Then: Should return at most 3 events
		require.NoError(t, err)
		assert.LessOrEqual(t, len(got), 3)
	})

	t.Run("ignores limit when both Start and End specified", func(t *testing.T) {
		// Given: Storage with many events
		store := newMockStorage()
		events := []*event.Event{
			{ChatRoomID: "room-1", CreatorID: "user-1", Title: "Event 1", StartTime: testTime1, EndTime: testTime2, Fee: "Free", Capacity: 10, Description: "1", ShowCreator: true},
			{ChatRoomID: "room-2", CreatorID: "user-1", Title: "Event 2", StartTime: testTime2, EndTime: testTime3, Fee: "Free", Capacity: 10, Description: "2", ShowCreator: true},
			{ChatRoomID: "room-3", CreatorID: "user-1", Title: "Event 3", StartTime: testTime3, EndTime: testTime4, Fee: "Free", Capacity: 10, Description: "3", ShowCreator: true},
			{ChatRoomID: "room-4", CreatorID: "user-1", Title: "Event 4", StartTime: testTime4, EndTime: testTime5, Fee: "Free", Capacity: 10, Description: "4", ShowCreator: true},
			{ChatRoomID: "room-5", CreatorID: "user-1", Title: "Event 5", StartTime: testTime5, EndTime: testTime6, Fee: "Free", Capacity: 10, Description: "5", ShowCreator: true},
		}

		var lines []string
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["events.jsonl"] = []byte(strings.Join(lines, "\n"))
		store.generation["events.jsonl"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: List with both Start+End and Limit
		start := testTime1
		end := testTime6
		opts := event.ListOptions{
			Start: &start,
			End:   &end,
			Limit: 2, // Should be ignored
		}
		got, err := svc.List(context.Background(), opts)

		// Then: Should return all matching events (no limit)
		require.NoError(t, err)
		assert.Equal(t, 5, len(got)) // All events in range
	})
}

func TestService_List_StorageError(t *testing.T) {
	t.Run("returns error when storage read fails", func(t *testing.T) {
		store := newMockStorage()
		store.readErr = errors.New("storage read error")
		svc, err := event.NewService(store)
		require.NoError(t, err)

		opts := event.ListOptions{}
		got, err := svc.List(context.Background(), opts)

		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "failed to read")
	})
}

// =============================================================================
// Mock Storage
// =============================================================================

type mockStorage struct {
	data                     map[string][]byte
	generation               map[string]int64
	readErr                  error
	writeErr                 error
	readCallCount            int
	writeCallCount           int
	lastWriteKey             string
	lastWriteMIMEType        string
	lastWriteData            []byte
	simulateConcurrentWrite  bool
	concurrentWriteAttempted bool
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data:       make(map[string][]byte),
		generation: make(map[string]int64),
	}
}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	m.readCallCount++
	if m.readErr != nil {
		return nil, 0, m.readErr
	}
	data, exists := m.data[key]
	if !exists {
		return nil, 0, nil
	}
	return data, m.generation[key], nil
}

func (m *mockStorage) Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) (int64, error) {
	m.writeCallCount++
	m.lastWriteKey = key
	m.lastWriteMIMEType = mimetype
	m.lastWriteData = data

	if m.writeErr != nil {
		return 0, m.writeErr
	}

	currentGen := m.generation[key]

	// Simulate concurrent write detection
	if m.simulateConcurrentWrite && m.concurrentWriteAttempted {
		// Second write fails with generation mismatch
		return 0, errors.New("generation mismatch: concurrent write detected")
	}

	if currentGen != expectedGeneration {
		return 0, errors.New("generation mismatch")
	}

	m.data[key] = data
	newGen := expectedGeneration + 1
	m.generation[key] = newGen

	if m.simulateConcurrentWrite {
		m.concurrentWriteAttempted = true
	}

	return newGen, nil
}

func (m *mockStorage) GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	return "", nil
}
