package agent

import (
	"context"
	"log/slog"
	"testing"

	"google.golang.org/genai"
)

func TestExtractResponseToAssistantMessage_NilResponse(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	_, err := g.extractResponseToAssistantMessage(nil)
	if err == nil {
		t.Fatal("expected error for nil response")
	}
}

func TestExtractResponseToAssistantMessage_NoCandidates(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	_, err := g.extractResponseToAssistantMessage(&genai.GenerateContentResponse{
		Candidates: nil,
	})
	if err == nil {
		t.Fatal("expected error for no candidates")
	}
}

func TestExtractResponseToAssistantMessage_NilContent(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	_, err := g.extractResponseToAssistantMessage(&genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{Content: nil},
		},
	})
	if err == nil {
		t.Fatal("expected error for nil content")
	}
}

func TestExtractResponseToAssistantMessage_EmptyParts(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	_, err := g.extractResponseToAssistantMessage(&genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{Parts: []*genai.Part{}}},
		},
	})
	if err == nil {
		t.Fatal("expected error for empty parts")
	}
}

func TestExtractResponseToAssistantMessage_InvalidParts(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	_, err := g.extractResponseToAssistantMessage(&genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{Parts: []*genai.Part{
				{}, // no Text, no FileData
			}}},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid parts")
	}
}

func TestExtractResponseToAssistantMessage_NilPartSkipped(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	msg, err := g.extractResponseToAssistantMessage(&genai.GenerateContentResponse{
		ModelVersion: "test-model",
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{Parts: []*genai.Part{
				nil,
				{Text: "valid"},
				nil,
			}}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(msg.Parts))
	}
}

func TestExtractResponseToAssistantMessage_NilCandidate(t *testing.T) {
	g := &GeminiAgent{logger: slog.New(slog.DiscardHandler)}

	_, err := g.extractResponseToAssistantMessage(&genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{nil},
	})
	if err == nil {
		t.Fatal("expected error for nil candidate")
	}
}

// =============================================================================
// Lifecycle Tests
// =============================================================================

func TestGeminiAgent_Generate_AfterClose(t *testing.T) {
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

	// Generate should return error
	history := []Message{}
	userMessage := &UserMessage{Parts: []UserPart{&UserTextPart{Text: "hello"}}}
	_, err = g.Generate(context.Background(), history, userMessage)

	if err == nil {
		t.Fatal("expected error for Generate after Close")
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
