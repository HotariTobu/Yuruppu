package bot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
	"yuruppu/internal/storage"

	"golang.org/x/sync/errgroup"
)

const signedURLTTL = 60 * time.Second

// HistoryRepository provides access to conversation history.
type HistoryRepository interface {
	GetHistory(ctx context.Context, sourceID string) ([]history.Message, int64, error)
	PutHistory(ctx context.Context, sourceID string, messages []history.Message, expectedGeneration int64) (int64, error)
}

// Handler implements the server.Handler interface for handling LINE messages.
type Handler struct {
	history         HistoryRepository
	mediaDownloader MediaDownloader
	mediaStorage    storage.Storage
	agent           agent.Agent
	logger          *slog.Logger
}

// NewHandler creates a new Handler with the given dependencies.
// Returns error if any dependency is nil.
func NewHandler(historyRepo HistoryRepository, mediaDownloader MediaDownloader, mediaStor storage.Storage, agent agent.Agent, logger *slog.Logger) (*Handler, error) {
	if historyRepo == nil {
		return nil, fmt.Errorf("historyRepo is required")
	}
	if mediaDownloader == nil {
		return nil, fmt.Errorf("mediaDownloader is required")
	}
	if mediaStor == nil {
		return nil, fmt.Errorf("mediaStorage is required")
	}
	if agent == nil {
		return nil, fmt.Errorf("agent is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &Handler{
		history:         historyRepo,
		mediaDownloader: mediaDownloader,
		mediaStorage:    mediaStor,
		agent:           agent,
		logger:          logger,
	}, nil
}

func (h *Handler) HandleText(ctx context.Context, text string) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("userID not found in context")
	}
	userMsg := &history.UserMessage{
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: text}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleImage(ctx context.Context, messageID string) error {
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("sourceID not found in context")
	}
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("userID not found in context")
	}
	var parts []history.UserPart

	storageKey, mimeType, err := h.uploadMedia(ctx, sourceID, messageID)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to upload image, using placeholder",
			slog.String("messageID", messageID),
			slog.Any("error", err),
		)
		parts = []history.UserPart{&history.UserTextPart{Text: "[User sent an image, but an error occurred while loading]"}}
	} else {
		parts = []history.UserPart{&history.UserFileDataPart{StorageKey: storageKey, MIMEType: mimeType}}
	}

	userMsg := &history.UserMessage{
		UserID:    userID,
		Parts:     parts,
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleSticker(ctx context.Context, packageID, stickerID string) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("userID not found in context")
	}
	userMsg := &history.UserMessage{
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: "[User sent a sticker]"}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleVideo(ctx context.Context, messageID string) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("userID not found in context")
	}
	userMsg := &history.UserMessage{
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: "[User sent a video]"}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleAudio(ctx context.Context, messageID string) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("userID not found in context")
	}
	userMsg := &history.UserMessage{
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: "[User sent an audio]"}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleLocation(ctx context.Context, latitude, longitude float64) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("userID not found in context")
	}
	userMsg := &history.UserMessage{
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: "[User sent a location]"}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleUnknown(ctx context.Context) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("userID not found in context")
	}
	userMsg := &history.UserMessage{
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: "[User sent a message]"}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) handleMessage(ctx context.Context, userMsg *history.UserMessage) error {
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("sourceID not found in context")
	}

	// Step 1: Load history
	hist, gen, err := h.history.GetHistory(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	// Step 2: Save user message to history
	hist = append(hist, userMsg)
	_, err = h.history.PutHistory(ctx, sourceID, hist, gen)
	if err != nil {
		return fmt.Errorf("failed to save user message to history: %w", err)
	}

	// Step 3: Convert history to agent format and generate response
	agentHistory, err := h.convertToAgentHistory(ctx, hist)
	if err != nil {
		return fmt.Errorf("failed to convert history: %w", err)
	}

	response, err := h.agent.Generate(ctx, agentHistory)
	if err != nil {
		return fmt.Errorf("failed to generate response: %w", err)
	}

	// Step 4: Log response contents for debugging
	h.logger.DebugContext(ctx, "agent response",
		slog.String("sourceID", sourceID),
		slog.Any("response", response),
	)

	return nil
}

