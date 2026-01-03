package bot

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/google/uuid"
)

// sourceIDPattern validates LINE source IDs (user IDs, group IDs, room IDs).
// LINE IDs are alphanumeric strings, typically 33 characters (U/C/R prefix + 32 hex).
// Pattern allows alphanumeric and hyphens but prevents path traversal sequences.
var sourceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

// uploadMedia downloads media content and stores it in storage.
// sourceID is the LINE source identifier (user or group ID).
// messageID is the LINE message ID containing the media.
// Returns the storage key and MIME type of the stored media.
func (h *Handler) uploadMedia(ctx context.Context, sourceID, messageID string) (string, string, error) {
	h.logger.DebugContext(ctx, "uploading media",
		slog.String("sourceID", sourceID),
		slog.String("messageID", messageID),
	)

	// Validate sourceID to prevent path traversal attacks
	if sourceID == "" || !sourceIDPattern.MatchString(sourceID) {
		return "", "", fmt.Errorf("invalid sourceID: %q", sourceID)
	}

	// Download media content from LINE
	data, mimeType, err := h.lineClient.GetMessageContent(messageID)
	if err != nil {
		return "", "", fmt.Errorf("failed to download media content: %w", err)
	}

	// Generate storage key: {sourceID}/{uuidv7}
	id, err := uuid.NewV7()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate UUIDv7: %w", err)
	}
	storageKey := sourceID + "/" + id.String()

	// Write to storage
	_, err = h.mediaStorage.Write(ctx, storageKey, mimeType, data, 0)
	if err != nil {
		return "", "", fmt.Errorf("failed to write media to storage: %w", err)
	}

	h.logger.DebugContext(ctx, "media uploaded successfully",
		slog.String("storageKey", storageKey),
		slog.String("mimeType", mimeType),
		slog.Int("dataSize", len(data)),
	)

	return storageKey, mimeType, nil
}
