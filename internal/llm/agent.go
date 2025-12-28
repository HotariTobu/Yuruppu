package llm

import (
	"context"
	"log/slog"
	"strings"
	"sync"
)

// Agent manages system prompt and caching for LLM interactions.
// AC-002: Agent Interface Defined
// The Agent component is responsible for:
// - Storing the system prompt
// - Managing cache lifecycle (creation, recreation, deletion)
// - Delegating API calls to Provider
type Agent interface {
	// GenerateText generates a text response given a user message.
	// The system prompt is managed internally by the Agent.
	// Returns LLMClosedError if the Agent has been closed.
	GenerateText(ctx context.Context, userMessage string) (string, error)

	// Close cleans up the Agent's resources (deletes cache).
	// Does not close the Provider - the caller is responsible for Provider lifecycle.
	// Close is idempotent and safe to call multiple times.
	Close(ctx context.Context) error
}

// agent is the implementation of Agent interface.
// AC-003: Agent Manages Cache
type agent struct {
	provider     Provider
	systemPrompt string
	logger       *slog.Logger

	// Cache state
	cacheName string
	mu        sync.Mutex   // Protects cache recreation
	closedMu  sync.RWMutex // Protects closed state
	closed    bool
}

// NewAgent creates a new Agent with the given Provider and system prompt.
// AC-002: NewAgent(provider Provider, systemPrompt string, logger *slog.Logger) Agent
// AC-003: Cache created during NewAgent() via provider.CreateCache() with 60-minute TTL
//
// Returns Agent (no error) even if cache creation fails - Agent operates in fallback mode.
// If logger is nil, a discard logger is created.
func NewAgent(provider Provider, systemPrompt string, logger *slog.Logger) Agent {
	// Handle nil logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	a := &agent{
		provider:     provider,
		systemPrompt: systemPrompt,
		logger:       logger,
	}

	// AC-003: Attempt to create cache during initialization
	// If creation fails, Agent operates in fallback mode (no error returned)
	ctx := context.Background()
	cacheName, err := provider.CreateCache(ctx, systemPrompt)
	if err != nil {
		logger.Warn("initial cache creation failed, operating in fallback mode",
			slog.Any("error", err),
		)
		// Fallback mode: cacheName remains empty
	} else {
		a.cacheName = cacheName
		logger.Debug("cache created during initialization",
			slog.String("cacheName", cacheName),
		)
	}

	return a
}

// GenerateText generates a text response given a user message.
// AC-003: Agent calls provider.GenerateTextCached() when cacheName is set, otherwise provider.GenerateText()
// AC-003: Cache errors during GenerateTextCached() trigger automatic recreation
func (a *agent) GenerateText(ctx context.Context, userMessage string) (string, error) {
	// AC-004: Check if Agent is closed
	a.closedMu.RLock()
	if a.closed {
		a.closedMu.RUnlock()
		return "", &LLMClosedError{Message: "agent is closed"}
	}
	a.closedMu.RUnlock()

	// AC-003: Use cached path if cache exists, otherwise non-cached path
	a.mu.Lock()
	cacheName := a.cacheName
	a.mu.Unlock()

	if cacheName == "" {
		// Fallback mode: use non-cached path
		return a.provider.GenerateText(ctx, a.systemPrompt, userMessage)
	}

	return a.generateWithCache(ctx, cacheName, userMessage)
}

// generateWithCache attempts to generate text using the cache.
// If cache error is detected, it attempts recreation and retry.
func (a *agent) generateWithCache(ctx context.Context, cacheName, userMessage string) (string, error) {
	response, err := a.provider.GenerateTextCached(ctx, cacheName, userMessage)
	if err == nil {
		return response, nil
	}

	// Not a cache error, return original error
	if !isCacheError(err) {
		return "", err
	}

	// AC-003: Cache error detected, attempt recreation
	return a.handleCacheErrorAndRetry(ctx, userMessage, err)
}

// handleCacheErrorAndRetry handles cache errors by attempting cache recreation.
// AC-003: Cache errors during GenerateTextCached() trigger automatic recreation
func (a *agent) handleCacheErrorAndRetry(ctx context.Context, userMessage string, originalErr error) (string, error) {
	a.logger.Warn("cache error detected, attempting recreation",
		slog.Any("error", originalErr),
	)

	// AC-003: Attempt cache recreation (protected by mutex)
	newCacheName, recreateErr := a.recreateCacheOnce(ctx)
	if recreateErr != nil {
		// AC-003: If recreation fails, fall back to non-cached mode for this call
		a.logger.Warn("cache recreation failed, falling back to non-cached mode",
			slog.Any("error", recreateErr),
		)
		return a.provider.GenerateText(ctx, a.systemPrompt, userMessage)
	}

	// Retry with recreated cache
	response, retryErr := a.provider.GenerateTextCached(ctx, newCacheName, userMessage)
	if retryErr != nil {
		// If retry also fails, fall back to non-cached mode
		a.logger.Warn("retry with recreated cache failed, falling back to non-cached mode",
			slog.Any("error", retryErr),
		)
		return a.provider.GenerateText(ctx, a.systemPrompt, userMessage)
	}

	return response, nil
}

// recreateCacheOnce recreates the cache, protected by mutex to prevent concurrent recreation.
// AC-003: Concurrent recreation attempts prevented by mutex
func (a *agent) recreateCacheOnce(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check: another goroutine might have already recreated the cache
	// For simplicity, we always recreate when called (single recreation per error is handled by caller logic)

	a.logger.Debug("recreating cache")
	newCacheName, err := a.provider.CreateCache(ctx, a.systemPrompt)
	if err != nil {
		a.logger.Error("cache recreation failed",
			slog.Any("error", err),
		)
		// Clear cache name to enter fallback mode
		a.cacheName = ""
		return "", err
	}

	// Update cache name
	a.cacheName = newCacheName
	a.logger.Info("cache recreated successfully",
		slog.String("cacheName", newCacheName),
	)

	return newCacheName, nil
}

// Close cleans up the Agent's resources by deleting the cache.
// AC-003: Close() deletes cache via provider.DeleteCache() (does not close Provider)
// AC-003: If cache deletion fails during Close(), error is logged but Close() completes successfully
func (a *agent) Close(ctx context.Context) error {
	a.closedMu.Lock()
	defer a.closedMu.Unlock()

	// Idempotent: safe to call multiple times
	if a.closed {
		return nil
	}

	a.closed = true

	// AC-003: Delete cache if it exists
	a.mu.Lock()
	cacheName := a.cacheName
	a.mu.Unlock()

	if cacheName != "" {
		err := a.provider.DeleteCache(ctx, cacheName)
		if err != nil {
			// AC-003: Log error but don't return it - Close() completes successfully
			a.logger.Warn("cache deletion failed during close",
				slog.String("cacheName", cacheName),
				slog.Any("error", err),
			)
		} else {
			a.logger.Debug("cache deleted during close",
				slog.String("cacheName", cacheName),
			)
		}
	}

	// AC-003: Agent does not close Provider - caller manages Provider lifecycle
	return nil
}

// isCacheError checks if an error is related to cache (expired, not found, etc.).
// Cache errors typically contain specific keywords in the error message.
func isCacheError(err error) bool {
	if err == nil {
		return false
	}

	// Check error message for cache-related keywords
	errMsg := strings.ToLower(err.Error())
	cacheKeywords := []string{
		"cache not found",
		"cache expired",
		"cached content not found",
		"invalid cache",
		"cache error",
	}

	for _, keyword := range cacheKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}
