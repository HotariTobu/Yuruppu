package bot

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed template/system_prompt.txt
var systemPromptTemplateText string
var systemPromptTemplate = template.Must(template.New("system_prompt").Parse(systemPromptTemplateText))

// BuildSystemPrompt builds a system prompt by injecting the character prompt into the template.
func BuildSystemPrompt(characterPrompt string) (string, error) {
	var buf bytes.Buffer
	data := struct {
		CharacterPrompt string
	}{
		CharacterPrompt: characterPrompt,
	}
	if err := systemPromptTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute system prompt template: %w", err)
	}
	return buf.String(), nil
}
