package main

import (
	"errors"
	"os"
	"strings"
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

func main() {
	// Load configuration
	_, err := loadConfig()
	if err != nil {
		// Error handling will be expanded in later requirements
		panic(err)
	}

	// Server initialization will be implemented in later requirements
}
