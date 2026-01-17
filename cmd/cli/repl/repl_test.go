package repl_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
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
	callDelay         time.Duration // Simulates processing delay
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
	if m.callDelay > 0 {
		time.Sleep(m.callDelay)
	}

	// Check context cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Run custom context checker if provided
	if m.ctxChecker != nil {
		if err := m.ctxChecker(ctx); err != nil {
			return err
		}
	}

	// Extract userID from context
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
	// Extract chat type and source ID from context
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
	// Extract chat type and source ID from context
	chatType, _ := line.ChatTypeFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)

	m.mu.Lock()
	m.memberJoinedCalls = append(m.memberJoinedCalls, handleMemberJoinedCall{
		chatType:      chatType,
		sourceID:      sourceID,
		joinedUserIDs: append([]string{}, joinedUserIDs...), // Copy slice
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

// TestRun_QuitCommand tests /quit command exits cleanly
// AC-009: Graceful exit with /quit [FR-010]
func TestRun_QuitCommand(t *testing.T) {
	t.Run("should exit cleanly when /quit is entered", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err, "should exit cleanly without errors")
		assert.Equal(t, 0, handler.callCount(), "should not call HandleText for /quit command")
		assert.Contains(t, stdout.String(), "> ", "should display prompt before /quit")
	})
}

// TestRun_EmptyInput tests that empty lines are ignored
func TestRun_EmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty line",
			input: "\n/quit\n",
		},
		{
			name:  "whitespace only",
			input: "   \n/quit\n",
		},
		{
			name:  "multiple empty lines",
			input: "\n\n\n/quit\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stdin := strings.NewReader(tt.input)
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			handler := &mockHandler{}
			logger := slog.New(slog.NewTextHandler(stderr, nil))

			ctx := context.Background()
			cfg := repl.Config{
				UserID:  "test-user",
				Handler: handler,
				Logger:  logger,
				Stdin:   stdin,
				Stdout:  stdout,
			}

			// When
			err := repl.Run(ctx, cfg)

			// Then
			require.NoError(t, err)
			assert.Equal(t, 0, handler.callCount(), "should not call HandleText for empty lines")
		})
	}
}

// TestRun_TextInput tests that text input is passed to Handler
// AC-001: Interactive REPL mode [FR-001, FR-003]
func TestRun_TextInput(t *testing.T) {
	t.Run("should call HandleText with correct text", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("Hello\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		require.Equal(t, 1, handler.callCount(), "should call HandleText once")
		calls := handler.getCalls()
		assert.Equal(t, "Hello", calls[0].text)
	})
}

// TestRun_ContextValues tests that userID, sourceID, and replyToken are set in context
func TestRun_ContextValues(t *testing.T) {
	t.Run("should set userID, sourceID, and replyToken in context", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("test message\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		var capturedCtx context.Context
		handler := &mockHandler{
			ctxChecker: func(ctx context.Context) error {
				capturedCtx = ctx
				return nil
			},
		}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user-123",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		require.NotNil(t, capturedCtx, "context should be captured")

		// Check userID
		userID, ok := line.UserIDFromContext(capturedCtx)
		assert.True(t, ok, "userID should be in context")
		assert.Equal(t, "test-user-123", userID)

		// Check sourceID (should equal userID in CLI mode per TR-003)
		sourceID, ok := line.SourceIDFromContext(capturedCtx)
		assert.True(t, ok, "sourceID should be in context")
		assert.Equal(t, "test-user-123", sourceID)

		// Check replyToken
		replyToken, ok := line.ReplyTokenFromContext(capturedCtx)
		assert.True(t, ok, "replyToken should be in context")
		assert.NotEmpty(t, replyToken, "replyToken should not be empty")
	})
}

// TestRun_HandlerError tests that handler errors are logged but loop continues
func TestRun_HandlerError(t *testing.T) {
	t.Run("should log error but continue loop when handler returns error", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("message1\nmessage2\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{
			returnErr: errors.New("handler processing error"),
		}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err, "should not exit on handler error")
		assert.Equal(t, 2, handler.callCount(), "should continue processing after error")
		assert.Contains(t, stderr.String(), "handler processing error", "should log handler error")
	})
}

// TestRun_ContextCancellation tests that context cancellation causes Run to return
func TestRun_ContextCancellation(t *testing.T) {
	t.Run("should exit when context is cancelled", func(t *testing.T) {
		// Given
		// Use a custom reader that blocks until cancelled
		pipeReader, pipeWriter := createBlockingPipe()
		defer pipeWriter.Close()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx, cancel := context.WithCancel(context.Background())
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   pipeReader,
			Stdout:  stdout,
		}

		// When
		done := make(chan error, 1)
		go func() {
			done <- repl.Run(ctx, cfg)
		}()

		// Give REPL time to start and display prompt
		time.Sleep(50 * time.Millisecond)

		// Cancel context
		cancel()

		// Wait for Run to exit
		select {
		case err := <-done:
			// Then
			assert.Error(t, err, "should return error when context is cancelled")
			assert.ErrorIs(t, err, context.Canceled)
		case <-time.After(1 * time.Second):
			t.Fatal("Run did not exit after context cancellation")
		}
	})
}

