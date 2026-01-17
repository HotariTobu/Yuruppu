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

type Runner struct {
	UserID          string
	GroupID         string
	ProfileService  ProfileService
	GroupSimService GroupSimService
	Handler         MessageHandler
	Logger          *slog.Logger
	Stdin           io.Reader
	Stdout          io.Writer
	Stderr          io.Writer
	currentUserID   string
}

func NewRunner(
	userID string,
	groupID string,
	profileService ProfileService,
	groupSimService GroupSimService,
	handler MessageHandler,
	logger *slog.Logger,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) (*Runner, error) {
	if userID == "" {
		return nil, errors.New("userID must not be empty")
	}
	if handler == nil {
		return nil, errors.New("handler must not be nil")
	}
	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}
	if stdin == nil {
		return nil, errors.New("stdin must not be nil")
	}
	if stdout == nil {
		return nil, errors.New("stdout must not be nil")
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	return &Runner{
		UserID:          userID,
		GroupID:         groupID,
		ProfileService:  profileService,
		GroupSimService: groupSimService,
		Handler:         handler,
		Logger:          logger,
		Stdin:           stdin,
		Stdout:          stdout,
		Stderr:          stderr,
		currentUserID:   userID,
	}, nil
}

func (r *Runner) formatUser(ctx context.Context, userID string) string {
	if r.ProfileService != nil {
		if p, err := r.ProfileService.GetUserProfile(ctx, userID); err == nil {
			return fmt.Sprintf("%s(%s)", p.DisplayName, userID)
		}
	}
	return fmt.Sprintf("(%s)", userID)
}

func (r *Runner) buildMessageContext(ctx context.Context) context.Context {
	var msgCtx context.Context
	if r.GroupID != "" {
		msgCtx = line.WithChatType(ctx, line.ChatTypeGroup)
		msgCtx = line.WithSourceID(msgCtx, r.GroupID)
	} else {
		msgCtx = line.WithChatType(ctx, line.ChatTypeOneOnOne)
		msgCtx = line.WithSourceID(msgCtx, r.currentUserID)
	}
	msgCtx = line.WithUserID(msgCtx, r.currentUserID)
	msgCtx = line.WithReplyToken(msgCtx, "cli-reply-token")
	return msgCtx
}

func (r *Runner) buildPrompt(ctx context.Context) string {
	return r.formatUser(ctx, r.currentUserID) + "> "
}

func (r *Runner) handleSwitch(ctx context.Context, targetUserID string) {
	if r.GroupID == "" || r.GroupSimService == nil {
		_, _ = fmt.Fprintln(r.Stderr, "/switch is not available")
		return
	}

	isMember, err := r.GroupSimService.IsMember(ctx, r.GroupID, targetUserID)
	if err != nil {
		r.Logger.ErrorContext(ctx, "failed to check membership", "error", err)
		return
	}

	if !isMember {
		_, _ = fmt.Fprintf(r.Stderr, "'%s' is not a member of this group\n", targetUserID)
		return
	}

	r.currentUserID = targetUserID
}

func (r *Runner) handleUsers(ctx context.Context) {
	if r.GroupID == "" || r.GroupSimService == nil {
		_, _ = fmt.Fprintln(r.Stderr, "/users is not available")
		return
	}

	members, err := r.GroupSimService.GetMembers(ctx, r.GroupID)
	if err != nil {
		r.Logger.ErrorContext(ctx, "failed to get members", "error", err)
		return
	}

	memberStrings := make([]string, 0, len(members))
	for _, memberID := range members {
		memberStrings = append(memberStrings, r.formatUser(ctx, memberID))
	}

	_, _ = fmt.Fprintln(r.Stdout, strings.Join(memberStrings, ", "))
}

func (r *Runner) handleInvite(ctx context.Context, invitedUserID string) {
	if r.GroupID == "" || r.GroupSimService == nil {
		_, _ = fmt.Fprintln(r.Stderr, "/invite is not available")
		return
	}

	invitedUserID = strings.TrimSpace(invitedUserID)
	if invitedUserID == "" {
		_, _ = fmt.Fprintln(r.Stderr, "usage: /invite <user-id>")
		return
	}

	err := r.GroupSimService.AddMember(ctx, r.GroupID, invitedUserID)
	if err != nil {
		if strings.Contains(err.Error(), "already a member") {
			_, _ = fmt.Fprintf(r.Stderr, "%s is already a member of this group\n", invitedUserID)
			return
		}
		r.Logger.ErrorContext(ctx, "failed to add member", "error", err)
		return
	}

	botInGroup, err := r.GroupSimService.IsBotInGroup(ctx, r.GroupID)
	if err != nil {
		r.Logger.ErrorContext(ctx, "failed to check bot presence", "error", err)
	} else if botInGroup {
		memberJoinedCtx := line.WithChatType(ctx, line.ChatTypeGroup)
		memberJoinedCtx = line.WithSourceID(memberJoinedCtx, r.GroupID)
		memberJoinedCtx = line.WithUserID(memberJoinedCtx, r.currentUserID)
		memberJoinedCtx = line.WithReplyToken(memberJoinedCtx, "cli-reply-token")

		if err := r.Handler.HandleMemberJoined(memberJoinedCtx, []string{invitedUserID}); err != nil {
			r.Logger.ErrorContext(memberJoinedCtx, "HandleMemberJoined error", "error", err)
		}
	}

	_, _ = fmt.Fprintf(r.Stdout, "%s has been invited to the group\n", invitedUserID)
}

