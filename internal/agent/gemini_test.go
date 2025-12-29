package agent

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/url"
	"testing"
	"yuruppu/internal/message"

	"google.golang.org/genai"
)

func TestBuildContentsFromHistory_SingleMessage(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}
	history := []message.Message{
		{Role: "user", Content: "hello"},
	}
	contents := g.buildContentsFromHistory(history)

	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}
	if contents[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", contents[0].Role)
	}
	if contents[0].Parts[0].Text != "hello" {
		t.Errorf("expected text 'hello', got '%s'", contents[0].Parts[0].Text)
	}
}

func TestBuildContentsFromHistory_MultipleMessages(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}
	history := []message.Message{
		{Role: "user", Content: "first message"},
		{Role: "assistant", Content: "first response"},
		{Role: "user", Content: "second message"},
	}
	contents := g.buildContentsFromHistory(history)

	if len(contents) != 3 {
		t.Fatalf("expected 3 contents, got %d", len(contents))
	}

	// First history message
	if contents[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", contents[0].Role)
	}
	if contents[0].Parts[0].Text != "first message" {
		t.Errorf("expected text 'first message', got '%s'", contents[0].Parts[0].Text)
	}

	// Second history message (assistant -> model)
	if contents[1].Role != "model" {
		t.Errorf("expected role 'model', got '%s'", contents[1].Role)
	}
	if contents[1].Parts[0].Text != "first response" {
		t.Errorf("expected text 'first response', got '%s'", contents[1].Parts[0].Text)
	}

	// Third message
	if contents[2].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", contents[2].Role)
	}
	if contents[2].Parts[0].Text != "second message" {
		t.Errorf("expected text 'second message', got '%s'", contents[2].Parts[0].Text)
	}
}

func TestBuildContentsFromHistory_Empty(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}
	contents := g.buildContentsFromHistory(nil)

	if len(contents) != 0 {
		t.Fatalf("expected 0 contents, got %d", len(contents))
	}
}

func TestExtractTextFromResponse_NilResponse(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	_, err := g.extractTextFromResponse(nil)
	if err == nil {
		t.Fatal("expected error for nil response")
	}

	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseError, got %T", err)
	}
}

func TestExtractTextFromResponse_NoCandidates(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	_, err := g.extractTextFromResponse(&genai.GenerateContentResponse{
		Candidates: nil,
	})
	if err == nil {
		t.Fatal("expected error for no candidates")
	}

	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseError, got %T", err)
	}
}

