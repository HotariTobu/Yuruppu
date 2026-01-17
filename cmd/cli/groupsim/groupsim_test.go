package groupsim_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
	"yuruppu/cmd/cli/groupsim"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewService Tests
// =============================================================================

func TestNewService(t *testing.T) {
	t.Run("creates service with storage", func(t *testing.T) {
		// Given
		store := newMockStorage()

		// When
		svc, err := groupsim.NewService(store)

		// Then
		require.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("returns error when storage is nil", func(t *testing.T) {
		// When
		svc, err := groupsim.NewService(nil)

		// Then
		require.Error(t, err)
		assert.Nil(t, svc)
		assert.Contains(t, err.Error(), "storage cannot be nil")
	})
}

// =============================================================================
// Exists Tests
// =============================================================================

func TestService_Exists(t *testing.T) {
	// AC-001: Service must support group existence checking (FR-001)
	t.Run("returns false for non-existent group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		exists, err := svc.Exists(ctx, "nonexistent-group")

		// Then
		require.NoError(t, err)
		assert.False(t, exists)
		assert.Equal(t, 1, store.readCallCount)
	})

	t.Run("returns true for existing group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Pre-populate storage with group data
		groupData := &groupSim{
			Members:    []string{"alice"},
			BotInGroup: false,
		}
		data, _ := json.Marshal(groupData)
		store.data["mygroup"] = data

		// When
		exists, err := svc.Exists(ctx, "mygroup")

		// Then
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns error when storage read fails", func(t *testing.T) {
		// Given
		store := newMockStorage()
		store.readErr = errors.New("storage error")
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		exists, err := svc.Exists(ctx, "mygroup")

		// Then
		require.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "failed to check group existence")
	})
}

// =============================================================================
// Create Tests
// =============================================================================

func TestService_Create(t *testing.T) {
	// AC-001: Create new group with first member (FR-002)
	t.Run("creates new group with first member", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		err := svc.Create(ctx, "mygroup", "alice")

		// Then
		require.NoError(t, err)
		assert.Equal(t, 1, store.writeCallCount)
		assert.Equal(t, "mygroup", store.lastWriteKey)
		assert.Equal(t, "application/json", store.lastWriteMIMEType)

		// Verify JSON structure
		var stored groupSim
		err = json.Unmarshal(store.lastWriteData, &stored)
		require.NoError(t, err)
		assert.Equal(t, []string{"alice"}, stored.Members)
		assert.False(t, stored.BotInGroup, "bot should not be in group by default (FR-014)")
	})

	// AC-013: Bot not in group by default (FR-014)
	t.Run("bot is not in group by default", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		err := svc.Create(ctx, "newgroup", "alice")

		// Then
		require.NoError(t, err)

		var stored groupSim
		err = json.Unmarshal(store.lastWriteData, &stored)
		require.NoError(t, err)
		assert.False(t, stored.BotInGroup)
	})

	t.Run("returns error when storage write fails", func(t *testing.T) {
		// Given
		store := newMockStorage()
		store.writeErr = errors.New("write failed")
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		err := svc.Create(ctx, "mygroup", "alice")

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create group")
	})

	t.Run("returns error when group already exists", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Pre-populate storage with existing group
		groupData := &groupSim{
			Members:    []string{"bob"},
			BotInGroup: false,
		}
		data, _ := json.Marshal(groupData)
		store.data["mygroup"] = data

		// When
		err := svc.Create(ctx, "mygroup", "alice")

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create group")
	})
}

// =============================================================================
// GetMembers Tests
// =============================================================================

func TestService_GetMembers(t *testing.T) {
	t.Run("returns member list for existing group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Pre-populate storage with group
		groupData := &groupSim{
			Members:    []string{"alice", "bob", "charlie"},
			BotInGroup: true,
		}
		data, _ := json.Marshal(groupData)
		store.data["mygroup"] = data

		// When
		members, err := svc.GetMembers(ctx, "mygroup")

		// Then
		require.NoError(t, err)
		assert.Equal(t, []string{"alice", "bob", "charlie"}, members)
	})

	t.Run("returns error for non-existent group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		members, err := svc.GetMembers(ctx, "nonexistent")

		// Then
		require.Error(t, err)
		assert.Nil(t, members)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("returns error when storage read fails", func(t *testing.T) {
		// Given
		store := newMockStorage()
		store.readErr = errors.New("storage error")
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		members, err := svc.GetMembers(ctx, "mygroup")

		// Then
		require.Error(t, err)
		assert.Nil(t, members)
		assert.Contains(t, err.Error(), "failed to read group")
	})

	t.Run("returns error when JSON unmarshal fails", func(t *testing.T) {
		// Given
		store := newMockStorage()
		store.data["mygroup"] = []byte("invalid json")
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		members, err := svc.GetMembers(ctx, "mygroup")

		// Then
		require.Error(t, err)
		assert.Nil(t, members)
		assert.Contains(t, err.Error(), "failed to unmarshal group data")
	})
}

// =============================================================================
// IsMember Tests
// =============================================================================

