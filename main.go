package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"yuruppu/internal/gcp"
	"yuruppu/internal/line"
	"yuruppu/internal/llm"
	"yuruppu/internal/yuruppu"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	Port                      string // Server port (default: 8080)
	ChannelSecret             string
	ChannelAccessToken        string
	GCPMetadataTimeoutSeconds int    // GCP metadata server timeout in seconds (default: 2)
	GCPProjectID              string // Optional: auto-detected on Cloud Run
	GCPRegion                 string // Optional: auto-detected on Cloud Run
	LLMTimeoutSeconds         int    // LLM API timeout in seconds (default: 30)
	LLMModel                  string // Required: LLM model name (e.g., "gemini-2.5-flash-lite")
}

const (
	// defaultPort is the default server port.
	defaultPort = "8080"

	// defaultGCPMetadataTimeoutSeconds is the default GCP metadata server timeout in seconds.
	defaultGCPMetadataTimeoutSeconds = 2

	// defaultLLMTimeoutSeconds is the default LLM API timeout in seconds.
	defaultLLMTimeoutSeconds = 30
)

// loadConfig loads configuration from environment variables.
// It reads PORT, LINE_CHANNEL_SECRET, LINE_CHANNEL_ACCESS_TOKEN, GCP_METADATA_TIMEOUT_SECONDS, GCP_PROJECT_ID, GCP_REGION, LLM_TIMEOUT_SECONDS, and LLM_MODEL from environment.
// Returns error if required environment variables (LINE credentials, LLM_MODEL) are missing or empty after trimming whitespace.
// GCP_PROJECT_ID and GCP_REGION are optional (auto-detected on Cloud Run).
// Returns error if timeout values are invalid (non-positive or non-integer).
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

	// Parse LLM timeout
	llmTimeoutSeconds := defaultLLMTimeoutSeconds
	if env := os.Getenv("LLM_TIMEOUT_SECONDS"); env != "" {
		parsed, err := strconv.Atoi(env)
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("LLM_TIMEOUT_SECONDS must be a positive integer: %s", env)
		}
		llmTimeoutSeconds = parsed
	}

	// Load and validate LLM_MODEL (required, no default)
	llmModel := strings.TrimSpace(os.Getenv("LLM_MODEL"))
	if llmModel == "" {
		return nil, errors.New("LLM_MODEL is required")
	}

	return &Config{
		Port:                      port,
		ChannelSecret:             channelSecret,
		ChannelAccessToken:        channelAccessToken,
		GCPMetadataTimeoutSeconds: gcpMetadataTimeoutSeconds,
		GCPProjectID:              gcpProjectID,
		GCPRegion:                 gcpRegion,
		LLMTimeoutSeconds:         llmTimeoutSeconds,
		LLMModel:                  llmModel,
	}, nil
}

// createMessageCallback creates a callback that adapts line.Message to yuruppu.Message.
// This adapter bridges the line and yuruppu packages without creating circular imports.
func createMessageCallback(handler *yuruppu.Handler) line.MessageHandler {
	return func(ctx context.Context, msg line.Message) error {
		// Convert line.Message to yuruppu.Message
		yMsg := yuruppu.Message{
			ReplyToken: msg.ReplyToken,
			Type:       msg.Type,
			Text:       msg.Text,
			UserID:     msg.UserID,
		}
		return handler.HandleMessage(ctx, yMsg)
	}
}

// createHandler creates and returns an http.Handler with registered routes.
// AC-004: /webhook endpoint is registered with server.HandleWebhook.
func createHandler(server *line.Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", server.HandleWebhook)
	return mux
}

func main() {
	// Create logger with JSON handler for structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		logger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize components
	llmTimeout := time.Duration(config.LLMTimeoutSeconds) * time.Second
	server, err := line.NewServer(config.ChannelSecret, llmTimeout, logger)
	if err != nil {
		logger.Error("failed to initialize server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	client, err := line.NewClient(config.ChannelAccessToken, logger)
	if err != nil {
		logger.Error("failed to initialize client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Resolve project ID and region from Cloud Run metadata with env var fallback
	gcpMetadataTimeout := time.Duration(config.GCPMetadataTimeoutSeconds) * time.Second
	metadataHTTPClient := &http.Client{Timeout: gcpMetadataTimeout}
	metadataClient := gcp.NewMetadataClient(gcp.DefaultMetadataServerURL, metadataHTTPClient, logger)
	projectID := metadataClient.GetProjectID(config.GCPProjectID)
	region := metadataClient.GetRegion(config.GCPRegion)

	llmProvider, err := llm.NewVertexAIClient(context.Background(), projectID, region, config.LLMModel, logger)
	if err != nil {
		logger.Error("failed to initialize LLM", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create yuruppu handler and register callback
	yHandler := yuruppu.NewHandler(llmProvider, client, logger)
	server.OnMessage(createMessageCallback(yHandler))

	// Start HTTP server
	handler := createHandler(server)
	logger.Info("server starting", slog.String("port", config.Port))
	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
