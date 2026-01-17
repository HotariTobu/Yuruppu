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

// CLIReplyToken is a dummy reply token used for CLI messages.
const CLIReplyToken = "cli-reply-token"

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
	userID          string
	groupID         string
	profileService  ProfileService
	groupSimService GroupSimService
	handler         MessageHandler
	logger          *slog.Logger
	stdin           io.Reader
	stdout          io.Writer
	stderr          io.Writer
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
		userID:          userID,
		groupID:         groupID,
		profileService:  profileService,
		groupSimService: groupSimService,
		handler:         handler,
		logger:          logger,
		stdin:           stdin,
		stdout:          stdout,
		stderr:          stderr,
	}, nil
}

func (r *Runner) formatUser(ctx context.Context, userID string) string {
	if r.profileService != nil {
		if p, err := r.profileService.GetUserProfile(ctx, userID); err == nil {
			return fmt.Sprintf("%s(%s)", p.DisplayName, userID)
		}
	}
	return fmt.Sprintf("(%s)", userID)
}

func (r *Runner) buildMessageContext(ctx context.Context) context.Context {
	var msgCtx context.Context
	if r.groupID != "" {
		msgCtx = line.WithChatType(ctx, line.ChatTypeGroup)
		msgCtx = line.WithSourceID(msgCtx, r.groupID)
	} else {
		msgCtx = line.WithChatType(ctx, line.ChatTypeOneOnOne)
		msgCtx = line.WithSourceID(msgCtx, r.userID)
	}
	msgCtx = line.WithUserID(msgCtx, r.userID)
	msgCtx = line.WithReplyToken(msgCtx, CLIReplyToken)
	return msgCtx
}

func (r *Runner) buildPrompt(ctx context.Context) string {
	return r.formatUser(ctx, r.userID) + "> "
}

func (r *Runner) handleSwitch(ctx context.Context, targetUserID string) {
	if r.groupID == "" || r.groupSimService == nil {
		_, _ = fmt.Fprintln(r.stderr, "/switch is not available")
		return
	}

	isMember, err := r.groupSimService.IsMember(ctx, r.groupID, targetUserID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to check membership", "error", err)
		return
	}

	if !isMember {
		_, _ = fmt.Fprintf(r.stderr, "'%s' is not a member of this group\n", targetUserID)
		return
	}

	r.userID = targetUserID
}

func (r *Runner) handleUsers(ctx context.Context) {
	if r.groupID == "" || r.groupSimService == nil {
		_, _ = fmt.Fprintln(r.stderr, "/users is not available")
		return
	}

	members, err := r.groupSimService.GetMembers(ctx, r.groupID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to get members", "error", err)
		return
	}

	memberStrings := make([]string, 0, len(members))
	for _, memberID := range members {
		memberStrings = append(memberStrings, r.formatUser(ctx, memberID))
	}

	_, _ = fmt.Fprintln(r.stdout, strings.Join(memberStrings, ", "))
}

func (r *Runner) handleInvite(ctx context.Context, invitedUserID string) {
	if r.groupID == "" || r.groupSimService == nil {
		_, _ = fmt.Fprintln(r.stderr, "/invite is not available")
		return
	}

	invitedUserID = strings.TrimSpace(invitedUserID)
	if invitedUserID == "" {
		_, _ = fmt.Fprintln(r.stderr, "usage: /invite <user-id>")
		return
	}

	err := r.groupSimService.AddMember(ctx, r.groupID, invitedUserID)
	if err != nil {
		_, _ = fmt.Fprintln(r.stderr, err)
		return
	}

	botInGroup, err := r.groupSimService.IsBotInGroup(ctx, r.groupID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to check bot presence", "error", err)
	} else if botInGroup {
		memberJoinedCtx := r.buildMessageContext(ctx)
		if err := r.handler.HandleMemberJoined(memberJoinedCtx, []string{invitedUserID}); err != nil {
			r.logger.ErrorContext(memberJoinedCtx, "HandleMemberJoined error", "error", err)
		}
	}

	_, _ = fmt.Fprintf(r.stdout, "%s has been invited to the group\n", invitedUserID)
}

func (r *Runner) handleInviteBot(ctx context.Context) {
	if r.groupID == "" || r.groupSimService == nil {
		_, _ = fmt.Fprintln(r.stderr, "/invite-bot is not available")
		return
	}

	err := r.groupSimService.AddBot(ctx, r.groupID)
	if err != nil {
		_, _ = fmt.Fprintln(r.stderr, err)
		return
	}

	joinCtx := r.buildMessageContext(ctx)
	if err := r.handler.HandleJoin(joinCtx); err != nil {
		r.logger.ErrorContext(joinCtx, "HandleJoin error", "error", err)
	}

	_, _ = fmt.Fprintln(r.stdout, "Bot has been invited to the group")
}

func (r *Runner) handleText(ctx context.Context, text string) {
	msgCtx := r.buildMessageContext(ctx)

	if r.groupID != "" && r.groupSimService != nil {
		botInGroup, err := r.groupSimService.IsBotInGroup(msgCtx, r.groupID)
		if err != nil {
			r.logger.ErrorContext(msgCtx, "failed to check bot presence", "error", err)
			return
		}
		if !botInGroup {
			return
		}
	}

	if err := r.handler.HandleText(msgCtx, text); err != nil {
		r.logger.ErrorContext(msgCtx, "handler error", "error", err)
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

	scanner := bufio.NewScanner(r.stdin)

	inputChan := make(chan string, 1)
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
		_, _ = fmt.Fprint(r.stdout, r.buildPrompt(ctx))

		select {
		case <-ctx.Done():
			return context.Canceled

		case <-sigChan:
			ctrlCCount++
			if ctrlCCount == 1 {
				_, _ = fmt.Fprintln(r.stderr, "Press Ctrl+C again to exit")
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
					_, _ = fmt.Fprintln(r.stderr, "usage: /switch <user-id>")
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
				_, _ = fmt.Fprintln(r.stderr, "usage: /invite <user-id>")
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
