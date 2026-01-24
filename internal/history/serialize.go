package history

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func serializeJSONL(messages []Message) ([]byte, error) {
	var buf bytes.Buffer
	for _, msg := range messages {
		var m message
		switch v := msg.(type) {
		case *UserMessage:
			parts, err := convertUserPartsToJSON(v.Parts)
			if err != nil {
				return nil, err
			}
			m = message{
				Role:      "user",
				MessageID: v.MessageID,
				UserID:    v.UserID,
				Parts:     parts,
				Timestamp: v.Timestamp,
			}
		case *AssistantMessage:
			parts, err := convertAssistantPartsToJSON(v.Parts)
			if err != nil {
				return nil, err
			}
			m = message{
				Role:      "assistant",
				ModelName: v.ModelName,
				Parts:     parts,
				Timestamp: v.Timestamp,
			}
		default:
			return nil, fmt.Errorf("unknown message type: %T", msg)
		}
		data, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

func convertUserPartsToJSON(parts []UserPart) ([]part, error) {
	result := make([]part, 0, len(parts))
	for _, p := range parts {
		switch v := p.(type) {
		case *UserTextPart:
			result = append(result, part{
				Type: "text",
				Text: v.Text,
			})
		case *UserFileDataPart:
			filePart := part{
				Type:        "file_data",
				StorageKey:  v.StorageKey,
				MIMEType:    v.MIMEType,
				DisplayName: v.DisplayName,
			}
			if v.VideoMetadata != nil {
				filePart.VideoMetadata = &videoMetadata{
					StartOffset: v.VideoMetadata.StartOffset,
					EndOffset:   v.VideoMetadata.EndOffset,
					FPS:         v.VideoMetadata.FPS,
				}
			}
			result = append(result, filePart)
		default:
			return nil, fmt.Errorf("unknown user part type: %T", p)
		}
	}
	return result, nil
}

func convertAssistantPartsToJSON(parts []AssistantPart) ([]part, error) {
	result := make([]part, 0, len(parts))
	for _, p := range parts {
		switch v := p.(type) {
		case *AssistantTextPart:
			result = append(result, part{
				Type:             "text",
				Text:             v.Text,
				Thought:          v.Thought,
				ThoughtSignature: v.ThoughtSignature,
			})
		case *AssistantFileDataPart:
			result = append(result, part{
				Type:        "file_data",
				StorageKey:  v.StorageKey,
				MIMEType:    v.MIMEType,
				DisplayName: v.DisplayName,
			})
		default:
			return nil, fmt.Errorf("unknown assistant part type: %T", p)
		}
	}
	return result, nil
}
