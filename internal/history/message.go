package history

import "time"

// ============================================================
// User Parts
// ============================================================

// UserPart is a marker interface for user message parts.
type UserPart interface {
	userPart()
}

// UserTextPart represents a text part in a user message.
type UserTextPart struct {
	Text string
}

func (*UserTextPart) userPart() {}

// UserFileDataPart represents a file data part in a user message.
type UserFileDataPart struct {
	StorageKey       string
	MIMEType      string
	DisplayName   string
	VideoMetadata *VideoMetadata
}

func (*UserFileDataPart) userPart() {}

// VideoMetadata contains metadata for video files.
type VideoMetadata struct {
	StartOffset time.Duration
	EndOffset   time.Duration
	FPS         *float64
}

// ============================================================
// Assistant Parts
// ============================================================

// AssistantPart is a marker interface for assistant message parts.
type AssistantPart interface {
	assistantPart()
}

// AssistantTextPart represents a text part in an assistant message.
type AssistantTextPart struct {
	Text             string
	Thought          bool
	ThoughtSignature string
}

func (*AssistantTextPart) assistantPart() {}

// AssistantFileDataPart represents a file data part in an assistant message.
type AssistantFileDataPart struct {
	StorageKey  string
	MIMEType    string
	DisplayName string
}

func (*AssistantFileDataPart) assistantPart() {}

// ============================================================
// Message Types
// ============================================================

// Message is a marker interface for messages.
type Message interface {
	message()
}

// UserMessage represents a message from a user.
type UserMessage struct {
	UserID    string
	Parts     []UserPart
	Timestamp time.Time
}

func (*UserMessage) message() {}

// AssistantMessage represents a message from an assistant.
type AssistantMessage struct {
	ModelName string
	Parts     []AssistantPart
	Timestamp time.Time
}

func (*AssistantMessage) message() {}




// ============================================================
// Internal JSON structs
// ============================================================

type part struct {
	Type             string         `json:"type"`
	Text             string         `json:"text,omitempty"`
	StorageKey       string         `json:"storageKey,omitempty"`
	MIMEType         string         `json:"mimeType,omitempty"`
	DisplayName      string         `json:"displayName,omitempty"`
	VideoMetadata    *VideoMetadata `json:"videoMetadata,omitempty"`
	Thought          bool           `json:"thought,omitempty"`
	ThoughtSignature string         `json:"thoughtSignature,omitempty"`
}

type message struct {
	Role      string    `json:"role"`
	UserID    string    `json:"userId,omitempty"`
	ModelName string    `json:"modelName,omitempty"`
	Parts     []part    `json:"parts"`
	Timestamp time.Time `json:"timestamp"`
}