// TestRun_MultipleMessages tests multiple message exchanges
// AC-001: Interactive REPL mode [FR-001, FR-003]
func TestRun_MultipleMessages(t *testing.T) {
	t.Run("should handle multiple messages in sequence", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("Hello\nHow are you?\nGoodbye\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		require.Equal(t, 3, handler.callCount())
		calls := handler.getCalls()
		assert.Equal(t, "Hello", calls[0].text)
		assert.Equal(t, "How are you?", calls[1].text)
		assert.Equal(t, "Goodbye", calls[2].text)

		// Check that prompt appears multiple times
		promptCount := strings.Count(stdout.String(), "> ")
		assert.GreaterOrEqual(t, promptCount, 3, "should display prompt before each input")
	})
}

// TestRun_PromptDisplay tests that "> " prompt is displayed
// AC-001: Interactive REPL mode [FR-001, FR-003]
func TestRun_PromptDisplay(t *testing.T) {
	t.Run("should display '> ' prompt on stdout", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "> ", "should display '> ' prompt")
	})
}

// TestRun_CtrlC_Single tests that single Ctrl+C shows warning message
// AC-011: Graceful exit with Ctrl+C twice [FR-010]
// NOTE: This test is skipped as it sends real SIGINT signals which interfere with test execution.
// SIGINT handling should be tested manually or with integration tests.
func TestRun_CtrlC_Single(t *testing.T) {
	t.Skip("Skipping SIGINT test - sends real signals to test process")
	t.Run("should show warning message on first Ctrl+C", func(t *testing.T) {
		// Given
		pipeReader, pipeWriter := createBlockingPipe()
		defer pipeWriter.Close()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   pipeReader,
			Stdout:  stdout,
		}

		// When
		done := make(chan error, 1)
		go func() {
			done <- repl.Run(ctx, cfg)
		}()

		// Give REPL time to start
		time.Sleep(50 * time.Millisecond)

		// Send first SIGINT
		sendSigInt()
		time.Sleep(100 * time.Millisecond)

		// Check stderr for warning message
		stderrContent := stderr.String()
		assert.Contains(t, stderrContent, "Ctrl+C", "should show Ctrl+C warning")
		assert.Contains(t, stderrContent, "again", "should show 'again' instruction")

		// Send second SIGINT to exit
		sendSigInt()

		// Wait for exit
		select {
		case err := <-done:
			require.NoError(t, err, "should exit cleanly after second Ctrl+C")
		case <-time.After(1 * time.Second):
			t.Fatal("Run did not exit after second Ctrl+C")
		}
	})
}

// TestRun_CtrlC_Twice tests that two Ctrl+C signals exit cleanly
// AC-011: Graceful exit with Ctrl+C twice [FR-010]
// NOTE: This test is skipped as it sends real SIGINT signals which interfere with test execution.
// SIGINT handling should be tested manually or with integration tests.
func TestRun_CtrlC_Twice(t *testing.T) {
	t.Skip("Skipping SIGINT test - sends real signals to test process")
	t.Run("should exit cleanly after two Ctrl+C signals", func(t *testing.T) {
		// Given
		pipeReader, pipeWriter := createBlockingPipe()
		defer pipeWriter.Close()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   pipeReader,
			Stdout:  stdout,
		}

		// When
		done := make(chan error, 1)
		go func() {
			done <- repl.Run(ctx, cfg)
		}()

		// Give REPL time to start
		time.Sleep(50 * time.Millisecond)

		// Send two SIGINT signals
		sendSigInt()
		time.Sleep(50 * time.Millisecond)
		sendSigInt()

		// Then
		select {
		case err := <-done:
			require.NoError(t, err, "should exit cleanly without errors")
			assert.Equal(t, 0, handler.callCount(), "should not process any messages")
		case <-time.After(1 * time.Second):
			t.Fatal("Run did not exit after two Ctrl+C signals")
		}
	})
}

// TestRun_CtrlC_Reset tests that Ctrl+C counter resets after user input
// NOTE: This test is skipped as it sends real SIGINT signals which interfere with test execution.
// SIGINT handling should be tested manually or with integration tests.
func TestRun_CtrlC_Reset(t *testing.T) {
	t.Skip("Skipping SIGINT test - sends real signals to test process")
	t.Run("should reset Ctrl+C counter after user provides input", func(t *testing.T) {
		// Given
		pipeReader, pipeWriter := createBlockingPipe()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   pipeReader,
			Stdout:  stdout,
		}

		// When
		done := make(chan error, 1)
		go func() {
			done <- repl.Run(ctx, cfg)
		}()

		// Give REPL time to start
		time.Sleep(50 * time.Millisecond)

		// Send first Ctrl+C
		sendSigInt()
		time.Sleep(50 * time.Millisecond)

		// Send user input (should reset Ctrl+C counter)
		_, err := fmt.Fprintln(pipeWriter, "Hello")
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)

		// Send first Ctrl+C again (should show warning, not exit)
		sendSigInt()
		time.Sleep(100 * time.Millisecond)

		// Verify still running (warning should be shown)
		select {
		case <-done:
			t.Fatal("REPL should not have exited after single Ctrl+C following user input")
		default:
			// Expected: still running
		}

		// Send second Ctrl+C to exit
		sendSigInt()

		// Then
		select {
		case err := <-done:
			require.NoError(t, err)
			pipeWriter.Close()
		case <-time.After(1 * time.Second):
			pipeWriter.Close()
			t.Fatal("Run did not exit after second Ctrl+C")
		}
	})
}

