package repl_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"yuruppu/cmd/cli/repl"
	"yuruppu/internal/line"
	"yuruppu/internal/profile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHandler implements a test handler for HandleText calls.
type mockHandler struct {
	mu                sync.Mutex
	calls             []handleTextCall
	joinCalls         []handleJoinCall
	memberJoinedCalls []handleMemberJoinedCall
	returnErr         error
	ctxChecker        func(context.Context) error
}

type handleTextCall struct {
	text   string
	userID string
}

type handleJoinCall struct {
	chatType line.ChatType
	sourceID string
}

type handleMemberJoinedCall struct {
	chatType      line.ChatType
	sourceID      string
	joinedUserIDs []string
}

func (m *mockHandler) HandleText(ctx context.Context, text string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if m.ctxChecker != nil {
		if err := m.ctxChecker(ctx); err != nil {
			return err
		}
	}

	userID, _ := line.UserIDFromContext(ctx)

	m.mu.Lock()
	m.calls = append(m.calls, handleTextCall{
		text:   text,
		userID: userID,
	})
	m.mu.Unlock()

	return m.returnErr
}

func (m *mockHandler) HandleJoin(ctx context.Context) error {
	chatType, _ := line.ChatTypeFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)

	m.mu.Lock()
	m.joinCalls = append(m.joinCalls, handleJoinCall{
		chatType: chatType,
		sourceID: sourceID,
	})
	m.mu.Unlock()

	return m.returnErr
}

func (m *mockHandler) HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error {
	chatType, _ := line.ChatTypeFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)

	m.mu.Lock()
	m.memberJoinedCalls = append(m.memberJoinedCalls, handleMemberJoinedCall{
		chatType:      chatType,
		sourceID:      sourceID,
		joinedUserIDs: append([]string{}, joinedUserIDs...),
	})
	m.mu.Unlock()

	return m.returnErr
}

func (m *mockHandler) getCalls() []handleTextCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]handleTextCall{}, m.calls...)
}

func (m *mockHandler) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockHandler) getJoinCalls() []handleJoinCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]handleJoinCall{}, m.joinCalls...)
}

func (m *mockHandler) joinCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.joinCalls)
}

func (m *mockHandler) getMemberJoinedCalls() []handleMemberJoinedCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]handleMemberJoinedCall{}, m.memberJoinedCalls...)
}

func (m *mockHandler) memberJoinedCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.memberJoinedCalls)
}

type mockProfileService struct {
	profiles map[string]*profile.UserProfile
	err      error
}

func (m *mockProfileService) GetUserProfile(_ context.Context, userID string) (*profile.UserProfile, error) {
	if m.err != nil {
		return nil, m.err
	}
	if p, ok := m.profiles[userID]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("profile not found: %s", userID)
}

type mockGroupSimService struct {
	members    map[string][]string
	botInGroup map[string]bool
	err        error
}

func newMockGroupSimService() *mockGroupSimService {
	return &mockGroupSimService{
		members:    make(map[string][]string),
		botInGroup: make(map[string]bool),
	}
}

func (m *mockGroupSimService) GetMembers(_ context.Context, groupID string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	members, ok := m.members[groupID]
	if !ok {
		return nil, fmt.Errorf("group not found: %s", groupID)
	}
	return members, nil
}

func (m *mockGroupSimService) IsMember(_ context.Context, groupID, userID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	members, ok := m.members[groupID]
	if !ok {
		return false, fmt.Errorf("group not found: %s", groupID)
	}
	for _, member := range members {
		if member == userID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockGroupSimService) AddMember(_ context.Context, groupID, userID string) error {
	if m.err != nil {
		return m.err
	}
	members, ok := m.members[groupID]
	if !ok {
		return fmt.Errorf("group not found: %s", groupID)
	}
	for _, member := range members {
		if member == userID {
			return fmt.Errorf("already a member")
		}
	}
	m.members[groupID] = append(members, userID)
	return nil
}

func (m *mockGroupSimService) IsBotInGroup(_ context.Context, groupID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	botIn, ok := m.botInGroup[groupID]
	if !ok {
		return false, fmt.Errorf("group not found: %s", groupID)
	}
	return botIn, nil
}

func (m *mockGroupSimService) AddBot(_ context.Context, groupID string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.botInGroup[groupID]; !ok {
		return fmt.Errorf("group not found: %s", groupID)
	}
	if m.botInGroup[groupID] {
		return fmt.Errorf("bot is already in the group")
	}
	m.botInGroup[groupID] = true
	return nil
}

func ptr(s string) *string { return &s }

func createBlockingPipe() (*os.File, *os.File) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	return r, w
}

