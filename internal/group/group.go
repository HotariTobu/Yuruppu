package group

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"yuruppu/internal/storage"
)

// Group represents a LINE group with membership tracking.
type Group struct {
	ID         string   `json:"id"`
	Members    []string `json:"members"`
	BotInGroup bool     `json:"botInGroup"`
}

// Service provides group management operations.
type Service struct {
	storage storage.Storage
}

// NewService creates a new group service.
func NewService(s storage.Storage) (*Service, error) {
	if s == nil {
		return nil, errors.New("storage cannot be nil")
	}
	return &Service{storage: s}, nil
}

// GetGroup retrieves a group by ID. Returns nil if not found.
func (s *Service) GetGroup(ctx context.Context, groupID string) (*Group, int64, error) {
	data, gen, err := s.storage.Read(ctx, groupID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read group: %w", err)
	}
	if data == nil {
		return nil, 0, nil
	}

	var group Group
	if err := json.Unmarshal(data, &group); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal group: %w", err)
	}
	return &group, gen, nil
}

// CreateGroup creates a new group with the first member.
func (s *Service) CreateGroup(ctx context.Context, groupID, firstMemberID string) error {
	group := &Group{
		ID:         groupID,
		Members:    []string{firstMemberID},
		BotInGroup: false,
	}

	data, err := json.Marshal(group)
	if err != nil {
		return fmt.Errorf("failed to marshal group: %w", err)
	}

	_, err = s.storage.Write(ctx, groupID, "application/json", data, 0)
	if err != nil {
		return fmt.Errorf("failed to write group: %w", err)
	}
	return nil
}

// GetMembers returns the list of member user IDs.
func (s *Service) GetMembers(ctx context.Context, groupID string) ([]string, error) {
	group, _, err := s.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, nil
	}
	return group.Members, nil
}

// IsMember checks if a user is a member of the group.
func (s *Service) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	members, err := s.GetMembers(ctx, groupID)
	if err != nil {
		return false, err
	}
	return slices.Contains(members, userID), nil
}

// AddMember adds a user to the group.
func (s *Service) AddMember(ctx context.Context, groupID, userID string) error {
	group, gen, err := s.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("group not found")
	}

	if slices.Contains(group.Members, userID) {
		return fmt.Errorf("%s is already a member of this group", userID)
	}

	group.Members = append(group.Members, userID)
	data, err := json.Marshal(group)
	if err != nil {
		return fmt.Errorf("failed to marshal group: %w", err)
	}

	_, err = s.storage.Write(ctx, groupID, "application/json", data, gen)
	return err
}

// IsBotInGroup checks if the bot is in the group.
func (s *Service) IsBotInGroup(ctx context.Context, groupID string) (bool, error) {
	group, _, err := s.GetGroup(ctx, groupID)
	if err != nil {
		return false, err
	}
	if group == nil {
		return false, nil
	}
	return group.BotInGroup, nil
}

// AddBot marks the bot as joined to the group.
func (s *Service) AddBot(ctx context.Context, groupID string) error {
	group, gen, err := s.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("group not found")
	}

	group.BotInGroup = true
	data, err := json.Marshal(group)
	if err != nil {
		return fmt.Errorf("failed to marshal group: %w", err)
	}

	_, err = s.storage.Write(ctx, groupID, "application/json", data, gen)
	return err
}
