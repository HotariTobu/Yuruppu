package main

import (
	// Standard library
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

	// Internal packages
	"yuruppu/internal/gcp"
	"yuruppu/internal/line/client"
	"yuruppu/internal/line/server"
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
	LLMModel                  string // Required: LLM model name
	LLMCacheTTLMinutes        int    // LLM cache TTL in minutes (default: 60)
	LLMTimeoutSeconds         int    // LLM API timeout in seconds (default: 30)
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
// It reads PORT, LINE_CHANNEL_SECRET, LINE_CHANNEL_ACCESS_TOKEN, GCP_METADATA_TIMEOUT_SECONDS, GCP_PROJECT_ID, GCP_REGION, LLM_MODEL, LLM_CACHE_TTL_MINUTES, and LLM_TIMEOUT_SECONDS from environment.
// Returns error if required environment variables (LINE credentials, LLM_MODEL) are missing or empty after trimming whitespace.
// GCP_PROJECT_ID and GCP_REGION are optional (auto-detected on Cloud Run).
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
	}, nil
}

// messageHandler implements the MessageHandler interface from yuruppu/internal/line.
type messageHandler struct {
	yuruppu *yuruppu.Yuruppu
	client  *client.Client
	logger  *slog.Logger
}

func (h *messageHandler) handleMessage(ctx context.Context, replyToken, userID, text string) error {
	response, err := h.yuruppu.GenerateText(ctx, text)
	if err != nil {
		h.logger.ErrorContext(ctx, "LLM call failed",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	if err := h.client.SendReply(replyToken, response); err != nil {
		h.logger.ErrorContext(ctx, "failed to send reply",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

func (h *messageHandler) HandleText(ctx context.Context, replyToken, userID, text string) error {
	return h.handleMessage(ctx, replyToken, userID, text)
}

func (h *messageHandler) HandleImage(ctx context.Context, replyToken, userID, messageID string) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent an image]")
}

func (h *messageHandler) HandleSticker(ctx context.Context, replyToken, userID, packageID, stickerID string) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent a sticker]")
}

func (h *messageHandler) HandleVideo(ctx context.Context, replyToken, userID, messageID string) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent a video]")
}

func (h *messageHandler) HandleAudio(ctx context.Context, replyToken, userID, messageID string) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent an audio]")
}

func (h *messageHandler) HandleLocation(ctx context.Context, replyToken, userID string, latitude, longitude float64) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent a location]")
}

func (h *messageHandler) HandleUnknown(ctx context.Context, replyToken, userID string) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent a message]")
}

// createHandler creates and returns an http.Handler with registered routes.
// AC-004: /webhook endpoint is registered with server.HandleWebhook.
func createHandler(srv *server.Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", srv.HandleWebhook)
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
	srv, err := server.New(config.ChannelSecret, llmTimeout, logger)
	if err != nil {
		logger.Error("failed to initialize server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	lineClient, err := client.New(config.ChannelAccessToken, logger)
	if err != nil {
		logger.Error("failed to initialize client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Resolve project ID and region from Cloud Run metadata with env var fallback
	gcpMetadataTimeout := time.Duration(config.GCPMetadataTimeoutSeconds) * time.Second
	metadataHTTPClient := &http.Client{Timeout: gcpMetadataTimeout}
	metadataClient := gcp.New(gcp.DefaultMetadataServerURL, metadataHTTPClient, logger)
	projectID := metadataClient.GetProjectID(config.GCPProjectID)
	region := metadataClient.GetRegion(config.GCPRegion)

	// Create LLM provider (pure API layer)
	llmProvider, err := llm.New(context.Background(), projectID, region, config.LLMModel, logger)
	if err != nil {
		logger.Error("failed to initialize LLM provider", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create Yuruppu agent (manages system prompt and caching)
	llmCacheTTL := time.Duration(config.LLMCacheTTLMinutes) * time.Minute
	yuruppuAgent := yuruppu.New(llmProvider, llmCacheTTL, logger)

	// Register message handler
	srv.RegisterHandler(&messageHandler{
		yuruppu: yuruppuAgent,
		client:  lineClient,
		logger:  logger,
	})

	// Create HTTP server with graceful shutdown support
	handler := createHandler(srv)
	httpServer := &http.Server{
		Addr:              ":" + config.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second, // Prevent Slowloris attacks
	}

	// Setup signal handling for graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		logger.Info("server starting", slog.String("port", config.Port))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", slog.String("error", err.Error()))
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
		logger.Error("failed to shutdown HTTP server gracefully", slog.String("error", err.Error()))
	}

	// Close Yuruppu agent first (cleans up cache)
	if err := yuruppuAgent.Close(shutdownCtx); err != nil {
		logger.Error("failed to close Yuruppu agent", slog.String("error", err.Error()))
	}

	// Close Provider (cleans up API connections)
	if err := llmProvider.Close(shutdownCtx); err != nil {
		logger.Error("failed to close LLM provider", slog.String("error", err.Error()))
	}

	logger.Info("graceful shutdown completed")
}
