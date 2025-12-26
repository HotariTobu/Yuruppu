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
	Port               string // Server port (default: 8080)
	ChannelSecret      string
	ChannelAccessToken string
	GCPProjectID       string // Optional: auto-detected on Cloud Run
	GCPRegion          string // Optional: auto-detected on Cloud Run
	LLMTimeoutSeconds  int    // LLM API timeout in seconds (default: 30)
}

const (
	// defaultLLMTimeoutSeconds is the default LLM API timeout in seconds.
	defaultLLMTimeoutSeconds = 30

	// defaultPort is the default server port.
	defaultPort = "8080"
)

// loadConfig loads configuration from environment variables.
// It reads PORT, LINE_CHANNEL_SECRET, LINE_CHANNEL_ACCESS_TOKEN, GCP_PROJECT_ID, GCP_REGION, and LLM_TIMEOUT_SECONDS from environment.
// Returns error if required LINE environment variables are missing or empty after trimming whitespace.
// GCP_PROJECT_ID and GCP_REGION are optional (auto-detected on Cloud Run).
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

	gcpProjectID := strings.TrimSpace(os.Getenv("GCP_PROJECT_ID"))
	gcpRegion := strings.TrimSpace(os.Getenv("GCP_REGION"))

	// Parse LLM timeout
	llmTimeoutSeconds := defaultLLMTimeoutSeconds
	if llmTimeoutEnv := os.Getenv("LLM_TIMEOUT_SECONDS"); llmTimeoutEnv != "" {
		if parsed, err := strconv.Atoi(llmTimeoutEnv); err == nil && parsed > 0 {
			llmTimeoutSeconds = parsed
		}
		// Invalid values fall back to default
	}

	return &Config{
		Port:               port,
		ChannelSecret:      channelSecret,
		ChannelAccessToken: channelAccessToken,
		GCPProjectID:       gcpProjectID,
		GCPRegion:          gcpRegion,
		LLMTimeoutSeconds:  llmTimeoutSeconds,
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

	// Initialize components
	server, err := initServer(config)
	if err != nil {
		log.Fatal(err)
	}

	client, err := initClient(config)
	if err != nil {
		log.Fatal(err)
	}

	llmProvider, err := initLLM(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}

	// Configure server
	llmTimeout := time.Duration(config.LLMTimeoutSeconds) * time.Second
	server.SetCallbackTimeout(llmTimeout)
	server.SetLogger(logger)

	// Create yuruppu handler and register callback
	yHandler := yuruppu.NewHandler(llmProvider, client, logger)
	server.OnMessage(createMessageCallback(yHandler))

	// Start HTTP server
	handler := createHandler(server)
	log.Printf("Server listening on port %s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		log.Fatal(err)
	}
}
