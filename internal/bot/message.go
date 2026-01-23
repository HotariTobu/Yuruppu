package bot

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"text/template"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/history"
	"yuruppu/internal/line"

	"golang.org/x/sync/errgroup"
)

//go:embed template/chat_context.txt
var chatContextTemplateText string
var chatContextTemplate = template.Must(template.New("chat_context").Parse(chatContextTemplateText))

//go:embed template/user_profile.txt
var userProfileTemplateText string
var userProfileTemplate = template.Must(template.New("user_profile").Parse(userProfileTemplateText))

//go:embed template/user_header.txt
var userHeaderTemplateText string
var userHeaderTemplate = template.Must(template.New("user_header").Parse(userHeaderTemplateText))

var jst = time.FixedZone("JST", 9*60*60)

const signedURLTTL = 60 * time.Second

func (h *Handler) HandleText(ctx context.Context, text string) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return errors.New("userID not found in context")
	}
	messageID, _ := line.MessageIDFromContext(ctx)
	userMsg := &history.UserMessage{
		MessageID: messageID,
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: text}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleImage(ctx context.Context, messageID string) error {
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return errors.New("sourceID not found in context")
	}
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return errors.New("userID not found in context")
	}
	lineMessageID, _ := line.MessageIDFromContext(ctx)

	var parts []history.UserPart
	data, mimeType, err := h.lineClient.GetMessageContent(messageID)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to download image, using placeholder",
			slog.String("messageID", messageID),
			slog.Any("error", err),
		)
		parts = []history.UserPart{&history.UserTextPart{Text: "[User sent an image, but an error occurred while loading]"}}
	} else if storageKey, err := h.media.Store(ctx, sourceID, data, mimeType); err != nil {
		h.logger.WarnContext(ctx, "failed to store image, using placeholder",
			slog.String("messageID", messageID),
			slog.Any("error", err),
		)
		parts = []history.UserPart{&history.UserTextPart{Text: "[User sent an image, but an error occurred while loading]"}}
	} else {
		parts = []history.UserPart{&history.UserFileDataPart{StorageKey: storageKey, MIMEType: mimeType}}
	}

	userMsg := &history.UserMessage{
		MessageID: lineMessageID,
		UserID:    userID,
		Parts:     parts,
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleSticker(ctx context.Context, packageID, stickerID string) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return errors.New("userID not found in context")
	}
	messageID, _ := line.MessageIDFromContext(ctx)
	userMsg := &history.UserMessage{
		MessageID: messageID,
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: "[User sent a sticker]"}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleVideo(ctx context.Context, messageID string) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return errors.New("userID not found in context")
	}
	lineMessageID, _ := line.MessageIDFromContext(ctx)
	userMsg := &history.UserMessage{
		MessageID: lineMessageID,
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: "[User sent a video]"}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleAudio(ctx context.Context, messageID string) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return errors.New("userID not found in context")
	}
	lineMessageID, _ := line.MessageIDFromContext(ctx)
	userMsg := &history.UserMessage{
		MessageID: lineMessageID,
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: "[User sent an audio]"}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleLocation(ctx context.Context, latitude, longitude float64) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return errors.New("userID not found in context")
	}
	messageID, _ := line.MessageIDFromContext(ctx)
	userMsg := &history.UserMessage{
		MessageID: messageID,
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: "[User sent a location]"}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) HandleFile(ctx context.Context, messageID, fileName string, fileSize int64) error {
	userID, ok := line.UserIDFromContext(ctx)
	if !ok {
		return errors.New("userID not found in context")
	}
	messageID, _ := line.MessageIDFromContext(ctx)
	userMsg := &history.UserMessage{
		MessageID: messageID,
		UserID:    userID,
		Parts:     []history.UserPart{&history.UserTextPart{Text: fmt.Sprintf("[User sent a file: %s]", fileName)}},
		Timestamp: time.Now(),
	}
	return h.handleMessage(ctx, userMsg)
}

