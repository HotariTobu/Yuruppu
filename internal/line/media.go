package line

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
)

// MaxMediaSize is the maximum size of media content that can be downloaded.
// LINE transcodes video/audio, so actual sizes are smaller than upload limits.
const MaxMediaSize = 100 * 1024 * 1024 // 100MB

// MediaContent represents media content downloaded from LINE.
type MediaContent struct {
	Data     []byte
	MIMEType string
}

// GetMessageContent downloads media content from LINE using a message ID.
// messageID is the LINE message ID for the media content.
// Returns the media content with binary data and MIME type, or an error.
func (c *Client) GetMessageContent(messageID string) (*MediaContent, error) {
	// Validate messageID is not empty
	if messageID == "" {
		return nil, errors.New("messageID cannot be empty")
	}

	c.logger.Debug("downloading media content",
		slog.String("messageID", messageID),
	)

	// Call LINE GetMessageContent API
	httpResp, err := c.blobAPI.GetMessageContent(messageID)
	if httpResp != nil && httpResp.Body != nil {
		defer httpResp.Body.Close()
	}

	// Extract x-line-request-id for debugging (available even on error)
	var requestID string
	if httpResp != nil {
		requestID = httpResp.Header.Get("X-Line-Request-Id")
	}

	if err != nil {
		return nil, fmt.Errorf("LINE API GetMessageContent failed (x-line-request-id=%s): %w", requestID, err)
	}

	// Extract Content-Type header
	mimeType := httpResp.Header.Get("Content-Type")
	if mimeType == "" {
		return nil, fmt.Errorf("Content-Type header missing (x-line-request-id=%s)", requestID)
	}

	// Read body content with size limit to prevent memory exhaustion
	limitedReader := io.LimitReader(httpResp.Body, MaxMediaSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read media content body (x-line-request-id=%s): %w", requestID, err)
	}

	if len(data) > MaxMediaSize {
		return nil, fmt.Errorf("media content exceeds size limit of %d bytes (x-line-request-id=%s)", MaxMediaSize, requestID)
	}

	c.logger.Debug("media content downloaded successfully",
		slog.String("x-line-request-id", requestID),
		slog.Int("dataSize", len(data)),
		slog.String("mimeType", mimeType),
	)

	return &MediaContent{
		Data:     data,
		MIMEType: mimeType,
	}, nil
}
