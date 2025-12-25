package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"yuruppu/internal/bot"
	"yuruppu/internal/llm"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	ChannelSecret      string
	ChannelAccessToken string
	GCPProjectID       string
	LLMTimeoutSeconds  int    // NFR-001: LLM API timeout in seconds (default: 30)
	Port               string // SC-001: Server port (default: 8080)
	GCPRegion          string // SC-002: GCP region for Vertex AI (default: us-central1)
}

const (
	// defaultLLMTimeoutSeconds is the default LLM API timeout in seconds.
	// NFR-001: LLM API total request timeout should be configurable (default: 30 seconds)
	defaultLLMTimeoutSeconds = 30

	// defaultPort is the default server port.
	// SC-001: Server reads PORT from environment with 8080 as default.
	defaultPort = "8080"

	// defaultRegion is the default GCP region for Vertex AI API calls.
	// SC-002: GCP_REGION is read from environment with us-central1 as default.
	defaultRegion = "us-central1"
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

	// SC-002: Default GCP_REGION to us-central1 if empty
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

// initBot initializes a Bot instance using the provided configuration.
// Returns the Bot instance or an error if initialization fails.
func initBot(config *Config) (*bot.Bot, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	return bot.NewBot(config.ChannelSecret, config.ChannelAccessToken)
}

// initLLM initializes an LLM provider using the provided configuration.
// Returns the LLM provider or an error if initialization fails.
// FR-003: Bot fails to start during initialization if credentials are missing
func initLLM(ctx context.Context, config *Config) (llm.Provider, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	return llm.NewVertexAIClient(ctx, config.GCPProjectID)
}

// stdLogger implements bot.Logger interface using standard log package.
type stdLogger struct{}

func (l *stdLogger) Info(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

func (l *stdLogger) Debug(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

func (l *stdLogger) Warn(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

func (l *stdLogger) Error(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// setupPackageLevel sets up package-level Bot, Logger, LLM provider, and timeout instances.
// AC-007: bot.SetDefaultBot() and bot.SetLogger() are called.
// FR-003: LLM provider is set during initialization.
// NFR-001: LLM timeout is set during initialization.
func setupPackageLevel(b *bot.Bot, llmProvider llm.Provider, llmTimeout time.Duration) {
	bot.SetDefaultBot(b)
	bot.SetLogger(&stdLogger{})
	bot.SetDefaultLLMProvider(llmProvider)
	bot.SetLLMTimeout(llmTimeout)
}

// getPort returns the port to listen on from the PORT environment variable.
// If PORT is not set or empty, returns the default port "8080".
// AC-005, AC-006: Server reads PORT from environment with 8080 as default.
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8080"
	}
	return port
}

// createHandler creates and returns an http.Handler with registered routes.
// AC-004: /webhook endpoint is registered with bot.HandleWebhook.
func createHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", bot.HandleWebhook)
	return mux
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize bot
	b, err := initBot(config)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize LLM provider (FR-003)
	llmProvider, err := initLLM(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}

	// Setup package-level bot, logger, LLM provider, and timeout (NFR-001)
	llmTimeout := time.Duration(config.LLMTimeoutSeconds) * time.Second
	setupPackageLevel(b, llmProvider, llmTimeout)

	// Create HTTP handler and start server
	handler := createHandler()
	port := getPort()

	// AC-004: Log startup message
	log.Printf("Server listening on port %s", port)

	// Start HTTP server
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
