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

// GroupSimService provides group simulation operations.
type GroupSimService interface {
	GetMembers(ctx context.Context, groupID string) ([]string, error)
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
	AddMember(ctx context.Context, groupID, userID string) error
}

// Config holds REPL configuration.
type Config struct {
	UserID  string
	Handler MessageHandler
	Logger  *slog.Logger
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer // If nil, defaults to os.Stderr

	// Group mode (optional)
	GroupID         string          // If set, REPL runs in group chat mode
	ProfileGetter   ProfileGetter   // If set, displays user's display name in prompt
	GroupSimService GroupSimService // If set, enables group commands (/switch, /users)
}

// getStderr returns cfg.Stderr if set, otherwise os.Stderr.
func getStderr(cfg Config) io.Writer {
	if cfg.Stderr != nil {
		return cfg.Stderr
	}
	return os.Stderr
}

// buildPrompt constructs the REPL prompt based on current user and profile.
// Returns "DisplayName(user-id)> " if profile exists, or "(user-id)> " otherwise.
func buildPrompt(ctx context.Context, cfg Config, currentUserID string) string {
	displayName := ""
	if cfg.ProfileGetter != nil {
		if profile, err := cfg.ProfileGetter.GetUserProfile(ctx, currentUserID); err == nil {
			displayName = profile.GetDisplayName()
		}
	}

	if displayName != "" {
		return fmt.Sprintf("%s(%s)> ", displayName, currentUserID)
	}
	return fmt.Sprintf("(%s)> ", currentUserID)
}

// handleSwitchCommand handles the /switch <user-id> command.
// Returns the new user ID on success, or the current user ID on error.
func handleSwitchCommand(ctx context.Context, cfg Config, currentUserID, targetUserID string) string {
	stderr := getStderr(cfg)

	// Check if in group mode
	if cfg.GroupID == "" || cfg.GroupSimService == nil {
		_, _ = fmt.Fprintln(stderr, "/switch is not available")
		return currentUserID
	}

	// Check if target user is a member
	isMember, err := cfg.GroupSimService.IsMember(ctx, cfg.GroupID, targetUserID)
	if err != nil {
		cfg.Logger.ErrorContext(ctx, "failed to check membership", "error", err)
		return currentUserID
	}

	if !isMember {
		_, _ = fmt.Fprintf(stderr, "'%s' is not a member of this group\n", targetUserID)
		return currentUserID
	}

	return targetUserID
}

// handleUsersCommand handles the /users command.
// Lists all group members in the format: DisplayName(user-id), ...
func handleUsersCommand(ctx context.Context, cfg Config) {
	// Check if in group mode
	if cfg.GroupID == "" || cfg.GroupSimService == nil {
		_, _ = fmt.Fprintln(getStderr(cfg), "/users is not available")
		return
	}

	// Get all members
	members, err := cfg.GroupSimService.GetMembers(ctx, cfg.GroupID)
	if err != nil {
		cfg.Logger.ErrorContext(ctx, "failed to get members", "error", err)
		return
	}

	// Build output string
	var memberStrings []string
	for _, memberID := range members {
		displayName := ""
		if cfg.ProfileGetter != nil {
			if profile, err := cfg.ProfileGetter.GetUserProfile(ctx, memberID); err == nil {
				displayName = profile.GetDisplayName()
			}
		}

		if displayName != "" {
			memberStrings = append(memberStrings, fmt.Sprintf("%s(%s)", displayName, memberID))
		} else {
			memberStrings = append(memberStrings, fmt.Sprintf("(%s)", memberID))
		}
	}

	// Print to stdout
	_, _ = fmt.Fprintln(cfg.Stdout, strings.Join(memberStrings, ", "))
}

// handleInviteCommand handles the /invite <user-id> command.
// Adds a new user to the group.
func handleInviteCommand(ctx context.Context, cfg Config, userID string) {
	stderr := getStderr(cfg)

	// Check if in group mode
	if cfg.GroupID == "" || cfg.GroupSimService == nil {
		_, _ = fmt.Fprintln(stderr, "/invite is not available")
		return
	}

	// Validate user ID is not empty
	userID = strings.TrimSpace(userID)
	if userID == "" {
		_, _ = fmt.Fprintln(stderr, "usage: /invite <user-id>")
		return
	}

	// Add member to group
	err := cfg.GroupSimService.AddMember(ctx, cfg.GroupID, userID)
	if err != nil {
		// Check if error is because user is already a member
		if strings.Contains(err.Error(), "already a member") {
			_, _ = fmt.Fprintf(stderr, "%s is already a member of this group\n", userID)
			return
		}
		cfg.Logger.ErrorContext(ctx, "failed to add member", "error", err)
		return
	}

	// Success message to stdout
	_, _ = fmt.Fprintf(cfg.Stdout, "%s has been invited to the group\n", userID)
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

	// Track current user (can be changed with /switch command)
	currentUserID := cfg.UserID

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
		prompt := buildPrompt(ctx, cfg, currentUserID)
		_, _ = fmt.Fprint(cfg.Stdout, prompt)

		// Wait for input or signals
		select {
		case <-ctx.Done():
			return context.Canceled

		case <-sigChan:
			ctrlCCount++
			if ctrlCCount == 1 {
				// First Ctrl+C: show warning
				_, _ = fmt.Fprintln(getStderr(cfg), "Press Ctrl+C again to exit")
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

			// Handle /switch command
			if targetUserID, ok := strings.CutPrefix(trimmed, "/switch "); ok {
				targetUserID = strings.TrimSpace(targetUserID)
				if targetUserID == "" {
					_, _ = fmt.Fprintln(getStderr(cfg), "usage: /switch <user-id>")
					continue
				}
				currentUserID = handleSwitchCommand(ctx, cfg, currentUserID, targetUserID)
				continue
			}

			// Handle /users command
			if trimmed == "/users" {
				handleUsersCommand(ctx, cfg)
				continue
			}

			// Handle /invite command
			if targetUserID, ok := strings.CutPrefix(trimmed, "/invite "); ok {
				handleInviteCommand(ctx, cfg, targetUserID)
				continue
			}
			if trimmed == "/invite" {
				_, _ = fmt.Fprintln(getStderr(cfg), "usage: /invite <user-id>")
				continue
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
				msgCtx = line.WithSourceID(msgCtx, currentUserID)
			}
			msgCtx = line.WithUserID(msgCtx, currentUserID)
			msgCtx = line.WithReplyToken(msgCtx, "cli-reply-token")

			// Call handler
			if err := cfg.Handler.HandleText(msgCtx, trimmed); err != nil {
				cfg.Logger.ErrorContext(msgCtx, "handler error", "error", err)
			}
		}
	}
}
