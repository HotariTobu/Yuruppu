package main

import (
	"errors"
	"log"
	"os"
	"strings"

	"yuruppu/internal/bot"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	ChannelSecret      string
	ChannelAccessToken string
}

// loadConfig loads configuration from environment variables.
// It reads LINE_CHANNEL_SECRET and LINE_CHANNEL_ACCESS_TOKEN from environment.
// Returns error if either required environment variable is missing or empty after trimming whitespace.
func loadConfig() (*Config, error) {
	// Load and trim environment variables
	channelSecret := strings.TrimSpace(os.Getenv("LINE_CHANNEL_SECRET"))
	channelAccessToken := strings.TrimSpace(os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"))

	// Validate LINE_CHANNEL_SECRET first
	if channelSecret == "" {
		return nil, errors.New("LINE_CHANNEL_SECRET is required")
	}

	// Validate LINE_CHANNEL_ACCESS_TOKEN
	if channelAccessToken == "" {
		return nil, errors.New("LINE_CHANNEL_ACCESS_TOKEN is required")
	}

	return &Config{
		ChannelSecret:      channelSecret,
		ChannelAccessToken: channelAccessToken,
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

// setupPackageLevel sets up package-level Bot and Logger instances.
// AC-007: bot.SetDefaultBot() and bot.SetLogger() are called.
func setupPackageLevel(b *bot.Bot) {
	bot.SetDefaultBot(b)
	bot.SetLogger(&stdLogger{})
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		// Error handling will be expanded in later requirements
		panic(err)
	}

	// Initialize bot
	b, err := initBot(config)
	if err != nil {
		// Error handling will be expanded in later requirements
		panic(err)
	}

	// Setup package-level bot and logger
	setupPackageLevel(b)

	// Server initialization will be implemented in later requirements
}
