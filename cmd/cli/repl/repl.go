package repl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"yuruppu/internal/line"
)

// MessageHandler defines the interface for handling text messages.
// This allows the REPL to work with bot.Handler without direct coupling.
type MessageHandler interface {
	HandleText(ctx context.Context, text string) error
}

// UserProfile is the minimal interface for user profile with display name.
// This interface is satisfied by both real profile.UserProfile and test mocks.
type UserProfile interface {
	GetDisplayName() string
}

// ProfileGetter retrieves user profiles for display name lookup.
type ProfileGetter interface {
	GetUserProfile(ctx context.Context, userID string) (UserProfile, error)
}

// Config holds REPL configuration.
type Config struct {
	UserID  string
	Handler MessageHandler
	Logger  *slog.Logger
	Stdin   io.Reader
	Stdout  io.Writer

	// Group mode (optional)
	GroupID       string        // If set, REPL runs in group chat mode
	ProfileGetter ProfileGetter // If set, displays user's display name in prompt
}

// buildPrompt constructs the REPL prompt based on current user and profile.
// Returns "DisplayName(user-id)> " if profile exists, or "(user-id)> " otherwise.
func buildPrompt(ctx context.Context, cfg Config) string {
	displayName := ""
	if cfg.ProfileGetter != nil {
		if profile, err := cfg.ProfileGetter.GetUserProfile(ctx, cfg.UserID); err == nil {
			displayName = profile.GetDisplayName()
		}
	}

	if displayName != "" {
		return fmt.Sprintf("%s(%s)> ", displayName, cfg.UserID)
	}
	return fmt.Sprintf("(%s)> ", cfg.UserID)
}

// Run starts the REPL loop.
// Exits on /quit, Ctrl+C twice, or context cancellation.
func Run(ctx context.Context, cfg Config) error {
	// Validate config
	if ctx == nil {
		return errors.New("context must not be nil")
	}
	if cfg.UserID == "" {
		return errors.New("userID must not be empty")
	}
	if cfg.Handler == nil {
		return errors.New("handler must not be nil")
	}
	if cfg.Logger == nil {
		return errors.New("logger must not be nil")
	}
	if cfg.Stdin == nil {
		return errors.New("stdin must not be nil")
	}
	if cfg.Stdout == nil {
		return errors.New("stdout must not be nil")
	}

	// Setup signal handler for SIGINT (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	defer signal.Stop(sigChan)

	// Track Ctrl+C count
	ctrlCCount := 0

	// Create scanner for reading input
	scanner := bufio.NewScanner(cfg.Stdin)

	// Channel to signal when input is available
	inputChan := make(chan string)
	doneChan := make(chan error, 1)

	// Start scanning goroutine
	go func() {
		for scanner.Scan() {
			inputChan <- scanner.Text()
		}
		// Check for scanner error
		if err := scanner.Err(); err != nil {
			doneChan <- err
		} else {
			doneChan <- nil // EOF
		}
		close(inputChan)
	}()

	for {
		// Display prompt
		prompt := buildPrompt(ctx, cfg)
		_, _ = fmt.Fprint(cfg.Stdout, prompt)

		// Wait for input or signals
		select {
		case <-ctx.Done():
			return context.Canceled

		case <-sigChan:
			ctrlCCount++
			if ctrlCCount == 1 {
				// First Ctrl+C: show warning
				fmt.Fprintln(os.Stderr, "Press Ctrl+C again to exit")
			} else {
				// Second Ctrl+C: exit cleanly
				return nil
			}

		case err := <-doneChan:
			// Scanner finished (EOF or error)
			return err

		case text, ok := <-inputChan:
			if !ok {
				// Channel closed, wait for done signal
				return <-doneChan
			}

			// Reset Ctrl+C count on user input
			ctrlCCount = 0

			// Trim whitespace
			trimmed := strings.TrimSpace(text)

			// Skip empty lines
			if trimmed == "" {
				continue
			}

			// Handle /quit command
			if trimmed == "/quit" {
				return nil
			}

			// Prepare context with LINE context values
			var msgCtx context.Context
			if cfg.GroupID != "" {
				// Group mode: chat type is "group", source ID is group ID
				msgCtx = line.WithChatType(ctx, line.ChatTypeGroup)
				msgCtx = line.WithSourceID(msgCtx, cfg.GroupID)
			} else {
				// 1-on-1 mode: chat type is "1-on-1", source ID is user ID
				msgCtx = line.WithChatType(ctx, line.ChatTypeOneOnOne)
				msgCtx = line.WithSourceID(msgCtx, cfg.UserID)
			}
			msgCtx = line.WithUserID(msgCtx, cfg.UserID)
			msgCtx = line.WithReplyToken(msgCtx, "cli-reply-token")

			// Call handler
			if err := cfg.Handler.HandleText(msgCtx, trimmed); err != nil {
				cfg.Logger.ErrorContext(msgCtx, "handler error", "error", err)
			}
		}
	}
}
