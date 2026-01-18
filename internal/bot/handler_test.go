package bot_test

import (
	"context"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/bot"
	"yuruppu/internal/groupprofile"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
	lineclient "yuruppu/internal/line/client"
	lineserver "yuruppu/internal/line/server"
	"yuruppu/internal/userprofile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface satisfaction checks
var (
	_ agent.Agent        = (*mockAgent)(nil)
	_ lineserver.Handler = (*bot.Handler)(nil)
)

// =============================================================================
// NewHandler Tests
// =============================================================================

func TestNewHandler(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		mockAg := &mockAgent{}
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)

		require.NoError(t, err)
		require.NotNil(t, h)
	})
}

func TestNewHandler_NilDependencies(t *testing.T) {
	validConfig := bot.HandlerConfig{
		TypingIndicatorDelay:   3 * time.Second,
		TypingIndicatorTimeout: 30 * time.Second,
	}

	t.Run("returns error when lineClient is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(nil, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "lineClient is required")
	})

	t.Run("returns error when profileService is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(&mockLineClient{}, nil, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "userProfileSvc is required")
	})

	t.Run("returns error when groupProfileService is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, nil, historyRepo, &mockMediaService{}, &mockAgent{}, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "groupProfileSvc is required")
	})

	t.Run("returns error when historySvc is nil", func(t *testing.T) {
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, nil, &mockMediaService{}, &mockAgent{}, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "historySvc is required")
	})

	t.Run("returns error when mediaSvc is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, nil, &mockAgent{}, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "mediaSvc is required")
	})

	t.Run("returns error when agent is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, nil, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "agent is required")
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validConfig, nil)

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "logger is required")
	})
}

// =============================================================================
// Helpers
// =============================================================================

// withLineContext creates a context with LINE-specific values
func withLineContext(ctx context.Context, replyToken, sourceID, userID string) context.Context {
	chatType := line.ChatTypeGroup
	if sourceID == userID {
		chatType = line.ChatTypeOneOnOne
	}
	ctx = line.WithChatType(ctx, chatType)
	ctx = line.WithSourceID(ctx, sourceID)
	ctx = line.WithUserID(ctx, userID)
	ctx = line.WithReplyToken(ctx, replyToken)
	return ctx
}

// validHandlerConfig returns a valid HandlerConfig for tests
func validHandlerConfig() bot.HandlerConfig {
	return bot.HandlerConfig{
		TypingIndicatorDelay:   3 * time.Second,
		TypingIndicatorTimeout: 30 * time.Second,
	}
}

// =============================================================================
// Test Handler Builder
// =============================================================================

// testHandlerBuilder provides a fluent API for constructing test handlers
type testHandlerBuilder struct {
	t            *testing.T
	lineClient   *mockLineClient
	profile      *mockProfileService
	groupProfile *mockGroupProfileService
	media        *mockMediaService
	agent        *mockAgent
	storage      *mockStorage
}

// newTestHandler creates a new test handler builder with sensible defaults
func newTestHandler(t *testing.T) *testHandlerBuilder {
	return &testHandlerBuilder{
		t:            t,
		lineClient:   &mockLineClient{},
		profile:      &mockProfileService{},
		groupProfile: &mockGroupProfileService{},
		media:        &mockMediaService{},
		agent:        &mockAgent{},
		storage:      newMockStorage(),
	}
}

// WithLineClient sets a custom LINE client mock
func (b *testHandlerBuilder) WithLineClient(lc *mockLineClient) *testHandlerBuilder {
	b.lineClient = lc
	return b
}

// WithGroupSummary configures the LINE client to return a specific group summary
func (b *testHandlerBuilder) WithGroupSummary(groupID, groupName, pictureURL string) *testHandlerBuilder {
	b.lineClient.groupSummary = &lineclient.GroupSummary{
		GroupID:    groupID,
		GroupName:  groupName,
		PictureURL: pictureURL,
	}
	return b
}

// WithGroupSummaryError configures the LINE client to return an error for GetGroupSummary
func (b *testHandlerBuilder) WithGroupSummaryError(err error) *testHandlerBuilder {
	b.lineClient.groupSummaryErr = err
	return b
}

// WithMemberCount configures the LINE client to return a specific member count
func (b *testHandlerBuilder) WithMemberCount(count int) *testHandlerBuilder {
	b.lineClient.groupMemberCount = count
	return b
}

// WithMemberCountError configures the LINE client to return an error for GetGroupMemberCount
func (b *testHandlerBuilder) WithMemberCountError(err error) *testHandlerBuilder {
	b.lineClient.groupMemberCountErr = err
	return b
}

// WithGroupProfile sets a custom group profile service mock
func (b *testHandlerBuilder) WithGroupProfile(gps *mockGroupProfileService) *testHandlerBuilder {
	b.groupProfile = gps
	return b
}

// WithGroupProfileError configures the group profile service to return errors
func (b *testHandlerBuilder) WithGroupProfileError(getErr, setErr error) *testHandlerBuilder {
	b.groupProfile.getErr = getErr
	b.groupProfile.setErr = setErr
	return b
}

