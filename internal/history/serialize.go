package history

import (
	"bytes"
	"encoding/json"
)

func (r *Repository) serializeJSONL(messages []Message) ([]byte, error) {
	var buf bytes.Buffer
	for _, msg := range messages {
		var m message
		switch v := msg.(type) {
		case *UserMessage:
			m = message{
				Role:      "user",
				UserID:    v.UserID,
				Parts:     convertUserPartsToJSON(v.Parts),
				Timestamp: v.Timestamp,
			}
		case *AssistantMessage:
			m = message{
				Role:      "assistant",
				ModelName: v.ModelName,
				Parts:     convertAssistantPartsToJSON(v.Parts),
				Timestamp: v.Timestamp,
			}
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

func convertUserPartsToJSON(parts []UserPart) []part {
	result := make([]part, 0, len(parts))
	for _, p := range parts {
		switch v := p.(type) {
		case *UserTextPart:
			result = append(result, part{
				Type: "text",
				Text: v.Text,
			})
		case *UserFileDataPart:
			result = append(result, part{
				Type:          "file_data",
				StorageKey:    v.StorageKey,
				MIMEType:      v.MIMEType,
				DisplayName:   v.DisplayName,
				VideoMetadata: v.VideoMetadata,
			})
		}
	}
	return result
}

func convertAssistantPartsToJSON(parts []AssistantPart) []part {
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
		}
	}
	return result
}
