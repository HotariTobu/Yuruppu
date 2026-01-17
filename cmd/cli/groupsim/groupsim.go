package groupsim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"yuruppu/internal/storage"
)

// groupSim is internal storage structure.
type groupSim struct {
	Members    []string `json:"members"`
	BotInGroup bool     `json:"botInGroup"`
}

// Service provides group simulation operations.
type Service struct {
	storage storage.Storage
}

// NewService creates a new group simulation service.
func NewService(s storage.Storage) (*Service, error) {
	if s == nil {
		return nil, errors.New("storage cannot be nil")
	}
	return &Service{storage: s}, nil
}

// Exists checks if a group exists.
func (s *Service) Exists(ctx context.Context, groupID string) (bool, error) {
	if groupID == "" {
		return false, errors.New("groupID must not be empty")
	}

	data, _, err := s.storage.Read(ctx, groupID)
	if err != nil {
		return false, fmt.Errorf("failed to check group existence: %w", err)
	}
	return data != nil, nil
}

// Create creates a new group with the first member.
func (s *Service) Create(ctx context.Context, groupID, firstMemberID string) error {
	if groupID == "" {
		return errors.New("groupID must not be empty")
	}
	if firstMemberID == "" {
		return errors.New("firstMemberID must not be empty")
	}

	group := groupSim{
		Members:    []string{firstMemberID},
		BotInGroup: false,
	}

	data, err := json.Marshal(group)
	if err != nil {
		return fmt.Errorf("failed to marshal group data: %w", err)
	}

	_, err = s.storage.Write(ctx, groupID, "application/json", data, 0)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	return nil
}

// GetMembers returns the list of members in a group.
func (s *Service) GetMembers(ctx context.Context, groupID string) ([]string, error) {
	if groupID == "" {
		return nil, errors.New("groupID must not be empty")
	}

	group, _, err := s.readGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return group.Members, nil
}

// IsMember checks if a user is a member of a group.
func (s *Service) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	if groupID == "" {
		return false, errors.New("groupID must not be empty")
	}
	if userID == "" {
		return false, errors.New("userID must not be empty")
	}

	group, _, err := s.readGroup(ctx, groupID)
	if err != nil {
		return false, err
	}

	return slices.Contains(group.Members, userID), nil
}

// AddMember adds a new member to a group.
func (s *Service) AddMember(ctx context.Context, groupID, userID string) error {
	if groupID == "" {
		return errors.New("groupID must not be empty")
	}
	if userID == "" {
		return errors.New("userID must not be empty")
	}

	group, gen, err := s.readGroup(ctx, groupID)
	if err != nil {
		return err
	}

	// Check if user is already a member
	if slices.Contains(group.Members, userID) {
		return fmt.Errorf("%s is already a member of this group", userID)
	}

	// Add member
	group.Members = append(group.Members, userID)

	// Write back
	data, err := json.Marshal(group)
	if err != nil {
		return fmt.Errorf("failed to marshal group data: %w", err)
	}

	_, err = s.storage.Write(ctx, groupID, "application/json", data, gen)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

// IsBotInGroup checks if the bot is in a group.
func (s *Service) IsBotInGroup(ctx context.Context, groupID string) (bool, error) {
	if groupID == "" {
		return false, errors.New("groupID must not be empty")
	}

	group, _, err := s.readGroup(ctx, groupID)
	if err != nil {
		return false, err
	}
	return group.BotInGroup, nil
}

// AddBot adds the bot to a group.
func (s *Service) AddBot(ctx context.Context, groupID string) error {
	if groupID == "" {
		return errors.New("groupID must not be empty")
	}

	group, gen, err := s.readGroup(ctx, groupID)
	if err != nil {
		return err
	}

	// Check if bot is already in group
	if group.BotInGroup {
		return fmt.Errorf("bot is already in group '%s'", groupID)
	}

	// Add bot
	group.BotInGroup = true

	// Write back
	data, err := json.Marshal(group)
	if err != nil {
		return fmt.Errorf("failed to marshal group data: %w", err)
	}

	_, err = s.storage.Write(ctx, groupID, "application/json", data, gen)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

// readGroup reads group data from storage along with its generation.
func (s *Service) readGroup(ctx context.Context, groupID string) (*groupSim, int64, error) {
	data, gen, err := s.storage.Read(ctx, groupID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read group: %w", err)
	}
	if data == nil {
		return nil, 0, fmt.Errorf("group '%s' not found", groupID)
	}

	var group groupSim
	if err := json.Unmarshal(data, &group); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal group data: %w", err)
	}

	return &group, gen, nil
}
