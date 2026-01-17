package event_test

import (
	"context"
	"log/slog"
	"testing"
	"yuruppu/internal/agent"
	"yuruppu/internal/event"
	"yuruppu/internal/profile"
	eventtoolset "yuruppu/internal/toolset/event"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// mockEventService is a test double for EventService interface.
type mockEventService struct{}

func (m *mockEventService) Create(ctx context.Context, ev *event.Event) error {
	return nil
}

func (m *mockEventService) Get(ctx context.Context, chatRoomID string) (*event.Event, error) {
	return &event.Event{}, nil
}

func (m *mockEventService) List(ctx context.Context, opts event.ListOptions) ([]*event.Event, error) {
	return []*event.Event{}, nil
}

func (m *mockEventService) Update(ctx context.Context, chatRoomID string, description string) error {
	return nil
}

func (m *mockEventService) Delete(ctx context.Context, chatRoomID string) error {
	return nil
}

// mockProfileService is a test double for ProfileService interface.
type mockProfileService struct{}

func (m *mockProfileService) GetUserProfile(ctx context.Context, userID string) (*profile.UserProfile, error) {
	return &profile.UserProfile{DisplayName: "Test User"}, nil
}

// =============================================================================
// NewTools() Tests
// =============================================================================

func TestNewTools(t *testing.T) {
	t.Run("creates all three event tools with valid parameters", func(t *testing.T) {
		// Given: Valid service and configuration
		eventService := &mockEventService{}
		profileService := &mockProfileService{}
		listMaxPeriodDays := 366
		listLimit := 5

		// When: NewTools is called
		tools, err := eventtoolset.NewTools(eventService, profileService, listMaxPeriodDays, listLimit, slog.New(slog.DiscardHandler))

		// Then: Should return 5 tools without error
		require.NoError(t, err)
		require.NotNil(t, tools)
		assert.Len(t, tools, 5, "should return exactly 5 tools")

		// Verify tool names
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			// Each tool should implement agent.Tool interface
			require.Implements(t, (*agent.Tool)(nil), tool)
			toolNames[tool.Name()] = true
		}

		// Verify all expected tools are present
		assert.True(t, toolNames["create_event"], "should include create_event tool")
		assert.True(t, toolNames["get_event"], "should include get_event tool")
		assert.True(t, toolNames["list_events"], "should include list_events tool")
		assert.True(t, toolNames["update_event"], "should include update_event tool")
		assert.True(t, toolNames["delete_event"], "should include delete_event tool")
	})

	t.Run("each tool has valid metadata", func(t *testing.T) {
		// Given: Valid service and configuration
		eventService := &mockEventService{}
		profileService := &mockProfileService{}

		// When: NewTools is called
		tools, err := eventtoolset.NewTools(eventService, profileService, 366, 5, slog.New(slog.DiscardHandler))

		// Then: Each tool should have valid metadata
		require.NoError(t, err)
		for _, tool := range tools {
			assert.NotEmpty(t, tool.Name(), "tool name should not be empty")
			assert.NotEmpty(t, tool.Description(), "tool description should not be empty")
			assert.NotEmpty(t, tool.ParametersJsonSchema(), "parameters schema should not be empty")
			assert.NotEmpty(t, tool.ResponseJsonSchema(), "response schema should not be empty")
		}
	})
}

// =============================================================================
// NewTools() Error Cases
// =============================================================================

