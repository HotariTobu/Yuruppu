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
		store.data["all"] = existingJSON
		store.generation["all"] = 1

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
		store.data["all"] = existingJSON
		store.generation["all"] = 1

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
		store.data["all"] = existingJSON
		store.generation["all"] = 1

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

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

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
		store.data["all"] = existingJSON
		store.generation["all"] = 1

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

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

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

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

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
		store.data["all"] = existingJSON
		store.generation["all"] = 1

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

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

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

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

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

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

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

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

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

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

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

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

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

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

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
// Update Tests (FR-002, FR-003, FR-006, NFR-001)
// =============================================================================

// AC-001: Event description update (FR-002, FR-003, FR-005)
func TestService_Update(t *testing.T) {
	t.Run("successfully updates event description", func(t *testing.T) {
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
			Description: "Original description",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existingEvent)
		store.data["all"] = existingJSON
		store.generation["all"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Update event description
		newDescription := "Updated description with new details"
		err = svc.Update(context.Background(), "chatroom-001", newDescription)

		// Then: Update should succeed
		require.NoError(t, err)
		assert.Equal(t, 1, store.writeCallCount)

		// Verify updated data
		storedData := store.lastWriteData
		lines := strings.Split(strings.TrimSpace(string(storedData)), "\n")
		assert.Len(t, lines, 1)

		var updated event.Event
		err = json.Unmarshal([]byte(lines[0]), &updated)
		require.NoError(t, err)

		// Then: Description should be updated, other fields unchanged
		assert.Equal(t, newDescription, updated.Description)
		assert.Equal(t, "chatroom-001", updated.ChatRoomID)
		assert.Equal(t, "user-123", updated.CreatorID)
		assert.Equal(t, "Go Meetup", updated.Title)
		assert.Equal(t, testTime1, updated.StartTime)
		assert.Equal(t, testTime2, updated.EndTime)
		assert.Equal(t, "Free", updated.Fee)
		assert.Equal(t, 50, updated.Capacity)
		assert.True(t, updated.ShowCreator)
	})

	t.Run("updates correct event when multiple events exist", func(t *testing.T) {
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
				Description: "First description",
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
				Description: "Second description",
				ShowCreator: false,
			},
			{
				ChatRoomID:  "chatroom-003",
				CreatorID:   "user-789",
				Title:       "Third Event",
				StartTime:   testTime5,
				EndTime:     testTime6,
				Fee:         "$5",
				Capacity:    15,
				Description: "Third description",
				ShowCreator: true,
			},
		}

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Update middle event
		err = svc.Update(context.Background(), "chatroom-002", "Updated second event")

		// Then: Only target event should be updated
		require.NoError(t, err)

		storedData := store.lastWriteData
		storedLines := strings.Split(strings.TrimSpace(string(storedData)), "\n")
		require.Len(t, storedLines, 3)

		// Verify first event unchanged
		var first event.Event
		err = json.Unmarshal([]byte(storedLines[0]), &first)
		require.NoError(t, err)
		assert.Equal(t, "First description", first.Description)

		// Verify second event updated
		var second event.Event
		err = json.Unmarshal([]byte(storedLines[1]), &second)
		require.NoError(t, err)
		assert.Equal(t, "Updated second event", second.Description)
		assert.Equal(t, "chatroom-002", second.ChatRoomID)

		// Verify third event unchanged
		var third event.Event
		err = json.Unmarshal([]byte(storedLines[2]), &third)
		require.NoError(t, err)
		assert.Equal(t, "Third description", third.Description)
	})
}

func TestService_Update_InvalidInput(t *testing.T) {
	t.Run("returns error when chatRoomID is empty", func(t *testing.T) {
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		err = svc.Update(context.Background(), "", "New description")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "chatRoomID cannot be empty")
		assert.Equal(t, 0, store.writeCallCount)
	})
}

