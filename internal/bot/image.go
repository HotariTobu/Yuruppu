package bot

import (
	"context"
	"fmt"
	"log/slog"
	"yuruppu/internal/history"
	"yuruppu/internal/line"

	"github.com/google/uuid"
)

// MediaDownloader downloads media content from LINE.
type MediaDownloader interface {
	GetMessageContent(messageID string) (*line.MediaContent, error)
}

// processImage downloads and stores an image, returning a UserFileDataPart on success.
// On any error, logs a warning and returns nil to trigger placeholder fallback.
func (h *Handler) processImage(ctx context.Context, sourceID, messageID string) *history.UserFileDataPart {
	// Download from LINE
	media, err := h.mediaDownloader.GetMessageContent(messageID)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to download image, using placeholder",
			slog.String("sourceID", sourceID),
			slog.String("messageID", messageID),
			slog.String("error", err.Error()),
		)
		return nil
	}

	// Generate storage key: media/{sourceID}/{uuidv7}
	id, err := uuid.NewV7()
	if err != nil {
		h.logger.WarnContext(ctx, "failed to generate UUID, using placeholder",
			slog.String("sourceID", sourceID),
			slog.String("messageID", messageID),
			slog.String("error", err.Error()),
		)
		return nil
	}
	storageKey := fmt.Sprintf("media/%s/%s", sourceID, id.String())

	// Store to GCS (expectedGeneration=0 means create new, fail if exists)
	_, err = h.mediaStorage.Write(ctx, storageKey, media.MIMEType, media.Data, 0)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to store image, using placeholder",
			slog.String("sourceID", sourceID),
			slog.String("storageKey", storageKey),
			slog.String("messageID", messageID),
			slog.String("error", err.Error()),
		)
		return nil
	}

	h.logger.DebugContext(ctx, "image processed successfully",
		slog.String("storageKey", storageKey),
		slog.String("mimeType", media.MIMEType),
		slog.Int("size", len(media.Data)),
	)

	return &history.UserFileDataPart{
		StorageKey:  storageKey,
		MIMEType:    media.MIMEType,
		DisplayName: "image",
	}
}