func TestNewTools_ErrorCases(t *testing.T) {
	tests := []struct {
		name              string
		eventService      eventtoolset.EventService
		profileService    eventtoolset.ProfileService
		listMaxPeriodDays int
		listLimit         int
		expectError       string
	}{
		{
			name:              "returns error when eventService is nil",
			eventService:      nil,
			profileService:    &mockProfileService{},
			listMaxPeriodDays: 366,
			listLimit:         5,
			expectError:       "eventService",
		},
		{
			name:              "returns error when profileService is nil",
			eventService:      &mockEventService{},
			profileService:    nil,
			listMaxPeriodDays: 366,
			listLimit:         5,
			expectError:       "profileService",
		},
		{
			name:              "returns error when listMaxPeriodDays is zero",
			eventService:      &mockEventService{},
			profileService:    &mockProfileService{},
			listMaxPeriodDays: 0,
			listLimit:         5,
			expectError:       "listMaxPeriodDays",
		},
		{
			name:              "returns error when listMaxPeriodDays is negative",
			eventService:      &mockEventService{},
			profileService:    &mockProfileService{},
			listMaxPeriodDays: -1,
			listLimit:         5,
			expectError:       "listMaxPeriodDays",
		},
		{
			name:              "returns error when listLimit is zero",
			eventService:      &mockEventService{},
			profileService:    &mockProfileService{},
			listMaxPeriodDays: 366,
			listLimit:         0,
			expectError:       "listLimit",
		},
		{
			name:              "returns error when listLimit is negative",
			eventService:      &mockEventService{},
			profileService:    &mockProfileService{},
			listMaxPeriodDays: 366,
			listLimit:         -1,
			expectError:       "listLimit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: NewTools is called with invalid parameters
			tools, err := eventtoolset.NewTools(tt.eventService, tt.profileService, tt.listMaxPeriodDays, tt.listLimit, slog.New(slog.DiscardHandler))

			// Then: Should return error and nil tools
			require.Error(t, err)
			assert.Nil(t, tools)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}

	t.Run("returns error when logger is nil", func(t *testing.T) {
		eventService := &mockEventService{}
		profileService := &mockProfileService{}

		tools, err := eventtoolset.NewTools(eventService, profileService, 366, 5, nil)

		require.Error(t, err)
		assert.Nil(t, tools)
		assert.Contains(t, err.Error(), "logger")
	})
}

// =============================================================================
// NewTools() Edge Cases
// =============================================================================

func TestNewTools_EdgeCases(t *testing.T) {
	t.Run("accepts minimum valid configuration values", func(t *testing.T) {
		// Given: Minimum valid values (1 for both period and limit)
		eventService := &mockEventService{}
		profileService := &mockProfileService{}
		listMaxPeriodDays := 1
		listLimit := 1

		// When: NewTools is called
		tools, err := eventtoolset.NewTools(eventService, profileService, listMaxPeriodDays, listLimit, slog.New(slog.DiscardHandler))

		// Then: Should succeed
		require.NoError(t, err)
		assert.Len(t, tools, 5)
	})

	t.Run("accepts large configuration values", func(t *testing.T) {
		// Given: Large valid values
		eventService := &mockEventService{}
		profileService := &mockProfileService{}
		listMaxPeriodDays := 10000
		listLimit := 1000

		// When: NewTools is called
		tools, err := eventtoolset.NewTools(eventService, profileService, listMaxPeriodDays, listLimit, slog.New(slog.DiscardHandler))

		// Then: Should succeed
		require.NoError(t, err)
		assert.Len(t, tools, 5)
	})
}

// =============================================================================
// Tool Interface Compliance Tests
// =============================================================================

func TestNewTools_ToolInterfaceCompliance(t *testing.T) {
	t.Run("all returned tools implement agent.Tool interface", func(t *testing.T) {
		// Given: Valid configuration
		eventService := &mockEventService{}
		profileService := &mockProfileService{}

		// When: NewTools is called
		tools, err := eventtoolset.NewTools(eventService, profileService, 366, 5, slog.New(slog.DiscardHandler))

		// Then: All tools should implement the agent.Tool interface
		require.NoError(t, err)
		for _, tool := range tools {
			assert.Implements(t, (*agent.Tool)(nil), tool,
				"tool %s should implement agent.Tool interface", tool.Name())
		}
	})

	t.Run("event tools do not implement agent.FinalAction interface", func(t *testing.T) {
		// Given: Valid configuration
		eventService := &mockEventService{}
		profileService := &mockProfileService{}

		// When: NewTools is called
		tools, err := eventtoolset.NewTools(eventService, profileService, 366, 5, slog.New(slog.DiscardHandler))

		// Then: Event tools should NOT implement the agent.FinalAction interface
		// because they require a follow-up reply tool call
		require.NoError(t, err)
		for _, tool := range tools {
			_, implementsFinalAction := tool.(agent.FinalAction)
			assert.False(t, implementsFinalAction,
				"tool %s should NOT implement agent.FinalAction interface", tool.Name())
		}
	})
}

// =============================================================================
// Tool Return Order Tests
// =============================================================================

func TestNewTools_ReturnOrder(t *testing.T) {
	t.Run("returns tools in consistent order", func(t *testing.T) {
		// Given: Valid configuration
		eventService := &mockEventService{}
		profileService := &mockProfileService{}

		// When: NewTools is called multiple times
		tools1, err1 := eventtoolset.NewTools(eventService, profileService, 366, 5, slog.New(slog.DiscardHandler))
		require.NoError(t, err1)

		tools2, err2 := eventtoolset.NewTools(eventService, profileService, 366, 5, slog.New(slog.DiscardHandler))
		require.NoError(t, err2)

		// Then: Tools should be returned in the same order
		require.Len(t, tools1, 5)
		require.Len(t, tools2, 5)
		for i := range 5 {
			assert.Equal(t, tools1[i].Name(), tools2[i].Name(),
				"tool at index %d should have the same name", i)
		}
	})

	t.Run("expected tool order is create, get, list", func(t *testing.T) {
		// Given: Valid configuration
		eventService := &mockEventService{}
		profileService := &mockProfileService{}

		// When: NewTools is called
		tools, err := eventtoolset.NewTools(eventService, profileService, 366, 5, slog.New(slog.DiscardHandler))

		// Then: Tools should follow the expected order
		require.NoError(t, err)
		require.Len(t, tools, 5)

		// Expected order based on implementation
		expectedOrder := []string{"create_event", "get_event", "list_events", "update_event", "delete_event"}
		for i, expectedName := range expectedOrder {
			assert.Equal(t, expectedName, tools[i].Name(),
				"tool at index %d should be %s", i, expectedName)
		}
	})
}
