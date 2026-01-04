package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/bot"
	"yuruppu/internal/history"
	lineclient "yuruppu/internal/line/client"
	lineserver "yuruppu/internal/line/server"
	"yuruppu/internal/media"
	"yuruppu/internal/profile"
	"yuruppu/internal/storage"
	"yuruppu/internal/toolset/reply"
	"yuruppu/internal/toolset/skip"
	"yuruppu/internal/toolset/weather"
	"yuruppu/internal/yuruppu"

	"cloud.google.com/go/compute/metadata"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	LogLevel           slog.Level // Log level (default: INFO)
	Endpoint           string     // Webhook endpoint path (required)
	Port               string     // Server port (default: 8080)
	ChannelSecret      string
	ChannelAccessToken string
	GCPProjectID       string // Optional: auto-detected on Cloud Run
	GCPRegion          string // Optional: auto-detected on Cloud Run
	LLMModel           string // Required: LLM model name
	LLMCacheTTLMinutes int    // LLM cache TTL in minutes (default: 60)
	LLMTimeoutSeconds  int    // LLM API timeout in seconds (default: 30)
	ProfileBucket      string // GCS bucket for user profiles
	HistoryBucket      string // GCS bucket for chat history
	MediaBucket        string // GCS bucket for media files
}

const (
	// defaultPort is the default server port.
	defaultPort = "8080"

	// defaultLLMCacheTTLMinutes is the default LLM cache TTL in minutes.
	defaultLLMCacheTTLMinutes = 60

	// defaultLLMTimeoutSeconds is the default LLM API timeout in seconds.
	defaultLLMTimeoutSeconds = 30
)

// loadConfig loads configuration from environment variables.
// It reads LOG_LEVEL, ENDPOINT, PORT, LINE_CHANNEL_SECRET, LINE_CHANNEL_ACCESS_TOKEN, GCP_PROJECT_ID, GCP_REGION, LLM_MODEL, LLM_CACHE_TTL_MINUTES, LLM_TIMEOUT_SECONDS, PROFILE_BUCKET, HISTORY_BUCKET, and MEDIA_BUCKET from environment.
// Returns error if required environment variables (ENDPOINT, LINE credentials, LLM_MODEL, PROFILE_BUCKET, HISTORY_BUCKET, MEDIA_BUCKET) are missing or empty after trimming whitespace.
// GCP_PROJECT_ID and GCP_REGION are optional (auto-detected on Cloud Run).
// LOG_LEVEL is optional (default: INFO, valid values: DEBUG, INFO, WARN, ERROR).
// Returns error if timeout/TTL values are invalid (non-positive or non-integer).
func loadConfig() (*Config, error) {
	// Load and trim environment variables (order matches Config struct)
	logLevel := slog.LevelInfo
	if env := strings.TrimSpace(os.Getenv("LOG_LEVEL")); env != "" {
		switch strings.ToUpper(env) {
		case "DEBUG":
			logLevel = slog.LevelDebug
		case "INFO":
			logLevel = slog.LevelInfo
		case "WARN":
			logLevel = slog.LevelWarn
		case "ERROR":
			logLevel = slog.LevelError
		default:
			return nil, fmt.Errorf("LOG_LEVEL must be one of DEBUG, INFO, WARN, ERROR: %s", env)
		}
	}

	endpoint := strings.TrimSpace(os.Getenv("ENDPOINT"))
	if endpoint == "" {
		return nil, errors.New("ENDPOINT is required")
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	channelSecret := strings.TrimSpace(os.Getenv("LINE_CHANNEL_SECRET"))
	channelAccessToken := strings.TrimSpace(os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"))

	// Validate LINE_CHANNEL_SECRET
	if channelSecret == "" {
		return nil, errors.New("LINE_CHANNEL_SECRET is required")
	}

	// Validate LINE_CHANNEL_ACCESS_TOKEN
	if channelAccessToken == "" {
		return nil, errors.New("LINE_CHANNEL_ACCESS_TOKEN is required")
	}

	gcpProjectID := strings.TrimSpace(os.Getenv("GCP_PROJECT_ID"))
	gcpRegion := strings.TrimSpace(os.Getenv("GCP_REGION"))

	// Load and validate LLM_MODEL (required, no default)
	llmModel := strings.TrimSpace(os.Getenv("LLM_MODEL"))
	if llmModel == "" {
		return nil, errors.New("LLM_MODEL is required")
	}

	// Parse LLM cache TTL
	llmCacheTTLMinutes := defaultLLMCacheTTLMinutes
	if env := os.Getenv("LLM_CACHE_TTL_MINUTES"); env != "" {
		parsed, err := strconv.Atoi(env)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("LLM_CACHE_TTL_MINUTES must be a positive integer: %s", env)
		}
		llmCacheTTLMinutes = parsed
	}

	// Parse LLM timeout
	llmTimeoutSeconds := defaultLLMTimeoutSeconds
	if env := os.Getenv("LLM_TIMEOUT_SECONDS"); env != "" {
		parsed, err := strconv.Atoi(env)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("LLM_TIMEOUT_SECONDS must be a positive integer: %s", env)
		}
		llmTimeoutSeconds = parsed
	}

	// Load and validate PROFILE_BUCKET (required)
	profileBucket := strings.TrimSpace(os.Getenv("PROFILE_BUCKET"))
	if profileBucket == "" {
		return nil, errors.New("PROFILE_BUCKET is required")
	}

	// Load and validate HISTORY_BUCKET (required)
	historyBucket := strings.TrimSpace(os.Getenv("HISTORY_BUCKET"))
	if historyBucket == "" {
		return nil, errors.New("HISTORY_BUCKET is required")
	}

	// Load and validate MEDIA_BUCKET (required)
	mediaBucket := strings.TrimSpace(os.Getenv("MEDIA_BUCKET"))
	if mediaBucket == "" {
		return nil, errors.New("MEDIA_BUCKET is required")
	}

	return &Config{
		LogLevel:           logLevel,
		Endpoint:           endpoint,
		Port:               port,
		ChannelSecret:      channelSecret,
		ChannelAccessToken: channelAccessToken,
		GCPProjectID:       gcpProjectID,
		GCPRegion:          gcpRegion,
		LLMModel:           llmModel,
		LLMCacheTTLMinutes: llmCacheTTLMinutes,
		LLMTimeoutSeconds:  llmTimeoutSeconds,
		ProfileBucket:      profileBucket,
		HistoryBucket:      historyBucket,
		MediaBucket:        mediaBucket,
	}, nil
}

