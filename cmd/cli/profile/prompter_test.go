package profile_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"yuruppu/cmd/cli/profile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrompter(t *testing.T) {
	t.Run("should create prompter with valid reader and writer", func(t *testing.T) {
		// Given
		reader := strings.NewReader("")
		var writer bytes.Buffer

		// When
		prompter := profile.NewPrompter(reader, &writer)

		// Then
		require.NotNil(t, prompter)
	})

	t.Run("should panic when reader is nil", func(t *testing.T) {
		// Given
		var writer bytes.Buffer

		// When/Then
		assert.Panics(t, func() {
			profile.NewPrompter(nil, &writer)
		})
	})

	t.Run("should panic when writer is nil", func(t *testing.T) {
		// Given
		reader := strings.NewReader("")

		// When/Then
		assert.Panics(t, func() {
			profile.NewPrompter(reader, nil)
		})
	})
}

func TestPrompter_FetchProfile(t *testing.T) {
	t.Run("should prompt for profile and return it", func(t *testing.T) {
		// Given
		input := "Test User\nhttps://example.com/pic.jpg\nHello world\n"
		reader := strings.NewReader(input)
		var writer bytes.Buffer
		prompter := profile.NewPrompter(reader, &writer)
		ctx := context.Background()

		// When
		p, err := prompter.FetchProfile(ctx, "user123")

		// Then
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, "Test User", p.DisplayName)
		assert.Equal(t, "https://example.com/pic.jpg", p.PictureURL)
		assert.Equal(t, "Hello world", p.StatusMessage)
	})

	t.Run("should re-prompt if display name is empty", func(t *testing.T) {
		// Given
		input := "\n\nValid Name\n\n\n"
		reader := strings.NewReader(input)
		var writer bytes.Buffer
		prompter := profile.NewPrompter(reader, &writer)
		ctx := context.Background()

		// When
		p, err := prompter.FetchProfile(ctx, "user123")

		// Then
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, "Valid Name", p.DisplayName)
	})

	t.Run("should allow empty optional fields", func(t *testing.T) {
		// Given
		input := "Test User\n\n\n"
		reader := strings.NewReader(input)
		var writer bytes.Buffer
		prompter := profile.NewPrompter(reader, &writer)
		ctx := context.Background()

		// When
		p, err := prompter.FetchProfile(ctx, "user123")

		// Then
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, "Test User", p.DisplayName)
		assert.Empty(t, p.PictureURL)
		assert.Empty(t, p.StatusMessage)
	})

	t.Run("should return EOF if input ends early", func(t *testing.T) {
		// Given
		input := "Test User\n"
		reader := strings.NewReader(input)
		var writer bytes.Buffer
		prompter := profile.NewPrompter(reader, &writer)
		ctx := context.Background()

		// When
		p, err := prompter.FetchProfile(ctx, "user123")

		// Then
		require.Error(t, err)
		assert.Nil(t, p)
	})

	t.Run("should display prompts to writer", func(t *testing.T) {
		// Given
		input := "Test User\n\n\n"
		reader := strings.NewReader(input)
		var writer bytes.Buffer
		prompter := profile.NewPrompter(reader, &writer)
		ctx := context.Background()

		// When
		_, _ = prompter.FetchProfile(ctx, "user123")

		// Then
		output := writer.String()
		assert.Contains(t, output, "Enter display name:")
		assert.Contains(t, output, "Enter picture URL")
		assert.Contains(t, output, "Enter status message")
	})

	t.Run("should trim whitespace from input", func(t *testing.T) {
		// Given
		input := "  Test User  \n  https://example.com  \n  Hello  \n"
		reader := strings.NewReader(input)
		var writer bytes.Buffer
		prompter := profile.NewPrompter(reader, &writer)
		ctx := context.Background()

		// When
		p, err := prompter.FetchProfile(ctx, "user123")

		// Then
		require.NoError(t, err)
		assert.Equal(t, "Test User", p.DisplayName)
		assert.Equal(t, "https://example.com", p.PictureURL)
		assert.Equal(t, "Hello", p.StatusMessage)
	})

	t.Run("should return error when context is cancelled", func(t *testing.T) {
		// Given
		input := "Test User\n\n\n"
		reader := strings.NewReader(input)
		var writer bytes.Buffer
		prompter := profile.NewPrompter(reader, &writer)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// When
		p, err := prompter.FetchProfile(ctx, "user123")

		// Then
		require.Error(t, err)
		assert.Nil(t, p)
		assert.Equal(t, context.Canceled, err)
	})
}