func TestService_IsMember(t *testing.T) {
	// AC-002: Allow chat if user is a member (FR-003)
	// AC-003: Error if user is not a member (FR-004)
	tests := []struct {
		name       string
		groupID    string
		userID     string
		members    []string
		want       bool
		wantErrMsg string
	}{
		{
			name:    "returns true for member",
			groupID: "mygroup",
			userID:  "alice",
			members: []string{"alice", "bob"},
			want:    true,
		},
		{
			name:    "returns false for non-member",
			groupID: "mygroup",
			userID:  "charlie",
			members: []string{"alice", "bob"},
			want:    false,
		},
		{
			name:    "returns false for empty member list",
			groupID: "mygroup",
			userID:  "alice",
			members: []string{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			store := newMockStorage()
			svc, _ := groupsim.NewService(store)
			ctx := context.Background()

			// Pre-populate storage with group
			groupData := &groupSim{
				Members:    tt.members,
				BotInGroup: false,
			}
			data, _ := json.Marshal(groupData)
			store.data[tt.groupID] = data

			// When
			isMember, err := svc.IsMember(ctx, tt.groupID, tt.userID)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.want, isMember)
		})
	}

	t.Run("returns error for non-existent group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		isMember, err := svc.IsMember(ctx, "nonexistent", "alice")

		// Then
		require.Error(t, err)
		assert.False(t, isMember)
		assert.Contains(t, err.Error(), "group not found")
	})
}

// =============================================================================
// AddMember Tests
// =============================================================================

func TestService_AddMember(t *testing.T) {
	// AC-010: /invite new user (FR-010)
	t.Run("adds new member to group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Pre-populate storage with group
		groupData := &groupSim{
			Members:    []string{"alice"},
			BotInGroup: false,
		}
		data, _ := json.Marshal(groupData)
		store.data["mygroup"] = data
		store.generation = 1

		// When
		err := svc.AddMember(ctx, "mygroup", "bob")

		// Then
		require.NoError(t, err)
		assert.Equal(t, 1, store.writeCallCount)

		// Verify updated members
		var updated groupSim
		err = json.Unmarshal(store.lastWriteData, &updated)
		require.NoError(t, err)
		assert.Equal(t, []string{"alice", "bob"}, updated.Members)
	})

	// AC-012: /invite existing member (FR-012)
	t.Run("returns error if user is already a member", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Pre-populate storage with group
		groupData := &groupSim{
			Members:    []string{"alice", "bob"},
			BotInGroup: false,
		}
		data, _ := json.Marshal(groupData)
		store.data["mygroup"] = data

		// When
		err := svc.AddMember(ctx, "mygroup", "bob")

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already a member")
		assert.Equal(t, 0, store.writeCallCount, "should not write to storage")
	})

	t.Run("returns error for non-existent group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		err := svc.AddMember(ctx, "nonexistent", "bob")

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("returns error when storage write fails", func(t *testing.T) {
		// Given
		store := newMockStorage()
		store.writeErr = errors.New("write failed")
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Pre-populate storage with group
		groupData := &groupSim{
			Members:    []string{"alice"},
			BotInGroup: false,
		}
		data, _ := json.Marshal(groupData)
		store.data["mygroup"] = data
		store.generation = 1

		// When
		err := svc.AddMember(ctx, "mygroup", "bob")

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update group")
	})
}

// =============================================================================
// IsBotInGroup Tests
// =============================================================================

func TestService_IsBotInGroup(t *testing.T) {
	// AC-013: Bot not in group by default (FR-014, FR-016)
	t.Run("returns false by default", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Create group (bot not in group by default)
		err := svc.Create(ctx, "mygroup", "alice")
		require.NoError(t, err)

		// When
		inGroup, err := svc.IsBotInGroup(ctx, "mygroup")

		// Then
		require.NoError(t, err)
		assert.False(t, inGroup)
	})

	// AC-014: Invite bot to group (FR-015, FR-017)
	t.Run("returns true after AddBot", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Create group and add bot
		err := svc.Create(ctx, "mygroup", "alice")
		require.NoError(t, err)

		err = svc.AddBot(ctx, "mygroup")
		require.NoError(t, err)

		// When
		inGroup, err := svc.IsBotInGroup(ctx, "mygroup")

		// Then
		require.NoError(t, err)
		assert.True(t, inGroup)
	})

	t.Run("returns error for non-existent group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		inGroup, err := svc.IsBotInGroup(ctx, "nonexistent")

		// Then
		require.Error(t, err)
		assert.False(t, inGroup)
		assert.Contains(t, err.Error(), "group not found")
	})
}

// =============================================================================
// AddBot Tests
// =============================================================================

