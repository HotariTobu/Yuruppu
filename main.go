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
	"yuruppu/internal/toolset/event"
	"yuruppu/internal/toolset/reply"
	"yuruppu/internal/toolset/skip"
	"yuruppu/internal/toolset/weather"
	"yuruppu/internal/yuruppu"

	eventdomain "yuruppu/internal/event"

	"cloud.google.com/go/compute/metadata"
	gcsstorage "cloud.google.com/go/storage"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	LogLevel                      slog.Level // Log level (default: INFO)
	Endpoint                      string     // Webhook endpoint path (required)
	Port                          string     // Server port (default: 8080)
	ChannelSecret                 string
	ChannelAccessToken            string
	GCPProjectID                  string // Optional: auto-detected on Cloud Run
	GCPRegion                     string // Optional: auto-detected on Cloud Run
	LLMModel                      string // Required: LLM model name
	LLMCacheTTLMinutes            int    // LLM cache TTL in minutes (default: 60)
	LLMTimeoutSeconds             int    // LLM API timeout in seconds (default: 30)
	BucketName                    string // GCS bucket for storage
	TypingIndicatorDelaySeconds   int    // Delay before showing typing indicator (default: 3)
	TypingIndicatorTimeoutSeconds int    // Typing indicator display duration (default: 30, range: 5-60)
	EventListMaxPeriodDays        int    // Max period in days for list_events
	EventListLimit                int    // Max items for list_events (default: 5)
}

const (
	// defaultPort is the default server port.
	defaultPort = "8080"

	// defaultLLMCacheTTLMinutes is the default LLM cache TTL in minutes.
	defaultLLMCacheTTLMinutes = 60

	// defaultLLMTimeoutSeconds is the default LLM API timeout in seconds.
	defaultLLMTimeoutSeconds = 30

	// defaultTypingIndicatorDelaySeconds is the delay before showing typing indicator.
	defaultTypingIndicatorDelaySeconds = 5

	// defaultTypingIndicatorTimeoutSeconds is the typing indicator display duration.
	defaultTypingIndicatorTimeoutSeconds = 30

	// defaultEventListMaxPeriodDays is the max period in days for list_events.
	defaultEventListMaxPeriodDays = 366

	// defaultEventListLimit is the max items for list_events.
	defaultEventListLimit = 5
)

// parsePositiveInt parses an environment variable as a positive integer.
// Returns the default value if the environment variable is not set.
// Returns an error if the value is invalid or not positive.
func parsePositiveInt(envName string, defaultValue int) (int, error) {
	env := os.Getenv(envName)
	if env == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(env)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer: %s", envName, env)
	}
	return parsed, nil
}

