package line

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// MediaContent represents downloaded media content from LINE.
type MediaContent struct {
	Data     []byte
	MIMEType string
}

// GetMessageContent downloads media content from LINE using a message ID.
// Returns the binary content and MIME type, or an error if download fails.
func (c *Client) GetMessageContent(ctx context.Context, messageID string) (*MediaContent, error) {
	c.logger.Debug("downloading media content",
		slog.String("messageID", messageID),
	)

	// Create blob API client for content operations
	blobAPI, err := messaging_api.NewMessagingApiBlobAPI(c.channelToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob API client: %w", err)
	}

	// Get message content from LINE
	httpResp, err := blobAPI.GetMessageContent(messageID)
	if httpResp != nil && httpResp.Body != nil {
		defer httpResp.Body.Close()
	}

	// Extract x-line-request-id for debugging (available even on error)
	var requestID string
	if httpResp != nil {
		requestID = httpResp.Header.Get("X-Line-Request-Id")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get message content (messageID=%s, x-line-request-id=%s): %w",
			messageID, requestID, err)
	}

	// Extract MIME type from Content-Type header
	mimeType := httpResp.Header.Get("Content-Type")
	if mimeType == "" {
		return nil, fmt.Errorf("missing Content-Type header (messageID=%s, x-line-request-id=%s)",
			messageID, requestID)
	}

	// Read binary content
	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body (messageID=%s, x-line-request-id=%s): %w",
			messageID, requestID, err)
	}

	c.logger.Debug("media content downloaded",
		slog.String("messageID", messageID),
		slog.String("mimeType", mimeType),
		slog.Int("size", len(data)),
		slog.String("x-line-request-id", requestID),
	)

	return &MediaContent{
		Data:     data,
		MIMEType: mimeType,
	}, nil
}
