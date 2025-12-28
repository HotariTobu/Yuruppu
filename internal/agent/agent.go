// Package agent provides a generic agent abstraction for LLM interactions.
package agent

import (
	// Standard library
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	// Internal packages
	"yuruppu/internal/llm"
)

// ClosedError represents an error when using a closed agent.
type ClosedError struct {
	Message string
}

func (e *ClosedError) Error() string {
	return e.Message
}

// Agent manages system prompt and caching for LLM interactions.
// The Agent component is responsible for:
// - Storing the system prompt
// - Managing cache lifecycle (creation, recreation, deletion)
// - Delegating API calls to Provider
type Agent struct {
	provider     llm.Provider
	systemPrompt string
	cacheTTL     time.Duration
	logger       *slog.Logger

	// Cache state
	cacheName string
	mu        sync.Mutex   // Protects cache recreation
	closedMu  sync.RWMutex // Protects closed state
	closed    bool
}

// New creates a new Agent with the given Provider and system prompt.
// Returns Agent (no error) even if cache creation fails - Agent operates in fallback mode.
// If logger is nil, a discard logger is created.
// cacheTTL specifies the TTL for the cached system prompt.
func New(provider llm.Provider, systemPrompt string, cacheTTL time.Duration, logger *slog.Logger) *Agent {
	// Handle nil logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	a := &Agent{
		provider:     provider,
		systemPrompt: systemPrompt,
		cacheTTL:     cacheTTL,
		logger:       logger,
	}

	// Attempt to create cache during initialization
	// If creation fails, Agent operates in fallback mode (no error returned)
	ctx := context.Background()
	cacheName, err := provider.CreateCachedConfig(ctx, systemPrompt, cacheTTL)
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
// Returns ClosedError if the Agent has been closed.
func (a *Agent) GenerateText(ctx context.Context, userMessage string) (string, error) {
	// Check if Agent is closed
	a.closedMu.RLock()
	if a.closed {
		a.closedMu.RUnlock()
		return "", &ClosedError{Message: "agent is closed"}
	}
	a.closedMu.RUnlock()

	// Use cached path if cache exists, otherwise non-cached path
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
func (a *Agent) generateWithCache(ctx context.Context, cacheName, userMessage string) (string, error) {
	response, err := a.provider.GenerateTextCached(ctx, cacheName, userMessage)
	if err == nil {
		return response, nil
	}

	// Not a cache error, return original error
	if !isCacheError(err) {
		return "", err
	}

	// Cache error detected, attempt recreation
	return a.handleCacheErrorAndRetry(ctx, userMessage, err)
}

// handleCacheErrorAndRetry handles cache errors by attempting cache recreation.
func (a *Agent) handleCacheErrorAndRetry(ctx context.Context, userMessage string, originalErr error) (string, error) {
	a.logger.Warn("cache error detected, attempting recreation",
		slog.Any("error", originalErr),
	)

	// Attempt cache recreation (protected by mutex)
	newCacheName, recreateErr := a.recreateCacheOnce(ctx)
	if recreateErr != nil {
		// If recreation fails, fall back to non-cached mode for this call
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
func (a *Agent) recreateCacheOnce(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Debug("recreating cache")
	newCacheName, err := a.provider.CreateCachedConfig(ctx, a.systemPrompt, a.cacheTTL)
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
// Does not close the Provider - the caller is responsible for Provider lifecycle.
func (a *Agent) Close(ctx context.Context) error {
	// Get cache name first to minimize lock holding time
	a.mu.Lock()
	cacheName := a.cacheName
	a.mu.Unlock()

	// Set closed flag atomically
	a.closedMu.Lock()
	if a.closed {
		a.closedMu.Unlock()
		return nil
	}
	a.closed = true
	a.closedMu.Unlock()

	// Delete cache if it exists (no locks held during API call)
	if cacheName != "" {
		err := a.provider.DeleteCachedConfig(ctx, cacheName)
		if err != nil {
			// Log error but don't return it - Close() completes successfully
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

	// Agent does not close Provider - caller manages Provider lifecycle
	return nil
}

// isCacheError checks if an error is related to cache (expired, not found, etc.).
func isCacheError(err error) bool {
	if err == nil {
		return false
	}

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
