package prompter_test

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"
	"yuruppu/cmd/cli/prompter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrompter(t *testing.T) {
	t.Run("should create prompter with valid scanner and writer", func(t *testing.T) {
		// Given
		scanner := bufio.NewScanner(strings.NewReader(""))
		var writer bytes.Buffer

		// When
		p := prompter.NewPrompter(scanner, &writer)

		// Then
		require.NotNil(t, p)
	})

	t.Run("should panic when scanner is nil", func(t *testing.T) {
		// Given
		var writer bytes.Buffer

		// When/Then
		assert.Panics(t, func() {
			prompter.NewPrompter(nil, &writer)
		})
	})

	t.Run("should panic when writer is nil", func(t *testing.T) {
		// Given
		scanner := bufio.NewScanner(strings.NewReader(""))

		// When/Then
		assert.Panics(t, func() {
			prompter.NewPrompter(scanner, nil)
		})
	})
}

func TestPrompter_FetchUserProfile(t *testing.T) {
	t.Run("should prompt for profile and return it", func(t *testing.T) {
		// Given
		input := "Test User\nhttps://example.com/pic.jpg\nHello world\n"
		scanner := bufio.NewScanner(strings.NewReader(input))
		var writer bytes.Buffer
		p := prompter.NewPrompter(scanner, &writer)
		ctx := context.Background()

		// When
		profile, err := p.FetchUserProfile(ctx, "user123")

		// Then
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, "Test User", profile.DisplayName)
		assert.Equal(t, "https://example.com/pic.jpg", profile.PictureURL)
		assert.Equal(t, "Hello world", profile.StatusMessage)
	})

	t.Run("should re-prompt if display name is empty", func(t *testing.T) {
		// Given
		input := "\n\nValid Name\n\n\n"
		scanner := bufio.NewScanner(strings.NewReader(input))
		var writer bytes.Buffer
		p := prompter.NewPrompter(scanner, &writer)
		ctx := context.Background()

		// When
		profile, err := p.FetchUserProfile(ctx, "user123")

		// Then
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, "Valid Name", profile.DisplayName)
	})

	t.Run("should allow empty optional fields", func(t *testing.T) {
		// Given
		input := "Test User\n\n\n"
		scanner := bufio.NewScanner(strings.NewReader(input))
		var writer bytes.Buffer
		p := prompter.NewPrompter(scanner, &writer)
		ctx := context.Background()

		// When
		profile, err := p.FetchUserProfile(ctx, "user123")

		// Then
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, "Test User", profile.DisplayName)
		assert.Empty(t, profile.PictureURL)
		assert.Empty(t, profile.StatusMessage)
	})

	t.Run("should return EOF if input ends early", func(t *testing.T) {
		// Given
		input := "Test User\n"
		scanner := bufio.NewScanner(strings.NewReader(input))
		var writer bytes.Buffer
		p := prompter.NewPrompter(scanner, &writer)
		ctx := context.Background()

		// When
		profile, err := p.FetchUserProfile(ctx, "user123")

		// Then
		require.Error(t, err)
		assert.Nil(t, profile)
	})

	t.Run("should display prompts to writer", func(t *testing.T) {
		// Given
		input := "Test User\n\n\n"
		scanner := bufio.NewScanner(strings.NewReader(input))
		var writer bytes.Buffer
		p := prompter.NewPrompter(scanner, &writer)
		ctx := context.Background()

		// When
		_, _ = p.FetchUserProfile(ctx, "user123")

		// Then
		output := writer.String()
		assert.Contains(t, output, "Enter user display name:")
		assert.Contains(t, output, "Enter user picture URL")
		assert.Contains(t, output, "Enter user status message")
	})

	t.Run("should trim whitespace from input", func(t *testing.T) {
		// Given
		input := "  Test User  \n  https://example.com  \n  Hello  \n"
		scanner := bufio.NewScanner(strings.NewReader(input))
		var writer bytes.Buffer
		p := prompter.NewPrompter(scanner, &writer)
		ctx := context.Background()

		// When
		profile, err := p.FetchUserProfile(ctx, "user123")

		// Then
		require.NoError(t, err)
		assert.Equal(t, "Test User", profile.DisplayName)
		assert.Equal(t, "https://example.com", profile.PictureURL)
		assert.Equal(t, "Hello", profile.StatusMessage)
	})

	t.Run("should return error when context is cancelled", func(t *testing.T) {
		// Given
		input := "Test User\n\n\n"
		scanner := bufio.NewScanner(strings.NewReader(input))
		var writer bytes.Buffer
		p := prompter.NewPrompter(scanner, &writer)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// When
		profile, err := p.FetchUserProfile(ctx, "user123")

		// Then
		require.Error(t, err)
		assert.Nil(t, profile)
		assert.Equal(t, context.Canceled, err)
	})
}
