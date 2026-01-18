package repl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"yuruppu/internal/line"
	"yuruppu/internal/userprofile"
)

// CLIReplyToken is a dummy reply token used for CLI messages.
const CLIReplyToken = "dummy"

type MessageHandler interface {
	HandleText(ctx context.Context, text string) error
	HandleJoin(ctx context.Context) error
	HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error
}

type UserProfileService interface {
	GetUserProfile(ctx context.Context, userID string) (*userprofile.UserProfile, error)
}

type GroupSimService interface {
	GetMembers(ctx context.Context, groupID string) ([]string, error)
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
	AddMember(ctx context.Context, groupID, userID string) error
	IsBotInGroup(ctx context.Context, groupID string) (bool, error)
	AddBot(ctx context.Context, groupID string) error
}

type Runner struct {
	userID             string
	groupID            string
	userProfileService UserProfileService
	groupSimService    GroupSimService
	handler            MessageHandler
	logger             *slog.Logger
	scanner            *bufio.Scanner
	writer             io.Writer
}

func NewRunner(
	userID string,
	groupID string,
	userProfileService UserProfileService,
	groupSimService GroupSimService,
	handler MessageHandler,
	logger *slog.Logger,
	scanner *bufio.Scanner,
	writer io.Writer,
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
	if scanner == nil {
		return nil, errors.New("scanner must not be nil")
	}
	if writer == nil {
		return nil, errors.New("writer must not be nil")
	}

	return &Runner{
		userID:             userID,
		groupID:            groupID,
		userProfileService: userProfileService,
		groupSimService:    groupSimService,
		handler:            handler,
		logger:             logger,
		scanner:            scanner,
		writer:             writer,
	}, nil
}

func (r *Runner) formatUser(ctx context.Context, userID string) string {
	if r.userProfileService != nil {
		if p, err := r.userProfileService.GetUserProfile(ctx, userID); err == nil {
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
		r.logger.WarnContext(ctx, "/switch is not available")
		return
	}

	isMember, err := r.groupSimService.IsMember(ctx, r.groupID, targetUserID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to check membership", slog.Any("error", err))
		return
	}

	if !isMember {
		r.logger.WarnContext(ctx, "user is not a member of this group", slog.String("userID", targetUserID))
		return
	}

	r.userID = targetUserID
}

func (r *Runner) handleUsers(ctx context.Context) {
	if r.groupID == "" || r.groupSimService == nil {
		r.logger.WarnContext(ctx, "/users is not available")
		return
	}

	members, err := r.groupSimService.GetMembers(ctx, r.groupID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to get members", slog.Any("error", err))
		return
	}

	memberStrings := make([]string, 0, len(members))
	for _, memberID := range members {
		memberStrings = append(memberStrings, r.formatUser(ctx, memberID))
	}

	r.logger.InfoContext(ctx, "group members", slog.String("members", strings.Join(memberStrings, ", ")))
}

func (r *Runner) handleInvite(ctx context.Context, invitedUserID string) {
	if r.groupID == "" || r.groupSimService == nil {
		r.logger.WarnContext(ctx, "/invite is not available")
		return
	}

	invitedUserID = strings.TrimSpace(invitedUserID)
	if invitedUserID == "" {
		r.logger.WarnContext(ctx, "usage: /invite <user-id>")
		return
	}

	err := r.groupSimService.AddMember(ctx, r.groupID, invitedUserID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to add member", slog.Any("error", err))
		return
	}

	botInGroup, err := r.groupSimService.IsBotInGroup(ctx, r.groupID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to check bot presence", slog.Any("error", err))
	} else if botInGroup {
		memberJoinedCtx := r.buildMessageContext(ctx)
		if err := r.handler.HandleMemberJoined(memberJoinedCtx, []string{invitedUserID}); err != nil {
			r.logger.ErrorContext(memberJoinedCtx, "HandleMemberJoined error", slog.Any("error", err))
		}
	}

	r.logger.InfoContext(ctx, "user invited to group", slog.String("userID", invitedUserID))
}

func (r *Runner) handleInviteBot(ctx context.Context) {
	if r.groupID == "" || r.groupSimService == nil {
		r.logger.WarnContext(ctx, "/invite-bot is not available")
		return
	}

	err := r.groupSimService.AddBot(ctx, r.groupID)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to add bot", slog.Any("error", err))
		return
	}

	joinCtx := r.buildMessageContext(ctx)
	if err := r.handler.HandleJoin(joinCtx); err != nil {
		r.logger.ErrorContext(joinCtx, "HandleJoin error", slog.Any("error", err))
	}

	r.logger.InfoContext(ctx, "bot invited to group")
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
// Exits on /quit, EOF, or Ctrl+C.
func (r *Runner) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context must not be nil")
	}

	for {
		if err := ctx.Err(); err != nil {
			return context.Canceled
		}

		_, _ = fmt.Fprint(r.writer, r.buildPrompt(ctx))

		if !r.scanner.Scan() {
			if err := r.scanner.Err(); err != nil {
				return err
			}
			return nil // EOF
		}

		trimmed := strings.TrimSpace(r.scanner.Text())

		if trimmed == "" {
			continue
		}

		if trimmed == "/quit" {
			return nil
		}

		if targetUserID, ok := strings.CutPrefix(trimmed, "/switch "); ok {
			targetUserID = strings.TrimSpace(targetUserID)
			if targetUserID == "" {
				r.logger.WarnContext(ctx, "usage: /switch <user-id>")
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
			r.logger.WarnContext(ctx, "usage: /invite <user-id>")
			continue
		}

		if trimmed == "/invite-bot" {
			r.handleInviteBot(ctx)
			continue
		}

		r.handleText(ctx, trimmed)
	}
}
