package agent

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
	FileURI       string
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
	FileURI     string
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
	UserName  string
	Parts     []UserPart
	LocalTime string
}

func (*UserMessage) message() {}

// AssistantMessage represents a message from an assistant.
type AssistantMessage struct {
	ModelName string
	Parts     []AssistantPart
	LocalTime string
}

func (*AssistantMessage) message() {}

// ============================================================
// FileDataPart Interface
// ============================================================

// FileDataPart is an interface for file data parts that need FileURI.
type FileDataPart interface {
	SetFileURI(uri string)
}

// SetFileURI sets the FileURI for UserFileDataPart.
func (p *UserFileDataPart) SetFileURI(uri string) { p.FileURI = uri }

// SetFileURI sets the FileURI for AssistantFileDataPart.
func (p *AssistantFileDataPart) SetFileURI(uri string) { p.FileURI = uri }