// TestRun_StdinError tests behavior when stdin read fails
func TestRun_StdinError(t *testing.T) {
	t.Run("should return error when stdin read fails", func(t *testing.T) {
		// Given
		pipeReader, pipeWriter := createBlockingPipe()
		// Close writer immediately to cause EOF
		pipeWriter.Close()

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   pipeReader,
			Stdout:  stdout,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		// EOF is acceptable (user closed stdin)
		// This should exit cleanly, not error
		require.NoError(t, err, "EOF should be treated as clean exit")
	})
}

// TestRun_ConfigValidation tests that invalid config returns error
func TestRun_ConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		modifyCtx func(context.Context) context.Context
		modifyCfg func(*repl.Config)
		wantErr   string
	}{
		{
			name: "nil context",
			modifyCtx: func(ctx context.Context) context.Context {
				return nil
			},
			wantErr: "context",
		},
		{
			name: "empty user ID",
			modifyCfg: func(cfg *repl.Config) {
				cfg.UserID = ""
			},
			wantErr: "userID",
		},
		{
			name: "nil handler",
			modifyCfg: func(cfg *repl.Config) {
				cfg.Handler = nil
			},
			wantErr: "handler",
		},
		{
			name: "nil logger",
			modifyCfg: func(cfg *repl.Config) {
				cfg.Logger = nil
			},
			wantErr: "logger",
		},
		{
			name: "nil stdin",
			modifyCfg: func(cfg *repl.Config) {
				cfg.Stdin = nil
			},
			wantErr: "stdin",
		},
		{
			name: "nil stdout",
			modifyCfg: func(cfg *repl.Config) {
				cfg.Stdout = nil
			},
			wantErr: "stdout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stdin := strings.NewReader("/quit\n")
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			handler := &mockHandler{}
			logger := slog.New(slog.NewTextHandler(stderr, nil))

			ctx := context.Background()
			if tt.modifyCtx != nil {
				ctx = tt.modifyCtx(ctx)
			}

			cfg := repl.Config{
				UserID:  "test-user",
				Handler: handler,
				Logger:  logger,
				Stdin:   stdin,
				Stdout:  stdout,
			}
			if tt.modifyCfg != nil {
				tt.modifyCfg(&cfg)
			}

			// When
			err := repl.Run(ctx, cfg)

			// Then
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// Helper functions

// createBlockingPipe creates a pipe that blocks on reads until data is written
func createBlockingPipe() (*os.File, *os.File) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	return r, w
}

// sendSigInt sends SIGINT to the current process
func sendSigInt() {
	// Get the current process
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		panic(err)
	}

	// Send SIGINT
	if err := proc.Signal(syscall.SIGINT); err != nil {
		panic(err)
	}
}

// Alternative approach: Test SIGINT handling through signal channel
// NOTE: This test is skipped as it sends real SIGINT signals which interfere with test execution.
// SIGINT handling should be tested manually or with integration tests.
func TestRun_CtrlC_SignalChannel(t *testing.T) {
	t.Skip("Skipping SIGINT test - sends real signals to test process")
	t.Run("should handle SIGINT through signal channel", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("") // No stdin input
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		// Setup signal notification
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)
		defer signal.Stop(sigChan)

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "test-user",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
		}

		// When
		done := make(chan error, 1)
		go func() {
			done <- repl.Run(ctx, cfg)
		}()

		// Give REPL time to start
		time.Sleep(50 * time.Millisecond)

		// Simulate two SIGINT signals
		sigChan <- syscall.SIGINT
		time.Sleep(50 * time.Millisecond)
		sigChan <- syscall.SIGINT

		// Then
		select {
		case err := <-done:
			// May return error or nil depending on implementation
			// The important thing is that it exits
			_ = err
		case <-time.After(1 * time.Second):
			t.Fatal("Run did not exit after SIGINT signals")
		}
	})
}

func ptr(s string) *string { return &s }

type mockProfileService struct {
	profiles map[string]*profile.UserProfile
	err      error
}

func (m *mockProfileService) GetUserProfile(ctx context.Context, userID string) (*profile.UserProfile, error) {
	if m.err != nil {
		return nil, m.err
	}
	if p, ok := m.profiles[userID]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("profile not found: %s", userID)
}

