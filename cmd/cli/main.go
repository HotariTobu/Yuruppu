package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"

	"yuruppu/cmd/cli/mock"
	cliprofile "yuruppu/cmd/cli/profile"
	"yuruppu/cmd/cli/repl"
	"yuruppu/cmd/cli/setup"
	"yuruppu/internal/agent"
	"yuruppu/internal/bot"
	eventdomain "yuruppu/internal/event"
	"yuruppu/internal/group"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
	"yuruppu/internal/media"
	"yuruppu/internal/profile"
	"yuruppu/internal/toolset/event"
	"yuruppu/internal/toolset/reply"
	"yuruppu/internal/toolset/skip"
	"yuruppu/internal/toolset/weather"
	"yuruppu/internal/yuruppu"
)

// userIDPattern validates user ID format: [0-9a-z_]+
var userIDPattern = regexp.MustCompile(`^[0-9a-z_]+$`)

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run implements the CLI logic with testable I/O.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Validate I/O
	if stdin == nil {
		return errors.New("stdin cannot be nil")
	}
	if stdout == nil {
		return errors.New("stdout cannot be nil")
	}
	if stderr == nil {
		return errors.New("stderr cannot be nil")
	}

	// Parse flags
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)

	userID := fs.String("user-id", "default", "User ID for the conversation")
	dataDir := fs.String("data-dir", ".yuruppu/", "Data directory for storage")
	message := fs.String("message", "", "Single message to send (single-turn mode)")
	groupID := fs.String("group-id", "", "Group ID for group chat mode (optional)")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	// Validate user ID
	if !userIDPattern.MatchString(*userID) {
		return fmt.Errorf("invalid user ID: must match pattern [0-9a-z_]+")
	}

	// Configure logger to write to stderr
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Check required environment variables
	gcpProjectID := os.Getenv("GCP_PROJECT_ID")
	if gcpProjectID == "" {
		return errors.New("GCP_PROJECT_ID environment variable is required")
	}

	gcpRegion := os.Getenv("GCP_REGION")
	if gcpRegion == "" {
		return errors.New("GCP_REGION environment variable is required")
	}

	llmModel := os.Getenv("LLM_MODEL")
	if llmModel == "" {
		return errors.New("LLM_MODEL environment variable is required")
	}

	// Ensure data directory exists
	if err := setup.EnsureDataDir(*dataDir, stdin, stderr); err != nil {
		return err
	}

	// Create FileStorage instances with key prefixes
	profileStorage := mock.NewFileStorage(*dataDir, "profile/")
	historyStorage := mock.NewFileStorage(*dataDir, "history/")
	mediaStorage := mock.NewFileStorage(*dataDir, "media/")
	groupStorage := mock.NewFileStorage(*dataDir, "group/")

	// Create profile service
	profileService, err := profile.NewService(profileStorage, logger)
	if err != nil {
		return fmt.Errorf("failed to create profile service: %w", err)
	}

	// Create group service
	groupService, err := group.NewService(groupStorage)
	if err != nil {
		return fmt.Errorf("failed to create group service: %w", err)
	}

	ctx := context.Background()

	// Create mock LINE client with profile prompter
	lineClient := mock.NewLineClient(stdout)
	lineClient.RegisterProfileFetcher(cliprofile.NewPrompter(stdin, stderr))

	// Create history service
	historyService, err := history.NewService(historyStorage)
	if err != nil {
		return fmt.Errorf("failed to create history service: %w", err)
	}

	// Create media service
	mediaService, err := media.NewService(mediaStorage, logger)
	if err != nil {
		return fmt.Errorf("failed to create media service: %w", err)
	}

	// Create tools
	replyTool, err := reply.NewTool(lineClient, historyService, logger)
	if err != nil {
		return fmt.Errorf("failed to create reply tool: %w", err)
	}

	weatherTool, err := weather.NewTool(http.DefaultClient, logger)
	if err != nil {
		return fmt.Errorf("failed to create weather tool: %w", err)
	}

	skipTool, err := skip.NewTool(logger)
	if err != nil {
		return fmt.Errorf("failed to create skip tool: %w", err)
	}

	// Create event service and tools
	eventStorage := mock.NewFileStorage(*dataDir, "event/")
	eventService, err := eventdomain.NewService(eventStorage)
	if err != nil {
		return fmt.Errorf("failed to create event service: %w", err)
	}
	eventTools, err := event.NewTools(eventService, profileService, 366, 5, logger)
	if err != nil {
		return fmt.Errorf("failed to create event tools: %w", err)
	}

	// Collect all tools
	toolset := append([]agent.Tool{replyTool, weatherTool, skipTool}, eventTools...)

	// Create GeminiAgent with tools
	systemPrompt, err := yuruppu.GetSystemPrompt()
	if err != nil {
		return fmt.Errorf("failed to get system prompt: %w", err)
	}
	geminiAgent, err := agent.NewGeminiAgent(ctx, agent.GeminiConfig{
		ProjectID:        gcpProjectID,
		Region:           gcpRegion,
		Model:            llmModel,
		SystemPrompt:     systemPrompt,
		Tools:            toolset,
		FunctionCallOnly: true,
		CacheDisplayName: "yuruppu-cli",
		CacheTTL:         1 * time.Hour,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to create Gemini agent with tools: %w", err)
	}
	defer func() { _ = geminiAgent.Close(ctx) }()

	// Create bot handler
	handlerConfig := bot.HandlerConfig{
		TypingIndicatorDelay:   3 * time.Second,
		TypingIndicatorTimeout: 30 * time.Second,
	}
	handler, err := bot.NewHandler(lineClient, profileService, historyService, mediaService, geminiAgent, handlerConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	// Check if profile exists, if not call HandleFollow to create it
	_, err = profileService.GetUserProfile(ctx, *userID)
	if err != nil {
		logger.Info("profile not found, prompting for new profile", slog.String("userID", *userID))

		followCtx := line.WithUserID(ctx, *userID)
		if err := handler.HandleFollow(followCtx); err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}

		logger.Info("profile created successfully", slog.String("userID", *userID))
	}

	// Handle group mode startup
	if *groupID != "" {
		existingGroup, _, err := groupService.GetGroup(ctx, *groupID)
		if err != nil {
			return fmt.Errorf("failed to check group: %w", err)
		}

		if existingGroup == nil {
			// FR-002: Create new group with user as first member
			if err := groupService.CreateGroup(ctx, *groupID, *userID); err != nil {
				return fmt.Errorf("failed to create group: %w", err)
			}
			logger.Info("group created", slog.String("groupID", *groupID), slog.String("firstMember", *userID))
		} else {
			// FR-003/FR-004: Validate membership
			isMember, err := groupService.IsMember(ctx, *groupID, *userID)
			if err != nil {
				return fmt.Errorf("failed to check membership: %w", err)
			}
			if !isMember {
				return fmt.Errorf("user '%s' is not a member of group '%s'", *userID, *groupID)
			}
		}
	}

	// Single-turn mode or REPL mode
	if *message != "" {
		// Single-turn mode
		var msgCtx context.Context
		if *groupID != "" {
			// Group chat mode
			msgCtx = line.WithChatType(ctx, line.ChatTypeGroup)
			msgCtx = line.WithSourceID(msgCtx, *groupID)
			msgCtx = line.WithUserID(msgCtx, *userID)
		} else {
			// 1-on-1 chat mode
			msgCtx = line.WithChatType(ctx, line.ChatTypeOneOnOne)
			msgCtx = line.WithSourceID(msgCtx, *userID)
			msgCtx = line.WithUserID(msgCtx, *userID)
		}
		msgCtx = line.WithReplyToken(msgCtx, "cli-reply-token")

		// In group mode, skip if bot is not in group
		if *groupID != "" {
			botInGroup, err := groupService.IsBotInGroup(ctx, *groupID)
			if err != nil {
				return fmt.Errorf("failed to check bot membership: %w", err)
			}
			if !botInGroup {
				logger.Info("bot is not in group, message not sent")
				return nil
			}
		}

		if err := handler.HandleText(msgCtx, *message); err != nil {
			return fmt.Errorf("failed to handle message: %w", err)
		}
	} else {
		// REPL mode
		if err := repl.Run(ctx, repl.Config{
			UserID:         *userID,
			Handler:        handler,
			Logger:         logger,
			Stdin:          stdin,
			Stdout:         stdout,
			GroupID:        *groupID,
			GroupService:   groupService,
			ProfileService: profileService,
		}); err != nil {
			return fmt.Errorf("REPL error: %w", err)
		}
	}

	return nil
}
