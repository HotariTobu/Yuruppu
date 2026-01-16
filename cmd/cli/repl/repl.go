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

	"yuruppu/internal/group"
	"yuruppu/internal/line"
	"yuruppu/internal/profile"
)

// MessageHandler defines the interface for handling text messages.
// This allows the REPL to work with bot.Handler without direct coupling.
type MessageHandler interface {
	HandleText(ctx context.Context, text string) error
	HandleJoin(ctx context.Context) error
	HandleMemberJoined(ctx context.Context, memberUserIDs []string) error
}

// Config holds REPL configuration.
type Config struct {
	UserID  string
	Handler MessageHandler
	Logger  *slog.Logger
	Stdin   io.Reader
	Stdout  io.Writer

	// Group mode (optional, nil for 1-on-1 mode)
	GroupID        string
	GroupService   *group.Service
	ProfileService *profile.Service
}

// repl holds the REPL state.
type repl struct {
	cfg          Config
	ctx          context.Context
	activeUserID string
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

	r := &repl{
		cfg:          cfg,
		ctx:          ctx,
		activeUserID: cfg.UserID,
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
		_, _ = fmt.Fprint(cfg.Stdout, r.getPrompt())

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

			// Handle commands
			if strings.HasPrefix(trimmed, "/") {
				quit, err := r.handleCommand(trimmed)
				if err != nil {
					fmt.Fprintln(os.Stderr, err.Error())
				}
				if quit {
					return nil
				}
				continue
			}

			// In group mode, skip message if bot is not in group
			if r.isGroupMode() {
				botInGroup, err := cfg.GroupService.IsBotInGroup(ctx, cfg.GroupID)
				if err != nil {
					cfg.Logger.ErrorContext(ctx, "failed to check bot membership", "error", err)
					continue
				}
				if !botInGroup {
					// Silently skip - bot is not in group
					continue
				}
			}

			// Prepare context with LINE context values
			msgCtx := r.buildMessageContext()

			// Call handler
			if err := cfg.Handler.HandleText(msgCtx, trimmed); err != nil {
				cfg.Logger.ErrorContext(msgCtx, "handler error", "error", err)
			}
		}
	}
}

func (r *repl) isGroupMode() bool {
	return r.cfg.GroupID != ""
}

func (r *repl) getPrompt() string {
	if !r.isGroupMode() {
		return "> "
	}

	// Group mode: try to get display name
	if r.cfg.ProfileService != nil {
		if p, err := r.cfg.ProfileService.GetUserProfile(r.ctx, r.activeUserID); err == nil {
			return fmt.Sprintf("%s(%s)> ", p.DisplayName, r.activeUserID)
		}
	}
	return fmt.Sprintf("(%s)> ", r.activeUserID)
}

func (r *repl) buildMessageContext() context.Context {
	var msgCtx context.Context
	if r.isGroupMode() {
		msgCtx = line.WithChatType(r.ctx, line.ChatTypeGroup)
		msgCtx = line.WithSourceID(msgCtx, r.cfg.GroupID)
		msgCtx = line.WithUserID(msgCtx, r.activeUserID)
	} else {
		msgCtx = line.WithChatType(r.ctx, line.ChatTypeOneOnOne)
		msgCtx = line.WithSourceID(msgCtx, r.cfg.UserID)
		msgCtx = line.WithUserID(msgCtx, r.cfg.UserID)
	}
	msgCtx = line.WithReplyToken(msgCtx, "cli-reply-token")
	return msgCtx
}

func (r *repl) handleCommand(text string) (quit bool, err error) {
	parts := strings.SplitN(text, " ", 2)
	cmd := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "/quit":
		return true, nil
	case "/switch":
		return false, r.handleSwitch(arg)
	case "/users":
		return false, r.handleUsers()
	case "/invite":
		return false, r.handleInvite(arg)
	case "/invite-bot":
		return false, r.handleInviteBot()
	default:
		return false, fmt.Errorf("unknown command: %s", cmd)
	}
}

func (r *repl) handleSwitch(userID string) error {
	if !r.isGroupMode() {
		return errors.New("/switch is only available in group mode")
	}
	if userID == "" {
		return errors.New("usage: /switch <user-id>")
	}

	isMember, err := r.cfg.GroupService.IsMember(r.ctx, r.cfg.GroupID, userID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return fmt.Errorf("'%s' is not a member of this group", userID)
	}

	r.activeUserID = userID
	return nil
}

func (r *repl) handleUsers() error {
	if !r.isGroupMode() {
		return errors.New("/users is only available in group mode")
	}

	members, err := r.cfg.GroupService.GetMembers(r.ctx, r.cfg.GroupID)
	if err != nil {
		return fmt.Errorf("failed to get members: %w", err)
	}

	var names []string
	for _, uid := range members {
		if r.cfg.ProfileService != nil {
			if p, err := r.cfg.ProfileService.GetUserProfile(r.ctx, uid); err == nil {
				names = append(names, fmt.Sprintf("%s(%s)", p.DisplayName, uid))
				continue
			}
		}
		names = append(names, fmt.Sprintf("(%s)", uid))
	}
	fmt.Fprintln(r.cfg.Stdout, strings.Join(names, ", "))
	return nil
}

func (r *repl) handleInvite(userID string) error {
	if !r.isGroupMode() {
		return errors.New("/invite is only available in group mode")
	}
	if userID == "" {
		return errors.New("usage: /invite <user-id>")
	}

	if err := r.cfg.GroupService.AddMember(r.ctx, r.cfg.GroupID, userID); err != nil {
		return err
	}
	fmt.Fprintf(r.cfg.Stdout, "%s has been invited to the group\n", userID)

	// Trigger HandleMemberJoined if bot is in group
	botInGroup, err := r.cfg.GroupService.IsBotInGroup(r.ctx, r.cfg.GroupID)
	if err != nil {
		r.cfg.Logger.ErrorContext(r.ctx, "failed to check bot membership", "error", err)
		return nil
	}
	if botInGroup {
		msgCtx := r.buildMessageContext()
		if err := r.cfg.Handler.HandleMemberJoined(msgCtx, []string{userID}); err != nil {
			r.cfg.Logger.ErrorContext(r.ctx, "HandleMemberJoined error", "error", err)
		}
	}
	return nil
}

func (r *repl) handleInviteBot() error {
	if !r.isGroupMode() {
		return errors.New("/invite-bot is only available in group mode")
	}

	botInGroup, err := r.cfg.GroupService.IsBotInGroup(r.ctx, r.cfg.GroupID)
	if err != nil {
		return fmt.Errorf("failed to check bot membership: %w", err)
	}
	if botInGroup {
		return errors.New("bot is already in the group")
	}

	if err := r.cfg.GroupService.AddBot(r.ctx, r.cfg.GroupID); err != nil {
		return fmt.Errorf("failed to add bot: %w", err)
	}
	fmt.Fprintln(r.cfg.Stdout, "Bot has been invited to the group")

	// Trigger HandleJoin
	msgCtx := r.buildMessageContext()
	if err := r.cfg.Handler.HandleJoin(msgCtx); err != nil {
		r.cfg.Logger.ErrorContext(r.ctx, "HandleJoin error", "error", err)
	}
	return nil
}
