package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"yuruppu/internal/line"
	"yuruppu/internal/llm"
	"yuruppu/internal/yuruppu"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	ChannelSecret      string
	ChannelAccessToken string
	GCPProjectID       string
	LLMTimeoutSeconds  int    // NFR-001: LLM API timeout in seconds (default: 30)
	Port               string // SC-001: Server port (default: 8080)
	GCPRegion          string // SC-002: GCP region for Vertex AI (default: asia-northeast1)
}

const (
	// defaultLLMTimeoutSeconds is the default LLM API timeout in seconds.
	// NFR-001: LLM API total request timeout should be configurable (default: 30 seconds)
	defaultLLMTimeoutSeconds = 30

	// defaultPort is the default server port.
	// SC-001: Server reads PORT from environment with 8080 as default.
	defaultPort = "8080"

	// defaultRegion is the default GCP region for Vertex AI API calls.
	// SC-002: GCP_REGION is read from environment with asia-northeast1 as default.
	defaultRegion = "asia-northeast1"
)

// loadConfig loads configuration from environment variables.
// It reads LINE_CHANNEL_SECRET, LINE_CHANNEL_ACCESS_TOKEN, GCP_PROJECT_ID, LLM_TIMEOUT_SECONDS, PORT, and GCP_REGION from environment.
// Returns error if any required environment variable is missing or empty after trimming whitespace.
// FR-003: Load LLM API credentials from environment variables
// NFR-001: Load LLM timeout configuration
// SC-001: Load PORT configuration
// SC-002: Load GCP_REGION configuration
func loadConfig() (*Config, error) {
	// Load and trim environment variables
	channelSecret := strings.TrimSpace(os.Getenv("LINE_CHANNEL_SECRET"))
	channelAccessToken := strings.TrimSpace(os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"))
	gcpProjectID := strings.TrimSpace(os.Getenv("GCP_PROJECT_ID"))
	port := strings.TrimSpace(os.Getenv("PORT"))
	gcpRegion := strings.TrimSpace(os.Getenv("GCP_REGION"))

	// Validate LINE_CHANNEL_SECRET first
	if channelSecret == "" {
		return nil, errors.New("LINE_CHANNEL_SECRET is required")
	}

	// Validate LINE_CHANNEL_ACCESS_TOKEN
	if channelAccessToken == "" {
		return nil, errors.New("LINE_CHANNEL_ACCESS_TOKEN is required")
	}

	// Validate GCP_PROJECT_ID (FR-003)
	if gcpProjectID == "" {
		return nil, errors.New("GCP_PROJECT_ID is required")
	}

	// Parse LLM timeout (NFR-001)
	llmTimeoutSeconds := defaultLLMTimeoutSeconds
	if llmTimeoutEnv := os.Getenv("LLM_TIMEOUT_SECONDS"); llmTimeoutEnv != "" {
		if parsed, err := strconv.Atoi(llmTimeoutEnv); err == nil && parsed > 0 {
			llmTimeoutSeconds = parsed
		}
		// Invalid values fall back to default
	}

	// SC-001: Default PORT to 8080 if empty
	if port == "" {
		port = defaultPort
	}

	// SC-002: Default GCP_REGION to asia-northeast1 if empty
	if gcpRegion == "" {
		gcpRegion = defaultRegion
	}

	return &Config{
		ChannelSecret:      channelSecret,
		ChannelAccessToken: channelAccessToken,
		GCPProjectID:       gcpProjectID,
		LLMTimeoutSeconds:  llmTimeoutSeconds,
		Port:               port,
		GCPRegion:          gcpRegion,
	}, nil
}

// initServer initializes a LINE webhook server using the provided configuration.
// Returns the Server instance or an error if initialization fails.
func initServer(config *Config) (*line.Server, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	return line.NewServer(config.ChannelSecret)
}

// initClient initializes a LINE messaging client using the provided configuration.
// Returns the Client instance or an error if initialization fails.
func initClient(config *Config) (*line.Client, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	return line.NewClient(config.ChannelAccessToken)
}

// initLLM initializes an LLM provider using the provided configuration.
// Returns the LLM provider or an error if initialization fails.
// FR-003: Bot fails to start during initialization if credentials are missing
// SC-003: Pass GCPRegion to NewVertexAIClient as fallback region
func initLLM(ctx context.Context, config *Config) (llm.Provider, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	return llm.NewVertexAIClient(ctx, config.GCPProjectID, config.GCPRegion)
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
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize LINE server (webhook handler)
	server, err := initServer(config)
	if err != nil {
		log.Fatal(err)
	}
	llmTimeout := time.Duration(config.LLMTimeoutSeconds) * time.Second
	server.SetCallbackTimeout(llmTimeout)
	server.SetLogger(logger)

	// Initialize LINE client (message sender)
	client, err := initClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize LLM provider (FR-003)
	llmProvider, err := initLLM(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}

	// Create yuruppu handler and register callback
	yHandler := yuruppu.NewHandler(llmProvider, client, logger)
	server.OnMessage(createMessageCallback(yHandler))

	// Create HTTP handler and start server
	handler := createHandler(server)

	// Log startup message
	log.Printf("Server listening on port %s", config.Port)

	// Start HTTP server
	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		log.Fatal(err)
	}
}