// WithInitialGroupProfile sets the initial profile that GetGroupProfile will return
func (b *testHandlerBuilder) WithInitialGroupProfile(profile *groupprofile.GroupProfile) *testHandlerBuilder {
	b.groupProfile.profile = profile
	return b
}

// WithAgent sets a custom agent mock
func (b *testHandlerBuilder) WithAgent(ag *mockAgent) *testHandlerBuilder {
	b.agent = ag
	return b
}

// WithStorage sets a custom storage mock
func (b *testHandlerBuilder) WithStorage(s *mockStorage) *testHandlerBuilder {
	b.storage = s
	return b
}

// Build creates the handler with configured mocks
func (b *testHandlerBuilder) Build() *bot.Handler {
	historyRepo, err := history.NewService(b.storage)
	require.NoError(b.t, err)

	handler, err := bot.NewHandler(
		b.lineClient,
		b.profile,
		b.groupProfile,
		historyRepo,
		b.media,
		b.agent,
		validHandlerConfig(),
		slog.New(slog.DiscardHandler),
	)
	require.NoError(b.t, err)
	return handler
}

// BuildWithMocks creates the handler and returns it along with key mocks for verification
func (b *testHandlerBuilder) BuildWithMocks() (*bot.Handler, *mockLineClient, *mockGroupProfileService) {
	return b.Build(), b.lineClient, b.groupProfile
}

// =============================================================================
// Mocks
// =============================================================================

type mockAgent struct {
	response            string
	err                 error
	lastUserMessageText string
	lastContextText     string        // Captures the first message if it's a context message
	processDelay        time.Duration // Delay to simulate slow processing
}

