package bot_test

import (
	"errors"
	"log/slog"
	"testing"
	"yuruppu/internal/bot"
	"yuruppu/internal/history"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HandleImage Media Tests
// =============================================================================

func TestHandleImage_UploadMedia(t *testing.T) {
	t.Run("success - downloads and stores image", func(t *testing.T) {
		mockStore := newMockStorage()
		mockDownloader := &mockMediaDownloader{
			data:     []byte("fake-image-data"),
			mimeType: "image/jpeg",
		}
		mockAg := &mockAgent{response: "Nice image!"}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(historyRepo, mockDownloader, mockStore, mockAg, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "msg-456", mockDownloader.lastMessageID)
		// Image upload + 1 history write (user msg only, assistant msg saved by reply tool)
		assert.Equal(t, 2, mockStore.writeCallCount)
		// Verify image was stored by checking first write
		assert.Equal(t, "image/jpeg", mockStore.writes[0].mimeType)
		assert.Equal(t, []byte("fake-image-data"), mockStore.writes[0].data)
	})

	t.Run("success - storage key format is sourceID/uuid", func(t *testing.T) {
		mockStore := newMockStorage()
		mockDownloader := &mockMediaDownloader{
			data:     []byte("image-data"),
			mimeType: "image/png",
		}
		mockAg := &mockAgent{response: "Nice!"}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(historyRepo, mockDownloader, mockStore, mockAg, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "group-789", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		// First write is the image upload
		assert.Contains(t, mockStore.writes[0].key, "group-789/")
	})

	t.Run("fallback - download error uses placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockDownloader := &mockMediaDownloader{
			err: errors.New("LINE API failed"),
		}
		mockAg := &mockAgent{response: "I see!"}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(historyRepo, mockDownloader, mockStore, mockAg, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an image, but an error occurred while loading]", mockAg.lastUserMessageText)
	})

	t.Run("fallback - storage error uses placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockStore.writeResults = []writeResult{{gen: 0, err: errors.New("GCS failed")}}
		mockDownloader := &mockMediaDownloader{
			data:     []byte("image-data"),
			mimeType: "image/jpeg",
		}
		mockAg := &mockAgent{response: "I see!"}
		historyRepo, err := history.NewRepository(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(historyRepo, mockDownloader, mockStore, mockAg, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an image, but an error occurred while loading]", mockAg.lastUserMessageText)
	})
}
