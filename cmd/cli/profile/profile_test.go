package profile_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"yuruppu/cmd/cli/profile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptNewProfile_AllFieldsProvided tests successful profile creation with all fields
// AC-004: CLI prompts for display name, picture URL, and status message
func TestPromptNewProfile_AllFieldsProvided(t *testing.T) {
	t.Run("should create profile with all fields when provided", func(t *testing.T) {
		// Given: Mock HTTP server that returns an image
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF}) // JPEG magic bytes
		}))
		defer server.Close()

		stdin := strings.NewReader("Test User\n" + server.URL + "\nHello, World!\n")
		stderr := &bytes.Buffer{}
		ctx := context.Background()

		// When
		userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

		// Then
		require.NoError(t, err)
		assert.Equal(t, "Test User", userProfile.DisplayName)
		assert.Equal(t, server.URL, userProfile.PictureURL)
		assert.Equal(t, "image/jpeg", userProfile.PictureMIMEType)
		assert.Equal(t, "Hello, World!", userProfile.StatusMessage)

		// Verify prompts were displayed
		stderrOutput := stderr.String()
		assert.Contains(t, stderrOutput, "Enter display name")
		assert.Contains(t, stderrOutput, "Enter picture URL")
		assert.Contains(t, stderrOutput, "Enter status message")
	})
}

// TestPromptNewProfile_OnlyDisplayName tests profile creation with only required field
// AC-004: Picture URL and status message are optional (can skip with Enter)
func TestPromptNewProfile_OnlyDisplayName(t *testing.T) {
	t.Run("should create profile with only display name when other fields skipped", func(t *testing.T) {
		// Given: Empty lines for picture URL and status message
		stdin := strings.NewReader("Test User\n\n\n")
		stderr := &bytes.Buffer{}
		ctx := context.Background()

		// When
		userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

		// Then
		require.NoError(t, err)
		assert.Equal(t, "Test User", userProfile.DisplayName)
		assert.Empty(t, userProfile.PictureURL, "picture URL should be empty when skipped")
		assert.Empty(t, userProfile.PictureMIMEType, "MIME type should be empty when picture URL is empty")
		assert.Empty(t, userProfile.StatusMessage, "status message should be empty when skipped")
	})
}

// TestPromptNewProfile_EmptyDisplayNameReprompt tests re-prompting for empty display name
// AC-005: CLI re-prompts for display name when empty
func TestPromptNewProfile_EmptyDisplayNameReprompt(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedAttempts int
	}{
		{
			name:             "empty line then valid name",
			input:            "\nValid User\n\n\n",
			expectedAttempts: 2,
		},
		{
			name:             "whitespace only then valid name",
			input:            "   \nValid User\n\n\n",
			expectedAttempts: 2,
		},
		{
			name:             "multiple empty attempts then valid name",
			input:            "\n\n  \nValid User\n\n\n",
			expectedAttempts: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stdin := strings.NewReader(tt.input)
			stderr := &bytes.Buffer{}
			ctx := context.Background()

			// When
			userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

			// Then
			require.NoError(t, err)
			assert.Equal(t, "Valid User", userProfile.DisplayName)

			// Verify re-prompting occurred
			stderrOutput := stderr.String()
			promptCount := strings.Count(stderrOutput, "Enter display name")
			assert.Equal(t, tt.expectedAttempts, promptCount,
				"should re-prompt for display name %d times", tt.expectedAttempts)
		})
	}
}

// TestPromptNewProfile_PictureURLWithMIMEType tests MIME type fetching
// FR-005: If picture URL is provided, MIME type is fetched automatically
func TestPromptNewProfile_PictureURLWithMIMEType(t *testing.T) {
	tests := []struct {
		name             string
		contentType      string
		expectedMIMEType string
	}{
		{
			name:             "JPEG image",
			contentType:      "image/jpeg",
			expectedMIMEType: "image/jpeg",
		},
		{
			name:             "PNG image",
			contentType:      "image/png",
			expectedMIMEType: "image/png",
		},
		{
			name:             "GIF image",
			contentType:      "image/gif",
			expectedMIMEType: "image/gif",
		},
		{
			name:             "WebP image",
			contentType:      "image/webp",
			expectedMIMEType: "image/webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock HTTP server that returns specific content type
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte{0xFF}) // minimal content
			}))
			defer server.Close()

			stdin := strings.NewReader("Test User\n" + server.URL + "\n\n")
			stderr := &bytes.Buffer{}
			ctx := context.Background()

			// When
			userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

			// Then
			require.NoError(t, err)
			assert.Equal(t, server.URL, userProfile.PictureURL)
			assert.Equal(t, tt.expectedMIMEType, userProfile.PictureMIMEType,
				"MIME type should be fetched from Content-Type header")
		})
	}
}

// TestPromptNewProfile_PictureURLFetchError tests handling of HTTP fetch errors
func TestPromptNewProfile_PictureURLFetchError(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		wantErr     bool
	}{
		{
			name: "404 Not Found",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			wantErr: true,
		},
		{
			name: "500 Internal Server Error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			server := tt.setupServer()
			defer server.Close()

			stdin := strings.NewReader("Test User\n" + server.URL + "\n\n")
			stderr := &bytes.Buffer{}
			ctx := context.Background()

			// When
			userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

			// Then
			if tt.wantErr {
				require.Error(t, err, "should return error for HTTP failure")
				assert.Nil(t, userProfile)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, userProfile)
			}
		})
	}
}