func TestExtractTextFromResponse_ValidResponse(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	text, err := g.extractTextFromResponse(&genai.GenerateContentResponse{
		ModelVersion: "test-model",
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: "response text"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "response text" {
		t.Errorf("expected 'response text', got '%s'", text)
	}
}

func TestExtractTextFromResponse_MultipleParts(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	text, err := g.extractTextFromResponse(&genai.GenerateContentResponse{
		ModelVersion: "test-model",
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "first "},
						{Text: "second "},
						{Text: "third"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "first second third" {
		t.Errorf("expected 'first second third', got '%s'", text)
	}
}

func TestMapAPIError_ContextDeadlineExceeded(t *testing.T) {
	err := mapAPIError(context.DeadlineExceeded)

	var timeoutErr *TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError, got %T", err)
	}
}

func TestMapAPIError_ContextCanceled(t *testing.T) {
	err := mapAPIError(context.Canceled)

	var timeoutErr *TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("expected TimeoutError, got %T", err)
	}
}

func TestMapAPIError_NetworkError(t *testing.T) {
	netErr := &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	err := mapAPIError(netErr)

	var networkErr *NetworkError
	if !errors.As(err, &networkErr) {
		t.Fatalf("expected NetworkError, got %T", err)
	}
}

func TestMapAPIError_DNSError(t *testing.T) {
	dnsErr := &net.DNSError{Err: "no such host", Name: "example.com"}
	err := mapAPIError(dnsErr)

	var networkErr *NetworkError
	if !errors.As(err, &networkErr) {
		t.Fatalf("expected NetworkError, got %T", err)
	}
}

func TestMapAPIError_URLError(t *testing.T) {
	urlErr := &url.Error{Op: "Get", URL: "https://example.com", Err: errors.New("connection refused")}
	err := mapAPIError(urlErr)

	var networkErr *NetworkError
	if !errors.As(err, &networkErr) {
		t.Fatalf("expected NetworkError, got %T", err)
	}
}

func TestMapAPIError_NilError(t *testing.T) {
	err := mapAPIError(nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestMapHTTPStatusCode_Auth(t *testing.T) {
	tests := []struct {
		code int
	}{
		{401},
		{403},
	}

	for _, tt := range tests {
		err := mapHTTPStatusCode(tt.code, "test message")
		var authErr *AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected AuthError for code %d, got %T", tt.code, err)
		}
		if authErr.StatusCode != tt.code {
			t.Errorf("expected status code %d, got %d", tt.code, authErr.StatusCode)
		}
	}
}

func TestMapHTTPStatusCode_RateLimit(t *testing.T) {
	err := mapHTTPStatusCode(429, "rate limited")

	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
}

func TestMapHTTPStatusCode_ServerError(t *testing.T) {
	codes := []int{500, 502, 503, 504}

	for _, code := range codes {
		err := mapHTTPStatusCode(code, "server error")
		var respErr *ResponseError
		if !errors.As(err, &respErr) {
			t.Fatalf("expected ResponseError for code %d, got %T", code, err)
		}
	}
}

func TestMapHTTPStatusCode_Default(t *testing.T) {
	err := mapHTTPStatusCode(400, "bad request")

	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("expected ResponseError, got %T", err)
	}
}

func TestErrorTypes_Error(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"TimeoutError", &TimeoutError{Message: "timeout"}, "timeout"},
		{"RateLimitError", &RateLimitError{Message: "rate limit"}, "rate limit"},
		{"NetworkError", &NetworkError{Message: "network"}, "network"},
		{"ResponseError", &ResponseError{Message: "response"}, "response"},
		{"AuthError", &AuthError{Message: "auth", StatusCode: 401}, "auth"},
		{"ClosedError", &ClosedError{Message: "closed"}, "closed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Lifecycle Tests
// =============================================================================

func TestGeminiAgent_GenerateText_AfterClose(t *testing.T) {
	g := &GeminiAgent{
		logger:                    slog.New(slog.DiscardHandler),
		contentConfigWithCache:    &genai.GenerateContentConfig{},
		contentConfigWithoutCache: &genai.GenerateContentConfig{},
	}

	// Close the agent
	err := g.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on Close: %v", err)
	}

	// GenerateText should return ClosedError
	history := []message.Message{{Role: "user", Content: "hello"}}
	_, err = g.GenerateText(context.Background(), history)

	if err == nil {
		t.Fatal("expected error for GenerateText after Close")
	}

	var closedErr *ClosedError
	if !errors.As(err, &closedErr) {
		t.Fatalf("expected ClosedError, got %T: %v", err, err)
	}
}

func TestGeminiAgent_Close_Idempotent(t *testing.T) {
	g := &GeminiAgent{
		logger:                    slog.New(slog.DiscardHandler),
		contentConfigWithCache:    &genai.GenerateContentConfig{},
		contentConfigWithoutCache: &genai.GenerateContentConfig{},
	}

	// Close multiple times should not error
	for i := range 3 {
		err := g.Close(context.Background())
		if err != nil {
			t.Fatalf("unexpected error on Close call %d: %v", i+1, err)
		}
	}
}

func TestGeminiAgent_GenerateText_EmptyHistory(t *testing.T) {
	g := &GeminiAgent{
		logger: slog.New(slog.DiscardHandler),
	}

	_, err := g.GenerateText(context.Background(), []message.Message{})

	if err == nil {
		t.Fatal("expected error for empty history")
	}
	if err.Error() != "history is required" {
		t.Errorf("expected 'history is required', got '%s'", err.Error())
	}
}

func TestGeminiAgent_GenerateText_LastMessageNotUser(t *testing.T) {
	g := &GeminiAgent{
		logger: slog.New(slog.DiscardHandler),
	}

	history := []message.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}
	_, err := g.GenerateText(context.Background(), history)

	if err == nil {
		t.Fatal("expected error for last message not from user")
	}
	if err.Error() != "last message in history must be from user" {
		t.Errorf("expected 'last message in history must be from user', got '%s'", err.Error())
	}
}