// convertToAgentHistory converts history.Message slice to agent.Message slice.
// Fetches signed URLs in parallel for all file parts.
func (h *Handler) convertToAgentHistory(ctx context.Context, hist []history.Message) ([]agent.Message, error) {
	result := make([]agent.Message, 0, len(hist))
	pending := make(map[string]agent.FileDataPart)

	for _, msg := range hist {
		switch m := msg.(type) {
		case *history.UserMessage:
			agentMsg, p := convertUserMessage(m)
			for k, v := range p {
				pending[k] = v
			}
			result = append(result, agentMsg)
		case *history.AssistantMessage:
			agentMsg, p := convertAssistantMessage(m)
			for k, v := range p {
				pending[k] = v
			}
			result = append(result, agentMsg)
		}
	}

	if len(pending) > 0 {
		urls, err := h.batchGetSignedURLs(ctx, pending)
		if err != nil {
			return nil, fmt.Errorf("failed to get signed URLs for history: %w", err)
		}
		for k, part := range pending {
			part.SetFileURI(urls[k])
		}
	}

	return result, nil
}

// convertUserMessage converts history.UserMessage to agent.UserMessage.
// Returns pending file parts that need FileURI to be filled.
func convertUserMessage(m *history.UserMessage) (*agent.UserMessage, map[string]agent.FileDataPart) {
	parts := make([]agent.UserPart, 0, len(m.Parts))
	pending := make(map[string]agent.FileDataPart)

	for _, p := range m.Parts {
		switch v := p.(type) {
		case *history.UserTextPart:
			parts = append(parts, &agent.UserTextPart{Text: v.Text})
		case *history.UserFileDataPart:
			var videoMeta *agent.VideoMetadata
			if v.VideoMetadata != nil {
				videoMeta = &agent.VideoMetadata{
					StartOffset: v.VideoMetadata.StartOffset,
					EndOffset:   v.VideoMetadata.EndOffset,
					FPS:         v.VideoMetadata.FPS,
				}
			}
			filePart := &agent.UserFileDataPart{
				MIMEType:      v.MIMEType,
				DisplayName:   v.DisplayName,
				VideoMetadata: videoMeta,
			}
			pending[v.StorageKey] = filePart
			parts = append(parts, filePart)
		}
	}

	return &agent.UserMessage{
		UserName:  m.UserID,
		Parts:     parts,
		LocalTime: m.Timestamp.Format(time.RFC3339),
	}, pending
}

// convertAssistantMessage converts history.AssistantMessage to agent.AssistantMessage.
// Returns pending file parts that need FileURI to be filled.
func convertAssistantMessage(m *history.AssistantMessage) (*agent.AssistantMessage, map[string]agent.FileDataPart) {
	parts := make([]agent.AssistantPart, 0, len(m.Parts))
	pending := make(map[string]agent.FileDataPart)

	for _, p := range m.Parts {
		switch v := p.(type) {
		case *history.AssistantTextPart:
			parts = append(parts, &agent.AssistantTextPart{
				Text:             v.Text,
				Thought:          v.Thought,
				ThoughtSignature: v.ThoughtSignature,
			})
		case *history.AssistantFileDataPart:
			filePart := &agent.AssistantFileDataPart{
				MIMEType:    v.MIMEType,
				DisplayName: v.DisplayName,
			}
			pending[v.StorageKey] = filePart
			parts = append(parts, filePart)
		}
	}

	return &agent.AssistantMessage{
		ModelName: m.ModelName,
		Parts:     parts,
		LocalTime: m.Timestamp.Format(time.RFC3339),
	}, pending
}

// batchGetSignedURLs fetches signed URLs for multiple storage keys in parallel.
func (h *Handler) batchGetSignedURLs(ctx context.Context, pending map[string]agent.FileDataPart) (map[string]string, error) {
	if len(pending) == 0 {
		return make(map[string]string), nil
	}

	var (
		mu   sync.Mutex
		urls = make(map[string]string, len(pending))
	)

	g, ctx := errgroup.WithContext(ctx)

	for key := range pending {
		k := key
		g.Go(func() error {
			url, err := h.mediaStorage.GetSignedURL(ctx, k, "GET", signedURLTTL)
			if err != nil {
				return fmt.Errorf("failed to get signed URL for storage key %s: %w", k, err)
			}
			mu.Lock()
			urls[k] = url
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return urls, nil
}