// loadConfig loads configuration from environment variables.
// It reads LOG_LEVEL, ENDPOINT, PORT, LINE_CHANNEL_SECRET, LINE_CHANNEL_ACCESS_TOKEN, GCP_PROJECT_ID, GCP_REGION, LLM_MODEL, LLM_CACHE_TTL_MINUTES, LLM_TIMEOUT_SECONDS, and BUCKET_NAME from environment.
// Returns error if required environment variables (ENDPOINT, LINE credentials, LLM_MODEL, BUCKET_NAME) are missing or empty after trimming whitespace.
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
	llmCacheTTLMinutes, err := parsePositiveInt("LLM_CACHE_TTL_MINUTES", defaultLLMCacheTTLMinutes)
	if err != nil {
		return nil, err
	}

	// Parse LLM timeout
	llmTimeoutSeconds, err := parsePositiveInt("LLM_TIMEOUT_SECONDS", defaultLLMTimeoutSeconds)
	if err != nil {
		return nil, err
	}

	// Load and validate BUCKET_NAME (required)
	bucketName := strings.TrimSpace(os.Getenv("BUCKET_NAME"))
	if bucketName == "" {
		return nil, errors.New("BUCKET_NAME is required")
	}

	// Parse typing indicator delay
	typingIndicatorDelaySeconds, err := parsePositiveInt("TYPING_INDICATOR_DELAY_SECONDS", defaultTypingIndicatorDelaySeconds)
	if err != nil {
		return nil, err
	}

	// Parse typing indicator timeout (must be 5-60 seconds per LINE API)
	typingIndicatorTimeoutSeconds := defaultTypingIndicatorTimeoutSeconds
	if env := os.Getenv("TYPING_INDICATOR_TIMEOUT_SECONDS"); env != "" {
		parsed, err := strconv.Atoi(env)
		if err != nil || parsed < 5 || parsed > 60 {
			return nil, fmt.Errorf("TYPING_INDICATOR_TIMEOUT_SECONDS must be between 5 and 60 seconds: %s", env)
		}
		typingIndicatorTimeoutSeconds = parsed
	}

	// Parse event list max period days
	eventListMaxPeriodDays, err := parsePositiveInt("EVENT_LIST_MAX_PERIOD_DAYS", defaultEventListMaxPeriodDays)
	if err != nil {
		return nil, err
	}

	// Parse event list limit
	eventListLimit, err := parsePositiveInt("EVENT_LIST_LIMIT", defaultEventListLimit)
	if err != nil {
		return nil, err
	}

	return &Config{
		LogLevel:                      logLevel,
		Endpoint:                      endpoint,
		Port:                          port,
		ChannelSecret:                 channelSecret,
		ChannelAccessToken:            channelAccessToken,
		GCPProjectID:                  gcpProjectID,
		GCPRegion:                     gcpRegion,
		LLMModel:                      llmModel,
		LLMCacheTTLMinutes:            llmCacheTTLMinutes,
		LLMTimeoutSeconds:             llmTimeoutSeconds,
		BucketName:                    bucketName,
		TypingIndicatorDelaySeconds:   typingIndicatorDelaySeconds,
		TypingIndicatorTimeoutSeconds: typingIndicatorTimeoutSeconds,
		EventListMaxPeriodDays:        eventListMaxPeriodDays,
		EventListLimit:                eventListLimit,
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
		slog.Error("failed to load configuration", slog.Any("error", err))
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

	// Create shared GCS client
	gcsClient, err := gcsstorage.NewClient(context.Background())
	if err != nil {
		logger.Error("failed to create GCS client", slog.Any("error", err))
		os.Exit(1)
	}

	// Create history repository (needed by reply tool and handler)
	historyStorage, err := storage.NewGCSStorage(gcsClient, config.BucketName, "history/")
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

	// Create profile service (needed by event tools and handler)
	profileStorage, err := storage.NewGCSStorage(gcsClient, config.BucketName, "profile/")
	if err != nil {
		logger.Error("failed to create profile storage", slog.Any("error", err))
		os.Exit(1)
	}
	profileService, err := profile.NewService(profileStorage, logger)
	if err != nil {
		logger.Error("failed to create profile service", slog.Any("error", err))
		os.Exit(1)
	}

	// Create event service and tools
	eventStorage, err := storage.NewGCSStorage(gcsClient, config.BucketName, "event/")
	if err != nil {
		logger.Error("failed to create event storage", slog.Any("error", err))
		os.Exit(1)
	}
	eventService, err := eventdomain.NewService(eventStorage)
	if err != nil {
		logger.Error("failed to create event service", slog.Any("error", err))
		os.Exit(1)
	}
	eventTools, err := event.NewTools(eventService, profileService, config.EventListMaxPeriodDays, config.EventListLimit, logger)
	if err != nil {
		logger.Error("failed to create event tools", slog.Any("error", err))
		os.Exit(1)
	}

	// Collect all tools
	toolset := append([]agent.Tool{weatherTool, replyTool, skipTool}, eventTools...)

	// Create Gemini agent with Yuruppu system prompt
	systemPrompt, err := yuruppu.GetSystemPrompt()
	if err != nil {
		logger.Error("failed to get system prompt", slog.Any("error", err))
		os.Exit(1)
	}
	llmCacheTTL := time.Duration(config.LLMCacheTTLMinutes) * time.Minute
	geminiAgent, err := agent.NewGeminiAgent(context.Background(), agent.GeminiConfig{
		ProjectID:        projectID,
		Region:           region,
		Model:            config.LLMModel,
		SystemPrompt:     systemPrompt,
		Tools:            toolset,
		FunctionCallOnly: true,
		CacheDisplayName: "yuruppu-system-prompt",
		CacheTTL:         llmCacheTTL,
	}, logger)
	if err != nil {
		logger.Error("failed to initialize Gemini agent", slog.Any("error", err))
		os.Exit(1)
	}

	// Create media service
	mediaStorage, err := storage.NewGCSStorage(gcsClient, config.BucketName, "media/")
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
	handlerConfig := bot.HandlerConfig{
		TypingIndicatorDelay:   time.Duration(config.TypingIndicatorDelaySeconds) * time.Second,
		TypingIndicatorTimeout: time.Duration(config.TypingIndicatorTimeoutSeconds) * time.Second,
	}
	messageHandler, err := bot.NewHandler(lineClient, profileService, historySvc, mediaSvc, geminiAgent, handlerConfig, logger)
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

	// Close GCS client
	if err := gcsClient.Close(); err != nil {
		logger.Error("failed to close GCS client", slog.Any("error", err))
	}

	logger.Info("graceful shutdown completed")
}
