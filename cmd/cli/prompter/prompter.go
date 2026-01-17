package prompter

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	lineclient "yuruppu/internal/line/client"
)

// Prompter prompts for profile information via stdin.
// Implements mock.Fetcher interface.
type Prompter struct {
	reader io.Reader
	writer io.Writer
}

// NewPrompter creates a new prompter.
func NewPrompter(r io.Reader, w io.Writer) *Prompter {
	if r == nil {
		panic("reader cannot be nil")
	}
	if w == nil {
		panic("writer cannot be nil")
	}
	return &Prompter{
		reader: r,
		writer: w,
	}
}

// FetchUserProfile prompts the user for profile information.
// Display name is required (re-prompts if empty).
// Picture URL and status message are optional.
func (p *Prompter) FetchUserProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error) {
	scanner := bufio.NewScanner(p.reader)

	// Display name (required)
	var displayName string
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		_, _ = fmt.Fprint(p.writer, "Enter display name: ")
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
	}

	// Picture URL (optional)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprint(p.writer, "Enter picture URL (optional): ")
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	pictureURL := strings.TrimSpace(scanner.Text())

	// Status message (optional)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprint(p.writer, "Enter status message (optional): ")
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	statusMessage := strings.TrimSpace(scanner.Text())

	return &lineclient.UserProfile{
		DisplayName:   displayName,
		PictureURL:    pictureURL,
		StatusMessage: statusMessage,
	}, nil
}

// FetchGroupSummary prompts the user for group information.
func (p *Prompter) FetchGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error) {
	scanner := bufio.NewScanner(p.reader)

	// Group name (required)
	var groupName string
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		_, _ = fmt.Fprint(p.writer, "Enter group name: ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, err
			}
			return nil, io.EOF
		}
		groupName = strings.TrimSpace(scanner.Text())
		if groupName != "" {
			break
		}
	}

	// Picture URL (optional)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprint(p.writer, "Enter picture URL (optional): ")
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	pictureURL := strings.TrimSpace(scanner.Text())

	return &lineclient.GroupSummary{
		GroupID:    groupID,
		GroupName:  groupName,
		PictureURL: pictureURL,
	}, nil
}