// TestRun_GroupMode_ChatContext tests group mode sets correct chat type and source ID
// AC-005: Group chat context [FR-006]
func TestRun_GroupMode_ChatContext(t *testing.T) {
	t.Run("should set chat type to group and source ID to group ID in group mode", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("Hello from group\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		var capturedCtx context.Context
		handler := &mockHandler{
			ctxChecker: func(ctx context.Context) error {
				capturedCtx = ctx
				return nil
			},
		}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "alice",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
			GroupID: ptr("mygroup"),
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		require.NotNil(t, capturedCtx, "context should be captured")

		// Check chat type is "group"
		chatType, ok := line.ChatTypeFromContext(capturedCtx)
		assert.True(t, ok, "chatType should be in context")
		assert.Equal(t, line.ChatTypeGroup, chatType)

		// Check source ID equals group ID
		sourceID, ok := line.SourceIDFromContext(capturedCtx)
		assert.True(t, ok, "sourceID should be in context")
		assert.Equal(t, "mygroup", sourceID)

		// Check user ID is still set correctly
		userID, ok := line.UserIDFromContext(capturedCtx)
		assert.True(t, ok, "userID should be in context")
		assert.Equal(t, "alice", userID)
	})
}

// TestRun_OneOnOneMode_ChatContext tests 1-on-1 mode maintains existing behavior
// AC-004: No group-id means 1-on-1 [FR-005]
func TestRun_OneOnOneMode_ChatContext(t *testing.T) {
	t.Run("should set chat type to 1-on-1 and source ID to user ID when no group ID", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("Hello\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		var capturedCtx context.Context
		handler := &mockHandler{
			ctxChecker: func(ctx context.Context) error {
				capturedCtx = ctx
				return nil
			},
		}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "alice",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
			// GroupID is empty (default)
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		require.NotNil(t, capturedCtx, "context should be captured")

		// Check chat type is "1-on-1"
		chatType, ok := line.ChatTypeFromContext(capturedCtx)
		assert.True(t, ok, "chatType should be in context")
		assert.Equal(t, line.ChatTypeOneOnOne, chatType)

		// Check source ID equals user ID
		sourceID, ok := line.SourceIDFromContext(capturedCtx)
		assert.True(t, ok, "sourceID should be in context")
		assert.Equal(t, "alice", sourceID)

		// Check user ID is set correctly
		userID, ok := line.UserIDFromContext(capturedCtx)
		assert.True(t, ok, "userID should be in context")
		assert.Equal(t, "alice", userID)
	})
}

// TestRun_Prompt_WithProfile tests prompt shows DisplayName(user-id) format
// AC-006: Prompt shows current user with profile [FR-007]
func TestRun_Prompt_WithProfile(t *testing.T) {
	t.Run("should display DisplayName(user-id)> when user has profile", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		ctx := context.Background()
		cfg := repl.Config{
			UserID:         "alice",
			Handler:        handler,
			Logger:         logger,
			Stdin:          stdin,
			Stdout:         stdout,
			GroupID:        ptr("mygroup"),
			ProfileService: profileService,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "Alice(alice)> ", "should display prompt with display name")
	})
}

// TestRun_Prompt_WithoutProfile tests prompt shows (user-id) format when no profile
// AC-006b: Prompt shows current user without profile [FR-007]
func TestRun_Prompt_WithoutProfile(t *testing.T) {
	t.Run("should display (user-id)> when user has no profile", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				// "bob" has no profile
			},
		}

		ctx := context.Background()
		cfg := repl.Config{
			UserID:         "bob",
			Handler:        handler,
			Logger:         logger,
			Stdin:          stdin,
			Stdout:         stdout,
			GroupID:        ptr("mygroup"),
			ProfileService: profileService,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "(bob)> ", "should display prompt with user-id only")
	})
}

// TestRun_Prompt_OneOnOneWithProfile tests 1-on-1 mode also uses new prompt format
// FR-007: Prompt format applies to all modes, not just group mode
func TestRun_Prompt_OneOnOneWithProfile(t *testing.T) {
	t.Run("should display DisplayName(user-id)> in 1-on-1 mode when user has profile", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"charlie": {DisplayName: "Charlie"},
			},
		}

		ctx := context.Background()
		cfg := repl.Config{
			UserID:         "charlie",
			Handler:        handler,
			Logger:         logger,
			Stdin:          stdin,
			Stdout:         stdout,
			ProfileService: profileService,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "Charlie(charlie)> ", "should display prompt with display name in 1-on-1 mode")
	})
}

// TestRun_Prompt_OneOnOneWithoutProfile tests 1-on-1 mode shows (user-id) without profile
// FR-007: Prompt format applies to all modes
func TestRun_Prompt_OneOnOneWithoutProfile(t *testing.T) {
	t.Run("should display (user-id)> in 1-on-1 mode when user has no profile", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				// "dave" has no profile
			},
		}

		ctx := context.Background()
		cfg := repl.Config{
			UserID:         "dave",
			Handler:        handler,
			Logger:         logger,
			Stdin:          stdin,
			Stdout:         stdout,
			ProfileService: profileService,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "(dave)> ", "should display prompt with user-id only in 1-on-1 mode")
	})
}

