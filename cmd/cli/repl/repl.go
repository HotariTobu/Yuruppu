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
	"yuruppu/internal/profile"
)

type MessageHandler interface {
	HandleText(ctx context.Context, text string) error
	HandleJoin(ctx context.Context) error
	HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error
}

type ProfileService interface {
	GetUserProfile(ctx context.Context, userID string) (*profile.UserProfile, error)
}

type GroupSimService interface {
	GetMembers(ctx context.Context, groupID string) ([]string, error)
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
	AddMember(ctx context.Context, groupID, userID string) error
	IsBotInGroup(ctx context.Context, groupID string) (bool, error)
	AddBot(ctx context.Context, groupID string) error
}

type Config struct {
	UserID          string
	GroupID         *string
	ProfileService  ProfileService
	GroupSimService GroupSimService
	Handler         MessageHandler
	Logger          *slog.Logger
	Stdin           io.Reader
	Stdout          io.Writer
	Stderr          io.Writer
}

func formatUser(ctx context.Context, cfg Config, userID string) string {
	if cfg.ProfileService != nil {
		if p, err := cfg.ProfileService.GetUserProfile(ctx, userID); err == nil {
			return fmt.Sprintf("%s(%s)", p.DisplayName, userID)
		}
	}
	return fmt.Sprintf("(%s)", userID)
}

func buildMessageContext(ctx context.Context, cfg Config, currentUserID string) context.Context {
	var msgCtx context.Context
	if cfg.GroupID != nil {
		msgCtx = line.WithChatType(ctx, line.ChatTypeGroup)
		msgCtx = line.WithSourceID(msgCtx, *cfg.GroupID)
	} else {
		msgCtx = line.WithChatType(ctx, line.ChatTypeOneOnOne)
		msgCtx = line.WithSourceID(msgCtx, currentUserID)
	}
	msgCtx = line.WithUserID(msgCtx, currentUserID)
	msgCtx = line.WithReplyToken(msgCtx, "cli-reply-token")
	return msgCtx
}

func buildPrompt(ctx context.Context, cfg Config, currentUserID string) string {
	return formatUser(ctx, cfg, currentUserID) + "> "
}

// handleSwitchCommand handles the /switch <user-id> command.
// Returns the new user ID on success, or the current user ID on error.
func handleSwitchCommand(ctx context.Context, cfg Config, currentUserID, targetUserID string) string {
	stderr := cfg.Stderr

	// Check if in group mode
	if cfg.GroupID == nil || cfg.GroupSimService == nil {
		_, _ = fmt.Fprintln(stderr, "/switch is not available")
		return currentUserID
	}

	// Check if target user is a member
	isMember, err := cfg.GroupSimService.IsMember(ctx, *cfg.GroupID, targetUserID)
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
	if cfg.GroupID == nil || cfg.GroupSimService == nil {
		_, _ = fmt.Fprintln(cfg.Stderr, "/users is not available")
		return
	}

	// Get all members
	members, err := cfg.GroupSimService.GetMembers(ctx, *cfg.GroupID)
	if err != nil {
		cfg.Logger.ErrorContext(ctx, "failed to get members", "error", err)
		return
	}

	memberStrings := make([]string, 0, len(members))
	for _, memberID := range members {
		memberStrings = append(memberStrings, formatUser(ctx, cfg, memberID))
	}

	// Print to stdout
	_, _ = fmt.Fprintln(cfg.Stdout, strings.Join(memberStrings, ", "))
}

