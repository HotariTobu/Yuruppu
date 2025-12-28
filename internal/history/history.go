package history

import "time"

// Message represents a single message in conversation history.
type Message struct {
	Role      string    `json:"Role"` // "user" or "assistant"
	Content   string    `json:"Content"`
	Timestamp time.Time `json:"Timestamp"`
}

// ConversationHistory holds messages for a specific source.
type ConversationHistory struct {
	SourceID string
	Messages []Message
}