// TestNewRunner_Validation tests that NewRunner validates inputs.
func TestNewRunner_Validation(t *testing.T) {
	t.Run("empty userID", func(t *testing.T) {
		_, err := repl.NewRunner(
			"",
			nil,
			nil,
			nil,
			&mockHandler{},
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			strings.NewReader(""),
			&bytes.Buffer{},
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "userID")
	})

	t.Run("nil handler", func(t *testing.T) {
		_, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			nil,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			strings.NewReader(""),
			&bytes.Buffer{},
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "handler")
	})

	t.Run("nil logger", func(t *testing.T) {
		_, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			&mockHandler{},
			nil,
			strings.NewReader(""),
			&bytes.Buffer{},
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "logger")
	})

	t.Run("nil stdin", func(t *testing.T) {
		_, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			&mockHandler{},
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			nil,
			&bytes.Buffer{},
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stdin")
	})

	t.Run("nil stdout", func(t *testing.T) {
		_, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			&mockHandler{},
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			strings.NewReader(""),
			nil,
			nil,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stdout")
	})
}

func TestNewRunner_Valid(t *testing.T) {
	t.Run("should create runner with valid inputs", func(t *testing.T) {
		r, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			&mockHandler{},
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			strings.NewReader(""),
			&bytes.Buffer{},
			nil,
		)
		require.NoError(t, err)
		assert.NotNil(t, r)
	})
}

// TestRun_QuitCommand tests /quit command exits cleanly.
func TestRun_QuitCommand(t *testing.T) {
	t.Run("should exit cleanly when /quit is entered", func(t *testing.T) {
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, handler.callCount())
		assert.Contains(t, stdout.String(), "> ")
	})
}

// TestRun_EmptyInput tests that empty lines are ignored.
func TestRun_EmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty line", input: "\n/quit\n"},
		{name: "whitespace only", input: "   \n/quit\n"},
		{name: "multiple empty lines", input: "\n\n\n/quit\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := strings.NewReader(tt.input)
			stdout := &bytes.Buffer{}
			handler := &mockHandler{}

			r, err := repl.NewRunner(
				"test-user",
				nil,
				nil,
				nil,
				handler,
				slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
				stdin,
				stdout,
				nil,
			)
			require.NoError(t, err)

			err = r.Run(context.Background())
			require.NoError(t, err)
			assert.Equal(t, 0, handler.callCount())
		})
	}
}

// TestRun_TextInput tests that text input is passed to Handler.
func TestRun_TextInput(t *testing.T) {
	t.Run("should call HandleText with correct text", func(t *testing.T) {
		stdin := strings.NewReader("Hello\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		require.Equal(t, 1, handler.callCount())
		calls := handler.getCalls()
		assert.Equal(t, "Hello", calls[0].text)
	})
}

// TestRun_ContextValues tests that userID, sourceID, and replyToken are set in context.
func TestRun_ContextValues(t *testing.T) {
	t.Run("should set userID, sourceID, and replyToken in context", func(t *testing.T) {
		stdin := strings.NewReader("test message\n/quit\n")
		stdout := &bytes.Buffer{}

		var capturedCtx context.Context
		handler := &mockHandler{
			ctxChecker: func(ctx context.Context) error {
				capturedCtx = ctx
				return nil
			},
		}

		r, err := repl.NewRunner(
			"test-user-123",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		require.NotNil(t, capturedCtx)

		userID, ok := line.UserIDFromContext(capturedCtx)
		assert.True(t, ok)
		assert.Equal(t, "test-user-123", userID)

		sourceID, ok := line.SourceIDFromContext(capturedCtx)
		assert.True(t, ok)
		assert.Equal(t, "test-user-123", sourceID)

		replyToken, ok := line.ReplyTokenFromContext(capturedCtx)
		assert.True(t, ok)
		assert.NotEmpty(t, replyToken)
	})
}

// TestRun_HandlerError tests that handler errors are logged but loop continues.
func TestRun_HandlerError(t *testing.T) {
	t.Run("should log error but continue loop when handler returns error", func(t *testing.T) {
		stdin := strings.NewReader("message1\nmessage2\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{
			returnErr: errors.New("handler processing error"),
		}

		r, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(stderr, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 2, handler.callCount())
		assert.Contains(t, stderr.String(), "handler processing error")
	})
}

// TestRun_ContextCancellation tests that context cancellation causes Run to return.
func TestRun_ContextCancellation(t *testing.T) {
	t.Run("should exit when context is cancelled", func(t *testing.T) {
		pipeReader, pipeWriter := createBlockingPipe()
		defer pipeWriter.Close()

		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			pipeReader,
			stdout,
			nil,
		)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- r.Run(ctx)
		}()

		time.Sleep(50 * time.Millisecond)
		cancel()

		select {
		case err := <-done:
			assert.Error(t, err)
			assert.ErrorIs(t, err, context.Canceled)
		case <-time.After(1 * time.Second):
			t.Fatal("Run did not exit after context cancellation")
		}
	})
}