// AC-003: Event update when event does not exist (FR-006)
func TestService_Update_EventNotFound(t *testing.T) {
	t.Run("returns error when event does not exist in empty storage", func(t *testing.T) {
		// Given: Empty storage
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Try to update non-existent event
		err = svc.Update(context.Background(), "chatroom-999", "New description")

		// Then: Should return error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
		assert.Contains(t, err.Error(), "chatroom-999")
		assert.Equal(t, 0, store.writeCallCount)
	})

	t.Run("returns error when event not found among existing events", func(t *testing.T) {
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
		store.data["all"] = existingJSON
		store.generation["all"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Try to update different chatRoomID
		err = svc.Update(context.Background(), "chatroom-999", "New description")

		// Then: Should return error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
		assert.Contains(t, err.Error(), "chatroom-999")
		assert.Equal(t, 0, store.writeCallCount)
	})
}

// NFR-001: Atomic update operation (optimistic locking)
func TestService_Update_Atomicity(t *testing.T) {
	t.Run("concurrent updates - one succeeds, one fails with conflict", func(t *testing.T) {
		// Given: Storage with existing event
		store := newMockStorage()
		existingEvent := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "Original",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existingEvent)
		store.data["all"] = existingJSON
		store.generation["all"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// Simulate concurrent updates by enabling conflict detection
		store.simulateConcurrentWrite = true

		// When: First update succeeds
		err1 := svc.Update(context.Background(), "chatroom-001", "Update 1")

		// When: Second update with stale generation fails
		err2 := svc.Update(context.Background(), "chatroom-001", "Update 2")

		// Then: One should succeed, one should fail
		if err1 == nil {
			require.Error(t, err2)
			assert.Contains(t, err2.Error(), "generation mismatch")
		} else {
			require.NoError(t, err2)
			assert.Contains(t, err1.Error(), "generation mismatch")
		}
	})

	t.Run("uses optimistic locking with generation check", func(t *testing.T) {
		// Given: Storage with event at generation 5
		store := newMockStorage()
		existingEvent := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "Original",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existingEvent)
		store.data["all"] = existingJSON
		store.generation["all"] = 5 // Current generation

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Update event
		err = svc.Update(context.Background(), "chatroom-001", "Updated")

		// Then: Should succeed and increment generation
		require.NoError(t, err)
		assert.Equal(t, int64(6), store.generation["all"])
	})
}

func TestService_Update_StorageErrors(t *testing.T) {
	t.Run("returns error when storage read fails", func(t *testing.T) {
		store := newMockStorage()
		store.readErr = errors.New("storage read error")
		svc, err := event.NewService(store)
		require.NoError(t, err)

		err = svc.Update(context.Background(), "chatroom-001", "New description")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
		assert.Equal(t, 0, store.writeCallCount)
	})

	t.Run("returns error when storage write fails", func(t *testing.T) {
		store := newMockStorage()
		existingEvent := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "Original",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existingEvent)
		store.data["all"] = existingJSON
		store.generation["all"] = 1

		store.writeErr = errors.New("storage write error")
		svc, err := event.NewService(store)
		require.NoError(t, err)

		err = svc.Update(context.Background(), "chatroom-001", "New description")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write")
	})
}

// =============================================================================
// Remove Tests (FR-007, FR-010, FR-011, NFR-001)
// =============================================================================

// AC-004: Event removal - physically removed from storage (FR-007, FR-009, FR-011)
func TestService_Remove(t *testing.T) {
	t.Run("successfully removes event and physically removes it", func(t *testing.T) {
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
		store.data["all"] = existingJSON
		store.generation["all"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Remove event
		err = svc.Remove(context.Background(), "chatroom-001")

		// Then: Remove should succeed
		require.NoError(t, err)
		assert.Equal(t, 1, store.writeCallCount)

		// Verify event is physically removed (empty storage)
		storedData := store.lastWriteData
		lines := strings.Split(strings.TrimSpace(string(storedData)), "\n")
		// Empty JSONL should have one empty line after trimming
		if len(lines) == 1 && lines[0] == "" {
			// Storage is empty
			assert.Equal(t, "", lines[0])
		} else {
			// No lines should remain
			assert.Empty(t, lines)
		}
	})

	t.Run("removes correct event when multiple events exist", func(t *testing.T) {
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
			{
				ChatRoomID:  "chatroom-003",
				CreatorID:   "user-789",
				Title:       "Third Event",
				StartTime:   testTime5,
				EndTime:     testTime6,
				Fee:         "$5",
				Capacity:    15,
				Description: "Third",
				ShowCreator: true,
			},
		}

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Remove middle event
		err = svc.Remove(context.Background(), "chatroom-002")

		// Then: Only target event should be removed
		require.NoError(t, err)

		storedData := store.lastWriteData
		storedLines := strings.Split(strings.TrimSpace(string(storedData)), "\n")
		require.Len(t, storedLines, 2)

		// Verify first event still exists
		var first event.Event
		err = json.Unmarshal([]byte(storedLines[0]), &first)
		require.NoError(t, err)
		assert.Equal(t, "chatroom-001", first.ChatRoomID)
		assert.Equal(t, "First Event", first.Title)

		// Verify third event still exists
		var third event.Event
		err = json.Unmarshal([]byte(storedLines[1]), &third)
		require.NoError(t, err)
		assert.Equal(t, "chatroom-003", third.ChatRoomID)
		assert.Equal(t, "Third Event", third.Title)

		// Verify second event is gone (not in results)
		for _, line := range storedLines {
			var ev event.Event
			json.Unmarshal([]byte(line), &ev)
			assert.NotEqual(t, "chatroom-002", ev.ChatRoomID)
		}
	})
}