func (h *Handler) handleMessage(ctx context.Context, userMsg *history.UserMessage) error {
	chatType, ok := line.ChatTypeFromContext(ctx)
	if !ok {
		return errors.New("chatType not found in context")
	}
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return errors.New("sourceID not found in context")
	}

	// Delayed loading indicator (FR-001, FR-002, FR-006, NFR-001, NFR-002)
	done := make(chan struct{})
	defer close(done)

	if chatType == line.ChatTypeOneOnOne {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					h.logger.WarnContext(ctx, "loading indicator goroutine panicked", slog.Any("panic", r))
				}
			}()
			select {
			case <-time.After(h.config.TypingIndicatorDelay):
				// Still processing → show indicator (FR-001)
				if err := h.lineClient.ShowLoadingAnimation(ctx, sourceID, h.config.TypingIndicatorTimeout); err != nil {
					h.logger.WarnContext(ctx, "failed to show loading animation", slog.Any("error", err))
				}
			case <-done:
				// Completed → do nothing (FR-006)
				return
			case <-ctx.Done():
				// Context cancelled → exit cleanly
				return
			}
		}()
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

	// Step 3: Build context message and convert history to agent format
	usernameCache := make(map[string]string)
	getUsername := func(userID string) string {
		if name, ok := usernameCache[userID]; ok {
			return name
		}
		var name string
		if p, err := h.userProfileService.GetUserProfile(ctx, userID); err == nil {
			name = p.DisplayName
		} else {
			name = "Unknown User"
			h.logger.InfoContext(ctx, "failed to get username",
				slog.String("userID", userID),
				slog.Any("error", err),
			)
		}
		usernameCache[userID] = name
		return name
	}

	var contextParts []agent.UserPart
	var agentHistory []agent.Message
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		contextParts, err = h.buildContextParts(gCtx, userMsg.UserID)
		return err
	})
	g.Go(func() error {
		var err error
		agentHistory, err = h.convertToAgentHistory(gCtx, hist, getUsername)
		return err
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to prepare agent input: %w", err)
	}

	agentInput := agentHistory
	if len(contextParts) > 0 {
		agentInput = append([]agent.Message{&agent.UserMessage{Parts: contextParts}}, agentHistory...)
	}
	response, err := h.agent.Generate(ctx, agentInput)
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

func (h *Handler) buildContextParts(ctx context.Context, userID string) ([]agent.UserPart, error) {
	chatType, ok := line.ChatTypeFromContext(ctx)
	if !ok {
		return nil, errors.New("chatType not found in context")
	}
	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		return nil, errors.New("sourceID not found in context")
	}

	// Get user count for group chats (FR-005)
	var userCount int
	if chatType == line.ChatTypeGroup {
		profile, err := h.groupProfileService.GetGroupProfile(ctx, sourceID)
		if err != nil {
			slog.WarnContext(ctx, "failed to get group profile for user count", "error", err)
		} else {
			userCount = profile.UserCount
		}
	}

	var buf bytes.Buffer
	if err := chatContextTemplate.Execute(&buf, struct {
		CurrentLocalTime string
		ChatType         line.ChatType
		UserCount        int
	}{
		CurrentLocalTime: time.Now().In(jst).Format("2006 Jan 2(Mon) 3:04PM"),
		ChatType:         chatType,
		UserCount:        userCount,
	}); err != nil {
		return nil, fmt.Errorf("failed to execute chat context template: %w", err)
	}
	parts := []agent.UserPart{&agent.UserTextPart{Text: buf.String()}}

	p, err := h.userProfileService.GetUserProfile(ctx, userID)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to get user profile",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return parts, nil
	}

	buf.Reset()
	if err := userProfileTemplate.Execute(&buf, p); err != nil {
		return nil, fmt.Errorf("failed to execute user profile template: %w", err)
	}
	parts = append(parts, &agent.UserTextPart{Text: buf.String()})

	if p.PictureURL != "" {
		parts = append(parts, &agent.UserFileDataPart{
			FileURI:     p.PictureURL,
			MIMEType:    p.PictureMIMEType,
			DisplayName: p.DisplayName + "'s avatar",
		})
	}

	return parts, nil
}

// convertToAgentHistory converts history.Message slice to agent.Message slice.
// Fetches signed URLs in parallel for all file parts.
func (h *Handler) convertToAgentHistory(ctx context.Context, hist []history.Message, getUsername func(string) string) ([]agent.Message, error) {
	result := make([]agent.Message, 0, len(hist))
	pending := make(map[string]agent.FileDataPart)

	for _, msg := range hist {
		switch m := msg.(type) {
		case *history.UserMessage:
			agentMsg, p := convertUserMessage(m, getUsername)
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
func convertUserMessage(m *history.UserMessage, getUsername func(string) string) (*agent.UserMessage, map[string]agent.FileDataPart) {
	parts := make([]agent.UserPart, 0, len(m.Parts)+1)
	pending := make(map[string]agent.FileDataPart)

	// Add header with username and timestamp
	var header bytes.Buffer
	_ = userHeaderTemplate.Execute(&header, map[string]string{
		"UserName":  getUsername(m.UserID),
		"LocalTime": m.Timestamp.In(jst).Format("Jan 2(Mon) 3:04PM"),
	})
	parts = append(parts, &agent.UserTextPart{Text: header.String()})

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
		Parts: parts,
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
		Parts: parts,
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
			url, err := h.media.GetSignedURL(ctx, k, signedURLTTL)
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
