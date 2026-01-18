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
	scanner *bufio.Scanner
	writer  io.Writer
}

// NewPrompter creates a new prompter.
func NewPrompter(scanner *bufio.Scanner, w io.Writer) *Prompter {
	if scanner == nil {
		panic("scanner cannot be nil")
	}
	if w == nil {
		panic("writer cannot be nil")
	}
	return &Prompter{
		scanner: scanner,
		writer:  w,
	}
}

// FetchUserProfile prompts the user for profile information.
// Display name is required (re-prompts if empty).
// Picture URL and status message are optional.
func (p *Prompter) FetchUserProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error) {
	// Display name (required)
	var displayName string
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		_, _ = fmt.Fprint(p.writer, "Enter user display name: ")
		if !p.scanner.Scan() {
			if err := p.scanner.Err(); err != nil {
				return nil, err
			}
			return nil, io.EOF
		}
		displayName = strings.TrimSpace(p.scanner.Text())
		if displayName != "" {
			break
		}
	}

	// Picture URL (optional)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprint(p.writer, "Enter user picture URL (optional): ")
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	pictureURL := strings.TrimSpace(p.scanner.Text())

	// Status message (optional)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprint(p.writer, "Enter user status message (optional): ")
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	statusMessage := strings.TrimSpace(p.scanner.Text())

	return &lineclient.UserProfile{
		DisplayName:   displayName,
		PictureURL:    pictureURL,
		StatusMessage: statusMessage,
	}, nil
}

// FetchGroupSummary prompts the user for group information.
func (p *Prompter) FetchGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error) {
	// Group name (required)
	var groupName string
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		_, _ = fmt.Fprint(p.writer, "Enter group display name: ")
		if !p.scanner.Scan() {
			if err := p.scanner.Err(); err != nil {
				return nil, err
			}
			return nil, io.EOF
		}
		groupName = strings.TrimSpace(p.scanner.Text())
		if groupName != "" {
			break
		}
	}

	// Picture URL (optional)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprint(p.writer, "Enter group picture URL (optional): ")
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	pictureURL := strings.TrimSpace(p.scanner.Text())

	return &lineclient.GroupSummary{
		GroupID:    groupID,
		GroupName:  groupName,
		PictureURL: pictureURL,
	}, nil
}
