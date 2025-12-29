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
	"yuruppu/internal/line"
	"yuruppu/internal/storage"
	"yuruppu/internal/yuruppu"

	"cloud.google.com/go/compute/metadata"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	Endpoint           string // Webhook endpoint path (required)
	Port               string // Server port (default: 8080)
	ChannelSecret      string
	ChannelAccessToken string
	GCPProjectID       string // Optional: auto-detected on Cloud Run
	GCPRegion          string // Optional: auto-detected on Cloud Run
	LLMModel           string // Required: LLM model name
	LLMCacheTTLMinutes int    // LLM cache TTL in minutes (default: 60)
	LLMTimeoutSeconds  int    // LLM API timeout in seconds (default: 30)
	HistoryBucket      string // GCS bucket for chat history
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
// It reads ENDPOINT, PORT, LINE_CHANNEL_SECRET, LINE_CHANNEL_ACCESS_TOKEN, GCP_PROJECT_ID, GCP_REGION, LLM_MODEL, LLM_CACHE_TTL_MINUTES, LLM_TIMEOUT_SECONDS, and HISTORY_BUCKET from environment.
// Returns error if required environment variables (ENDPOINT, LINE credentials, LLM_MODEL, HISTORY_BUCKET) are missing or empty after trimming whitespace.
// GCP_PROJECT_ID and GCP_REGION are optional (auto-detected on Cloud Run).
// Returns error if timeout/TTL values are invalid (non-positive or non-integer).
func loadConfig() (*Config, error) {
	// Load and trim environment variables (order matches Config struct)
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

	// Load and validate HISTORY_BUCKET (required)
	historyBucket := strings.TrimSpace(os.Getenv("HISTORY_BUCKET"))
	if historyBucket == "" {
		return nil, errors.New("HISTORY_BUCKET is required")
	}

	return &Config{
		Endpoint:           endpoint,
		Port:               port,
		ChannelSecret:      channelSecret,
		ChannelAccessToken: channelAccessToken,
		GCPProjectID:       gcpProjectID,
		GCPRegion:          gcpRegion,
		LLMModel:           llmModel,
		LLMCacheTTLMinutes: llmCacheTTLMinutes,
		LLMTimeoutSeconds:  llmTimeoutSeconds,
		HistoryBucket:      historyBucket,
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
	// Create logger with JSON handler for structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		logger.Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize components
	llmTimeout := time.Duration(config.LLMTimeoutSeconds) * time.Second
	lineServer, err := line.NewServer(config.ChannelSecret, llmTimeout, logger)
	if err != nil {
		logger.Error("failed to initialize server", slog.Any("error", err))
		os.Exit(1)
	}

	lineClient, err := line.NewClient(config.ChannelAccessToken, logger)
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

	// Create Gemini agent with Yuruppu system prompt
	llmCacheTTL := time.Duration(config.LLMCacheTTLMinutes) * time.Minute
	geminiAgent, err := agent.NewGeminiAgent(context.Background(), agent.GeminiConfig{
		ProjectID:        projectID,
		Region:           region,
		Model:            config.LLMModel,
		CacheTTL:         llmCacheTTL,
		CacheDisplayName: "yuruppu-system-prompt",
		SystemPrompt:     yuruppu.SystemPrompt,
	}, logger)
	if err != nil {
		logger.Error("failed to initialize Gemini agent", slog.Any("error", err))
		os.Exit(1)
	}

	// Create history repository
	gcsStorage, err := storage.NewGCSStorage(context.Background(), config.HistoryBucket)
	if err != nil {
		logger.Error("failed to create GCS storage", slog.Any("error", err))
		os.Exit(1)
	}
	historyRepo, err := history.NewRepository(gcsStorage)
	if err != nil {
		logger.Error("failed to create history repository", slog.Any("error", err))
		os.Exit(1)
	}

	// Create message handler
	messageHandler, err := bot.NewHandler(historyRepo, geminiAgent, lineClient, logger)
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
	if err := gcsStorage.Close(shutdownCtx); err != nil {
		logger.Error("failed to close GCS storage", slog.Any("error", err))
	}

	logger.Info("graceful shutdown completed")
}
