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
	"yuruppu/internal/gcp"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
	"yuruppu/internal/storage"
	"yuruppu/internal/yuruppu"

	gcsstorage "cloud.google.com/go/storage"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	Port                      string // Server port (default: 8080)
	ChannelSecret             string
	ChannelAccessToken        string
	GCPMetadataTimeoutSeconds int    // GCP metadata server timeout in seconds (default: 2)
	GCPProjectID              string // Optional: auto-detected on Cloud Run
	GCPRegion                 string // Optional: auto-detected on Cloud Run
	LLMModel                  string // Required: LLM model name
	LLMCacheTTLMinutes        int    // LLM cache TTL in minutes (default: 60)
	LLMTimeoutSeconds         int    // LLM API timeout in seconds (default: 30)
	HistoryBucket             string // GCS bucket for chat history (optional)
}

const (
	// defaultPort is the default server port.
	defaultPort = "8080"

	// defaultGCPMetadataTimeoutSeconds is the default GCP metadata server timeout in seconds.
	defaultGCPMetadataTimeoutSeconds = 2

	// defaultLLMCacheTTLMinutes is the default LLM cache TTL in minutes.
	defaultLLMCacheTTLMinutes = 60

	// defaultLLMTimeoutSeconds is the default LLM API timeout in seconds.
	defaultLLMTimeoutSeconds = 30
)

// loadConfig loads configuration from environment variables.
// It reads PORT, LINE_CHANNEL_SECRET, LINE_CHANNEL_ACCESS_TOKEN, GCP_METADATA_TIMEOUT_SECONDS, GCP_PROJECT_ID, GCP_REGION, LLM_MODEL, LLM_CACHE_TTL_MINUTES, LLM_TIMEOUT_SECONDS, and HISTORY_BUCKET from environment.
// Returns error if required environment variables (LINE credentials, LLM_MODEL) are missing or empty after trimming whitespace.
// GCP_PROJECT_ID, GCP_REGION, and HISTORY_BUCKET are optional (auto-detected on Cloud Run, history disabled if not set).
// Returns error if timeout/TTL values are invalid (non-positive or non-integer).
func loadConfig() (*Config, error) {
	// Load and trim environment variables (order matches Config struct)
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

	// Parse GCP metadata timeout
	gcpMetadataTimeoutSeconds := defaultGCPMetadataTimeoutSeconds
	if env := os.Getenv("GCP_METADATA_TIMEOUT_SECONDS"); env != "" {
		parsed, err := strconv.Atoi(env)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("GCP_METADATA_TIMEOUT_SECONDS must be a positive integer: %s", env)
		}
		gcpMetadataTimeoutSeconds = parsed
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

	// Load optional history bucket
	historyBucket := strings.TrimSpace(os.Getenv("HISTORY_BUCKET"))

	return &Config{
		Port:                      port,
		ChannelSecret:             channelSecret,
		ChannelAccessToken:        channelAccessToken,
		GCPMetadataTimeoutSeconds: gcpMetadataTimeoutSeconds,
		GCPProjectID:              gcpProjectID,
		GCPRegion:                 gcpRegion,
		LLMModel:                  llmModel,
		LLMCacheTTLMinutes:        llmCacheTTLMinutes,
		LLMTimeoutSeconds:         llmTimeoutSeconds,
		HistoryBucket:             historyBucket,
	}, nil
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
	srv, err := line.NewServer(config.ChannelSecret, llmTimeout, logger)
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
	gcpMetadataTimeout := time.Duration(config.GCPMetadataTimeoutSeconds) * time.Second
	metadataHTTPClient := &http.Client{Timeout: gcpMetadataTimeout}
	metadataClient := gcp.NewClient(gcp.DefaultMetadataServerURL, metadataHTTPClient, logger)
	projectID := metadataClient.GetProjectID(config.GCPProjectID)
	region := metadataClient.GetRegion(config.GCPRegion)

	// Create Gemini agent with Yuruppu system prompt
	llmCacheTTL := time.Duration(config.LLMCacheTTLMinutes) * time.Minute
	geminiAgent, err := agent.NewGeminiAgent(context.Background(), projectID, region, config.LLMModel, llmCacheTTL, yuruppu.SystemPrompt, logger)
	if err != nil {
		logger.Error("failed to initialize Gemini agent", slog.Any("error", err))
		os.Exit(1)
	}

	// Create history repository if bucket is configured
	var historyRepo *history.Repository
	var gcsClient *gcsstorage.Client
	if config.HistoryBucket != "" {
		var err error
		gcsClient, err = gcsstorage.NewClient(context.Background())
		if err != nil {
			logger.Error("failed to create GCS client", slog.Any("error", err))
			os.Exit(1)
		}
		gcsStorage := storage.NewGCSStorage(gcsClient, config.HistoryBucket)
		historyRepo, err = history.NewRepository(gcsStorage)
		if err != nil {
			logger.Error("failed to create history repository", slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("chat history enabled", slog.String("bucket", config.HistoryBucket))
	} else {
		logger.Info("chat history disabled (HISTORY_BUCKET not set)")
	}

	// Register message handler
	srv.RegisterHandler(bot.NewHandler(historyRepo, geminiAgent, lineClient, logger))

	// Create HTTP server with graceful shutdown support
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", srv.HandleWebhook)
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
		logger.Info("server starting", slog.String("port", config.Port))
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

	// Close GCS client if it was created
	if gcsClient != nil {
		if err := gcsClient.Close(); err != nil {
			logger.Error("failed to close GCS client", slog.Any("error", err))
		}
	}

	logger.Info("graceful shutdown completed")
}