func (m *mockAgent) Generate(ctx context.Context, hist []agent.Message) (*agent.AssistantMessage, error) {
	// Extract context from first message if it looks like a context message
	m.extractContextFromHistory(hist)

	// Extract text from last user message in history for testing
	m.extractLastUserMessageText(hist)
	// Simulate processing delay for testing delayed loading indicator
	if m.processDelay > 0 {
		select {
		case <-time.After(m.processDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.err != nil {
		return nil, m.err
	}
	return &agent.AssistantMessage{
		Parts: []agent.AssistantPart{&agent.AssistantTextPart{Text: m.response}},
	}, nil
}

func (m *mockAgent) Close(ctx context.Context) error {
	return nil
}

// extractContextFromHistory extracts context from the first message if it's a context message
func (m *mockAgent) extractContextFromHistory(hist []agent.Message) {
	if len(hist) == 0 {
		return
	}
	userMsg, ok := hist[0].(*agent.UserMessage)
	if !ok || len(userMsg.Parts) == 0 {
		return
	}
	textPart, ok := userMsg.Parts[0].(*agent.UserTextPart)
	if !ok || len(textPart.Text) == 0 {
		return
	}
	// Check if this is a context message (starts with "[context]")
	if textPart.Text[0] == '[' {
		m.lastContextText = textPart.Text
	}
}

// extractLastUserMessageText extracts text from the last user message
// Parts[0] is the header, Parts[1] is the actual message content
func (m *mockAgent) extractLastUserMessageText(hist []agent.Message) {
	if len(hist) == 0 {
		return
	}
	userMsg, ok := hist[len(hist)-1].(*agent.UserMessage)
	if !ok || len(userMsg.Parts) <= 1 {
		return
	}
	textPart, ok := userMsg.Parts[1].(*agent.UserTextPart)
	if !ok {
		return
	}
	m.lastUserMessageText = textPart.Text
}

type mockLineClient struct {
	data          []byte
	mimeType      string
	err           error
	lastMessageID string
	profile       *lineclient.UserProfile
	profileErr    error
	// ShowLoadingAnimation tracking
	showLoadingCalled  bool
	showLoadingChatID  string
	showLoadingTimeout time.Duration
	showLoadingDelay   time.Duration // Delay to simulate slow API call
	showLoadingErr     error
	// GroupSummary tracking
	groupSummary    *lineclient.GroupSummary
	groupSummaryErr error
	lastGroupID     string
	// GroupMemberCount tracking
	groupMemberCount    int
	groupMemberCountErr error
}

func (m *mockLineClient) GetMessageContent(messageID string) ([]byte, string, error) {
	m.lastMessageID = messageID
	if m.err != nil {
		return nil, "", m.err
	}
	return m.data, m.mimeType, nil
}

func (m *mockLineClient) GetUserProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error) {
	if m.profileErr != nil {
		return nil, m.profileErr
	}
	if m.profile != nil {
		return m.profile, nil
	}
	return &lineclient.UserProfile{
		DisplayName:   "Test User",
		PictureURL:    "",
		StatusMessage: "",
	}, nil
}

func (m *mockLineClient) GetGroupSummary(ctx context.Context, groupID string) (*lineclient.GroupSummary, error) {
	m.lastGroupID = groupID
	if m.groupSummaryErr != nil {
		return nil, m.groupSummaryErr
	}
	if m.groupSummary != nil {
		return m.groupSummary, nil
	}
	return &lineclient.GroupSummary{
		GroupID:    groupID,
		GroupName:  "Test Group",
		PictureURL: "",
	}, nil
}

func (m *mockLineClient) GetGroupMemberCount(ctx context.Context, groupID string) (int, error) {
	if m.groupMemberCountErr != nil {
		return 0, m.groupMemberCountErr
	}
	return m.groupMemberCount, nil
}

func (m *mockLineClient) ShowLoadingAnimation(ctx context.Context, chatID string, timeout time.Duration) error {
	m.showLoadingCalled = true
	m.showLoadingChatID = chatID
	m.showLoadingTimeout = timeout

	// Simulate API delay if configured
	if m.showLoadingDelay > 0 {
		select {
		case <-time.After(m.showLoadingDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return m.showLoadingErr
}

type mockProfileService struct {
	profile    *userprofile.UserProfile
	getErr     error
	setErr     error
	lastUserID string
}

func (m *mockProfileService) GetUserProfile(ctx context.Context, userID string) (*userprofile.UserProfile, error) {
	m.lastUserID = userID
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.profile != nil {
		return m.profile, nil
	}
	return &userprofile.UserProfile{
		DisplayName:   "Test User",
		PictureURL:    "",
		StatusMessage: "",
	}, nil
}

func (m *mockProfileService) SetUserProfile(ctx context.Context, userID string, p *userprofile.UserProfile) error {
	m.lastUserID = userID
	m.profile = p
	return m.setErr
}

// writeResult represents a single Write call result
type writeResult struct {
	gen int64
	err error
}

// writeRecord represents a recorded Write call
type writeRecord struct {
	key      string
	mimeType string
	data     []byte
}

// mockStorage implements storage.Storage interface
type mockStorage struct {
	// Read behavior
	data          map[string][]byte
	generation    map[string]int64
	readErr       error
	readCallCount int

	// Write behavior
	writeResults         []writeResult
	writeCallCount       int
	writes               []writeRecord
	lastWriteKey         string
	lastWriteMIMEType    string
	lastWriteData        []byte
	lastWriteExpectedGen int64
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data:       make(map[string][]byte),
		generation: make(map[string]int64),
	}
}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	m.readCallCount++
	if m.readErr != nil {
		return nil, 0, m.readErr
	}
	data, exists := m.data[key]
	if !exists {
		return nil, 0, nil
	}
	return data, m.generation[key], nil
}

func (m *mockStorage) Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) (int64, error) {
	m.writeCallCount++
	m.writes = append(m.writes, writeRecord{key: key, mimeType: mimetype, data: data})
	m.lastWriteKey = key
	m.lastWriteMIMEType = mimetype
	m.lastWriteData = data
	m.lastWriteExpectedGen = expectedGeneration

	if len(m.writeResults) > 0 {
		r := m.writeResults[0]
		m.writeResults = m.writeResults[1:]
		if r.err != nil {
			return 0, r.err
		}
		m.data[key] = data
		m.generation[key] = r.gen
		return r.gen, nil
	}
	m.data[key] = data
	newGen := expectedGeneration + 1
	m.generation[key] = newGen
	return newGen, nil
}

func (m *mockStorage) GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	return "https://example.com/signed/" + key, nil
}

// mockMediaService implements bot.MediaService interface
type mockMediaService struct {
	storeErr     error
	storeKey     string
	lastSourceID string
	lastData     []byte
	lastMIMEType string
}

func (m *mockMediaService) Store(ctx context.Context, sourceID string, data []byte, mimeType string) (string, error) {
	m.lastSourceID = sourceID
	m.lastData = data
	m.lastMIMEType = mimeType
	if m.storeErr != nil {
		return "", m.storeErr
	}
	if m.storeKey != "" {
		return m.storeKey, nil
	}
	return sourceID + "/test-uuid", nil
}

func (m *mockMediaService) GetSignedURL(ctx context.Context, storageKey string, ttl time.Duration) (string, error) {
	return "https://example.com/signed/" + storageKey, nil
}

// mockGroupProfileService implements bot.GroupProfileService interface
type mockGroupProfileService struct {
	profile     *groupprofile.GroupProfile
	getErr      error
	setErr      error
	lastGroupID string
}

func (m *mockGroupProfileService) GetGroupProfile(ctx context.Context, groupID string) (*groupprofile.GroupProfile, error) {
	m.lastGroupID = groupID
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.profile != nil {
		return m.profile, nil
	}
	return &groupprofile.GroupProfile{
		DisplayName: "Test Group",
		PictureURL:  "",
	}, nil
}

func (m *mockGroupProfileService) SetGroupProfile(ctx context.Context, groupID string, p *groupprofile.GroupProfile) error {
	m.lastGroupID = groupID
	m.profile = p
	return m.setErr
}
