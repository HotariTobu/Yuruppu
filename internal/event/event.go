package event

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yuruppu/internal/storage"
)

// Event represents an event in a group chat.
type Event struct {
	ChatRoomID  string    `json:"chat_room_id"`
	CreatorID   string    `json:"creator_id"`
	Title       string    `json:"title"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Capacity    int       `json:"capacity"`
	Fee         string    `json:"fee"`
	Description string    `json:"description"`
	ShowCreator bool      `json:"show_creator"`
}

// Service provides event operations.
type Service struct {
	storage storage.Storage
}

// NewService creates a new event service.
func NewService(s storage.Storage) (*Service, error) {
	if s == nil {
		return nil, fmt.Errorf("storage cannot be nil")
	}
	return &Service{storage: s}, nil
}

// Create stores a new event. Returns error if the event already exists.
func (s *Service) Create(ctx context.Context, ev *Event) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = s.storage.Write(ctx, ev.ChatRoomID, "application/json", data, 0)
	if err != nil {
		if isPreconditionFailed(err) {
			return fmt.Errorf("event already exists for chat room %s", ev.ChatRoomID)
		}
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

// Get retrieves an event by chat room ID. Returns nil if not found.
func (s *Service) Get(ctx context.Context, chatRoomID string) (*Event, error) {
	data, _, err := s.storage.Read(ctx, chatRoomID)
	if err != nil {
		return nil, fmt.Errorf("failed to read event: %w", err)
	}
	if data == nil {
		return nil, nil
	}

	var ev Event
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return &ev, nil
}

// List returns all events.
func (s *Service) List(_ context.Context) ([]*Event, error) {
	// TODO: Implement using storage list API
	return []*Event{}, nil
}

func isPreconditionFailed(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "precondition") || strings.Contains(errStr, "412")
}