func (r *Runner) handleInviteBot(ctx context.Context) {
	if r.GroupID == "" || r.GroupSimService == nil {
		_, _ = fmt.Fprintln(r.Stderr, "/invite-bot is not available")
		return
	}

	err := r.GroupSimService.AddBot(ctx, r.GroupID)
	if err != nil {
		if strings.Contains(err.Error(), "already in the group") {
			_, _ = fmt.Fprintln(r.Stderr, "bot is already in the group")
			return
		}
		r.Logger.ErrorContext(ctx, "failed to add bot to group", "error", err)
		return
	}

	joinCtx := line.WithChatType(ctx, line.ChatTypeGroup)
	joinCtx = line.WithSourceID(joinCtx, r.GroupID)
	joinCtx = line.WithUserID(joinCtx, r.currentUserID)
	joinCtx = line.WithReplyToken(joinCtx, "cli-reply-token")

	if err := r.Handler.HandleJoin(joinCtx); err != nil {
		r.Logger.ErrorContext(joinCtx, "HandleJoin error", "error", err)
	}

	_, _ = fmt.Fprintln(r.Stdout, "Bot has been invited to the group")
}

func (r *Runner) handleText(ctx context.Context, text string) {
	msgCtx := r.buildMessageContext(ctx)

	if r.GroupID != "" && r.GroupSimService != nil {
		botInGroup, err := r.GroupSimService.IsBotInGroup(msgCtx, r.GroupID)
		if err != nil {
			r.Logger.ErrorContext(msgCtx, "failed to check bot presence", "error", err)
			return
		}
		if !botInGroup {
			return
		}
	}

	if err := r.Handler.HandleText(msgCtx, text); err != nil {
		r.Logger.ErrorContext(msgCtx, "handler error", "error", err)
	}
}

// Run starts the REPL loop.
// Exits on /quit, Ctrl+C twice, or context cancellation.
func (r *Runner) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context must not be nil")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	defer signal.Stop(sigChan)

	ctrlCCount := 0

	scanner := bufio.NewScanner(r.Stdin)

	inputChan := make(chan string)
	doneChan := make(chan error, 1)

	go func() {
		for scanner.Scan() {
			inputChan <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			doneChan <- err
		} else {
			doneChan <- nil
		}
		close(inputChan)
	}()

	for {
		_, _ = fmt.Fprint(r.Stdout, r.buildPrompt(ctx))

		select {
		case <-ctx.Done():
			return context.Canceled

		case <-sigChan:
			ctrlCCount++
			if ctrlCCount == 1 {
				_, _ = fmt.Fprintln(r.Stderr, "Press Ctrl+C again to exit")
			} else {
				return nil
			}

		case err := <-doneChan:
			return err

		case text, ok := <-inputChan:
			if !ok {
				return <-doneChan
			}

			ctrlCCount = 0

			trimmed := strings.TrimSpace(text)

			if trimmed == "" {
				continue
			}

			if trimmed == "/quit" {
				return nil
			}

			if targetUserID, ok := strings.CutPrefix(trimmed, "/switch "); ok {
				targetUserID = strings.TrimSpace(targetUserID)
				if targetUserID == "" {
					_, _ = fmt.Fprintln(r.Stderr, "usage: /switch <user-id>")
					continue
				}
				r.handleSwitch(ctx, targetUserID)
				continue
			}

			if trimmed == "/users" {
				r.handleUsers(ctx)
				continue
			}

			if targetUserID, ok := strings.CutPrefix(trimmed, "/invite "); ok {
				r.handleInvite(ctx, targetUserID)
				continue
			}
			if trimmed == "/invite" {
				_, _ = fmt.Fprintln(r.Stderr, "usage: /invite <user-id>")
				continue
			}

			if trimmed == "/invite-bot" {
				r.handleInviteBot(ctx)
				continue
			}

			r.handleText(ctx, trimmed)
		}
	}
}
