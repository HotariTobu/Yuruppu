package profile

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"yuruppu/internal/profile"
)

// PromptNewProfile prompts user to enter profile fields.
// Display name is required (re-prompts if empty).
// Picture URL and status message are optional.
// If picture URL is provided, MIME type is fetched automatically.
//
// FR-005: For new user IDs, CLI prompts for all profile fields (display name required; picture URL, status message optional).
// If picture URL is provided, MIME type is fetched automatically
// AC-004: New user profile creation
// AC-005: Empty display name rejection
func PromptNewProfile(ctx context.Context, stdin io.Reader, stderr io.Writer) (*profile.UserProfile, error) {
	// Check if context is already cancelled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	scanner := bufio.NewScanner(stdin)

	// Prompt for display name (required, re-prompt if empty)
	var displayName string
	for {
		_, _ = fmt.Fprint(stderr, "Enter display name: ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, err
			}
			return nil, io.EOF
		}
		displayName = strings.TrimSpace(scanner.Text())
		if displayName != "" {
			break
		}
		// Re-prompt if empty
	}

	// Prompt for picture URL (optional)
	_, _ = fmt.Fprint(stderr, "Enter picture URL: ")
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	pictureURL := strings.TrimSpace(scanner.Text())

	// Fetch MIME type if picture URL provided
	var pictureMIMEType string
	if pictureURL != "" {
		// Check context before HTTP request
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		mimeType, err := fetchMIMEType(ctx, pictureURL)
		if err != nil {
			return nil, err
		}
		pictureMIMEType = mimeType
	}

	// Prompt for status message (optional)
	_, _ = fmt.Fprint(stderr, "Enter status message: ")
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	statusMessage := strings.TrimSpace(scanner.Text())

	return &profile.UserProfile{
		DisplayName:     displayName,
		PictureURL:      pictureURL,
		PictureMIMEType: pictureMIMEType,
		StatusMessage:   statusMessage,
	}, nil
}

// fetchMIMEType performs HTTP HEAD request to fetch MIME type from URL.
func fetchMIMEType(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
	}

	return resp.Header.Get("Content-Type"), nil
}
