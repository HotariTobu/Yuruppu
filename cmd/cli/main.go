package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"yuruppu/internal/agent"
	"yuruppu/internal/bot"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
	"yuruppu/internal/mock"
	"yuruppu/internal/profile"
	"yuruppu/internal/toolset/reply"
	"yuruppu/internal/toolset/skip"
	"yuruppu/internal/toolset/weather"
)

var (
	userIDFlag  = flag.String("user-id", "default", "User ID (pattern: [0-9a-z_]+)")
	dataDirFlag = flag.String("data-dir", ".yuruppu", "Data directory")
	messageFlag = flag.String("message", "", "Single message mode (send one message and exit)")
)

var userIDPattern = regexp.MustCompile(`^[0-9a-z_]+$`)

func main() {
	flag.Parse()

	// Setup logger to stderr
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Validate user ID
	if !userIDPattern.MatchString(*userIDFlag) {
		fmt.Fprintf(os.Stderr, "Error: Invalid user ID '%s'. Must match pattern [0-9a-z_]+\n", *userIDFlag)
		os.Exit(1)
	}

	// Check/create data directory
	if err := ensureDataDir(*dataDirFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create mock storage instances for each bucket
	profileStorage := mock.NewFileStorage(filepath.Join(*dataDirFlag, "profiles"))
	historyStorage := mock.NewFileStorage(filepath.Join(*dataDirFlag, "history"))
	mediaStorage := mock.NewFileStorage(filepath.Join(*dataDirFlag, "media"))

	// Create services
	profileService := profile.NewService(profileStorage, logger)
	historyRepo := history.NewRepository(historyStorage)

	// Check/create user profile
	ctx := context.Background()
	if err := ensureUserProfile(ctx, profileService, *userIDFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get LLM configuration from environment
	projectID := os.Getenv("GCP_PROJECT_ID")
	region := os.Getenv("GCP_REGION")
	model := os.Getenv("LLM_MODEL")
	if projectID == "" || region == "" || model == "" {
		fmt.Fprintln(os.Stderr, "Error: Missing required environment variables: GCP_PROJECT_ID, GCP_REGION, LLM_MODEL")
		os.Exit(1)
	}

	// Create LLM agent
	llmAgent, err := agent.NewGeminiAgent(ctx, projectID, region, model, 60*time.Minute, 30*time.Second, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating LLM agent: %v\n", err)
		os.Exit(1)
	}
	defer llmAgent.Close(ctx)

	// Create mock LINE client
	lineClient := mock.NewLineClient(logger)

	// Create tools
	replyTool := reply.NewTool(lineClient, historyRepo, logger)
	weatherTool := weather.NewTool(&http.Client{Timeout: 10 * time.Second}, logger)
	skipTool := skip.NewTool(logger)
	llmAgent.RegisterTools(replyTool, weatherTool, skipTool)

	// Create handler
	handler := bot.NewHandler(lineClient, profileService, historyRepo, mediaStorage, llmAgent, logger)

	// Run in single message mode or REPL mode
	if *messageFlag != "" {
		runSingleMessage(ctx, handler, *userIDFlag, *messageFlag)
	} else {
		runREPL(ctx, handler, *userIDFlag, logger)
	}
}

func ensureDataDir(dataDir string) error {
	info, err := os.Stat(dataDir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", dataDir)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	// Directory doesn't exist, prompt user
	fmt.Fprintf(os.Stderr, "Directory %s does not exist. Create it? [y/N] ", dataDir)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if response == "y" || response == "yes" {
			return os.MkdirAll(dataDir, 0755)
		}
	}
	return fmt.Errorf("directory creation cancelled")
}

func ensureUserProfile(ctx context.Context, profileService *profile.Service, userID string) error {
	_, err := profileService.GetUserProfile(ctx, userID)
	if err == nil {
		return nil // Profile exists
	}

	// Profile doesn't exist, prompt for creation
	fmt.Fprintln(os.Stderr, "Creating new user profile...")
	scanner := bufio.NewScanner(os.Stdin)

	// Display name (required)
	var displayName string
	for {
		fmt.Fprint(os.Stderr, "Display name (required): ")
		if !scanner.Scan() {
			return fmt.Errorf("failed to read display name")
		}
		displayName = strings.TrimSpace(scanner.Text())
		if displayName != "" {
			break
		}
		fmt.Fprintln(os.Stderr, "Display name cannot be empty. Please try again.")
	}

	// Picture URL (optional)
	fmt.Fprint(os.Stderr, "Picture URL (optional, press Enter to skip): ")
	var pictureURL, pictureMIMEType string
	if scanner.Scan() {
		pictureURL = strings.TrimSpace(scanner.Text())
		if pictureURL != "" {
			// Fetch MIME type
			pictureMIMEType = fetchPictureMIMEType(ctx, pictureURL)
			if pictureMIMEType == "" {
				fmt.Fprintln(os.Stderr, "Warning: Could not fetch MIME type, skipping picture URL")
				pictureURL = ""
			}
		}
	}

	// Status message (optional)
	fmt.Fprint(os.Stderr, "Status message (optional, press Enter to skip): ")
	var statusMessage string
	if scanner.Scan() {
		statusMessage = strings.TrimSpace(scanner.Text())
	}

	// Create and save profile
	userProfile := &profile.UserProfile{
		DisplayName:     displayName,
		PictureURL:      pictureURL,
		PictureMIMEType: pictureMIMEType,
		StatusMessage:   statusMessage,
	}
	return profileService.SetUserProfile(ctx, userID, userProfile)
}

func fetchPictureMIMEType(ctx context.Context, url string) string {
	// Append /small to minimize data transfer (LINE pattern)
	fetchURL := url
	if !strings.HasSuffix(url, "/small") {
		fetchURL = url + "/small"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return ""
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return "image/jpeg" // Default fallback
	}
	// Extract MIME type (remove charset etc.)
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = contentType[:idx]
	}
	return strings.TrimSpace(contentType)
}

func runSingleMessage(ctx context.Context, handler *bot.Handler, userID, message string) {
	ctx = line.WithUserID(ctx, userID)
	ctx = line.WithSourceID(ctx, userID) // TR-003: userID = sourceID
	ctx = line.WithReplyToken(ctx, "cli-single")

	if err := handler.HandleText(ctx, message); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runREPL(ctx context.Context, handler *bot.Handler, userID string, logger *slog.Logger) {
	scanner := bufio.NewScanner(os.Stdin)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	ctrlCCount := 0
	inputChan := make(chan string)
	errChan := make(chan error)

	// Start input goroutine
	go func() {
		for {
			fmt.Print("> ")
			if scanner.Scan() {
				inputChan <- scanner.Text()
			} else {
				errChan <- scanner.Err()
				return
			}
		}
	}()

	fmt.Fprintln(os.Stderr, "Ready. Type /quit or press Ctrl+C twice to exit.")

	for {
		select {
		case <-sigChan:
			ctrlCCount++
			if ctrlCCount >= 2 {
				fmt.Fprintln(os.Stderr, "\nExiting...")
				return
			}
			fmt.Fprintln(os.Stderr, "\nPress Ctrl+C again to exit")
		case text := <-inputChan:
			ctrlCCount = 0 // Reset on input
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			if text == "/quit" {
				fmt.Fprintln(os.Stderr, "Goodbye!")
				return
			}

			// Create context with user info
			msgCtx := line.WithUserID(ctx, userID)
			msgCtx = line.WithSourceID(msgCtx, userID)
			msgCtx = line.WithReplyToken(msgCtx, fmt.Sprintf("cli-%d", time.Now().UnixNano()))

			if err := handler.HandleText(msgCtx, text); err != nil {
				logger.Error("handler error", "error", err)
			}
		case err := <-errChan:
			if err != nil {
				logger.Error("input error", "error", err)
			}
			return
		}
	}
}
