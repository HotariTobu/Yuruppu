package history

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func parseJSONL(data []byte) ([]Message, error) {
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
			parts, err := convertJSONToUserParts(m.Parts)
			if err != nil {
				return nil, err
			}
			messages = append(messages, &UserMessage{
				MessageID: m.MessageID,
				UserID:    m.UserID,
				Parts:     parts,
				Timestamp: m.Timestamp,
			})
		case "assistant":
			parts, err := convertJSONToAssistantParts(m.Parts)
			if err != nil {
				return nil, err
			}
			messages = append(messages, &AssistantMessage{
				ModelName: m.ModelName,
				Parts:     parts,
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

func convertJSONToUserParts(parts []part) ([]UserPart, error) {
	result := make([]UserPart, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "text":
			result = append(result, &UserTextPart{
				Text: p.Text,
			})
		case "file_data":
			filePart := &UserFileDataPart{
				StorageKey:  p.StorageKey,
				MIMEType:    p.MIMEType,
				DisplayName: p.DisplayName,
			}
			if p.VideoMetadata != nil {
				filePart.VideoMetadata = &VideoMetadata{
					StartOffset: p.VideoMetadata.StartOffset,
					EndOffset:   p.VideoMetadata.EndOffset,
					FPS:         p.VideoMetadata.FPS,
				}
			}
			result = append(result, filePart)
		default:
			return nil, fmt.Errorf("unknown user part type: %s", p.Type)
		}
	}
	return result, nil
}

func convertJSONToAssistantParts(parts []part) ([]AssistantPart, error) {
	result := make([]AssistantPart, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "text":
			result = append(result, &AssistantTextPart{
				Text:             p.Text,
				Thought:          p.Thought,
				ThoughtSignature: p.ThoughtSignature,
			})
		case "file_data":
			result = append(result, &AssistantFileDataPart{
				StorageKey:  p.StorageKey,
				MIMEType:    p.MIMEType,
				DisplayName: p.DisplayName,
			})
		default:
			return nil, fmt.Errorf("unknown assistant part type: %s", p.Type)
		}
	}
	return result, nil
}
