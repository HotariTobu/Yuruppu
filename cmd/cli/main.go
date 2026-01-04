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
	cliProfile "yuruppu/cmd/cli/profile"
	"yuruppu/cmd/cli/repl"
	"yuruppu/cmd/cli/setup"
	"yuruppu/internal/agent"
	"yuruppu/internal/bot"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
	"yuruppu/internal/media"
	"yuruppu/internal/profile"
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
	// Validate I/O (check for nil interfaces including typed nils)
	if stdin == nil || (fmt.Sprintf("%v", stdin) == "<nil>") {
		return errors.New("stdin cannot be nil")
	}
	if stdout == nil || (fmt.Sprintf("%v", stdout) == "<nil>") {
		return errors.New("stdout cannot be nil")
	}
	if stderr == nil || (fmt.Sprintf("%v", stderr) == "<nil>") {
		return errors.New("stderr cannot be nil")
	}

	// Parse flags
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)

	userID := fs.String("user-id", "default", "User ID for the conversation")
	dataDir := fs.String("data-dir", ".yuruppu/", "Data directory for storage")
	message := fs.String("message", "", "Single message to send (single-turn mode)")

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

	// Create FileStorage instances for each bucket
	profileStorage := mock.NewFileStorage(*dataDir + "profiles/")
	historyStorage := mock.NewFileStorage(*dataDir + "history/")
	mediaStorage := mock.NewFileStorage(*dataDir + "media/")

	// Create profile service
	profileService, err := profile.NewService(profileStorage, logger)
	if err != nil {
		return fmt.Errorf("failed to create profile service: %w", err)
	}

	// Check if profile exists, if not prompt for new profile
	ctx := context.Background()
	_, err = profileService.GetUserProfile(ctx, *userID)
	if err != nil {
		// Profile not found, prompt for new profile
		logger.Info("profile not found, prompting for new profile", slog.String("userID", *userID))

		newProfile, err := cliProfile.PromptNewProfile(ctx, stdin, stderr)
		if err != nil {
			return fmt.Errorf("failed to prompt for profile: %w", err)
		}

		if err := profileService.SetUserProfile(ctx, *userID, newProfile); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}

		logger.Info("profile created successfully", slog.String("userID", *userID))
	}

	// Create GeminiAgent
	geminiAgent, err := agent.NewGeminiAgent(ctx, agent.GeminiConfig{
		ProjectID:        gcpProjectID,
		Region:           gcpRegion,
		Model:            llmModel,
		SystemPrompt:     yuruppu.SystemPrompt,
		Tools:            nil, // Tools will be set after creating them
		FunctionCallOnly: true,
		CacheDisplayName: "yuruppu-cli",
		CacheTTL:         1 * time.Hour,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to create Gemini agent: %w", err)
	}
	defer func() { _ = geminiAgent.Close(ctx) }()

	// Create mock LINE client
	lineClient := mock.NewLineClient(stdout)

	// Create history service
	historyService, err := history.NewService(historyStorage)
	if err != nil {
		return fmt.Errorf("failed to create history service: %w", err)
	}
	defer func() { _ = historyService.Close(ctx) }()

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

	// Recreate agent with tools
	_ = geminiAgent.Close(ctx)
	geminiAgent, err = agent.NewGeminiAgent(ctx, agent.GeminiConfig{
		ProjectID:        gcpProjectID,
		Region:           gcpRegion,
		Model:            llmModel,
		SystemPrompt:     yuruppu.SystemPrompt,
		Tools:            []agent.Tool{replyTool, weatherTool, skipTool},
		FunctionCallOnly: true,
		CacheDisplayName: "yuruppu-cli",
		CacheTTL:         1 * time.Hour,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to create Gemini agent with tools: %w", err)
	}
	defer func() { _ = geminiAgent.Close(ctx) }()

	// Create bot handler
	handler, err := bot.NewHandler(lineClient, profileService, historyService, mediaService, geminiAgent, logger)
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	// Single-turn mode or REPL mode
	if *message != "" {
		// Single-turn mode
		msgCtx := line.WithUserID(ctx, *userID)
		msgCtx = line.WithSourceID(msgCtx, *userID) // sourceID = userID in CLI mode
		msgCtx = line.WithReplyToken(msgCtx, "cli-reply-token")

		if err := handler.HandleText(msgCtx, *message); err != nil {
			return fmt.Errorf("failed to handle message: %w", err)
		}
	} else {
		// REPL mode
		if err := repl.Run(ctx, repl.Config{
			UserID:  *userID,
			Handler: handler,
			Logger:  logger,
			Stdin:   stdin,
			Stdout:  stdout,
		}); err != nil {
			return fmt.Errorf("REPL error: %w", err)
		}
	}

	return nil
}