// TestRun_Prompt_ProfileGetterError tests fallback when profile getter returns error
func TestRun_Prompt_ProfileGetterError(t *testing.T) {
	t.Run("should display (user-id)> when profile getter returns error", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			err: errors.New("profile service unavailable"),
		}

		ctx := context.Background()
		cfg := repl.Config{
			UserID:         "alice",
			Handler:        handler,
			Logger:         logger,
			Stdin:          stdin,
			Stdout:         stdout,
			GroupID:        ptr("mygroup"),
			ProfileService: profileService,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "(alice)> ", "should display prompt with user-id only on error")
	})
}

// TestRun_Prompt_NoProfileGetter tests backward compatibility when ProfileGetter is nil
func TestRun_Prompt_NoProfileGetter(t *testing.T) {
	t.Run("should display (user-id)> when ProfileGetter is nil", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "alice",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
			GroupID: ptr("mygroup"),
			// ProfileGetter is nil
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "(alice)> ", "should display prompt with user-id only when no ProfileGetter")
	})
}

// mockGroupSimService implements a test group simulation service for tests.
type mockGroupSimService struct {
	members    map[string][]string // groupID -> userIDs
	botInGroup map[string]bool     // groupID -> bot status
	err        error
}

func newMockGroupSimService() *mockGroupSimService {
	return &mockGroupSimService{
		members:    make(map[string][]string),
		botInGroup: make(map[string]bool),
	}
}

func (m *mockGroupSimService) GetMembers(ctx context.Context, groupID string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	members, ok := m.members[groupID]
	if !ok {
		return nil, fmt.Errorf("group not found: %s", groupID)
	}
	return members, nil
}

func (m *mockGroupSimService) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
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

func (m *mockGroupSimService) AddMember(ctx context.Context, groupID, userID string) error {
	if m.err != nil {
		return m.err
	}
	members, ok := m.members[groupID]
	if !ok {
		return fmt.Errorf("group not found: %s", groupID)
	}
	// Check if already member
	for _, member := range members {
		if member == userID {
			return fmt.Errorf("already a member")
		}
	}
	m.members[groupID] = append(members, userID)
	return nil
}

func (m *mockGroupSimService) IsBotInGroup(ctx context.Context, groupID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	botIn, ok := m.botInGroup[groupID]
	if !ok {
		return false, fmt.Errorf("group not found: %s", groupID)
	}
	return botIn, nil
}

func (m *mockGroupSimService) AddBot(ctx context.Context, groupID string) error {
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

// TestRun_SwitchCommand_Success tests /switch command successfully switches user
// AC-007: /switch command [FR-008]
func TestRun_SwitchCommand_Success(t *testing.T) {
	t.Run("should switch to specified user and update prompt when user is member", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/switch charlie\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

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

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		output := stdout.String()
		// Should start with Alice prompt
		assert.Contains(t, output, "Alice(alice)> ", "should display initial prompt for alice")
		// After /switch, should show Charlie prompt
		assert.Contains(t, output, "Charlie(charlie)> ", "should display updated prompt for charlie")
	})
}

// TestRun_SwitchCommand_InvalidUser tests /switch with invalid user
// AC-008: /switch with invalid user [FR-008, Error]
func TestRun_SwitchCommand_InvalidUser(t *testing.T) {
	t.Run("should show error and keep current user when switching to non-member", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/switch unknown\nHello\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
				"bob":   {DisplayName: "Bob"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice", "bob"}
		groupSim.botInGroup["mygroup"] = true

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "'unknown' is not a member of this group", "should display error message")

		// Check that user remains alice by verifying HandleText was called with alice
		require.Equal(t, 1, handler.callCount(), "should call HandleText for non-command message")
		calls := handler.getCalls()
		assert.Equal(t, "alice", calls[0].userID, "current user should remain alice")
	})
}

// TestRun_SwitchCommand_NotInGroupMode tests /switch in 1-on-1 mode
func TestRun_SwitchCommand_NotInGroupMode(t *testing.T) {
	t.Run("should show error when /switch is used in 1-on-1 mode", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/switch bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		ctx := context.Background()
		cfg := repl.Config{
			UserID:         "alice",
			Handler:        handler,
			Logger:         logger,
			Stdin:          stdin,
			Stdout:         stdout,
			Stderr:         stderr,
			ProfileService: profileService,
			// GroupID is empty (1-on-1 mode)
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "/switch is not available", "should display unavailable command error")
	})
}

// TestRun_UsersCommand_Success tests /users lists all group members
// AC-009: /users command [FR-009]
func TestRun_UsersCommand_Success(t *testing.T) {
	t.Run("should list all group members with display names", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/users\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

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

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "Alice(alice), Bob(bob), Charlie(charlie)", "should list all members with display names")
	})
}

// TestRun_UsersCommand_WithoutProfile tests /users with users without profiles
func TestRun_UsersCommand_WithoutProfile(t *testing.T) {
	t.Run("should list members showing (user-id) for users without profiles", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/users\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
				// bob has no profile
				"charlie": {DisplayName: "Charlie"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice", "bob", "charlie"}
		groupSim.botInGroup["mygroup"] = true

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "Alice(alice), (bob), Charlie(charlie)", "should list members with (user-id) for users without profiles")
	})
}

