package event

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"yuruppu/internal/storage"
)

const storageKey = "all"

// Event represents an event in a chat room.
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

// ListOptions specifies filtering and pagination options for listing events.
type ListOptions struct {
	CreatorID *string    // Filter by creator (nil = no filter)
	Start     *time.Time // Filter events with StartTime >= this time
	End       *time.Time // Filter events with StartTime <= this time
	Limit     int        // Max items to return (0 = no limit)
}

// Service provides event management operations.
type Service struct {
	storage storage.Storage
}

// NewService creates a new Service with the given storage backend.
// Returns error if storage is nil.
func NewService(s storage.Storage) (*Service, error) {
	if s == nil {
		return nil, errors.New("storage cannot be nil")
	}
	return &Service{storage: s}, nil
}

// Create creates a new event.
// Returns error if an event already exists for the chat room or if storage operations fail.
func (s *Service) Create(ctx context.Context, ev *Event) error {
	if ev == nil {
		return errors.New("event cannot be nil")
	}
	if ev.ChatRoomID == "" {
		return errors.New("chatRoomID cannot be empty")
	}

	// Read existing events
	events, generation, err := s.readEvents(ctx)
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	// Check for duplicate ChatRoomID
	for _, existing := range events {
		if existing.ChatRoomID == ev.ChatRoomID {
			return fmt.Errorf("event already exists: %s", ev.ChatRoomID)
		}
	}

	// Append new event
	events = append(events, ev)

	// Write back with generation
	if err := s.writeEvents(ctx, events, generation); err != nil {
		return fmt.Errorf("failed to write events: %w", err)
	}

	return nil
}

// Get retrieves an event by chat room ID.
// Returns error if the event is not found or if storage operations fail.
func (s *Service) Get(ctx context.Context, chatRoomID string) (*Event, error) {
	if chatRoomID == "" {
		return nil, errors.New("chatRoomID cannot be empty")
	}

	events, _, err := s.readEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}

	for _, ev := range events {
		if ev.ChatRoomID == chatRoomID {
			return ev, nil
		}
	}

	return nil, fmt.Errorf("event not found: %s", chatRoomID)
}

// List retrieves events with optional filtering and sorting.
// Sorting behavior:
//   - Start only or Start+End specified: ascending by StartTime
//   - End only specified: descending by StartTime
//
// Limit is applied after sorting and filtering.
func (s *Service) List(ctx context.Context, opts ListOptions) ([]*Event, error) {
	events, _, err := s.readEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}

	// Apply filters
	filtered := filterEvents(events, opts)

	// Sort
	sortEvents(filtered, opts)

	// Apply limit
	applyLimit(&filtered, opts)

	return filtered, nil
}

// readEvents reads and parses events from storage.
// Returns empty slice and generation 0 if no events exist.
func (s *Service) readEvents(ctx context.Context) ([]*Event, int64, error) {
	data, generation, err := s.storage.Read(ctx, storageKey)
	if err != nil {
		return nil, 0, err
	}

	if data == nil {
		return []*Event{}, generation, nil
	}

	events, err := parseJSONL(data)
	if err != nil {
		return nil, 0, err
	}

	return events, generation, nil
}

// writeEvents serializes and writes events to storage with optimistic locking.
func (s *Service) writeEvents(ctx context.Context, events []*Event, expectedGeneration int64) error {
	data, err := serializeJSONL(events)
	if err != nil {
		return err
	}

	_, err = s.storage.Write(ctx, storageKey, "application/jsonl", data, expectedGeneration)
	return err
}

// parseJSONL parses JSONL data into a slice of events.
func parseJSONL(data []byte) ([]*Event, error) {
	var events []*Event
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return nil, err
		}
		events = append(events, &ev)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// serializeJSONL serializes events to JSONL format.
func serializeJSONL(events []*Event) ([]byte, error) {
	var buf bytes.Buffer
	for _, ev := range events {
		data, err := json.Marshal(ev)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

// filterEvents applies CreatorID, Start, and End filters to events.
func filterEvents(events []*Event, opts ListOptions) []*Event {
	filtered := make([]*Event, 0, len(events))
	for _, ev := range events {
		// CreatorID filter
		if opts.CreatorID != nil && ev.CreatorID != *opts.CreatorID {
			continue
		}

		// Start filter
		if opts.Start != nil && ev.StartTime.Before(*opts.Start) {
			continue
		}

		// End filter
		if opts.End != nil && ev.StartTime.After(*opts.End) {
			continue
		}

		filtered = append(filtered, ev)
	}
	return filtered
}

// sortEvents sorts events based on ListOptions.
// Start only or Start+End: ascending by StartTime
// End only: descending by StartTime
func sortEvents(events []*Event, opts ListOptions) {
	hasStart := opts.Start != nil
	hasEnd := opts.End != nil

	if hasEnd && !hasStart {
		// End only: descending
		sort.Slice(events, func(i, j int) bool {
			return events[i].StartTime.After(events[j].StartTime)
		})
	} else {
		// Default or Start+End: ascending
		sort.Slice(events, func(i, j int) bool {
			return events[i].StartTime.Before(events[j].StartTime)
		})
	}
}

// applyLimit applies the limit to events if applicable.
// Limit is only applied when Start or End is specified (not both).
func applyLimit(events *[]*Event, opts ListOptions) {
	hasStart := opts.Start != nil
	hasEnd := opts.End != nil
	bothSpecified := hasStart && hasEnd

	// Only apply limit if Start or End (but not both) is specified
	if !bothSpecified && opts.Limit > 0 && len(*events) > opts.Limit {
		*events = (*events)[:opts.Limit]
	}
}