// handleInviteCommand handles the /invite <user-id> command.
// Adds a new user to the group.
func handleInviteCommand(ctx context.Context, cfg Config, currentUserID, invitedUserID string) {
	stderr := cfg.Stderr

	// Check if in group mode
	if cfg.GroupID == nil || cfg.GroupSimService == nil {
		_, _ = fmt.Fprintln(stderr, "/invite is not available")
		return
	}

	// Validate user ID is not empty
	invitedUserID = strings.TrimSpace(invitedUserID)
	if invitedUserID == "" {
		_, _ = fmt.Fprintln(stderr, "usage: /invite <user-id>")
		return
	}

	// Add member to group
	err := cfg.GroupSimService.AddMember(ctx, *cfg.GroupID, invitedUserID)
	if err != nil {
		// Check if error is because user is already a member
		if strings.Contains(err.Error(), "already a member") {
			_, _ = fmt.Fprintf(stderr, "%s is already a member of this group\n", invitedUserID)
			return
		}
		cfg.Logger.ErrorContext(ctx, "failed to add member", "error", err)
		return
	}

	// Check if bot is in group - if yes, trigger HandleMemberJoined
	botInGroup, err := cfg.GroupSimService.IsBotInGroup(ctx, *cfg.GroupID)
	if err != nil {
		cfg.Logger.ErrorContext(ctx, "failed to check bot presence", "error", err)
		// Continue even if check fails - user was still added
	} else if botInGroup {
		// Bot is in group - trigger HandleMemberJoined event
		// Build context with group chat context
		memberJoinedCtx := line.WithChatType(ctx, line.ChatTypeGroup)
		memberJoinedCtx = line.WithSourceID(memberJoinedCtx, *cfg.GroupID)
		memberJoinedCtx = line.WithUserID(memberJoinedCtx, currentUserID)
		memberJoinedCtx = line.WithReplyToken(memberJoinedCtx, "cli-reply-token")

		// Call HandleMemberJoined with the invited user's ID
		if err := cfg.Handler.HandleMemberJoined(memberJoinedCtx, []string{invitedUserID}); err != nil {
			cfg.Logger.ErrorContext(memberJoinedCtx, "HandleMemberJoined error", "error", err)
			// Continue even if handler fails - user was still added to group
		}
	}

	// Success message to stdout
	_, _ = fmt.Fprintf(cfg.Stdout, "%s has been invited to the group\n", invitedUserID)
}

// handleInviteBotCommand handles the /invite-bot command.
// Adds the bot to the group and triggers HandleJoin event.
func handleInviteBotCommand(ctx context.Context, cfg Config, currentUserID string) {
	stderr := cfg.Stderr

	// Check if in group mode
	if cfg.GroupID == nil || cfg.GroupSimService == nil {
		_, _ = fmt.Fprintln(stderr, "/invite-bot is not available")
		return
	}

	// Add bot to group
	err := cfg.GroupSimService.AddBot(ctx, *cfg.GroupID)
	if err != nil {
		// Check if error is because bot is already in group
		if strings.Contains(err.Error(), "already in the group") {
			_, _ = fmt.Fprintln(stderr, "bot is already in the group")
			return
		}
		cfg.Logger.ErrorContext(ctx, "failed to add bot to group", "error", err)
		return
	}

	// Build context for HandleJoin with group chat context
	joinCtx := line.WithChatType(ctx, line.ChatTypeGroup)
	joinCtx = line.WithSourceID(joinCtx, *cfg.GroupID)
	joinCtx = line.WithUserID(joinCtx, currentUserID)
	joinCtx = line.WithReplyToken(joinCtx, "cli-reply-token")

	// Call HandleJoin
	if err := cfg.Handler.HandleJoin(joinCtx); err != nil {
		cfg.Logger.ErrorContext(joinCtx, "HandleJoin error", "error", err)
		// Continue even if handler fails - bot was still added to group
	}

	// Success message to stdout
	_, _ = fmt.Fprintln(cfg.Stdout, "Bot has been invited to the group")
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
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
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
				_, _ = fmt.Fprintln(cfg.Stderr, "Press Ctrl+C again to exit")
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
					_, _ = fmt.Fprintln(cfg.Stderr, "usage: /switch <user-id>")
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
				handleInviteCommand(ctx, cfg, currentUserID, targetUserID)
				continue
			}
			if trimmed == "/invite" {
				_, _ = fmt.Fprintln(cfg.Stderr, "usage: /invite <user-id>")
				continue
			}

			// Handle /invite-bot command
			if trimmed == "/invite-bot" {
				handleInviteBotCommand(ctx, cfg, currentUserID)
				continue
			}

			// Prepare context with LINE context values
			msgCtx := buildMessageContext(ctx, cfg, currentUserID)

			if cfg.GroupID != nil && cfg.GroupSimService != nil {
				botInGroup, err := cfg.GroupSimService.IsBotInGroup(msgCtx, *cfg.GroupID)
				if err != nil {
					cfg.Logger.ErrorContext(msgCtx, "failed to check bot presence", "error", err)
					continue
				}
				if !botInGroup {
					continue
				}
			}

			// Call handler
			if err := cfg.Handler.HandleText(msgCtx, trimmed); err != nil {
				cfg.Logger.ErrorContext(msgCtx, "handler error", "error", err)
			}
		}
	}
}
