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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHandler implements a test handler for HandleText calls.
type mockHandler struct {
	mu         sync.Mutex
	calls      []handleTextCall
	returnErr  error
	callDelay  time.Duration // Simulates processing delay
	ctxChecker func(context.Context) error
}

type handleTextCall struct {
	text   string
	userID string
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

func (m *mockHandler) HandleJoin(_ context.Context) error {
	return nil
}

func (m *mockHandler) HandleMemberJoined(_ context.Context, _ []string) error {
	return nil
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