func TestService_AddBot(t *testing.T) {
	// AC-014: Invite bot to group (FR-015)
	t.Run("adds bot to group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Pre-populate storage with group (bot not in group)
		groupData := &groupSim{
			Members:    []string{"alice"},
			BotInGroup: false,
		}
		data, _ := json.Marshal(groupData)
		store.data["mygroup"] = data
		store.generation = 1

		// When
		err := svc.AddBot(ctx, "mygroup")

		// Then
		require.NoError(t, err)
		assert.Equal(t, 1, store.writeCallCount)

		// Verify bot is in group
		var updated groupSim
		err = json.Unmarshal(store.lastWriteData, &updated)
		require.NoError(t, err)
		assert.True(t, updated.BotInGroup)
		assert.Equal(t, []string{"alice"}, updated.Members, "members should not change")
	})

	t.Run("returns error if bot already in group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Pre-populate storage with group (bot already in group)
		groupData := &groupSim{
			Members:    []string{"alice"},
			BotInGroup: true,
		}
		data, _ := json.Marshal(groupData)
		store.data["mygroup"] = data

		// When
		err := svc.AddBot(ctx, "mygroup")

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bot is already in the group")
		assert.Equal(t, 0, store.writeCallCount, "should not write to storage")
	})

	t.Run("returns error for non-existent group", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When
		err := svc.AddBot(ctx, "nonexistent")

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("returns error when storage write fails", func(t *testing.T) {
		// Given
		store := newMockStorage()
		store.writeErr = errors.New("write failed")
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// Pre-populate storage with group
		groupData := &groupSim{
			Members:    []string{"alice"},
			BotInGroup: false,
		}
		data, _ := json.Marshal(groupData)
		store.data["mygroup"] = data
		store.generation = 1

		// When
		err := svc.AddBot(ctx, "mygroup")

		// Then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update group")
	})
}

// =============================================================================
// Persistence Tests
// =============================================================================

func TestService_Persistence(t *testing.T) {
	// AC-017: Group persists across restarts (FR-013)
	t.Run("data survives across service instances", func(t *testing.T) {
		// Given
		store := newMockStorage()
		ctx := context.Background()

		// Create first service instance and create group
		svc1, _ := groupsim.NewService(store)
		err := svc1.Create(ctx, "mygroup", "alice")
		require.NoError(t, err)

		err = svc1.AddMember(ctx, "mygroup", "bob")
		require.NoError(t, err)

		err = svc1.AddBot(ctx, "mygroup")
		require.NoError(t, err)

		// Simulate restart: create new service instance with same storage
		svc2, _ := groupsim.NewService(store)

		// When - read data from new service instance
		members, err := svc2.GetMembers(ctx, "mygroup")
		require.NoError(t, err)

		inGroup, err := svc2.IsBotInGroup(ctx, "mygroup")
		require.NoError(t, err)

		// Then - data should persist
		assert.Equal(t, []string{"alice", "bob"}, members)
		assert.True(t, inGroup)
	})

	t.Run("multiple groups persist independently", func(t *testing.T) {
		// Given
		store := newMockStorage()
		svc, _ := groupsim.NewService(store)
		ctx := context.Background()

		// When - create multiple groups
		err := svc.Create(ctx, "group1", "alice")
		require.NoError(t, err)

		err = svc.Create(ctx, "group2", "bob")
		require.NoError(t, err)

		err = svc.AddBot(ctx, "group1")
		require.NoError(t, err)

		// Then - each group has independent state
		members1, err := svc.GetMembers(ctx, "group1")
		require.NoError(t, err)
		assert.Equal(t, []string{"alice"}, members1)

		members2, err := svc.GetMembers(ctx, "group2")
		require.NoError(t, err)
		assert.Equal(t, []string{"bob"}, members2)

		inGroup1, err := svc.IsBotInGroup(ctx, "group1")
		require.NoError(t, err)
		assert.True(t, inGroup1)

		inGroup2, err := svc.IsBotInGroup(ctx, "group2")
		require.NoError(t, err)
		assert.False(t, inGroup2)
	})
}

// =============================================================================
// Mocks
// =============================================================================

// groupSim matches the internal storage structure from design.md
type groupSim struct {
	Members    []string `json:"members"`
	BotInGroup bool     `json:"botInGroup"`
}

type mockStorage struct {
	data              map[string][]byte
	generation        int64
	readErr           error
	writeErr          error
	readCallCount     int
	writeCallCount    int
	lastWriteKey      string
	lastWriteMIMEType string
	lastWriteData     []byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data:       make(map[string][]byte),
		generation: 1,
	}
}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	m.readCallCount++
	if m.readErr != nil {
		return nil, 0, m.readErr
	}
	data, ok := m.data[key]
	if !ok {
		return nil, 0, nil
	}
	return data, m.generation, nil
}

func (m *mockStorage) Write(ctx context.Context, key, mimeType string, data []byte, expectedGen int64) (int64, error) {
	m.writeCallCount++
	m.lastWriteKey = key
	m.lastWriteMIMEType = mimeType
	m.lastWriteData = data
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	if expectedGen == 0 {
		if _, exists := m.data[key]; exists {
			return 0, fmt.Errorf("precondition failed: object already exists")
		}
	}
	m.data[key] = data
	m.generation++
	return m.generation, nil
}

func (m *mockStorage) GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	return "", nil
}