// TestRun_MultipleMessages tests multiple message exchanges.
func TestRun_MultipleMessages(t *testing.T) {
	t.Run("should handle multiple messages in sequence", func(t *testing.T) {
		stdin := strings.NewReader("Hello\nHow are you?\nGoodbye\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		require.Equal(t, 3, handler.callCount())
		calls := handler.getCalls()
		assert.Equal(t, "Hello", calls[0].text)
		assert.Equal(t, "How are you?", calls[1].text)
		assert.Equal(t, "Goodbye", calls[2].text)

		promptCount := strings.Count(stdout.String(), "> ")
		assert.GreaterOrEqual(t, promptCount, 3)
	})
}

// TestRun_PromptDisplay tests that prompt is displayed.
func TestRun_PromptDisplay(t *testing.T) {
	t.Run("should display prompt on stdout", func(t *testing.T) {
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "> ")
	})
}

// TestRun_StdinEOF tests behavior when stdin reaches EOF.
func TestRun_StdinEOF(t *testing.T) {
	t.Run("should return nil when stdin reaches EOF", func(t *testing.T) {
		pipeReader, pipeWriter := createBlockingPipe()
		pipeWriter.Close()

		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"test-user",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			pipeReader,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
	})
}

// TestRun_GroupMode_ChatContext tests group mode sets correct chat type and source ID.
func TestRun_GroupMode_ChatContext(t *testing.T) {
	t.Run("should set chat type to group and source ID to group ID in group mode", func(t *testing.T) {
		stdin := strings.NewReader("Hello from group\n/quit\n")
		stdout := &bytes.Buffer{}

		var capturedCtx context.Context
		handler := &mockHandler{
			ctxChecker: func(ctx context.Context) error {
				capturedCtx = ctx
				return nil
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		require.NotNil(t, capturedCtx)

		chatType, ok := line.ChatTypeFromContext(capturedCtx)
		assert.True(t, ok)
		assert.Equal(t, line.ChatTypeGroup, chatType)

		sourceID, ok := line.SourceIDFromContext(capturedCtx)
		assert.True(t, ok)
		assert.Equal(t, "mygroup", sourceID)

		userID, ok := line.UserIDFromContext(capturedCtx)
		assert.True(t, ok)
		assert.Equal(t, "alice", userID)
	})
}

// TestRun_OneOnOneMode_ChatContext tests 1-on-1 mode maintains existing behavior.
func TestRun_OneOnOneMode_ChatContext(t *testing.T) {
	t.Run("should set chat type to 1-on-1 and source ID to user ID when no group ID", func(t *testing.T) {
		stdin := strings.NewReader("Hello\n/quit\n")
		stdout := &bytes.Buffer{}

		var capturedCtx context.Context
		handler := &mockHandler{
			ctxChecker: func(ctx context.Context) error {
				capturedCtx = ctx
				return nil
			},
		}

		r, err := repl.NewRunner(
			"alice",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		require.NotNil(t, capturedCtx)

		chatType, ok := line.ChatTypeFromContext(capturedCtx)
		assert.True(t, ok)
		assert.Equal(t, line.ChatTypeOneOnOne, chatType)

		sourceID, ok := line.SourceIDFromContext(capturedCtx)
		assert.True(t, ok)
		assert.Equal(t, "alice", sourceID)

		userID, ok := line.UserIDFromContext(capturedCtx)
		assert.True(t, ok)
		assert.Equal(t, "alice", userID)
	})
}

// TestRun_Prompt_WithProfile tests prompt shows DisplayName(user-id) format.
func TestRun_Prompt_WithProfile(t *testing.T) {
	t.Run("should display DisplayName(user-id)> when user has profile", func(t *testing.T) {
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			profileService,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "Alice(alice)> ")
	})
}

// TestRun_Prompt_WithoutProfile tests prompt shows (user-id) format when no profile.
func TestRun_Prompt_WithoutProfile(t *testing.T) {
	t.Run("should display (user-id)> when user has no profile", func(t *testing.T) {
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"bob"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"bob",
			ptr("mygroup"),
			profileService,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "(bob)> ")
	})
}

// TestRun_Prompt_NoProfileService tests backward compatibility when ProfileService is nil.
func TestRun_Prompt_NoProfileService(t *testing.T) {
	t.Run("should display (user-id)> when ProfileService is nil", func(t *testing.T) {
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "(alice)> ")
	})
}