// TestRun_UsersCommand_NotInGroupMode tests /users in 1-on-1 mode
func TestRun_UsersCommand_NotInGroupMode(t *testing.T) {
	t.Run("should show error when /users is used in 1-on-1 mode", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/users\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		ctx := context.Background()
		cfg := repl.Config{
			UserID:         "alice",
			Handler:        handler,
			Logger:         logger,
			Stdin:          stdin,
			Stdout:         stdout,
			Stderr:         stderr,
			ProfileService: profileService,
			// GroupID is empty (1-on-1 mode)
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "/users is not available", "should display unavailable command error")
	})
}

// TestRun_InviteCommand_Success tests /invite adds new user to group
// AC-010: /invite new user [FR-010]
func TestRun_InviteCommand_Success(t *testing.T) {
	t.Run("should add new user to group and display success message", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "bob has been invited to the group", "should display success message")

		// Verify bob was added to members
		members, err := groupSim.GetMembers(ctx, "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "bob", "bob should be added to group members")
	})
}

// TestRun_InviteCommand_UserWithoutProfile tests /invite works for users without profile
// AC-011: /invite user without profile [FR-010, FR-011]
func TestRun_InviteCommand_UserWithoutProfile(t *testing.T) {
	t.Run("should add user without profile without triggering profile creation", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite newuser\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
				// "newuser" has no profile
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "newuser has been invited to the group", "should display success message")

		// Verify newuser was added to members
		members, err := groupSim.GetMembers(ctx, "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "newuser", "newuser should be added to group members")

		// Verify no profile was created (profile getter not called for invitation)
		_, err = profileService.GetUserProfile(ctx, "newuser")
		assert.Error(t, err, "newuser should still have no profile")
	})
}

// TestRun_InviteCommand_ExistingMember tests /invite shows error for existing member
// AC-012: /invite existing member [FR-012]
func TestRun_InviteCommand_ExistingMember(t *testing.T) {
	t.Run("should show error when inviting existing member", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
				"bob":   {DisplayName: "Bob"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice", "bob"}
		groupSim.botInGroup["mygroup"] = false

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "bob is already a member of this group", "should display error message to stderr")

		// Verify membership unchanged (still 2 members)
		members, err := groupSim.GetMembers(ctx, "mygroup")
		require.NoError(t, err)
		assert.Equal(t, 2, len(members), "membership count should remain unchanged")
		assert.Contains(t, members, "alice", "alice should still be a member")
		assert.Contains(t, members, "bob", "bob should still be a member")
	})
}

// TestRun_InviteCommand_NotInGroupMode tests /invite in 1-on-1 mode
func TestRun_InviteCommand_NotInGroupMode(t *testing.T) {
	t.Run("should show error when /invite is used in 1-on-1 mode", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		ctx := context.Background()
		cfg := repl.Config{
			UserID:         "alice",
			Handler:        handler,
			Logger:         logger,
			Stdin:          stdin,
			Stdout:         stdout,
			Stderr:         stderr,
			ProfileService: profileService,
			// GroupID is empty (1-on-1 mode)
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "/invite is not available", "should display unavailable command error")
	})
}

// TestRun_InviteCommand_EmptyUserID tests /invite with empty user ID
func TestRun_InviteCommand_EmptyUserID(t *testing.T) {
	t.Run("should show usage error when /invite is called without user ID", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "usage: /invite <user-id>", "should display usage message")
	})
}

// TestRun_InviteCommand_WithWhitespace tests /invite handles whitespace correctly
func TestRun_InviteCommand_WithWhitespace(t *testing.T) {
	t.Run("should trim whitespace from user ID in /invite command", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite   bob   \n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "bob has been invited to the group", "should display success message")

		// Verify bob was added (whitespace trimmed)
		members, err := groupSim.GetMembers(ctx, "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "bob", "bob should be added to group members")
	})
}

// TestRun_BotNotInGroup_NoLLMCall tests that when bot is not in group, messages are not sent to LLM
// AC-013: Bot not in group by default [FR-014, FR-016]
func TestRun_BotNotInGroup_NoLLMCall(t *testing.T) {
	t.Run("should not call HandleText when bot is not in group", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("Hello\nAnother message\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false // Bot is NOT in group

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		assert.Equal(t, 0, handler.callCount(), "HandleText should NOT be called when bot is not in group")
	})
}

// TestRun_BotInGroup_LLMCalled tests that when bot is in group, messages are processed by LLM
// AC-014: Invite bot to group [FR-015, FR-017]
func TestRun_BotInGroup_LLMCalled(t *testing.T) {
	t.Run("should call HandleText when bot is in group", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("Hello\nHow are you?\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true // Bot IS in group

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		require.Equal(t, 2, handler.callCount(), "HandleText should be called for each message")
		calls := handler.getCalls()
		assert.Equal(t, "Hello", calls[0].text)
		assert.Equal(t, "How are you?", calls[1].text)
		assert.Equal(t, "alice", calls[0].userID)
		assert.Equal(t, "alice", calls[1].userID)
	})
}