func getProjectIDAndRegion(ctx context.Context) (string, string, error) {
	if !metadata.OnGCE() {
		return "", "", errors.New("not running on GCE")
	}
	projectID, err1 := metadata.ProjectIDWithContext(ctx)
	zone, err2 := metadata.ZoneWithContext(ctx)
	if err := errors.Join(err1, err2); err != nil {
		return "", "", err
	}
	return projectID, zone[:len(zone)-2], nil
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load configuration:", err)
		os.Exit(1)
	}

	// Create logger with JSON handler for structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: config.LogLevel,
	}))

	// Initialize components
	llmTimeout := time.Duration(config.LLMTimeoutSeconds) * time.Second
	lineServer, err := lineserver.NewServer(config.ChannelSecret, llmTimeout, logger)
	if err != nil {
		logger.Error("failed to initialize server", slog.Any("error", err))
		os.Exit(1)
	}

	lineClient, err := lineclient.NewClient(config.ChannelAccessToken, logger)
	if err != nil {
		logger.Error("failed to initialize client", slog.Any("error", err))
		os.Exit(1)
	}

	// Resolve project ID and region from Cloud Run metadata with env var fallback
	projectID, region, err := getProjectIDAndRegion(context.Background())
	if err != nil {
		logger.Warn("failed to get metadata from GCP, using fallback", slog.Any("error", err))
		projectID = config.GCPProjectID
		region = config.GCPRegion
	}

	// Create tools
	weatherTool, err := weather.NewTool(&http.Client{Timeout: 30 * time.Second}, logger)
	if err != nil {
		logger.Error("failed to create weather tool", slog.Any("error", err))
		os.Exit(1)
	}

	// Create history repository (needed by reply tool and handler)
	historyStorage, err := storage.NewGCSStorage(context.Background(), config.HistoryBucket)
	if err != nil {
		logger.Error("failed to create history storage", slog.Any("error", err))
		os.Exit(1)
	}
	historySvc, err := history.NewService(historyStorage)
	if err != nil {
		logger.Error("failed to create history service", slog.Any("error", err))
		os.Exit(1)
	}

	// Create reply tool
	replyTool, err := reply.NewTool(lineClient, historySvc, logger)
	if err != nil {
		logger.Error("failed to create reply tool", slog.Any("error", err))
		os.Exit(1)
	}

	// Create skip tool
	skipTool, err := skip.NewTool(logger)
	if err != nil {
		logger.Error("failed to create skip tool", slog.Any("error", err))
		os.Exit(1)
	}

	// Create Gemini agent with Yuruppu system prompt
	llmCacheTTL := time.Duration(config.LLMCacheTTLMinutes) * time.Minute
	geminiAgent, err := agent.NewGeminiAgent(context.Background(), agent.GeminiConfig{
		ProjectID:        projectID,
		Region:           region,
		Model:            config.LLMModel,
		SystemPrompt:     yuruppu.SystemPrompt,
		Tools:            []agent.Tool{weatherTool, replyTool, skipTool},
		FunctionCallOnly: true,
		CacheDisplayName: "yuruppu-system-prompt",
		CacheTTL:         llmCacheTTL,
	}, logger)
	if err != nil {
		logger.Error("failed to initialize Gemini agent", slog.Any("error", err))
		os.Exit(1)
	}

	// Create profile service
	profileStorage, err := storage.NewGCSStorage(context.Background(), config.ProfileBucket)
	if err != nil {
		logger.Error("failed to create profile storage", slog.Any("error", err))
		os.Exit(1)
	}
	profileService, err := profile.NewService(profileStorage, logger)
	if err != nil {
		logger.Error("failed to create profile service", slog.Any("error", err))
		os.Exit(1)
	}

	// Create media service
	mediaStorage, err := storage.NewGCSStorage(context.Background(), config.MediaBucket)
	if err != nil {
		logger.Error("failed to create media storage", slog.Any("error", err))
		os.Exit(1)
	}
	mediaSvc, err := media.NewService(mediaStorage, logger)
	if err != nil {
		logger.Error("failed to create media service", slog.Any("error", err))
		os.Exit(1)
	}

	// Create message handler
	messageHandler, err := bot.NewHandler(lineClient, profileService, historySvc, mediaSvc, geminiAgent, logger)
	if err != nil {
		logger.Error("failed to create message handler", slog.Any("error", err))
		os.Exit(1)
	}

	// Register message handler
	lineServer.RegisterHandler(messageHandler)

	// Create HTTP server with graceful shutdown support
	mux := http.NewServeMux()
	mux.HandleFunc(config.Endpoint, lineServer.HandleWebhook)
	httpServer := &http.Server{
		Addr:              ":" + config.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second, // Prevent Slowloris attacks
	}

	// Setup signal handling for graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		logger.Info("server starting",
			slog.String("endpoint", config.Endpoint),
			slog.String("port", config.Port),
		)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-shutdown
	logger.Info("shutdown signal received, initiating graceful shutdown")

	// Create context with timeout for graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server gracefully
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown HTTP server gracefully", slog.Any("error", err))
	}

	// Close Gemini agent (cleans up cache and API connections)
	if err := geminiAgent.Close(shutdownCtx); err != nil {
		logger.Error("failed to close Gemini agent", slog.Any("error", err))
	}

	// Close GCS storage
	if err := profileStorage.Close(shutdownCtx); err != nil {
		logger.Error("failed to close profile storage", slog.Any("error", err))
	}
	if err := historyStorage.Close(shutdownCtx); err != nil {
		logger.Error("failed to close history storage", slog.Any("error", err))
	}
	if err := mediaStorage.Close(shutdownCtx); err != nil {
		logger.Error("failed to close media storage", slog.Any("error", err))
	}

	logger.Info("graceful shutdown completed")
}