// TestRun_SwitchCommand_Success tests /switch command successfully switches user.
func TestRun_SwitchCommand_Success(t *testing.T) {
	t.Run("should switch to specified user and update prompt when user is member", func(t *testing.T) {
		stdin := strings.NewReader("/switch charlie\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice":   {DisplayName: "Alice"},
				"charlie": {DisplayName: "Charlie"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice", "charlie"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			profileService,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "Alice(alice)> ")
		assert.Contains(t, output, "Charlie(charlie)> ")
	})
}

// TestRun_SwitchCommand_InvalidUser tests /switch with invalid user.
func TestRun_SwitchCommand_InvalidUser(t *testing.T) {
	t.Run("should show error and keep current user when switching to non-member", func(t *testing.T) {
		stdin := strings.NewReader("/switch unknown\nHello\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			profileService,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "'unknown' is not a member of this group")

		require.Equal(t, 1, handler.callCount())
		calls := handler.getCalls()
		assert.Equal(t, "alice", calls[0].userID)
	})
}

// TestRun_SwitchCommand_NotInGroupMode tests /switch in 1-on-1 mode.
func TestRun_SwitchCommand_NotInGroupMode(t *testing.T) {
	t.Run("should show error when /switch is used in 1-on-1 mode", func(t *testing.T) {
		stdin := strings.NewReader("/switch bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"alice",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "/switch is not available")
	})
}

// TestRun_UsersCommand_Success tests /users lists all group members.
func TestRun_UsersCommand_Success(t *testing.T) {
	t.Run("should list all group members with display names", func(t *testing.T) {
		stdin := strings.NewReader("/users\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice":   {DisplayName: "Alice"},
				"bob":     {DisplayName: "Bob"},
				"charlie": {DisplayName: "Charlie"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice", "bob", "charlie"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			profileService,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "Alice(alice), Bob(bob), Charlie(charlie)")
	})
}

// TestRun_UsersCommand_WithoutProfile tests /users with users without profiles.
func TestRun_UsersCommand_WithoutProfile(t *testing.T) {
	t.Run("should list members showing (user-id) for users without profiles", func(t *testing.T) {
		stdin := strings.NewReader("/users\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice":   {DisplayName: "Alice"},
				"charlie": {DisplayName: "Charlie"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice", "bob", "charlie"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			profileService,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "Alice(alice), (bob), Charlie(charlie)")
	})
}

// TestRun_UsersCommand_NotInGroupMode tests /users in 1-on-1 mode.
func TestRun_UsersCommand_NotInGroupMode(t *testing.T) {
	t.Run("should show error when /users is used in 1-on-1 mode", func(t *testing.T) {
		stdin := strings.NewReader("/users\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"alice",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "/users is not available")
	})
}

// TestRun_InviteCommand_Success tests /invite adds new user to group.
func TestRun_InviteCommand_Success(t *testing.T) {
	t.Run("should add new user to group and display success message", func(t *testing.T) {
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "bob has been invited to the group")

		members, err := groupSim.GetMembers(context.Background(), "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "bob")
	})
}

// TestRun_InviteCommand_ExistingMember tests /invite shows error for existing member.
func TestRun_InviteCommand_ExistingMember(t *testing.T) {
	t.Run("should show error when inviting existing member", func(t *testing.T) {
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice", "bob"}
		groupSim.botInGroup["mygroup"] = false

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "bob is already a member of this group")
	})
}

// TestRun_InviteCommand_NotInGroupMode tests /invite in 1-on-1 mode.
func TestRun_InviteCommand_NotInGroupMode(t *testing.T) {
	t.Run("should show error when /invite is used in 1-on-1 mode", func(t *testing.T) {
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"alice",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "/invite is not available")
	})
}

// TestRun_InviteCommand_EmptyUserID tests /invite with empty user ID.
func TestRun_InviteCommand_EmptyUserID(t *testing.T) {
	t.Run("should show usage error when /invite is called without user ID", func(t *testing.T) {
		stdin := strings.NewReader("/invite\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "usage: /invite <user-id>")
	})
}

// TestRun_InviteCommand_WithWhitespace tests /invite handles whitespace correctly.
func TestRun_InviteCommand_WithWhitespace(t *testing.T) {
	t.Run("should trim whitespace from user ID in /invite command", func(t *testing.T) {
		stdin := strings.NewReader("/invite   bob   \n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "bob has been invited to the group")

		members, err := groupSim.GetMembers(context.Background(), "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "bob")
	})
}

// TestRun_BotNotInGroup_NoLLMCall tests that when bot is not in group, messages are not sent to LLM.
func TestRun_BotNotInGroup_NoLLMCall(t *testing.T) {
	t.Run("should not call HandleText when bot is not in group", func(t *testing.T) {
		stdin := strings.NewReader("Hello\nAnother message\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, handler.callCount())
	})
}

// TestRun_BotInGroup_LLMCalled tests that when bot is in group, messages are processed by LLM.
func TestRun_BotInGroup_LLMCalled(t *testing.T) {
	t.Run("should call HandleText when bot is in group", func(t *testing.T) {
		stdin := strings.NewReader("Hello\nHow are you?\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		require.Equal(t, 2, handler.callCount())
		calls := handler.getCalls()
		assert.Equal(t, "Hello", calls[0].text)
		assert.Equal(t, "How are you?", calls[1].text)
	})
}

// TestRun_OneOnOneMode_AlwaysProcessed tests that in 1-on-1 mode, messages are always processed.
func TestRun_OneOnOneMode_AlwaysProcessed(t *testing.T) {
	t.Run("should always call HandleText in 1-on-1 mode regardless of bot status", func(t *testing.T) {
		stdin := strings.NewReader("Hello\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"alice",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		require.Equal(t, 1, handler.callCount())
		calls := handler.getCalls()
		assert.Equal(t, "Hello", calls[0].text)
		assert.Equal(t, "alice", calls[0].userID)
	})
}

// TestRun_BotStatusCheck_ErrorHandling tests handling of IsBotInGroup errors.
func TestRun_BotStatusCheck_ErrorHandling(t *testing.T) {
	t.Run("should not call HandleText when IsBotInGroup returns error", func(t *testing.T) {
		stdin := strings.NewReader("Hello\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.err = errors.New("database connection failed")

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(stderr, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, handler.callCount())
	})
}

// TestRun_InviteBotCommand_Success tests /invite-bot successfully adds bot and calls HandleJoin.
func TestRun_InviteBotCommand_Success(t *testing.T) {
	t.Run("should add bot to group, call HandleJoin, and display success message", func(t *testing.T) {
		stdin := strings.NewReader("/invite-bot\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "Bot has been invited to the group")

		botInGroup, err := groupSim.IsBotInGroup(context.Background(), "mygroup")
		require.NoError(t, err)
		assert.True(t, botInGroup)

		require.Equal(t, 1, handler.joinCallCount())
		joinCalls := handler.getJoinCalls()
		assert.Equal(t, line.ChatTypeGroup, joinCalls[0].chatType)
		assert.Equal(t, "mygroup", joinCalls[0].sourceID)
	})
}

// TestRun_InviteBotCommand_AlreadyInGroup tests /invite-bot when bot is already in group.
func TestRun_InviteBotCommand_AlreadyInGroup(t *testing.T) {
	t.Run("should show error message when bot is already in group", func(t *testing.T) {
		stdin := strings.NewReader("/invite-bot\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "bot is already in the group")
		assert.Equal(t, 0, handler.joinCallCount())
	})
}

// TestRun_InviteBotCommand_NotInGroupMode tests /invite-bot in 1-on-1 mode.
func TestRun_InviteBotCommand_NotInGroupMode(t *testing.T) {
	t.Run("should show error when /invite-bot is used in 1-on-1 mode", func(t *testing.T) {
		stdin := strings.NewReader("/invite-bot\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}

		r, err := repl.NewRunner(
			"alice",
			nil,
			nil,
			nil,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "/invite-bot is not available")
		assert.Equal(t, 0, handler.joinCallCount())
	})
}

// TestRun_InviteBotCommand_EnablesMessageProcessing tests that messages are processed after bot is invited.
func TestRun_InviteBotCommand_EnablesMessageProcessing(t *testing.T) {
	t.Run("should process messages after bot is invited to group", func(t *testing.T) {
		stdin := strings.NewReader("/invite-bot\nHello bot!\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)

		require.Equal(t, 1, handler.joinCallCount())
		require.Equal(t, 1, handler.callCount())
		calls := handler.getCalls()
		assert.Equal(t, "Hello bot!", calls[0].text)
		assert.Equal(t, "alice", calls[0].userID)
	})
}

// TestRun_InviteCommand_TriggersHandleMemberJoined tests /invite triggers HandleMemberJoined when bot is in group.
func TestRun_InviteCommand_TriggersHandleMemberJoined(t *testing.T) {
	t.Run("should call HandleMemberJoined with invited user ID when bot is in group", func(t *testing.T) {
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "bob has been invited to the group")

		members, err := groupSim.GetMembers(context.Background(), "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "bob")

		require.Equal(t, 1, handler.memberJoinedCallCount())
		memberJoinedCalls := handler.getMemberJoinedCalls()

		assert.Equal(t, line.ChatTypeGroup, memberJoinedCalls[0].chatType)
		assert.Equal(t, "mygroup", memberJoinedCalls[0].sourceID)

		require.Len(t, memberJoinedCalls[0].joinedUserIDs, 1)
		assert.Equal(t, "bob", memberJoinedCalls[0].joinedUserIDs[0])
	})
}

// TestRun_InviteCommand_BotNotInGroup_NoHandleMemberJoined tests /invite does not trigger HandleMemberJoined when bot is not in group.
func TestRun_InviteCommand_BotNotInGroup_NoHandleMemberJoined(t *testing.T) {
	t.Run("should NOT call HandleMemberJoined when bot is not in group", func(t *testing.T) {
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		handler := &mockHandler{}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			stdin,
			stdout,
			nil,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "bob has been invited to the group")

		members, err := groupSim.GetMembers(context.Background(), "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "bob")

		assert.Equal(t, 0, handler.memberJoinedCallCount())
	})
}

// TestRun_InviteCommand_HandleMemberJoinedError tests /invite continues even if HandleMemberJoined returns error.
func TestRun_InviteCommand_HandleMemberJoinedError(t *testing.T) {
	t.Run("should continue and show success message even if HandleMemberJoined returns error", func(t *testing.T) {
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{
			returnErr: errors.New("HandleMemberJoined processing error"),
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true

		r, err := repl.NewRunner(
			"alice",
			ptr("mygroup"),
			nil,
			groupSim,
			handler,
			slog.New(slog.NewTextHandler(stderr, nil)),
			stdin,
			stdout,
			stderr,
		)
		require.NoError(t, err)

		err = r.Run(context.Background())
		require.NoError(t, err)

		assert.Contains(t, stdout.String(), "bob has been invited to the group")

		members, err := groupSim.GetMembers(context.Background(), "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "bob")

		require.Equal(t, 1, handler.memberJoinedCallCount())
		assert.Contains(t, stderr.String(), "HandleMemberJoined processing error")
	})
}