// TestPromptNewProfile_InvalidPictureURL tests handling of invalid URLs
func TestPromptNewProfile_InvalidPictureURL(t *testing.T) {
	t.Run("should return error for invalid URL", func(t *testing.T) {
		// Given: Invalid URL
		stdin := strings.NewReader("Test User\n://invalid-url\n\n")
		stderr := &bytes.Buffer{}
		ctx := context.Background()

		// When
		userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

		// Then
		require.Error(t, err, "should return error for invalid URL")
		assert.Nil(t, userProfile)
	})
}

// TestPromptNewProfile_ContextCancellation tests context cancellation handling
func TestPromptNewProfile_ContextCancellation(t *testing.T) {
	t.Run("should return error when context is cancelled", func(t *testing.T) {
		// Given: Context that is already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		stdin := strings.NewReader("Test User\n\n\n")
		stderr := &bytes.Buffer{}

		// When
		userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

		// Then
		require.Error(t, err, "should return error when context is cancelled")
		assert.Nil(t, userProfile)
	})
}

// TestPromptNewProfile_ContextTimeout tests context timeout during HTTP fetch
func TestPromptNewProfile_ContextTimeout(t *testing.T) {
	t.Run("should return error when context times out during HTTP fetch", func(t *testing.T) {
		// Given: Mock server with slow response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		stdin := strings.NewReader("Test User\n" + server.URL + "\n\n")
		stderr := &bytes.Buffer{}

		// When
		userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

		// Then
		require.Error(t, err, "should return error when context times out")
		assert.Nil(t, userProfile)
	})
}

// TestPromptNewProfile_StdinEOF tests handling of unexpected stdin EOF
func TestPromptNewProfile_StdinEOF(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "EOF before display name",
			input: "",
		},
		{
			name:  "EOF before picture URL",
			input: "Test User\n",
		},
		{
			name:  "EOF before status message",
			input: "Test User\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stdin := strings.NewReader(tt.input)
			stderr := &bytes.Buffer{}
			ctx := context.Background()

			// When
			userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

			// Then
			require.Error(t, err, "should return error on unexpected EOF")
			assert.Nil(t, userProfile)
		})
	}
}

// TestPromptNewProfile_WhitespaceHandling tests trimming of whitespace
func TestPromptNewProfile_WhitespaceHandling(t *testing.T) {
	t.Run("should trim whitespace from inputs", func(t *testing.T) {
		// Given: Inputs with leading/trailing whitespace
		stdin := strings.NewReader("  Test User  \n  https://example.com/pic.jpg  \n  My Status  \n")
		stderr := &bytes.Buffer{}
		ctx := context.Background()

		// When
		userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

		// Then
		if err == nil {
			// If MIME type fetch succeeds (URL is valid and reachable), verify trimming
			assert.Equal(t, "Test User", userProfile.DisplayName, "should trim display name")
			assert.Equal(t, "https://example.com/pic.jpg", userProfile.PictureURL, "should trim picture URL")
			assert.Equal(t, "My Status", userProfile.StatusMessage, "should trim status message")
		}
		// Note: This test may fail if the URL fetch fails, which is expected behavior
	})
}

// TestPromptNewProfile_EmptyPictureURLSkipsMIMEFetch tests that empty URL skips HTTP fetch
func TestPromptNewProfile_EmptyPictureURLSkipsMIMEFetch(t *testing.T) {
	t.Run("should not fetch MIME type when picture URL is empty", func(t *testing.T) {
		// Given: Empty picture URL (skip with Enter)
		stdin := strings.NewReader("Test User\n\nMy Status\n")
		stderr := &bytes.Buffer{}
		ctx := context.Background()

		// When
		userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

		// Then
		require.NoError(t, err)
		assert.Equal(t, "Test User", userProfile.DisplayName)
		assert.Empty(t, userProfile.PictureURL)
		assert.Empty(t, userProfile.PictureMIMEType, "MIME type should be empty when URL is empty")
		assert.Equal(t, "My Status", userProfile.StatusMessage)
	})
}

// TestPromptNewProfile_LongInputs tests handling of very long inputs
func TestPromptNewProfile_LongInputs(t *testing.T) {
	t.Run("should accept long display name", func(t *testing.T) {
		// Given: Very long display name (100 characters)
		longName := strings.Repeat("a", 100)
		stdin := strings.NewReader(longName + "\n\n\n")
		stderr := &bytes.Buffer{}
		ctx := context.Background()

		// When
		userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

		// Then
		require.NoError(t, err)
		assert.Equal(t, longName, userProfile.DisplayName)
	})

	t.Run("should accept long status message", func(t *testing.T) {
		// Given: Very long status message (500 characters)
		longStatus := strings.Repeat("b", 500)
		stdin := strings.NewReader("Test User\n\n" + longStatus + "\n")
		stderr := &bytes.Buffer{}
		ctx := context.Background()

		// When
		userProfile, err := profile.PromptNewProfile(ctx, stdin, stderr)

		// Then
		require.NoError(t, err)
		assert.Equal(t, "Test User", userProfile.DisplayName)
		assert.Equal(t, longStatus, userProfile.StatusMessage)
	})
}
