// Package message provides the Message type for conversation history.
package message

import "time"

// Message represents a single message in conversation history.
type Message struct {
	Role      string    `json:"Role"` // "user" or "assistant"
	Content   string    `json:"Content"`
	Timestamp time.Time `json:"Timestamp"`
}