// TestRun_OneOnOneMode_AlwaysProcessed tests that in 1-on-1 mode, messages are always processed
// FR-005, NFR-002: Existing single-user CLI behavior must not break
func TestRun_OneOnOneMode_AlwaysProcessed(t *testing.T) {
	t.Run("should always call HandleText in 1-on-1 mode regardless of bot status", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("Hello\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		ctx := context.Background()
		cfg := repl.Config{
			UserID:  "alice",
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
			Stderr:  stderr,
			// GroupID is empty (1-on-1 mode)
			// GroupSimService is nil
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		require.Equal(t, 1, handler.callCount(), "HandleText should be called in 1-on-1 mode")
		calls := handler.getCalls()
		assert.Equal(t, "Hello", calls[0].text)
		assert.Equal(t, "alice", calls[0].userID)
	})
}

// TestRun_BotStatusCheck_ErrorHandling tests handling of IsBotInGroup errors
func TestRun_BotStatusCheck_ErrorHandling(t *testing.T) {
	t.Run("should not call HandleText when IsBotInGroup returns error", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("Hello\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.err = errors.New("database connection failed")

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err, "REPL should continue even if bot status check fails")
		assert.Equal(t, 0, handler.callCount(), "HandleText should NOT be called when bot status check fails")
	})
}

// TestRun_InviteBotCommand_Success tests /invite-bot successfully adds bot and calls HandleJoin
// AC-014: Invite bot to group [FR-015, FR-017]
func TestRun_InviteBotCommand_Success(t *testing.T) {
	t.Run("should add bot to group, call HandleJoin, and display success message", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite-bot\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false // Bot is NOT in group initially

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "Bot has been invited to the group", "should display success message")

		// Verify bot was added
		botInGroup, err := groupSim.IsBotInGroup(ctx, "mygroup")
		require.NoError(t, err)
		assert.True(t, botInGroup, "bot should be added to group")

		// Verify HandleJoin was called with correct context
		require.Equal(t, 1, handler.joinCallCount(), "HandleJoin should be called once")
		joinCalls := handler.getJoinCalls()
		assert.Equal(t, line.ChatTypeGroup, joinCalls[0].chatType, "chat type should be 'group'")
		assert.Equal(t, "mygroup", joinCalls[0].sourceID, "source ID should be group ID")
	})
}

// TestRun_InviteBotCommand_AlreadyInGroup tests /invite-bot when bot is already in group
// AC-014: Error handling [FR-015]
func TestRun_InviteBotCommand_AlreadyInGroup(t *testing.T) {
	t.Run("should show error message when bot is already in group", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite-bot\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true // Bot IS already in group

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "bot is already in the group", "should display error message to stderr")

		// Verify HandleJoin was NOT called
		assert.Equal(t, 0, handler.joinCallCount(), "HandleJoin should NOT be called when bot is already in group")
	})
}

// TestRun_InviteBotCommand_NotInGroupMode tests /invite-bot in 1-on-1 mode
func TestRun_InviteBotCommand_NotInGroupMode(t *testing.T) {
	t.Run("should show error when /invite-bot is used in 1-on-1 mode", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite-bot\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		ctx := context.Background()
		cfg := repl.Config{
			UserID:         "alice",
			Handler:        handler,
			Logger:         logger,
			Stdin:          stdin,
			Stdout:         stdout,
			Stderr:         stderr,
			ProfileService: profileService,
			// GroupID is empty (1-on-1 mode)
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "/invite-bot is not available", "should display unavailable command error")

		// Verify HandleJoin was NOT called
		assert.Equal(t, 0, handler.joinCallCount(), "HandleJoin should NOT be called in 1-on-1 mode")
	})
}

// TestRun_InviteBotCommand_EnablesMessageProcessing tests that messages are processed after bot is invited
// AC-014: Invite bot to group [FR-015, FR-017]
func TestRun_InviteBotCommand_EnablesMessageProcessing(t *testing.T) {
	t.Run("should process messages after bot is invited to group", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite-bot\nHello bot!\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false // Bot is NOT in group initially

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)

		// Verify HandleJoin was called
		require.Equal(t, 1, handler.joinCallCount(), "HandleJoin should be called once")

		// Verify message was processed after bot was invited
		require.Equal(t, 1, handler.callCount(), "HandleText should be called for message sent after bot invitation")
		calls := handler.getCalls()
		assert.Equal(t, "Hello bot!", calls[0].text, "message should be processed by handler")
		assert.Equal(t, "alice", calls[0].userID, "message should be from alice")
	})
}

