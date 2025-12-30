package history

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func (r *Repository) parseJSONL(data []byte) ([]Message, error) {
	var messages []Message
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var m message
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return nil, err
		}

		switch m.Role {
		case "user":
			messages = append(messages, UserMessage{
				UserID:    m.UserID,
				Parts:     convertJSONToUserParts(m.Parts),
				Timestamp: m.Timestamp,
			})
		case "assistant":
			messages = append(messages, AssistantMessage{
				ModelName: m.ModelName,
				Parts:     convertJSONToAssistantParts(m.Parts),
				Timestamp: m.Timestamp,
			})
		default:
			return nil, fmt.Errorf("unknown role: %s", m.Role)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func convertJSONToUserParts(parts []part) []UserPart {
	result := make([]UserPart, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "text":
			result = append(result, UserTextPart{
				Text: p.Text,
			})
		case "file_data":
			result = append(result, UserFileDataPart{
				StorageKey:    p.StorageKey,
				MIMEType:      p.MIMEType,
				DisplayName:   p.DisplayName,
				VideoMetadata: p.VideoMetadata,
			})
		}
	}
	return result
}

func convertJSONToAssistantParts(parts []part) []AssistantPart {
	result := make([]AssistantPart, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "text":
			result = append(result, AssistantTextPart{
				Text:             p.Text,
				Thought:          p.Thought,
				ThoughtSignature: p.ThoughtSignature,
			})
		case "file_data":
			result = append(result, AssistantFileDataPart{
				StorageKey:  p.StorageKey,
				MIMEType:    p.MIMEType,
				DisplayName: p.DisplayName,
			})
		}
	}
	return result
}