func TestService_Remove_InvalidInput(t *testing.T) {
	t.Run("returns error when chatRoomID is empty", func(t *testing.T) {
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		err = svc.Remove(context.Background(), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "chatRoomID cannot be empty")
		assert.Equal(t, 0, store.writeCallCount)
	})
}

// AC-006: Event removal when event does not exist (FR-010)
func TestService_Remove_EventNotFound(t *testing.T) {
	t.Run("returns error when event does not exist in empty storage", func(t *testing.T) {
		// Given: Empty storage
		store := newMockStorage()
		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Try to remove non-existent event
		err = svc.Remove(context.Background(), "chatroom-999")

		// Then: Should return error (FR-010)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
		assert.Contains(t, err.Error(), "chatroom-999")
		assert.Equal(t, 0, store.writeCallCount)
	})

	t.Run("returns error when event not found among existing events", func(t *testing.T) {
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
		store.data["all"] = existingJSON
		store.generation["all"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Try to remove different chatRoomID
		err = svc.Remove(context.Background(), "chatroom-999")

		// Then: Should return error (FR-010)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
		assert.Contains(t, err.Error(), "chatroom-999")
		assert.Equal(t, 0, store.writeCallCount)
	})
}

// FR-011: Verify removed events don't appear in Get/List
func TestService_Remove_VerifyPhysicalRemoval(t *testing.T) {
	t.Run("removed event does not appear in Get", func(t *testing.T) {
		// Given: Storage with event
		store := newMockStorage()
		existingEvent := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Event to Delete",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "Will be deleted",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existingEvent)
		store.data["all"] = existingJSON
		store.generation["all"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// Verify event exists before removal
		ev, err := svc.Get(context.Background(), "chatroom-001")
		require.NoError(t, err)
		assert.Equal(t, "chatroom-001", ev.ChatRoomID)

		// When: Remove event
		err = svc.Remove(context.Background(), "chatroom-001")
		require.NoError(t, err)

		// Then: Get should return "not found" error (FR-011)
		ev, err = svc.Get(context.Background(), "chatroom-001")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
		assert.Nil(t, ev)
	})

	t.Run("removed event does not appear in List", func(t *testing.T) {
		// Given: Storage with multiple events
		store := newMockStorage()
		events := []*event.Event{
			{
				ChatRoomID:  "chatroom-001",
				CreatorID:   "user-123",
				Title:       "Event 1",
				StartTime:   testTime1,
				EndTime:     testTime2,
				Fee:         "Free",
				Capacity:    10,
				Description: "First",
				ShowCreator: true,
			},
			{
				ChatRoomID:  "chatroom-002",
				CreatorID:   "user-123",
				Title:       "Event to Delete",
				StartTime:   testTime3,
				EndTime:     testTime4,
				Fee:         "$10",
				Capacity:    20,
				Description: "Will be deleted",
				ShowCreator: false,
			},
			{
				ChatRoomID:  "chatroom-003",
				CreatorID:   "user-123",
				Title:       "Event 3",
				StartTime:   testTime5,
				EndTime:     testTime6,
				Fee:         "$5",
				Capacity:    15,
				Description: "Third",
				ShowCreator: true,
			},
		}

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// Verify 3 events exist before removal
		listResult, err := svc.List(context.Background(), event.ListOptions{})
		require.NoError(t, err)
		assert.Len(t, listResult, 3)

		// When: Remove middle event
		err = svc.Remove(context.Background(), "chatroom-002")
		require.NoError(t, err)

		// Then: List should return only 2 events (FR-011)
		listResult, err = svc.List(context.Background(), event.ListOptions{})
		require.NoError(t, err)
		assert.Len(t, listResult, 2)

		// Verify removed event is not in list
		for _, ev := range listResult {
			assert.NotEqual(t, "chatroom-002", ev.ChatRoomID)
		}

		// Verify remaining events
		assert.Equal(t, "chatroom-001", listResult[0].ChatRoomID)
		assert.Equal(t, "chatroom-003", listResult[1].ChatRoomID)
	})
}

// NFR-001: Atomic remove operation (optimistic locking)
func TestService_Remove_Atomicity(t *testing.T) {
	t.Run("concurrent removes on different events - generation check prevents race", func(t *testing.T) {
		// Given: Storage with multiple events
		store := newMockStorage()
		events := []*event.Event{
			{
				ChatRoomID:  "chatroom-001",
				CreatorID:   "user-123",
				Title:       "Event 1",
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
				Title:       "Event 2",
				StartTime:   testTime3,
				EndTime:     testTime4,
				Fee:         "$10",
				Capacity:    20,
				Description: "Second",
				ShowCreator: false,
			},
		}

		lines := make([]string, 0, len(events))
		for _, ev := range events {
			jsonData, _ := json.Marshal(ev)
			lines = append(lines, string(jsonData))
		}
		store.data["all"] = []byte(strings.Join(lines, "\n"))
		store.generation["all"] = 1

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// Simulate concurrent writes by enabling conflict detection
		store.simulateConcurrentWrite = true

		// When: First remove succeeds
		err1 := svc.Remove(context.Background(), "chatroom-001")

		// When: Second remove with stale generation fails
		err2 := svc.Remove(context.Background(), "chatroom-002")

		// Then: One should succeed, one should fail with generation mismatch (NFR-001: atomicity)
		if err1 == nil {
			require.Error(t, err2)
			assert.Contains(t, err2.Error(), "generation mismatch")
		} else {
			require.NoError(t, err2)
			assert.Contains(t, err1.Error(), "generation mismatch")
		}
	})

	t.Run("uses optimistic locking with generation check", func(t *testing.T) {
		// Given: Storage with event at generation 5
		store := newMockStorage()
		existingEvent := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "To remove",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existingEvent)
		store.data["all"] = existingJSON
		store.generation["all"] = 5 // Current generation

		svc, err := event.NewService(store)
		require.NoError(t, err)

		// When: Remove event
		err = svc.Remove(context.Background(), "chatroom-001")

		// Then: Should succeed and increment generation (NFR-001)
		require.NoError(t, err)
		assert.Equal(t, int64(6), store.generation["all"])
	})
}

func TestService_Remove_StorageErrors(t *testing.T) {
	t.Run("returns error when storage read fails", func(t *testing.T) {
		store := newMockStorage()
		store.readErr = errors.New("storage read error")
		svc, err := event.NewService(store)
		require.NoError(t, err)

		err = svc.Remove(context.Background(), "chatroom-001")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
		assert.Equal(t, 0, store.writeCallCount)
	})

	t.Run("returns error when storage write fails", func(t *testing.T) {
		store := newMockStorage()
		existingEvent := &event.Event{
			ChatRoomID:  "chatroom-001",
			CreatorID:   "user-123",
			Title:       "Event",
			StartTime:   testTime1,
			EndTime:     testTime2,
			Fee:         "Free",
			Capacity:    10,
			Description: "To remove",
			ShowCreator: true,
		}
		existingJSON, _ := json.Marshal(existingEvent)
		store.data["all"] = existingJSON
		store.generation["all"] = 1

		store.writeErr = errors.New("storage write error")
		svc, err := event.NewService(store)
		require.NoError(t, err)

		err = svc.Remove(context.Background(), "chatroom-001")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write")
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