// TestRun_InviteCommand_TriggersHandleMemberJoined tests /invite triggers HandleMemberJoined when bot is in group
// AC-015: Invite user triggers HandleMemberJoined [FR-018, FR-019]
func TestRun_InviteCommand_TriggersHandleMemberJoined(t *testing.T) {
	t.Run("should call HandleMemberJoined with invited user ID when bot is in group", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true // Bot IS in group

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "bob has been invited to the group", "should display success message")

		// Verify bob was added to members
		members, err := groupSim.GetMembers(ctx, "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "bob", "bob should be added to group members")

		// Verify HandleMemberJoined was called
		require.Equal(t, 1, handler.memberJoinedCallCount(), "HandleMemberJoined should be called once")
		memberJoinedCalls := handler.getMemberJoinedCalls()

		// Verify context values
		assert.Equal(t, line.ChatTypeGroup, memberJoinedCalls[0].chatType, "chat type should be 'group'")
		assert.Equal(t, "mygroup", memberJoinedCalls[0].sourceID, "source ID should be group ID")

		// Verify joined user IDs
		require.Len(t, memberJoinedCalls[0].joinedUserIDs, 1, "should have one joined user")
		assert.Equal(t, "bob", memberJoinedCalls[0].joinedUserIDs[0], "joined user should be 'bob'")
	})
}

// TestRun_InviteCommand_BotNotInGroup_NoHandleMemberJoined tests /invite does not trigger HandleMemberJoined when bot is not in group
// AC-016: Invite user without bot does not trigger event [FR-018]
func TestRun_InviteCommand_BotNotInGroup_NoHandleMemberJoined(t *testing.T) {
	t.Run("should NOT call HandleMemberJoined when bot is not in group", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = false // Bot is NOT in group

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "bob has been invited to the group", "should display success message")

		// Verify bob was added to members
		members, err := groupSim.GetMembers(ctx, "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "bob", "bob should be added to group members")

		// Verify HandleMemberJoined was NOT called
		assert.Equal(t, 0, handler.memberJoinedCallCount(), "HandleMemberJoined should NOT be called when bot is not in group")
	})
}

// TestRun_InviteCommand_UserWithoutProfile_TriggersHandleMemberJoined tests /invite for user without profile still triggers HandleMemberJoined
// FR-019: HandleMemberJoined is called with the invited user's ID (always included, regardless of profile existence)
func TestRun_InviteCommand_UserWithoutProfile_TriggersHandleMemberJoined(t *testing.T) {
	t.Run("should call HandleMemberJoined with user ID even when user has no profile", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite newuser\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
				// "newuser" has no profile
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true // Bot IS in group

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err)
		output := stdout.String()
		assert.Contains(t, output, "newuser has been invited to the group", "should display success message")

		// Verify newuser was added to members
		members, err := groupSim.GetMembers(ctx, "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "newuser", "newuser should be added to group members")

		// Verify HandleMemberJoined was called with newuser's ID
		require.Equal(t, 1, handler.memberJoinedCallCount(), "HandleMemberJoined should be called once")
		memberJoinedCalls := handler.getMemberJoinedCalls()

		// Verify context values
		assert.Equal(t, line.ChatTypeGroup, memberJoinedCalls[0].chatType, "chat type should be 'group'")
		assert.Equal(t, "mygroup", memberJoinedCalls[0].sourceID, "source ID should be group ID")

		// Verify joined user IDs (user ID is included regardless of profile existence)
		require.Len(t, memberJoinedCalls[0].joinedUserIDs, 1, "should have one joined user")
		assert.Equal(t, "newuser", memberJoinedCalls[0].joinedUserIDs[0], "joined user should be 'newuser' regardless of profile")
	})
}

// TestRun_InviteCommand_HandleMemberJoinedError tests /invite continues even if HandleMemberJoined returns error
func TestRun_InviteCommand_HandleMemberJoinedError(t *testing.T) {
	t.Run("should continue and show success message even if HandleMemberJoined returns error", func(t *testing.T) {
		// Given
		stdin := strings.NewReader("/invite bob\n/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		handler := &mockHandler{
			returnErr: errors.New("HandleMemberJoined processing error"),
		}
		logger := slog.New(slog.NewTextHandler(stderr, nil))

		profileService := &mockProfileService{
			profiles: map[string]*profile.UserProfile{
				"alice": {DisplayName: "Alice"},
			},
		}

		groupSim := newMockGroupSimService()
		groupSim.members["mygroup"] = []string{"alice"}
		groupSim.botInGroup["mygroup"] = true // Bot IS in group

		ctx := context.Background()
		cfg := repl.Config{
			UserID:          "alice",
			Handler:         handler,
			Logger:          logger,
			Stdin:           stdin,
			Stdout:          stdout,
			Stderr:          stderr,
			GroupID:         ptr("mygroup"),
			ProfileService:  profileService,
			GroupSimService: groupSim,
		}

		// When
		err := repl.Run(ctx, cfg)

		// Then
		require.NoError(t, err, "REPL should not exit on HandleMemberJoined error")

		// Verify success message is still shown
		output := stdout.String()
		assert.Contains(t, output, "bob has been invited to the group", "should display success message even if handler fails")

		// Verify bob was added to members
		members, err := groupSim.GetMembers(ctx, "mygroup")
		require.NoError(t, err)
		assert.Contains(t, members, "bob", "bob should be added to group members")

		// Verify HandleMemberJoined was called
		require.Equal(t, 1, handler.memberJoinedCallCount(), "HandleMemberJoined should be called")

		// Verify error was logged
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "HandleMemberJoined processing error", "error should be logged to stderr")
	})
}
